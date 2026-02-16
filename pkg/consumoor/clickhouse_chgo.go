//nolint:wsl_v5 // Conversion-heavy helper code intentionally stays compact.
package consumoor

import (
	"context"
	"crypto/tls"
	"fmt"
	"math"
	"net"
	"net/url"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ClickHouse/ch-go"
	"github.com/ClickHouse/ch-go/proto"
	"github.com/sirupsen/logrus"
)

var timeType = reflect.TypeOf(time.Time{})

// ChGoWriter manages batched inserts using the ch-go client.
type ChGoWriter struct {
	log      logrus.FieldLogger
	config   *ClickHouseConfig
	metrics  *Metrics
	client   *ch.Client
	clientMu sync.Mutex
	database string

	tables map[string]*chTableWriter
	mu     sync.RWMutex

	done chan struct{}
	wg   sync.WaitGroup
}

type chTableWriter struct {
	log      logrus.FieldLogger
	table    string
	database string
	config   TableConfig
	metrics  *Metrics
	client   *ch.Client
	clientMu *sync.Mutex
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

	client, err := ch.Dial(context.Background(), opts)
	if err != nil {
		return nil, fmt.Errorf("opening ch-go connection: %w", err)
	}

	return &ChGoWriter{
		log:      log.WithField("component", "chgo_writer"),
		config:   config,
		metrics:  metrics,
		client:   client,
		database: opts.Database,
		tables:   make(map[string]*chTableWriter, 16),
		done:     make(chan struct{}),
	}, nil
}

// Start verifies connectivity.
func (w *ChGoWriter) Start(ctx context.Context) error {
	if err := w.client.Ping(ctx); err != nil {
		return fmt.Errorf("ch-go ping failed: %w", err)
	}

	w.log.Info("ch-go ClickHouse connection established")

	return nil
}

// Stop drains buffers and closes connection.
func (w *ChGoWriter) Stop(_ context.Context) error {
	w.log.Info("Stopping ch-go writer, flushing remaining buffers")

	close(w.done)
	w.wg.Wait()

	if err := w.client.Close(); err != nil {
		return fmt.Errorf("closing ch-go connection: %w", err)
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
		client:   w.client,
		clientMu: &w.clientMu,
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

	if err := tw.do(ctx, &ch.Query{
		Body:  tw.input.Into(tw.table),
		Input: tw.input,
	}); err != nil {
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

	if err := tw.do(ctx, &ch.Query{
		Body: query,
		Result: proto.Results{
			{Name: "name", Data: &names},
			{Name: "type", Data: &types},
		},
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

func (tw *chTableWriter) do(ctx context.Context, query *ch.Query) error {
	tw.clientMu.Lock()
	defer tw.clientMu.Unlock()

	return tw.client.Do(ctx, *query)
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
