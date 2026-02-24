package clickhouse

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/ClickHouse/ch-go"
	"github.com/ethpandaops/xatu/pkg/consumoor/route"
	"github.com/ethpandaops/xatu/pkg/consumoor/telemetry"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/sirupsen/logrus"
)

// bufferWarningInterval is the minimum time between warning logs per table.
const bufferWarningInterval = time.Minute

// eventEntry holds a single event for buffering in the table writer.
type eventEntry struct {
	event *xatu.DecoratedEvent
}

type chTableWriter struct {
	log       logrus.FieldLogger
	table     string
	baseTable string
	database  string
	config    TableConfig
	metrics   *telemetry.Metrics
	writer    *ChGoWriter
	buffer    chan eventEntry
	flushReq  chan chan error // coordinator sends a response channel; tableWriter flushes and replies

	organicRetryInitDelay time.Duration
	organicRetryMaxDelay  time.Duration
	drainTimeout          time.Duration

	newBatch func() route.ColumnarBatch
	batch    route.ColumnarBatch // reused across flushes via Reset()

	// lastWarnAt tracks the last time a buffer warning was emitted (Unix nanos).
	// Accessed atomically from the Write goroutine for rate limiting.
	lastWarnAt atomic.Int64
}

func (tw *chTableWriter) run(done <-chan struct{}) {
	ticker := time.NewTicker(tw.config.FlushInterval)
	defer ticker.Stop()

	events := make([]eventEntry, 0, tw.config.BatchSize)
	flushBlocked := false
	retryAttempt := 0

	for {
		bufferCh := tw.buffer
		if flushBlocked {
			bufferCh = nil
		}

		select {
		case entry := <-bufferCh:
			tw.decrBuffer(1)

			events = append(events, entry)

			if len(events) >= tw.config.BatchSize {
				if err := tw.flush(context.Background(), events); err != nil {
					var flatErr *flattenError
					if errors.As(err, &flatErr) {
						events = events[:0]

						continue
					}

					if IsPermanentWriteError(err) {
						tw.log.WithError(err).
							WithField("events", len(events)).
							Warn("Dropping permanently invalid batch")
						events = events[:0]
						flushBlocked = false
						retryAttempt = 0

						continue
					}

					// Preserve events for retry on next flush cycle. While
					// pending, stop draining tw.buffer so channel backpressure
					// remains bounded by BufferSize.
					flushBlocked = true

					tw.scheduleOrganicRetry(ticker, retryAttempt)
					retryAttempt++

					continue
				}

				events = events[:0]
				flushBlocked = false
				retryAttempt = 0
			}

		case <-ticker.C:
			if len(events) > 0 {
				if err := tw.flush(context.Background(), events); err != nil {
					var flatErr *flattenError
					if errors.As(err, &flatErr) {
						events = events[:0]

						continue
					}

					if IsPermanentWriteError(err) {
						tw.log.WithError(err).
							WithField("events", len(events)).
							Warn("Dropping permanently invalid batch")
						events = events[:0]
						flushBlocked = false

						tw.resetRetryState(ticker, &retryAttempt)

						continue
					}

					flushBlocked = true

					tw.scheduleOrganicRetry(ticker, retryAttempt)
					retryAttempt++

					continue
				}

				events = events[:0]
				flushBlocked = false

				tw.resetRetryState(ticker, &retryAttempt)
			}

		case errCh := <-tw.flushReq:
			if !flushBlocked {
				// Drain anything still in the channel into the batch.
			drainReq:
				for {
					select {
					case entry := <-tw.buffer:
						tw.decrBuffer(1)

						events = append(events, entry)
					default:
						break drainReq
					}
				}
			}

			if len(events) > 0 {
				err := tw.flush(context.Background(), events)

				var flatErr *flattenError

				switch {
				case err == nil:
					events = events[:0]
					flushBlocked = false

					tw.resetRetryState(ticker, &retryAttempt)
				case errors.As(err, &flatErr):
					events = events[:0]
				case IsPermanentWriteError(err):
					tw.log.WithError(err).
						WithField("events", len(events)).
						Warn("Dropping permanently invalid batch")
					events = events[:0]
					flushBlocked = false

					tw.resetRetryState(ticker, &retryAttempt)
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
					tw.decrBuffer(1)

					events = append(events, entry)
				default:
					break drainLoop
				}
			}

			if len(events) > 0 {
				drainCtx, drainCancel := context.WithTimeout(context.Background(), tw.drainTimeout)

				if err := tw.flush(drainCtx, events); err != nil {
					tw.log.WithError(err).WithField("events", len(events)).Warn("Flush error during shutdown drain")
				}

				drainCancel()
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
			table: tw.baseTable,
			cause: &inputPrepError{cause: fmt.Errorf("no columnar batch factory for %s", tw.table)},
		}
	}

	if tw.batch == nil {
		tw.batch = tw.newBatch()
	} else {
		tw.batch.Reset()
	}

	batch := tw.batch

	var (
		flattenErrs int
		lastErr     error
	)

	for _, e := range events {
		if err := batch.FlattenTo(e.event); err != nil {
			if errors.Is(err, route.ErrInvalidEvent) {
				tw.log.WithError(err).Warn("Skipping invalid event")
				tw.metrics.WriteErrors().WithLabelValues(tw.table).Inc()

				continue
			}

			flattenErrs++
			lastErr = err

			tw.metrics.WriteErrors().WithLabelValues(tw.table).Inc()

			if !tw.config.SkipFlattenErrors {
				tw.log.WithError(err).
					WithField("events", len(events)).
					Error("Flatten failed (fail-fast)")

				return &tableWriteError{
					table: tw.baseTable,
					cause: &flattenError{cause: err},
				}
			}
		}
	}

	if flattenErrs > 0 && tw.config.SkipFlattenErrors {
		tw.log.WithError(lastErr).
			WithField("failed", flattenErrs).
			WithField("total", len(events)).
			Warn("Skipped unflattenable events")
	}

	if flattenErrs == len(events) {
		return &tableWriteError{
			table: tw.baseTable,
			cause: &inputPrepError{
				cause: fmt.Errorf("all %d events failed FlattenTo for %s", len(events), tw.table),
			},
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
			table: tw.baseTable,
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
			table: tw.baseTable,
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

// scheduleOrganicRetry resets the ticker interval to an exponential backoff
// delay so the next ticker.C fires after the computed retry delay.
func (tw *chTableWriter) scheduleOrganicRetry(ticker *time.Ticker, attempt int) {
	delay := min(
		tw.organicRetryInitDelay*time.Duration(1<<attempt),
		tw.organicRetryMaxDelay,
	)

	ticker.Reset(delay)

	tw.log.WithField("delay", delay).
		WithField("attempt", attempt+1).
		Debug("Scheduled organic retry")
}

// resetRetryState restores the ticker to the normal flush interval and zeroes
// the retry attempt counter when recovering from a blocked state.
func (tw *chTableWriter) resetRetryState(ticker *time.Ticker, attempt *int) {
	if *attempt > 0 {
		ticker.Reset(tw.config.FlushInterval)
	}

	*attempt = 0
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

		if fn := tw.writer.poolDoFn; fn != nil {
			return fn(attemptCtx, *query)
		}

		pool := tw.writer.getPool()
		if pool == nil {
			return ch.ErrClosed
		}

		return pool.Do(attemptCtx, *query)
	})
}

// checkBufferWarning emits a rate-limited warning when the buffer usage
// exceeds the configured BufferWarningThreshold. Safe to call from any
// goroutine.
func (tw *chTableWriter) checkBufferWarning() {
	threshold := tw.writer.config.BufferWarningThreshold
	if threshold <= 0 {
		return
	}

	current := len(tw.buffer)
	limit := tw.config.BufferSize

	if float64(current) < threshold*float64(limit) {
		return
	}

	now := time.Now().UnixNano()
	last := tw.lastWarnAt.Load()

	if now-last < int64(bufferWarningInterval) {
		return
	}

	if tw.lastWarnAt.CompareAndSwap(last, now) {
		tw.log.WithFields(logrus.Fields{
			"current":   current,
			"max":       limit,
			"threshold": threshold,
		}).Warn("Buffer usage exceeds warning threshold")
	}
}

// decrBuffer decrements both the per-table and aggregate buffer gauges.
func (tw *chTableWriter) decrBuffer(n int) {
	tw.metrics.BufferUsage().WithLabelValues(tw.table).Sub(float64(n))
	tw.metrics.BufferUsageTotal().Sub(float64(n))
}
