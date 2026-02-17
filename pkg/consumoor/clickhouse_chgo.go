//nolint:wsl_v5 // Conversion-heavy helper code intentionally stays compact.
package consumoor

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"math"
	"net"
	"net/url"
	"reflect"
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
	"github.com/sirupsen/logrus"
)

var timeType = reflect.TypeOf(time.Time{})

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

// ChGoWriter manages batched inserts using the ch-go client.
type ChGoWriter struct {
	log     logrus.FieldLogger
	config  *ClickHouseConfig
	metrics *Metrics

	options ch.Options
	chgoCfg ChGoConfig

	poolMu sync.RWMutex
	pool   *chpool.Pool

	database string

	tables map[string]*chTableWriter
	mu     sync.RWMutex

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
	metrics  *Metrics
	writer   *ChGoWriter
	buffer   chan []map[string]any
	flushReq chan chan error // coordinator sends a response channel; tableWriter flushes and replies

	columnsMu sync.RWMutex
	columns   []string
	input     proto.Input
	appenders []columnAppender
}

type columnAppender struct {
	name   string
	append reflect.Value
	arg    reflect.Type
}

// NewChGoWriter creates a new ch-go writer.
func NewChGoWriter(
	log logrus.FieldLogger,
	config *ClickHouseConfig,
	metrics *Metrics,
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
		log:      log.WithField("component", "chgo_writer"),
		config:   config,
		metrics:  metrics,
		options:  opts,
		chgoCfg:  chgoCfg,
		database: opts.Database,
		tables:   make(map[string]*chTableWriter, 16),
		done:     make(chan struct{}),
	}, nil
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

// Write enqueues rows for table batching.
// Write blocks if the buffer is full, propagating backpressure to the caller.
func (w *ChGoWriter) Write(table string, rows []map[string]any) {
	tw := w.getOrCreateTableWriter(table)

	select {
	case tw.buffer <- rows:
		w.metrics.bufferUsage.WithLabelValues(table).Add(float64(len(rows)))
	case <-w.done:
		// Shutting down, discard.
	}
}

func (w *ChGoWriter) getOrCreateTableWriter(table string) *chTableWriter {
	w.mu.RLock()
	tw, ok := w.tables[table]
	w.mu.RUnlock()

	if ok {
		return tw
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	if tw, ok = w.tables[table]; ok {
		return tw
	}

	cfg := w.config.TableConfigFor(table)
	tw = &chTableWriter{
		log:      w.log.WithField("table", table),
		table:    table,
		database: w.database,
		config:   cfg,
		metrics:  w.metrics,
		writer:   w,
		buffer:   make(chan []map[string]any, cfg.BufferSize),
		flushReq: make(chan chan error, 1),
	}

	w.tables[table] = tw

	w.wg.Go(func() {
		tw.run(w.done)
	})

	w.log.WithField("table", table).
		WithField("batch_size", cfg.BatchSize).
		WithField("flush_interval", cfg.FlushInterval).
		Info("Created ch-go table writer")

	return tw
}

func (tw *chTableWriter) run(done <-chan struct{}) {
	ticker := time.NewTicker(tw.config.FlushInterval)
	defer ticker.Stop()

	batch := make([]map[string]any, 0, tw.config.BatchSize)
	batchBytes := 0
	flushBlocked := false

	for {
		bufferCh := tw.buffer
		if flushBlocked {
			bufferCh = nil
		}

		select {
		case rows := <-bufferCh:
			tw.metrics.bufferUsage.WithLabelValues(tw.table).Sub(float64(len(rows)))

			for _, row := range rows {
				batch = append(batch, row)
				batchBytes += len(row) * 100
			}

			if len(batch) >= tw.config.BatchSize || batchBytes >= tw.config.BatchBytes {
				if err := tw.flush(context.Background(), batch); err != nil {
					// Preserve batch for retry on next flush cycle. While
					// pending, stop draining tw.buffer so channel backpressure
					// remains bounded by BufferSize.
					flushBlocked = true

					continue
				}

				batch = make([]map[string]any, 0, tw.config.BatchSize)
				batchBytes = 0
				flushBlocked = false
			}

		case <-ticker.C:
			if len(batch) > 0 {
				if err := tw.flush(context.Background(), batch); err != nil {
					flushBlocked = true

					continue
				}

				batch = make([]map[string]any, 0, tw.config.BatchSize)
				batchBytes = 0
				flushBlocked = false
			}

		case errCh := <-tw.flushReq:
			if !flushBlocked {
				// Drain anything still in the channel into the batch.
			drainReq:
				for {
					select {
					case rows := <-tw.buffer:
						tw.metrics.bufferUsage.WithLabelValues(tw.table).Sub(float64(len(rows)))

						for _, row := range rows {
							batch = append(batch, row)
							batchBytes += len(row) * 100
						}
					default:
						break drainReq
					}
				}
			}

			if len(batch) > 0 {
				err := tw.flush(context.Background(), batch)
				if err == nil {
					batch = make([]map[string]any, 0, tw.config.BatchSize)
					batchBytes = 0
					flushBlocked = false
				} else {
					flushBlocked = true
				}
				// On error: batch is PRESERVED for retry on next cycle
				errCh <- err
			} else {
				errCh <- nil
			}

		case <-done:
		drainLoop:
			for {
				select {
				case rows := <-tw.buffer:
					for _, row := range rows {
						batch = append(batch, row)
						batchBytes += len(row) * 100
					}
				default:
					break drainLoop
				}
			}

			if len(batch) > 0 {
				_ = tw.flush(context.Background(), batch)
			}

			return
		}
	}
}

func (tw *chTableWriter) flush(ctx context.Context, rows []map[string]any) error {
	if len(rows) == 0 {
		return nil
	}

	start := time.Now()
	columns := sortedColumns(rows)

	if err := tw.ensureInput(ctx, columns); err != nil {
		tw.log.WithError(err).
			WithField("rows", len(rows)).
			Error("Failed to prepare ch-go input")
		tw.metrics.writeErrors.WithLabelValues(tw.table).Add(float64(len(rows)))

		return fmt.Errorf("preparing ch-go input for %s: %w", tw.table, err)
	}

	tw.input.Reset()

	skipped := 0
	firstErr := error(nil)
	converted := make([]reflect.Value, len(tw.appenders))
	wrote := 0

	for _, row := range rows {
		rowValid := true

		for i := range tw.appenders {
			value, ok := row[tw.appenders[i].name]
			if !ok {
				value = nil
			}

			cv, err := convertForType(value, tw.appenders[i].arg, tw.appenders[i].name)
			if err != nil {
				if firstErr == nil {
					firstErr = err
				}

				rowValid = false
				skipped++

				break
			}

			converted[i] = cv
		}

		if !rowValid {
			continue
		}

		for i := range tw.appenders {
			tw.appenders[i].append.Call([]reflect.Value{converted[i]})
		}

		wrote++
	}

	if skipped > 0 {
		tw.log.WithError(firstErr).
			WithField("rows_skipped", skipped).
			Warn("Skipping rows that could not be converted for ch-go input")
		tw.metrics.writeErrors.WithLabelValues(tw.table).Add(float64(skipped))
	}

	if wrote == 0 {
		return nil
	}

	insertBody, err := insertQueryWithSettings(tw.input.Into(tw.table), tw.config.InsertSettings)
	if err != nil {
		tw.log.WithError(err).
			WithField("rows", wrote).
			Error("Invalid insert settings")
		tw.metrics.writeErrors.WithLabelValues(tw.table).Add(float64(wrote))

		return fmt.Errorf("building insert query for %s: %w", tw.table, err)
	}

	if err := tw.do(ctx, "insert_"+tw.table, &ch.Query{
		Body:  insertBody,
		Input: tw.input,
	}, nil); err != nil {
		tw.log.WithError(err).
			WithField("rows", wrote).
			Error("Failed to send ch-go batch")
		tw.metrics.writeErrors.WithLabelValues(tw.table).Add(float64(wrote))

		return fmt.Errorf("sending ch-go batch for %s: %w", tw.table, err)
	}

	duration := time.Since(start)

	tw.metrics.rowsWritten.WithLabelValues(tw.table).Add(float64(wrote))
	tw.metrics.writeDuration.WithLabelValues(tw.table).Observe(duration.Seconds())
	tw.metrics.batchSize.WithLabelValues(tw.table).Observe(float64(wrote))

	tw.log.WithField("rows", wrote).
		WithField("duration", duration).
		Debug("Flushed ch-go batch")

	return nil
}

// FlushAll forces all table writers to drain their buffers and flush
// to ClickHouse synchronously. Returns the first error encountered;
// on failure, unflushed rows are preserved in the table writers.
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
			delay := w.chgoCfg.RetryBaseDelay * time.Duration(1<<(attempt-1))
			if delay > w.chgoCfg.RetryMaxDelay {
				delay = w.chgoCfg.RetryMaxDelay
			}

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

			w.metrics.chgoPoolAcquiredResources.Set(float64(stat.AcquiredResources()))
			w.metrics.chgoPoolIdleResources.Set(float64(stat.IdleResources()))
			w.metrics.chgoPoolConstructingResources.Set(float64(stat.ConstructingResources()))
			w.metrics.chgoPoolTotalResources.Set(float64(stat.TotalResources()))
			w.metrics.chgoPoolMaxResources.Set(float64(stat.MaxResources()))
			w.metrics.chgoPoolAcquireDuration.Set(stat.AcquireDuration().Seconds())
			w.metrics.chgoPoolEmptyAcquireWaitTime.Set(stat.EmptyAcquireWaitTime().Seconds())

			acquireCount := stat.AcquireCount()
			if delta := acquireCount - prevAcquireCount; delta > 0 {
				w.metrics.chgoPoolAcquireTotal.Add(float64(delta))
			}
			prevAcquireCount = acquireCount

			emptyAcquireCount := stat.EmptyAcquireCount()
			if delta := emptyAcquireCount - prevEmptyAcquireCount; delta > 0 {
				w.metrics.chgoPoolEmptyAcquireTotal.Add(float64(delta))
			}
			prevEmptyAcquireCount = emptyAcquireCount

			canceledAcquireCount := stat.CanceledAcquireCount()
			if delta := canceledAcquireCount - prevCanceledAcquireCount; delta > 0 {
				w.metrics.chgoPoolCanceledAcquireTotal.Add(float64(delta))
			}
			prevCanceledAcquireCount = canceledAcquireCount
		}
	}
}

func (tw *chTableWriter) ensureInput(ctx context.Context, columns []string) error {
	tw.columnsMu.RLock()
	if stringSlicesEqual(tw.columns, columns) && len(tw.input) == len(columns) {
		tw.columnsMu.RUnlock()

		return nil
	}
	tw.columnsMu.RUnlock()

	tw.columnsMu.Lock()
	defer tw.columnsMu.Unlock()

	if stringSlicesEqual(tw.columns, columns) && len(tw.input) == len(columns) {
		return nil
	}

	types, err := tw.loadColumnTypes(ctx, columns)
	if err != nil {
		return err
	}

	input := make(proto.Input, 0, len(columns))
	appenders := make([]columnAppender, 0, len(columns))

	for _, col := range columns {
		typ, ok := types[col]
		if !ok {
			return fmt.Errorf("missing clickhouse type for column %q", col)
		}

		auto := &proto.ColAuto{}
		if err := auto.Infer(typ); err != nil {
			return fmt.Errorf("infer column %q type %q: %w", col, typ, err)
		}

		app, err := newColumnAppender(col, auto.Data)
		if err != nil {
			return fmt.Errorf("create appender for column %q: %w", col, err)
		}

		input = append(input, proto.InputColumn{
			Name: col,
			Data: auto,
		})
		appenders = append(appenders, app)
	}

	tw.columns = append([]string(nil), columns...)
	tw.input = input
	tw.appenders = appenders

	return nil
}

func (tw *chTableWriter) loadColumnTypes(ctx context.Context, columns []string) (map[string]proto.ColumnType, error) {
	nameList := make([]string, 0, len(columns))
	for _, col := range columns {
		nameList = append(nameList, quoteCHString(col))
	}

	query := fmt.Sprintf(
		"SELECT name, type FROM system.columns WHERE database = %s AND table = %s AND name IN (%s)",
		quoteCHString(tw.database),
		quoteCHString(tw.table),
		strings.Join(nameList, ", "),
	)

	var names proto.ColStr
	var types proto.ColStr

	if err := tw.do(ctx, "load_column_types_"+tw.table, &ch.Query{
		Body: query,
		Result: proto.Results{
			{Name: "name", Data: &names},
			{Name: "type", Data: &types},
		},
	}, func() {
		names.Reset()
		types.Reset()
	}); err != nil {
		return nil, fmt.Errorf("querying system.columns: %w", err)
	}

	if names.Rows() != types.Rows() {
		return nil, fmt.Errorf(
			"unexpected system.columns result mismatch: names=%d types=%d",
			names.Rows(),
			types.Rows(),
		)
	}

	out := make(map[string]proto.ColumnType, names.Rows())
	for i := 0; i < names.Rows(); i++ {
		out[names.Row(i)] = proto.ColumnType(types.Row(i))
	}

	return out, nil
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

func newColumnAppender(name string, data any) (columnAppender, error) {
	rv := reflect.ValueOf(data)
	appendMethod := rv.MethodByName("Append")
	if !appendMethod.IsValid() {
		return columnAppender{}, fmt.Errorf("column type %T does not expose Append", data)
	}

	t := appendMethod.Type()
	if t.NumIn() != 1 || t.NumOut() != 0 {
		return columnAppender{}, fmt.Errorf("unexpected Append signature for %T", data)
	}

	return columnAppender{
		name:   name,
		append: appendMethod,
		arg:    t.In(0),
	}, nil
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

func sortedColumns(rows []map[string]any) []string {
	seen := make(map[string]struct{})
	for _, row := range rows {
		for col := range row {
			seen[col] = struct{}{}
		}
	}

	cols := make([]string, 0, len(seen))
	for col := range seen {
		cols = append(cols, col)
	}

	sort.Strings(cols)

	return cols
}

func stringSlicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}

	return true
}

func isTrue(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func convertForType(value any, target reflect.Type, columnName string) (reflect.Value, error) {
	if target == nil {
		return reflect.Value{}, fmt.Errorf("nil target type")
	}

	if value == nil {
		return reflect.Zero(target), nil
	}

	rv := reflect.ValueOf(value)
	if rv.IsValid() && rv.Type().AssignableTo(target) {
		return rv, nil
	}
	if rv.IsValid() && rv.Type().ConvertibleTo(target) {
		return rv.Convert(target), nil
	}

	if target == timeType {
		t, err := toTime(value, columnName)
		if err != nil {
			return reflect.Value{}, err
		}

		return reflect.ValueOf(t), nil
	}

	if target.Kind() == reflect.Struct &&
		target.PkgPath() == "github.com/ClickHouse/ch-go/proto" &&
		strings.HasPrefix(target.String(), "proto.Nullable[") {
		return toNullableValue(value, target, columnName)
	}

	switch target.Kind() {
	case reflect.String:
		return reflect.ValueOf(fmt.Sprint(value)).Convert(target), nil
	case reflect.Bool:
		v, err := toBool(value)
		if err != nil {
			return reflect.Value{}, err
		}

		out := reflect.New(target).Elem()
		out.SetBool(v)

		return out, nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		v, err := toInt64(value)
		if err != nil {
			return reflect.Value{}, err
		}

		out := reflect.New(target).Elem()
		if out.OverflowInt(v) {
			return reflect.Value{}, fmt.Errorf("value %d overflows %s", v, target)
		}
		out.SetInt(v)

		return out, nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		v, err := toUint64(value)
		if err != nil {
			return reflect.Value{}, err
		}

		out := reflect.New(target).Elem()
		if out.OverflowUint(v) {
			return reflect.Value{}, fmt.Errorf("value %d overflows %s", v, target)
		}
		out.SetUint(v)

		return out, nil
	case reflect.Float32, reflect.Float64:
		v, err := toFloat64(value)
		if err != nil {
			return reflect.Value{}, err
		}

		out := reflect.New(target).Elem()
		if out.OverflowFloat(v) {
			return reflect.Value{}, fmt.Errorf("value %f overflows %s", v, target)
		}
		out.SetFloat(v)

		return out, nil
	case reflect.Slice:
		return convertSlice(value, target, columnName)
	case reflect.Map:
		return convertMap(value, target, columnName)
	default:
		return reflect.Value{}, fmt.Errorf("unsupported conversion from %T to %s", value, target)
	}
}

func toNullableValue(value any, target reflect.Type, columnName string) (reflect.Value, error) {
	out := reflect.New(target).Elem()

	valueField := out.FieldByName("Value")
	validField := out.FieldByName("Valid")
	if !valueField.IsValid() || !validField.IsValid() {
		return reflect.Value{}, fmt.Errorf("invalid nullable type %s", target)
	}

	cv, err := convertForType(value, valueField.Type(), columnName)
	if err != nil {
		return reflect.Value{}, err
	}

	valueField.Set(cv)
	validField.SetBool(true)

	return out, nil
}

func convertSlice(value any, target reflect.Type, columnName string) (reflect.Value, error) {
	if value == nil {
		return reflect.Zero(target), nil
	}

	rv := reflect.ValueOf(value)
	if rv.Type().AssignableTo(target) {
		return rv, nil
	}
	if rv.Type().ConvertibleTo(target) {
		return rv.Convert(target), nil
	}
	if rv.Kind() != reflect.Slice && rv.Kind() != reflect.Array {
		return reflect.Value{}, fmt.Errorf("expected slice for %s, got %T", target, value)
	}

	out := reflect.MakeSlice(target, rv.Len(), rv.Len())
	for i := 0; i < rv.Len(); i++ {
		cv, err := convertForType(rv.Index(i).Interface(), target.Elem(), columnName)
		if err != nil {
			return reflect.Value{}, fmt.Errorf("slice index %d: %w", i, err)
		}
		out.Index(i).Set(cv)
	}

	return out, nil
}

func convertMap(value any, target reflect.Type, columnName string) (reflect.Value, error) {
	if value == nil {
		return reflect.Zero(target), nil
	}

	rv := reflect.ValueOf(value)
	if rv.Type().AssignableTo(target) {
		return rv, nil
	}
	if rv.Type().ConvertibleTo(target) {
		return rv.Convert(target), nil
	}
	if rv.Kind() != reflect.Map {
		return reflect.Value{}, fmt.Errorf("expected map for %s, got %T", target, value)
	}

	out := reflect.MakeMapWithSize(target, rv.Len())
	iter := rv.MapRange()
	for iter.Next() {
		kv, err := convertForType(iter.Key().Interface(), target.Key(), columnName)
		if err != nil {
			return reflect.Value{}, fmt.Errorf("map key: %w", err)
		}
		vv, err := convertForType(iter.Value().Interface(), target.Elem(), columnName)
		if err != nil {
			return reflect.Value{}, fmt.Errorf("map value: %w", err)
		}

		out.SetMapIndex(kv, vv)
	}

	return out, nil
}

func toBool(value any) (bool, error) {
	switch v := value.(type) {
	case bool:
		return v, nil
	case string:
		parsed, err := strconv.ParseBool(strings.TrimSpace(v))
		if err != nil {
			return false, fmt.Errorf("parse bool %q: %w", v, err)
		}

		return parsed, nil
	case int, int8, int16, int32, int64:
		n, err := toInt64(v)
		if err != nil {
			return false, err
		}

		return n != 0, nil
	case uint, uint8, uint16, uint32, uint64:
		n, err := toUint64(v)
		if err != nil {
			return false, err
		}

		return n != 0, nil
	default:
		return false, fmt.Errorf("cannot convert %T to bool", value)
	}
}

func toInt64(value any) (int64, error) {
	switch v := value.(type) {
	case int:
		return int64(v), nil
	case int8:
		return int64(v), nil
	case int16:
		return int64(v), nil
	case int32:
		return int64(v), nil
	case int64:
		return v, nil
	case uint:
		if uint64(v) > math.MaxInt64 {
			return 0, fmt.Errorf("uint overflow to int64: %d", v)
		}

		//nolint:gosec // guarded by explicit overflow check above.
		return int64(v), nil
	case uint8:
		return int64(v), nil
	case uint16:
		return int64(v), nil
	case uint32:
		return int64(v), nil
	case uint64:
		if v > math.MaxInt64 {
			return 0, fmt.Errorf("uint64 overflow to int64: %d", v)
		}

		return int64(v), nil
	case float32:
		return int64(v), nil
	case float64:
		return int64(v), nil
	case string:
		parsed, err := strconv.ParseInt(strings.TrimSpace(v), 10, 64)
		if err != nil {
			return 0, fmt.Errorf("parse int64 %q: %w", v, err)
		}

		return parsed, nil
	default:
		return 0, fmt.Errorf("cannot convert %T to int64", value)
	}
}

func toUint64(value any) (uint64, error) {
	switch v := value.(type) {
	case int:
		if v < 0 {
			return 0, fmt.Errorf("negative int %d to uint64", v)
		}

		return uint64(v), nil
	case int8:
		if v < 0 {
			return 0, fmt.Errorf("negative int8 %d to uint64", v)
		}

		return uint64(v), nil
	case int16:
		if v < 0 {
			return 0, fmt.Errorf("negative int16 %d to uint64", v)
		}

		return uint64(v), nil
	case int32:
		if v < 0 {
			return 0, fmt.Errorf("negative int32 %d to uint64", v)
		}

		return uint64(v), nil
	case int64:
		if v < 0 {
			return 0, fmt.Errorf("negative int64 %d to uint64", v)
		}

		return uint64(v), nil
	case uint:
		return uint64(v), nil
	case uint8:
		return uint64(v), nil
	case uint16:
		return uint64(v), nil
	case uint32:
		return uint64(v), nil
	case uint64:
		return v, nil
	case float32:
		if v < 0 {
			return 0, fmt.Errorf("negative float32 %f to uint64", v)
		}

		return uint64(v), nil
	case float64:
		if v < 0 {
			return 0, fmt.Errorf("negative float64 %f to uint64", v)
		}

		return uint64(v), nil
	case string:
		parsed, err := strconv.ParseUint(strings.TrimSpace(v), 10, 64)
		if err != nil {
			return 0, fmt.Errorf("parse uint64 %q: %w", v, err)
		}

		return parsed, nil
	default:
		return 0, fmt.Errorf("cannot convert %T to uint64", value)
	}
}

func toFloat64(value any) (float64, error) {
	switch v := value.(type) {
	case float32:
		return float64(v), nil
	case float64:
		return v, nil
	case int:
		return float64(v), nil
	case int8:
		return float64(v), nil
	case int16:
		return float64(v), nil
	case int32:
		return float64(v), nil
	case int64:
		return float64(v), nil
	case uint:
		return float64(v), nil
	case uint8:
		return float64(v), nil
	case uint16:
		return float64(v), nil
	case uint32:
		return float64(v), nil
	case uint64:
		return float64(v), nil
	case string:
		parsed, err := strconv.ParseFloat(strings.TrimSpace(v), 64)
		if err != nil {
			return 0, fmt.Errorf("parse float64 %q: %w", v, err)
		}

		return parsed, nil
	default:
		return 0, fmt.Errorf("cannot convert %T to float64", value)
	}
}

func toTime(value any, columnName string) (time.Time, error) {
	switch v := value.(type) {
	case time.Time:
		return v.UTC(), nil
	case int64:
		return unixNumericTime(v, columnName), nil
	case int32:
		return unixNumericTime(int64(v), columnName), nil
	case int:
		return unixNumericTime(int64(v), columnName), nil
	case uint64:
		if v > math.MaxInt64 {
			return time.Time{}, fmt.Errorf("uint64 overflow to int64 for time: %d", v)
		}

		return unixNumericTime(int64(v), columnName), nil
	case float64:
		return unixNumericTime(int64(v), columnName), nil
	case string:
		trimmed := strings.TrimSpace(v)
		if trimmed == "" {
			return time.Time{}, fmt.Errorf("empty time string")
		}

		if parsed, err := time.Parse(time.RFC3339Nano, trimmed); err == nil {
			return parsed.UTC(), nil
		}

		if unixMillis, err := strconv.ParseInt(trimmed, 10, 64); err == nil {
			return unixNumericTime(unixMillis, columnName), nil
		}

		return time.Time{}, fmt.Errorf("cannot parse time %q", v)
	default:
		return time.Time{}, fmt.Errorf("cannot convert %T to time.Time", value)
	}
}

func unixNumericTime(value int64, columnName string) time.Time {
	// event_date_time is emitted as Unix milliseconds; other *_date_time
	// fields are emitted as Unix seconds.
	if columnName == "event_date_time" {
		return time.UnixMilli(value).UTC()
	}

	return time.Unix(value, 0).UTC()
}
