package clickhouse

import (
	"context"
	"fmt"
	"time"

	"github.com/ClickHouse/ch-go"
	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/flattener"
	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/metadata"
	"github.com/ethpandaops/xatu/pkg/consumoor/telemetry"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/sirupsen/logrus"
)

// eventEntry holds a single event and its pre-extracted metadata for
// buffering in the table writer.
type eventEntry struct {
	event *xatu.DecoratedEvent
	meta  *metadata.CommonMetadata
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
					tw.metrics.BufferUsage().WithLabelValues(tw.table).Sub(1)

					events = append(events, entry)
				default:
					break drainLoop
				}
			}

			if len(events) > 0 {
				if err := tw.flush(context.Background(), events); err != nil {
					tw.log.WithError(err).WithField("events", len(events)).Warn("Flush error during shutdown drain")
				}
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
