package clickhouse

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/ClickHouse/ch-go"
	"github.com/ethpandaops/xatu/pkg/consumoor/route"
	"github.com/ethpandaops/xatu/pkg/consumoor/telemetry"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/sirupsen/logrus"
)

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

	drainTimeout time.Duration

	// drainMu serializes buffer drains so that the first goroutine gets a
	// large batch while subsequent concurrent flushers find an empty buffer.
	drainMu sync.Mutex

	newBatch func() route.ColumnarBatch

	// limiter is the per-table adaptive concurrency limiter.
	// nil when adaptive limiting is disabled.
	limiter *adaptiveConcurrencyLimiter
}

// flushBuffer drains the buffer channel and flushes all accumulated events.
// The drain is serialized via drainMu so that the first caller gets a large
// batch while concurrent callers find an empty buffer and return nil.
func (tw *chTableWriter) flushBuffer(ctx context.Context) error {
	tw.drainMu.Lock()

	events := make([]eventEntry, 0, len(tw.buffer))

	for {
		select {
		case entry := <-tw.buffer:
			events = append(events, entry)
		default:
			goto drained
		}
	}

drained:

	tw.drainMu.Unlock()

	if len(events) == 0 {
		return nil
	}

	return tw.flush(ctx, events)
}

// drainAndFlush drains the buffer and flushes with a timeout context.
// Used during shutdown to flush any remaining events.
func (tw *chTableWriter) drainAndFlush(timeout time.Duration) error {
	tw.drainMu.Lock()

	events := make([]eventEntry, 0, len(tw.buffer))

	for {
		select {
		case entry := <-tw.buffer:
			events = append(events, entry)
		default:
			goto drained
		}
	}

drained:

	tw.drainMu.Unlock()

	if len(events) == 0 {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	return tw.flush(ctx, events)
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

	batch := tw.newBatch()

	var (
		flattenErrs int
		lastErr     error
	)

	for _, e := range events {
		err := batch.FlattenTo(e.event)
		if err == nil {
			continue
		}

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

func (tw *chTableWriter) do(
	ctx context.Context,
	operation string,
	query *ch.Query,
	beforeAttempt func(),
) error {
	poolFn := func(attemptCtx context.Context) error {
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
	}

	return tw.writer.doWithRetry(ctx, operation, func(attemptCtx context.Context) error {
		if tw.limiter == nil {
			return poolFn(attemptCtx)
		}

		return tw.limiter.doWithLimiter(attemptCtx, poolFn)
	})
}
