package clickhouse

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/ClickHouse/ch-go"
	"github.com/ClickHouse/ch-go/chpool"
	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/flattener"
	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/metadata"
	"github.com/ethpandaops/xatu/pkg/consumoor/telemetry"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/sirupsen/logrus"
)

// ChGoWriter manages batched inserts using the ch-go client.
type ChGoWriter struct {
	log     logrus.FieldLogger
	config  *Config
	metrics *telemetry.Metrics

	options ch.Options
	chgoCfg ChGoConfig

	poolMu sync.RWMutex
	pool   *chpool.Pool

	database string

	tables map[string]*chTableWriter
	mu     sync.RWMutex

	// batchFactories maps base table names to ColumnarBatch constructors
	// registered by route initialization.
	batchFactories map[string]func() flattener.ColumnarBatch

	done chan struct{}
	wg   sync.WaitGroup

	poolMetricsDone chan struct{}
	poolMetricsWG   sync.WaitGroup
}

// NewChGoWriter creates a new ch-go writer.
func NewChGoWriter(
	log logrus.FieldLogger,
	config *Config,
	metrics *telemetry.Metrics,
) (*ChGoWriter, error) {
	opts, err := parseChGoOptions(config.DSN)
	if err != nil {
		return nil, fmt.Errorf("parsing clickhouse DSN for ch-go backend: %w", err)
	}

	chgoCfg := config.ChGo
	if err := chgoCfg.Validate(); err != nil {
		return nil, fmt.Errorf("validating clickhouse ch-go config: %w", err)
	}

	return &ChGoWriter{
		log:            log.WithField("component", "chgo_writer"),
		config:         config,
		metrics:        metrics,
		options:        opts,
		chgoCfg:        chgoCfg,
		database:       opts.Database,
		tables:         make(map[string]*chTableWriter, 16),
		batchFactories: make(map[string]func() flattener.ColumnarBatch, 64),
		done:           make(chan struct{}),
	}, nil
}

// RegisterBatchFactories extracts ColumnarBatch factories from routes and
// stores them keyed by base table name. This must be called before Write.
func (w *ChGoWriter) RegisterBatchFactories(routes []flattener.Route) {
	for _, route := range routes {
		batch := route.NewBatch()
		if batch == nil {
			continue
		}

		w.batchFactories[route.TableName()] = route.NewBatch
	}

	w.log.WithField("tables_with_batch", len(w.batchFactories)).
		Info("Registered columnar batch factories")
}

// Start dials the pool and verifies connectivity.
func (w *ChGoWriter) Start(ctx context.Context) error {
	if w.getPool() != nil {
		return nil
	}

	var pool *chpool.Pool

	if err := w.doWithRetry(ctx, "dial_pool", func(attemptCtx context.Context) error {
		p, err := chpool.Dial(attemptCtx, chpool.Options{
			ClientOptions:     w.options,
			MaxConns:          w.chgoCfg.MaxConns,
			MinConns:          w.chgoCfg.MinConns,
			MaxConnLifetime:   w.chgoCfg.ConnMaxLifetime,
			MaxConnIdleTime:   w.chgoCfg.ConnMaxIdleTime,
			HealthCheckPeriod: w.chgoCfg.HealthCheckPeriod,
		})
		if err != nil {
			return err
		}

		if err := p.Ping(attemptCtx); err != nil {
			p.Close()

			return err
		}

		pool = p

		return nil
	}); err != nil {
		return fmt.Errorf("opening ch-go connection pool: %w", err)
	}

	w.setPool(pool)

	if w.chgoCfg.PoolMetricsInterval > 0 {
		w.poolMetricsDone = make(chan struct{})
		w.poolMetricsWG.Go(w.collectPoolMetrics)
	}

	w.log.WithField("max_conns", w.chgoCfg.MaxConns).
		WithField("min_conns", w.chgoCfg.MinConns).
		Info("ch-go ClickHouse connection pool established")

	return nil
}

// Stop drains buffers and closes the connection pool.
func (w *ChGoWriter) Stop(_ context.Context) error {
	w.log.Info("Stopping ch-go writer, flushing remaining buffers")

	close(w.done)
	w.wg.Wait()

	if w.poolMetricsDone != nil {
		close(w.poolMetricsDone)
		w.poolMetricsWG.Wait()
	}

	if pool := w.getPool(); pool != nil {
		pool.Close()
	}

	return nil
}

// Write enqueues an event for table batching.
// Write blocks if the buffer is full, propagating backpressure to the caller.
func (w *ChGoWriter) Write(table string, event *xatu.DecoratedEvent, meta *metadata.CommonMetadata) {
	tw := w.getOrCreateTableWriter(table)

	select {
	case tw.buffer <- eventEntry{event: event, meta: meta}:
		w.metrics.BufferUsage().WithLabelValues(tw.table).Inc()
	case <-w.done:
		// Shutting down, discard.
	}
}

// FlushAll forces all table writers to drain their buffers and flush
// to ClickHouse synchronously. Returns the first error encountered;
// on failure, unflushed events are preserved in the table writers.
func (w *ChGoWriter) FlushAll(ctx context.Context) error {
	w.mu.RLock()
	writers := make([]*chTableWriter, 0, len(w.tables))

	for _, tw := range w.tables {
		writers = append(writers, tw)
	}

	w.mu.RUnlock()

	if len(writers) == 0 {
		return nil
	}

	// Send flush requests to all table writers in parallel
	errChs := make([]chan error, len(writers))

	for i, tw := range writers {
		errCh := make(chan error, 1)
		errChs[i] = errCh

		select {
		case tw.flushReq <- errCh:
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	// Collect results — return first error
	var firstErr error

	for _, errCh := range errChs {
		select {
		case err := <-errCh:
			if err != nil && firstErr == nil {
				firstErr = err
			}
		case <-ctx.Done():
			if firstErr == nil {
				firstErr = ctx.Err()
			}
		}
	}

	return firstErr
}

func (w *ChGoWriter) getOrCreateTableWriter(table string) *chTableWriter {
	writeTable := table + w.config.TableSuffix

	w.mu.RLock()
	tw, ok := w.tables[writeTable]
	w.mu.RUnlock()

	if ok {
		return tw
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	if tw, ok = w.tables[writeTable]; ok {
		return tw
	}

	// Config lookup uses the base table name so per-table overrides
	// don't need to include the suffix.
	cfg := w.config.TableConfigFor(table)
	tw = &chTableWriter{
		log:      w.log.WithField("table", writeTable),
		table:    writeTable,
		database: w.database,
		config:   cfg,
		metrics:  w.metrics,
		writer:   w,
		buffer:   make(chan eventEntry, cfg.BufferSize),
		flushReq: make(chan chan error, 1),
		newBatch: w.batchFactories[table],
	}

	w.tables[writeTable] = tw

	w.wg.Go(func() {
		tw.run(w.done)
	})

	w.log.WithField("table", writeTable).
		WithField("batch_size", cfg.BatchSize).
		WithField("flush_interval", cfg.FlushInterval).
		WithField("has_batch_factory", tw.newBatch != nil).
		Info("Created ch-go table writer")

	return tw
}

func (w *ChGoWriter) getPool() *chpool.Pool {
	w.poolMu.RLock()
	defer w.poolMu.RUnlock()

	return w.pool
}

func (w *ChGoWriter) setPool(pool *chpool.Pool) {
	w.poolMu.Lock()
	defer w.poolMu.Unlock()

	w.pool = pool
}

func (w *ChGoWriter) withQueryTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	if w.chgoCfg.QueryTimeout == 0 {
		return ctx, func() {}
	}

	if _, hasDeadline := ctx.Deadline(); hasDeadline {
		return ctx, func() {}
	}

	return context.WithTimeout(ctx, w.chgoCfg.QueryTimeout)
}

func (w *ChGoWriter) doWithRetry(
	ctx context.Context,
	operation string,
	fn func(context.Context) error,
) error {
	var lastErr error

	for attempt := 0; attempt <= w.chgoCfg.MaxRetries; attempt++ {
		if attempt > 0 {
			delay := min(
				w.chgoCfg.RetryBaseDelay*time.Duration(1<<(attempt-1)),
				w.chgoCfg.RetryMaxDelay,
			)

			w.log.WithFields(logrus.Fields{
				"operation": operation,
				"attempt":   attempt,
				"max":       w.chgoCfg.MaxRetries,
				"delay":     delay,
				"error":     lastErr,
			}).Debug("Retrying ch-go operation after transient error")

			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}
		}

		attemptCtx, cancel := w.withQueryTimeout(ctx)
		err := fn(attemptCtx)

		cancel()

		if err == nil {
			return nil
		}

		lastErr = err

		if !isRetryableError(err) {
			return err
		}
	}

	return fmt.Errorf(
		"ch-go operation %q exceeded max retries (%d): %w",
		operation,
		w.chgoCfg.MaxRetries,
		lastErr,
	)
}
