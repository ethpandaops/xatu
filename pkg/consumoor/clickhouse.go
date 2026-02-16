package consumoor

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/sirupsen/logrus"
)

// ClickHouseWriter manages batched inserts to ClickHouse tables.
// Each table gets its own buffer and flush goroutine with independently
// configurable batch sizes and flush intervals.
type ClickHouseWriter struct {
	log     logrus.FieldLogger
	config  *ClickHouseConfig
	metrics *Metrics

	conn   driver.Conn
	tables map[string]*tableWriter
	mu     sync.RWMutex

	done chan struct{}
	wg   sync.WaitGroup
}

// tableWriter manages buffered rows for a single ClickHouse table.
type tableWriter struct {
	log       logrus.FieldLogger
	table     string
	config    TableConfig
	metrics   *Metrics
	conn      driver.Conn
	buffer    chan []map[string]any
	columns   []string
	columnsMu sync.RWMutex
}

// NewClickHouseWriter creates a new ClickHouse writer. Call Start() to
// begin the flush goroutines.
func NewClickHouseWriter(
	log logrus.FieldLogger,
	config *ClickHouseConfig,
	metrics *Metrics,
) (*ClickHouseWriter, error) {
	opts, err := clickhouse.ParseDSN(config.DSN)
	if err != nil {
		return nil, fmt.Errorf("parsing clickhouse DSN: %w", err)
	}

	conn, err := clickhouse.Open(opts)
	if err != nil {
		return nil, fmt.Errorf("opening clickhouse connection: %w", err)
	}

	return &ClickHouseWriter{
		log:     log.WithField("component", "clickhouse_writer"),
		config:  config,
		metrics: metrics,
		conn:    conn,
		tables:  make(map[string]*tableWriter, 16),
		done:    make(chan struct{}),
	}, nil
}

// Start verifies the ClickHouse connection is alive.
func (w *ClickHouseWriter) Start(ctx context.Context) error {
	if err := w.conn.Ping(ctx); err != nil {
		return fmt.Errorf("clickhouse ping failed: %w", err)
	}

	w.log.Info("ClickHouse connection established")

	return nil
}

// Stop flushes all remaining buffered rows and closes the connection.
func (w *ClickHouseWriter) Stop(ctx context.Context) error {
	w.log.Info("Stopping ClickHouse writer, flushing remaining buffers")

	close(w.done)
	w.wg.Wait()

	if err := w.conn.Close(); err != nil {
		return fmt.Errorf("closing clickhouse connection: %w", err)
	}

	return nil
}

// Write buffers rows for the given table. If no writer exists for this
// table yet, one is created and its flush goroutine started.
func (w *ClickHouseWriter) Write(table string, rows []map[string]any) {
	tw := w.getOrCreateTableWriter(table)

	select {
	case tw.buffer <- rows:
		w.metrics.bufferUsage.WithLabelValues(table).Add(float64(len(rows)))
	default:
		w.log.WithField("table", table).
			WithField("rows", len(rows)).
			Warn("Buffer full, dropping rows")
		w.metrics.writeErrors.WithLabelValues(table).Add(float64(len(rows)))
	}
}

// getOrCreateTableWriter returns the writer for a table, creating it
// lazily if needed.
func (w *ClickHouseWriter) getOrCreateTableWriter(table string) *tableWriter {
	w.mu.RLock()
	tw, ok := w.tables[table]
	w.mu.RUnlock()

	if ok {
		return tw
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	// Double-check after acquiring write lock
	if tw, ok = w.tables[table]; ok {
		return tw
	}

	cfg := w.config.TableConfigFor(table)

	tw = &tableWriter{
		log:     w.log.WithField("table", table),
		table:   table,
		config:  cfg,
		metrics: w.metrics,
		conn:    w.conn,
		buffer:  make(chan []map[string]any, cfg.BufferSize),
	}

	w.tables[table] = tw

	// Start the flush goroutine for this table
	w.wg.Go(func() {
		tw.run(w.done)
	})

	w.log.WithField("table", table).
		WithField("batch_size", cfg.BatchSize).
		WithField("flush_interval", cfg.FlushInterval).
		Info("Created table writer")

	return tw
}

// run is the main flush loop for a table writer. It batches rows and
// flushes either when the batch is full or the flush interval elapses.
func (tw *tableWriter) run(done <-chan struct{}) {
	ticker := time.NewTicker(tw.config.FlushInterval)
	defer ticker.Stop()

	batch := make([]map[string]any, 0, tw.config.BatchSize)
	batchBytes := 0

	for {
		select {
		case rows := <-tw.buffer:
			tw.metrics.bufferUsage.WithLabelValues(tw.table).Sub(float64(len(rows)))

			for _, row := range rows {
				batch = append(batch, row)
				// Rough estimate: 100 bytes per field
				batchBytes += len(row) * 100
			}

			// Flush if batch size or byte limit reached
			if len(batch) >= tw.config.BatchSize || batchBytes >= tw.config.BatchBytes {
				tw.flush(context.Background(), batch)
				batch = make([]map[string]any, 0, tw.config.BatchSize)
				batchBytes = 0
			}

		case <-ticker.C:
			if len(batch) > 0 {
				tw.flush(context.Background(), batch)
				batch = make([]map[string]any, 0, tw.config.BatchSize)
				batchBytes = 0
			}

		case <-done:
			// Drain remaining buffer items
		drainLoop:
			for {
				select {
				case rows := <-tw.buffer:
					batch = append(batch, rows...)
				default:
					break drainLoop
				}
			}

			if len(batch) > 0 {
				tw.flush(context.Background(), batch)
			}

			return
		}
	}
}

// flush writes a batch of rows to ClickHouse.
func (tw *tableWriter) flush(ctx context.Context, rows []map[string]any) {
	if len(rows) == 0 {
		return
	}

	start := time.Now()

	// Determine column order from the first row (cached after first call)
	columns := tw.getColumns(rows[0])

	// Build INSERT statement
	query := fmt.Sprintf(
		"INSERT INTO %s (%s)",
		tw.table,
		strings.Join(columns, ", "),
	)

	batch, err := tw.conn.PrepareBatch(ctx, query)
	if err != nil {
		tw.log.WithError(err).
			WithField("rows", len(rows)).
			Error("Failed to prepare batch")
		tw.metrics.writeErrors.WithLabelValues(tw.table).Add(float64(len(rows)))

		return
	}

	for _, row := range rows {
		vals := make([]any, 0, len(columns))
		for _, col := range columns {
			vals = append(vals, row[col])
		}

		if err := batch.Append(vals...); err != nil {
			tw.log.WithError(err).
				WithField("rows", len(rows)).
				Error("Failed to append row to batch")
			tw.metrics.writeErrors.WithLabelValues(tw.table).Inc()

			continue
		}
	}

	if err := batch.Send(); err != nil {
		tw.log.WithError(err).
			WithField("rows", len(rows)).
			Error("Failed to send batch")
		tw.metrics.writeErrors.WithLabelValues(tw.table).Add(float64(len(rows)))

		return
	}

	duration := time.Since(start)

	tw.metrics.rowsWritten.WithLabelValues(tw.table).Add(float64(len(rows)))
	tw.metrics.writeDuration.WithLabelValues(tw.table).Observe(duration.Seconds())
	tw.metrics.batchSize.WithLabelValues(tw.table).Observe(float64(len(rows)))

	tw.log.WithField("rows", len(rows)).
		WithField("duration", duration).
		Debug("Flushed batch")
}

// getColumns returns a stable, sorted list of column names from a row.
// The column list is cached after the first call since all rows for a
// given table should have the same columns.
func (tw *tableWriter) getColumns(row map[string]any) []string {
	tw.columnsMu.RLock()

	if tw.columns != nil {
		cols := tw.columns
		tw.columnsMu.RUnlock()

		return cols
	}

	tw.columnsMu.RUnlock()

	tw.columnsMu.Lock()
	defer tw.columnsMu.Unlock()

	// Double-check
	if tw.columns != nil {
		return tw.columns
	}

	cols := make([]string, 0, len(row))
	for k := range row {
		cols = append(cols, k)
	}

	sort.Strings(cols)

	tw.columns = cols

	return cols
}
