package clickhouse

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/ClickHouse/ch-go"
	"github.com/ClickHouse/ch-go/chpool"
	"github.com/ethpandaops/xatu/pkg/consumoor/route"
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
	batchFactories map[string]func() route.ColumnarBatch

	stopOnce sync.Once

	poolMetricsDone chan struct{}
	poolMetricsWG   sync.WaitGroup

	// poolDoFn, when non-nil, replaces pool.Do in chTableWriter.do().
	// Used by benchmarks to inject a noop ClickHouse sink.
	poolDoFn func(ctx context.Context, query ch.Query) error
}

// NewChGoWriter creates a new ch-go writer.
func NewChGoWriter(
	log logrus.FieldLogger,
	config *Config,
	metrics *telemetry.Metrics,
) (*ChGoWriter, error) {
	opts, err := parseChGoOptions(
		config.DSN,
		config.ChGo.DialTimeout,
		config.ChGo.ReadTimeout,
		&config.TLS,
	)
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
		batchFactories: make(map[string]func() route.ColumnarBatch, 64),
	}, nil
}

// RegisterBatchFactories extracts ColumnarBatch factories from routes and
// stores them keyed by base table name. This must be called before Write.
func (w *ChGoWriter) RegisterBatchFactories(routes []route.Route) {
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

	// Validate that all registered route tables exist in the target
	// database. This catches missing migrations early before INSERTs
	// silently fail with ErrUnknownTable (classified as permanent).
	routeTableNames := make([]string, 0, len(w.batchFactories))
	for name := range w.batchFactories {
		routeTableNames = append(routeTableNames, name)
	}

	if err := w.ValidateTables(ctx, routeTableNames); err != nil {
		return fmt.Errorf("validating clickhouse tables: %w", err)
	}

	return nil
}

// Ping checks connectivity to the ClickHouse pool.
func (w *ChGoWriter) Ping(ctx context.Context) error {
	pool := w.getPool()
	if pool == nil {
		return fmt.Errorf("clickhouse pool not initialized")
	}

	return pool.Ping(ctx)
}

// Stop closes the connection pool. It is safe to call multiple times;
// only the first call performs cleanup.
func (w *ChGoWriter) Stop(_ context.Context) error {
	w.stopOnce.Do(func() {
		w.log.Info("Stopping ch-go writer")

		if w.poolMetricsDone != nil {
			close(w.poolMetricsDone)
			w.poolMetricsWG.Wait()
		}

		if pool := w.getPool(); pool != nil {
			pool.Close()
		}
	})

	return nil
}

// FlushTableEvents writes the given events directly to their respective
// ClickHouse tables concurrently. The map keys are base table names
// (without suffix). Returns a joined error containing all table failures.
func (w *ChGoWriter) FlushTableEvents(
	ctx context.Context,
	tableEvents map[string][]*xatu.DecoratedEvent,
) error {
	if len(tableEvents) == 0 {
		return nil
	}

	type tableFlush struct {
		tw     *chTableWriter
		events []*xatu.DecoratedEvent
	}

	flushes := make([]tableFlush, 0, len(tableEvents))

	for base, events := range tableEvents {
		if len(events) == 0 {
			continue
		}

		tw := w.getOrCreateTableWriter(base)
		flushes = append(flushes, tableFlush{tw: tw, events: events})
	}

	if len(flushes) == 0 {
		return nil
	}

	errs := make([]error, len(flushes))

	var wg sync.WaitGroup

	wg.Add(len(flushes))

	for i, f := range flushes {
		go func(idx int, f tableFlush) {
			defer wg.Done()

			errs[idx] = f.tw.flush(ctx, f.events)
		}(i, f)
	}

	wg.Wait()

	return errors.Join(errs...)
}

func (w *ChGoWriter) getOrCreateTableWriter(table string) *chTableWriter {
	var writeTable string
	if w.config.TableSuffix == "" {
		writeTable = table
	} else {
		writeTable = table + w.config.TableSuffix
	}

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
		log:        w.log.WithField("table", writeTable),
		table:      writeTable,
		baseTable:  table,
		database:   w.database,
		config:     cfg,
		metrics:    w.metrics,
		writer:     w,
		newBatch:   w.batchFactories[table],
		limiter:    newAdaptiveConcurrencyLimiter(w.chgoCfg.AdaptiveLimiter),
		logSampler: telemetry.NewLogSampler(30 * time.Second),
	}

	w.tables[writeTable] = tw

	w.log.WithField("table", writeTable).
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
			w.metrics.WriteRetries().WithLabelValues(operation).Inc()

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

		if IsLimiterRejected(err) {
			return err
		}

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
