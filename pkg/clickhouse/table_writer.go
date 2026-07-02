package clickhouse

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/ClickHouse/ch-go"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/ethpandaops/xatu/pkg/clickhouse/route"
	"github.com/ethpandaops/xatu/pkg/clickhouse/telemetry"
	"github.com/ethpandaops/xatu/pkg/observability"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

type chTableWriter struct {
	log       observability.ContextualLogger
	table     string
	baseTable string
	database  string
	config    TableConfig
	metrics   *telemetry.Metrics
	writer    *Writer

	newBatch func() route.ColumnarBatch

	// limiter is the per-table adaptive concurrency limiter.
	// nil when adaptive limiting is disabled.
	limiter *adaptiveConcurrencyLimiter

	// logSampler rate-limits repetitive per-event error logs (e.g. invalid
	// events or flatten failures) to avoid flooding when a topic starts
	// producing bad data.
	logSampler *telemetry.LogSampler

	// Cached per-table strings computed once and reused every flush.
	// queryInit ensures thread-safe lazy initialization when maxInFlight > 1.
	queryInit     sync.Once
	queryInitErr  error
	operationName string
	insertQuery   string
	insertQueryOK bool
}

// isCanonical reports whether this writer targets a canonical (cannon-derived,
// backfilled) table. Canonical data is authoritative and gap-sensitive, so
// invalid events halt rather than drop. Live/sentry tables are not prefixed
// canonical_ and drop invalid events instead.
func (tw *chTableWriter) isCanonical() bool {
	return strings.HasPrefix(tw.baseTable, "canonical_")
}

func (tw *chTableWriter) flush(ctx context.Context, events []*xatu.DecoratedEvent) ([]*xatu.DecoratedEvent, error) {
	if len(events) == 0 {
		return nil, nil
	}

	start := time.Now()

	if tw.newBatch == nil {
		tw.log.WithField("events", len(events)).WithContext(ctx).
			Error("No columnar batch factory registered")
		tw.metrics.WriteErrors().WithLabelValues(tw.table).Add(float64(len(events)))

		return nil, &tableWriteError{
			table: tw.baseTable,
			cause: &inputPrepError{cause: fmt.Errorf("no columnar batch factory for %s", tw.table)},
		}
	}

	batch := tw.newBatch()

	var (
		flattenErrs   int
		lastErr       error
		invalidEvents []*xatu.DecoratedEvent
	)

	for _, event := range events {
		err := batch.FlattenTo(event)
		if err == nil {
			continue
		}

		if errors.Is(err, route.ErrInvalidEvent) {
			tw.metrics.WriteErrors().WithLabelValues(tw.table).Inc()

			canonical := tw.isCanonical()

			if ok, suppressed := tw.logSampler.Allow("invalid_event"); ok {
				entry := tw.log.WithError(err).
					WithField("event_name", event.GetEvent().GetName().String()).
					WithField("canonical", canonical)
				if meta := event.GetMeta(); meta != nil && meta.GetClient() != nil {
					entry = entry.WithField("client_name", meta.GetClient().GetName())
				}

				if suppressed > 0 {
					entry = entry.WithField("suppressed", suppressed)
				}

				if jsonBytes, jsonErr := protojson.Marshal(event); jsonErr == nil {
					entry = entry.WithField("event_json", string(jsonBytes))
				}

				if canonical {
					entry.Error("Halting on invalid canonical event (refusing to advance past a gap)")
				} else {
					entry.Warn("Skipping invalid event")
				}
			}

			// Canonical (cannon-backfilled) data is authoritative: dropping an
			// invalid row leaves a permanent gap. Halt so the caller retries
			// against complete upstream data rather than advancing past it.
			if canonical {
				return invalidEvents, &tableWriteError{
					table: tw.baseTable,
					cause: &invalidEventError{cause: err},
				}
			}

			// Non-canonical (live/sentry) data is point-in-time and cannot be
			// re-fetched — drop it.
			invalidEvents = append(invalidEvents, event)

			continue
		}

		flattenErrs++
		lastErr = err

		tw.metrics.WriteErrors().WithLabelValues(tw.table).Inc()

		if !tw.config.SkipFlattenErrors {
			if ok, suppressed := tw.logSampler.Allow("flatten_error"); ok {
				entry := tw.log.WithError(err).
					WithField("events", len(events)).
					WithField("event_name", event.GetEvent().GetName().String())
				if suppressed > 0 {
					entry = entry.WithField("suppressed", suppressed)
				}

				if jsonBytes, jsonErr := protojson.Marshal(event); jsonErr == nil {
					entry = entry.WithField("event_json", string(jsonBytes))
				}

				entry.Error("Flatten failed (fail-fast)")
			}

			return invalidEvents, &tableWriteError{
				table: tw.baseTable,
				cause: &flattenError{cause: err},
			}
		}
	}

	if flattenErrs > 0 && tw.config.SkipFlattenErrors {
		tw.log.WithError(lastErr).
			WithField("failed", flattenErrs).
			WithField("total", len(events)).WithContext(ctx).
			Warn("Skipped unflattenable events")
	}

	if flattenErrs == len(events) {
		return invalidEvents, &tableWriteError{
			table: tw.baseTable,
			cause: &inputPrepError{
				cause: fmt.Errorf("all %d events failed FlattenTo for %s", len(events), tw.table),
			},
		}
	}

	rows := batch.Rows()
	if rows == 0 {
		return invalidEvents, nil
	}

	input := batch.Input()

	// Cache the INSERT query body and operation name on first use.
	// Both are invariant between flushes for a given table.
	// sync.Once ensures safe concurrent initialization when maxInFlight > 1.
	tw.queryInit.Do(func() {
		body, qErr := insertQueryWithSettings(input.Into(tw.table), tw.config.InsertSettings)
		if qErr != nil {
			tw.queryInitErr = qErr

			return
		}

		tw.insertQuery = body
		tw.operationName = "insert_" + tw.table
		tw.insertQueryOK = true
	})

	if !tw.insertQueryOK {
		tw.log.WithError(tw.queryInitErr).
			WithField("rows", rows).WithContext(ctx).
			Error("Invalid insert settings")
		tw.metrics.WriteErrors().WithLabelValues(tw.table).Add(float64(rows))

		return invalidEvents, &tableWriteError{
			table: tw.baseTable,
			cause: fmt.Errorf("building insert query for %s: %w", tw.table, tw.queryInitErr),
		}
	}

	if err := tw.do(ctx, tw.operationName, &ch.Query{
		Body:  tw.insertQuery,
		Input: input,
	}, nil); err != nil {
		tw.log.WithError(err).
			WithField("rows", rows).WithContext(ctx).
			Error("Failed to send ch-go batch")
		tw.metrics.WriteErrors().WithLabelValues(tw.table).Add(float64(rows))

		return invalidEvents, &tableWriteError{
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
		WithField("duration", duration).WithContext(ctx).
		Debug("Flushed ch-go batch")

	return invalidEvents, nil
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
