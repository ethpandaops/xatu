package clickhouse

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ClickHouse/ch-go"
	"github.com/ethpandaops/xatu/pkg/consumoor/route"
	"github.com/ethpandaops/xatu/pkg/consumoor/telemetry"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/sirupsen/logrus"
)

type chTableWriter struct {
	log       logrus.FieldLogger
	table     string
	baseTable string
	database  string
	config    TableConfig
	metrics   *telemetry.Metrics
	writer    *ChGoWriter

	newBatch func() route.ColumnarBatch

	// limiter is the per-table adaptive concurrency limiter.
	// nil when adaptive limiting is disabled.
	limiter *adaptiveConcurrencyLimiter

	// Cached per-table strings computed once and reused every flush.
	operationName string
	insertQuery   string
	insertQueryOK bool
}

func (tw *chTableWriter) flush(ctx context.Context, events []*xatu.DecoratedEvent) error {
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

	for _, event := range events {
		err := batch.FlattenTo(event)
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

	// Cache the INSERT query body and operation name on first use.
	// Both are invariant between flushes for a given table.
	if !tw.insertQueryOK {
		body, err := insertQueryWithSettings(input.Into(tw.table), tw.config.InsertSettings)
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

		tw.insertQuery = body
		tw.operationName = "insert_" + tw.table
		tw.insertQueryOK = true
	}

	if err := tw.do(ctx, tw.operationName, &ch.Query{
		Body:  tw.insertQuery,
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

		err := tw.limiter.doWithLimiter(attemptCtx, poolFn)
		if IsLimiterRejected(err) {
			tw.metrics.AdaptiveLimiterRejections().WithLabelValues(tw.table).Inc()
		}

		return err
	})
}
