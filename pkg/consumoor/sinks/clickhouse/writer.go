package clickhouse

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/ClickHouse/ch-go"
	"github.com/ClickHouse/ch-go/chpool"
	"github.com/ClickHouse/ch-go/compress"
	"github.com/ClickHouse/ch-go/proto"
	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/flattener"
	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/metadata"
	"github.com/ethpandaops/xatu/pkg/consumoor/telemetry"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/sirupsen/logrus"
)

const (
	defaultChGoRetryBaseDelay       = 100 * time.Millisecond
	defaultChGoRetryMaxDelay        = 2 * time.Second
	defaultChGoMaxConns       int32 = 8
	defaultChGoConnMaxLife          = time.Hour
	defaultChGoConnMaxIdle          = 10 * time.Minute
	defaultChGoHealthCheck          = 30 * time.Second
)

func resolvedChGoConfig(cfg ChGoConfig) ChGoConfig {
	out := cfg

	if out.RetryBaseDelay == 0 {
		out.RetryBaseDelay = defaultChGoRetryBaseDelay
	}

	if out.RetryMaxDelay == 0 {
		out.RetryMaxDelay = defaultChGoRetryMaxDelay
	}

	if out.MaxConns == 0 {
		out.MaxConns = defaultChGoMaxConns
	}

	if out.ConnMaxLifetime == 0 {
		out.ConnMaxLifetime = defaultChGoConnMaxLife
	}

	if out.ConnMaxIdleTime == 0 {
		out.ConnMaxIdleTime = defaultChGoConnMaxIdle
	}

	if out.HealthCheckPeriod == 0 {
		out.HealthCheckPeriod = defaultChGoHealthCheck
	}

	return out
}

// eventEntry holds a single event and its pre-extracted metadata for
// buffering in the table writer.
type eventEntry struct {
	event *xatu.DecoratedEvent
	meta  *metadata.CommonMetadata
}

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

type chTableWriter struct {
	log      logrus.FieldLogger
	table    string
	database string
	config   TableConfig
	metrics  *telemetry.Metrics
	writer   *ChGoWriter
	buffer   chan eventEntry
	flushReq chan chan error // coordinator sends a response channel; tableWriter flushes and replies

	newBatch func() flattener.ColumnarBatch
}

type tableWriteError struct {
	table string
	cause error
}

// inputPrepError wraps errors from batch factory initialization.
// These are always permanent — retrying won't resolve them.
type inputPrepError struct {
	cause error
}

func (e *inputPrepError) Error() string { return e.cause.Error() }
func (e *inputPrepError) Unwrap() error { return e.cause }

// DefaultErrorClassifier classifies ClickHouse write errors for source retry
// and reject handling.
type DefaultErrorClassifier struct{}

func (DefaultErrorClassifier) IsPermanent(err error) bool {
	return IsPermanentWriteError(err)
}

func (DefaultErrorClassifier) Table(err error) string {
	return WriteErrorTable(err)
}

func (e *tableWriteError) Error() string {
	return fmt.Sprintf("table %s write failed: %v", e.table, e.cause)
}

func (e *tableWriteError) Unwrap() error {
	return e.cause
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

	chgoCfg := resolvedChGoConfig(config.ChGo)
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

func (tw *chTableWriter) run(done <-chan struct{}) {
	ticker := time.NewTicker(tw.config.FlushInterval)
	defer ticker.Stop()

	events := make([]eventEntry, 0, tw.config.BatchSize)
	flushBlocked := false

	for {
		bufferCh := tw.buffer
		if flushBlocked {
			bufferCh = nil
		}

		select {
		case entry := <-bufferCh:
			tw.metrics.BufferUsage().WithLabelValues(tw.table).Sub(1)

			events = append(events, entry)

			if len(events) >= tw.config.BatchSize {
				if err := tw.flush(context.Background(), events); err != nil {
					if IsPermanentWriteError(err) {
						tw.log.WithError(err).
							WithField("events", len(events)).
							Warn("Dropping permanently invalid batch")
						events = events[:0]
						flushBlocked = false

						continue
					}

					// Preserve events for retry on next flush cycle. While
					// pending, stop draining tw.buffer so channel backpressure
					// remains bounded by BufferSize.
					flushBlocked = true

					continue
				}

				events = events[:0]
				flushBlocked = false
			}

		case <-ticker.C:
			if len(events) > 0 {
				if err := tw.flush(context.Background(), events); err != nil {
					if IsPermanentWriteError(err) {
						tw.log.WithError(err).
							WithField("events", len(events)).
							Warn("Dropping permanently invalid batch")
						events = events[:0]
						flushBlocked = false

						continue
					}

					flushBlocked = true

					continue
				}

				events = events[:0]
				flushBlocked = false
			}

		case errCh := <-tw.flushReq:
			if !flushBlocked {
				// Drain anything still in the channel into the batch.
			drainReq:
				for {
					select {
					case entry := <-tw.buffer:
						tw.metrics.BufferUsage().WithLabelValues(tw.table).Sub(1)

						events = append(events, entry)
					default:
						break drainReq
					}
				}
			}

			if len(events) > 0 {
				err := tw.flush(context.Background(), events)
				switch {
				case err == nil:
					events = events[:0]
					flushBlocked = false
				case IsPermanentWriteError(err):
					tw.log.WithError(err).
						WithField("events", len(events)).
						Warn("Dropping permanently invalid batch")
					events = events[:0]
					flushBlocked = false
				default:
					flushBlocked = true
				}
				// On error: events are PRESERVED for retry on next cycle
				errCh <- err
			} else {
				errCh <- nil
			}

		case <-done:
		drainLoop:
			for {
				select {
				case entry := <-tw.buffer:
					events = append(events, entry)
				default:
					break drainLoop
				}
			}

			if len(events) > 0 {
				_ = tw.flush(context.Background(), events)
			}

			return
		}
	}
}

func (tw *chTableWriter) flush(ctx context.Context, events []eventEntry) error {
	if len(events) == 0 {
		return nil
	}

	start := time.Now()

	if tw.newBatch == nil {
		tw.log.WithField("events", len(events)).
			Error("No columnar batch factory registered")
		tw.metrics.WriteErrors().WithLabelValues(tw.table).Add(float64(len(events)))

		return &tableWriteError{
			table: tw.table,
			cause: &inputPrepError{cause: fmt.Errorf("no columnar batch factory for %s", tw.table)},
		}
	}

	batch := tw.newBatch()

	for _, e := range events {
		if err := batch.FlattenTo(e.event, e.meta); err != nil {
			tw.log.WithError(err).
				WithField("events", len(events)).
				Error("Failed to flatten event into columnar batch")
			tw.metrics.WriteErrors().WithLabelValues(tw.table).Add(float64(len(events)))

			return &tableWriteError{
				table: tw.table,
				cause: &inputPrepError{cause: fmt.Errorf("flattening event for %s: %w", tw.table, err)},
			}
		}
	}

	rows := batch.Rows()
	if rows == 0 {
		return nil
	}

	input := batch.Input()

	insertBody, err := insertQueryWithSettings(input.Into(tw.table), tw.config.InsertSettings)
	if err != nil {
		tw.log.WithError(err).
			WithField("rows", rows).
			Error("Invalid insert settings")
		tw.metrics.WriteErrors().WithLabelValues(tw.table).Add(float64(rows))

		return &tableWriteError{
			table: tw.table,
			cause: fmt.Errorf("building insert query for %s: %w", tw.table, err),
		}
	}

	if err := tw.do(ctx, "insert_"+tw.table, &ch.Query{
		Body:  insertBody,
		Input: input,
	}, nil); err != nil {
		tw.log.WithError(err).
			WithField("rows", rows).
			Error("Failed to send ch-go batch")
		tw.metrics.WriteErrors().WithLabelValues(tw.table).Add(float64(rows))

		return &tableWriteError{
			table: tw.table,
			cause: fmt.Errorf("sending ch-go batch for %s: %w", tw.table, err),
		}
	}

	duration := time.Since(start)

	tw.metrics.RowsWritten().WithLabelValues(tw.table).Add(float64(rows))
	tw.metrics.WriteDuration().WithLabelValues(tw.table).Observe(duration.Seconds())
	tw.metrics.BatchSize().WithLabelValues(tw.table).Observe(float64(rows))

	tw.log.WithField("rows", rows).
		WithField("events", len(events)).
		WithField("duration", duration).
		Debug("Flushed ch-go batch")

	return nil
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

func (w *ChGoWriter) collectPoolMetrics() {
	ticker := time.NewTicker(w.chgoCfg.PoolMetricsInterval)
	defer ticker.Stop()

	var prevAcquireCount int64

	var prevEmptyAcquireCount int64

	var prevCanceledAcquireCount int64

	for {
		select {
		case <-w.poolMetricsDone:
			return
		case <-ticker.C:
			pool := w.getPool()
			if pool == nil {
				continue
			}

			stat := pool.Stat()

			w.metrics.ChgoPoolAcquiredResources().Set(float64(stat.AcquiredResources()))
			w.metrics.ChgoPoolIdleResources().Set(float64(stat.IdleResources()))
			w.metrics.ChgoPoolConstructingResources().Set(float64(stat.ConstructingResources()))
			w.metrics.ChgoPoolTotalResources().Set(float64(stat.TotalResources()))
			w.metrics.ChgoPoolMaxResources().Set(float64(stat.MaxResources()))
			w.metrics.ChgoPoolAcquireDuration().Set(stat.AcquireDuration().Seconds())
			w.metrics.ChgoPoolEmptyAcquireWaitTime().Set(stat.EmptyAcquireWaitTime().Seconds())

			acquireCount := stat.AcquireCount()
			if delta := acquireCount - prevAcquireCount; delta > 0 {
				w.metrics.ChgoPoolAcquireTotal().Add(float64(delta))
			}

			prevAcquireCount = acquireCount

			emptyAcquireCount := stat.EmptyAcquireCount()
			if delta := emptyAcquireCount - prevEmptyAcquireCount; delta > 0 {
				w.metrics.ChgoPoolEmptyAcquireTotal().Add(float64(delta))
			}

			prevEmptyAcquireCount = emptyAcquireCount

			canceledAcquireCount := stat.CanceledAcquireCount()
			if delta := canceledAcquireCount - prevCanceledAcquireCount; delta > 0 {
				w.metrics.ChgoPoolCanceledAcquireTotal().Add(float64(delta))
			}

			prevCanceledAcquireCount = canceledAcquireCount
		}
	}
}

func (tw *chTableWriter) do(
	ctx context.Context,
	operation string,
	query *ch.Query,
	beforeAttempt func(),
) error {
	return tw.writer.doWithRetry(ctx, operation, func(attemptCtx context.Context) error {
		if beforeAttempt != nil {
			beforeAttempt()
		}

		pool := tw.writer.getPool()
		if pool == nil {
			return ch.ErrClosed
		}

		return pool.Do(attemptCtx, *query)
	})
}

func parseChGoOptions(dsn string) (ch.Options, error) {
	if !strings.Contains(dsn, "://") {
		dsn = "clickhouse://" + dsn
	}

	u, err := url.Parse(dsn)
	if err != nil {
		return ch.Options{}, err
	}

	switch u.Scheme {
	case "http", "https":
		return ch.Options{}, fmt.Errorf("ch-go backend supports native tcp only, got DSN scheme %q", u.Scheme)
	case "clickhouse", "tcp", "clickhouses":
	default:
		return ch.Options{}, fmt.Errorf("unsupported DSN scheme %q for ch-go backend", u.Scheme)
	}

	hostPort := u.Host
	if hostPort == "" {
		return ch.Options{}, fmt.Errorf("missing host in DSN")
	}

	if strings.Contains(hostPort, ",") {
		return ch.Options{}, fmt.Errorf("multiple hosts are not supported by ch-go backend")
	}

	if _, _, splitErr := net.SplitHostPort(hostPort); splitErr != nil {
		hostPort = net.JoinHostPort(hostPort, "9000")
	}

	database := strings.TrimPrefix(u.Path, "/")
	if database == "" {
		database = "default"
	}

	username := "default"
	password := ""

	if u.User != nil {
		if user := u.User.Username(); user != "" {
			username = user
		}

		if pass, ok := u.User.Password(); ok {
			password = pass
		}
	}

	q := u.Query()
	if user := q.Get("username"); user != "" {
		username = user
	}

	if pass := q.Get("password"); pass != "" {
		password = pass
	}

	if db := q.Get("database"); db != "" {
		database = db
	}

	var tlsConfig *tls.Config
	if isTrue(q.Get("secure")) || isTrue(q.Get("tls")) || u.Scheme == "clickhouses" {
		tlsConfig = &tls.Config{
			MinVersion: tls.VersionTLS12,
		}
	}

	return ch.Options{
		Address:     hostPort,
		Database:    database,
		User:        username,
		Password:    password,
		DialTimeout: 5 * time.Second,
		ReadTimeout: 30 * time.Second,
		TLS:         tlsConfig,
	}, nil
}

// IsPermanentWriteError returns true for errors that will never succeed on
// retry: schema mismatches, type errors, conversion failures.
func IsPermanentWriteError(err error) bool {
	if err == nil {
		return false
	}

	var prepErr *inputPrepError
	if errors.As(err, &prepErr) {
		return true
	}

	if exc, ok := ch.AsException(err); ok {
		return exc.IsCode(
			proto.ErrUnknownTable,
			proto.ErrUnknownDatabase,
			proto.ErrNoSuchColumnInTable,
			proto.ErrThereIsNoColumn,
			proto.ErrUnknownIdentifier,
			proto.ErrTypeMismatch,
			proto.ErrCannotConvertType,
			proto.ErrCannotParseText,
			proto.ErrCannotParseNumber,
			proto.ErrCannotParseDate,
			proto.ErrCannotParseDatetime,
			proto.ErrCannotInsertNullInOrdinaryColumn,
			proto.ErrIncorrectData,
			proto.ErrValueIsOutOfRangeOfDataType,
			proto.ErrIncorrectNumberOfColumns,
			proto.ErrNumberOfColumnsDoesntMatch,
			proto.ErrIllegalColumn,
			proto.ErrIllegalTypeOfArgument,
			proto.ErrUnknownSetting,
			proto.ErrBadArguments,
			proto.ErrSyntaxError,
		)
	}

	return false
}

// WriteErrorTable extracts the table name from a write error.
func WriteErrorTable(err error) string {
	if err == nil {
		return ""
	}

	var tableErr *tableWriteError
	if errors.As(err, &tableErr) {
		return tableErr.table
	}

	return ""
}

func isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	// Context cancellation/deadlines are terminal for this call path.
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}

	// Closed pool/client is terminal.
	if errors.Is(err, ch.ErrClosed) {
		return false
	}

	// Classify server-side exceptions by ClickHouse code.
	if exc, ok := ch.AsException(err); ok {
		return exc.IsCode(
			proto.ErrTimeoutExceeded,
			proto.ErrNoFreeConnection,
			proto.ErrTooManySimultaneousQueries,
			proto.ErrSocketTimeout,
			proto.ErrNetworkError,
		)
	}

	// Corrupted compression payloads should not be retried.
	var corruptedErr *compress.CorruptedDataErr
	if errors.As(err, &corruptedErr) {
		return false
	}

	// Common transient transport errors.
	if errors.Is(err, syscall.ECONNRESET) ||
		errors.Is(err, syscall.ECONNREFUSED) ||
		errors.Is(err, syscall.EPIPE) ||
		errors.Is(err, io.EOF) ||
		errors.Is(err, io.ErrUnexpectedEOF) {
		return true
	}

	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}

	errStr := strings.ToLower(err.Error())
	transientPatterns := []string{
		"connection reset",
		"connection refused",
		"broken pipe",
		"eof",
		"timeout",
		"temporary failure",
		"server is overloaded",
		"too many connections",
	}

	for _, pattern := range transientPatterns {
		if strings.Contains(errStr, pattern) {
			return true
		}
	}

	return false
}

func insertQueryWithSettings(insertQuery string, settings map[string]any) (string, error) {
	if len(settings) == 0 {
		return insertQuery, nil
	}

	keys := make([]string, 0, len(settings))
	for k := range settings {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		value, err := formatInsertSettingValue(settings[key])
		if err != nil {
			return "", fmt.Errorf("setting %q: %w", key, err)
		}

		parts = append(parts, fmt.Sprintf("%s = %s", key, value))
	}

	settingsClause := " SETTINGS " + strings.Join(parts, ", ")

	upper := strings.ToUpper(insertQuery)
	valuesIdx := strings.LastIndex(upper, " VALUES")

	if valuesIdx >= 0 {
		return insertQuery[:valuesIdx] + settingsClause + insertQuery[valuesIdx:], nil
	}

	return insertQuery + settingsClause, nil
}

func formatInsertSettingValue(value any) (string, error) {
	switch v := value.(type) {
	case string:
		return quoteCHString(v), nil
	case bool:
		if v {
			return "1", nil
		}

		return "0", nil
	case int:
		return strconv.FormatInt(int64(v), 10), nil
	case int8:
		return strconv.FormatInt(int64(v), 10), nil
	case int16:
		return strconv.FormatInt(int64(v), 10), nil
	case int32:
		return strconv.FormatInt(int64(v), 10), nil
	case int64:
		return strconv.FormatInt(v, 10), nil
	case uint:
		return strconv.FormatUint(uint64(v), 10), nil
	case uint8:
		return strconv.FormatUint(uint64(v), 10), nil
	case uint16:
		return strconv.FormatUint(uint64(v), 10), nil
	case uint32:
		return strconv.FormatUint(uint64(v), 10), nil
	case uint64:
		return strconv.FormatUint(v, 10), nil
	case float32:
		return strconv.FormatFloat(float64(v), 'f', -1, 32), nil
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64), nil
	default:
		return "", fmt.Errorf("unsupported value type %T", value)
	}
}

func quoteCHString(v string) string {
	return "'" + strings.ReplaceAll(v, "'", "''") + "'"
}

func isTrue(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}
