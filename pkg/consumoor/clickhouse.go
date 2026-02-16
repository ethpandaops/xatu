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
	log      logrus.FieldLogger
	table    string
	config   TableConfig
	metrics  *Metrics
	conn     driver.Conn
	buffer   chan []map[string]any
	flushReq chan chan error // coordinator sends a response channel; tableWriter flushes and replies
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
// Write blocks if the buffer is full, propagating backpressure to the caller.
func (w *ClickHouseWriter) Write(table string, rows []map[string]any) {
	tw := w.getOrCreateTableWriter(table)

	select {
	case tw.buffer <- rows:
		w.metrics.bufferUsage.WithLabelValues(table).Add(float64(len(rows)))
	case <-w.done:
		// Shutting down, discard.
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
		log:      w.log.WithField("table", table),
		table:    table,
		config:   cfg,
		metrics:  w.metrics,
		conn:     w.conn,
		buffer:   make(chan []map[string]any, cfg.BufferSize),
		flushReq: make(chan chan error, 1),
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
// flushes either when the batch is full, the flush interval elapses,
// or a FlushAll request arrives from the commit coordinator.
func (tw *tableWriter) run(done <-chan struct{}) {
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
				// Rough estimate: 100 bytes per field
				batchBytes += len(row) * 100
			}

			// Flush if batch size or byte limit reached
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
			// Drain remaining buffer items
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

// flush writes a batch of rows to ClickHouse. On error the caller
// should preserve the batch for retry.
func (tw *tableWriter) flush(ctx context.Context, rows []map[string]any) error {
	if len(rows) == 0 {
		return nil
	}

	start := time.Now()

	// Determine column order from the full batch so sparse/optional fields
	// in later rows are not silently dropped.
	columns := tw.getColumns(rows)

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

		return fmt.Errorf("preparing batch for %s: %w", tw.table, err)
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

		return fmt.Errorf("sending batch for %s: %w", tw.table, err)
	}

	duration := time.Since(start)

	tw.metrics.rowsWritten.WithLabelValues(tw.table).Add(float64(len(rows)))
	tw.metrics.writeDuration.WithLabelValues(tw.table).Observe(duration.Seconds())
	tw.metrics.batchSize.WithLabelValues(tw.table).Observe(float64(len(rows)))

	tw.log.WithField("rows", len(rows)).
		WithField("duration", duration).
		Debug("Flushed batch")

	return nil
}

// FlushAll forces all table writers to drain their buffers and flush
// to ClickHouse synchronously. Returns the first error encountered;
// on failure, unflushed rows are preserved in the table writers.
func (w *ClickHouseWriter) FlushAll(ctx context.Context) error {
	w.mu.RLock()
	writers := make([]*tableWriter, 0, len(w.tables))

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

// getColumns returns a stable, sorted list of column names for this batch.
// Rows can be sparse, so we must include keys from every row.
func (tw *tableWriter) getColumns(rows []map[string]any) []string {
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
