package source

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/redpanda-data/benthos/v4/public/service"

	"github.com/ethpandaops/xatu/pkg/consumoor/clickhouse"
	"github.com/ethpandaops/xatu/pkg/consumoor/router"
	"github.com/ethpandaops/xatu/pkg/consumoor/telemetry"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/sirupsen/logrus"
)

const (
	rejectReasonDecode         = "decode_error"
	rejectReasonRouteRejected  = "route_rejected"
	rejectReasonWritePermanent = "write_permanent"
	rejectReasonInvalidEvent   = "invalid_event"
)

var errBatchWriteFailed = errors.New("clickhouse batch write failed")

// groupMessage holds a successfully decoded message within an event group.
type groupMessage struct {
	batchIndex int
	raw        []byte
	event      *xatu.DecoratedEvent
	kafka      kafkaMessageMetadata
	tables     []string
}

// eventGroup collects all successfully decoded messages for one event type.
type eventGroup struct {
	messages []groupMessage
}

type xatuClickHouseOutput struct {
	log              logrus.FieldLogger
	encoding         string
	router           *router.Engine
	writer           Writer
	metrics          *telemetry.Metrics
	rejectSink       rejectSink
	ownsWriter       bool
	outputBatchCount int
	logSampler       *telemetry.LogSampler

	// Group-level retry config for partial table failures.
	groupRetryMaxAttempts int
	groupRetryBaseDelay   time.Duration
	groupRetryMaxDelay    time.Duration

	mu      sync.Mutex
	started bool
}

func (o *xatuClickHouseOutput) Connect(ctx context.Context) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	if o.started {
		return nil
	}

	if o.ownsWriter {
		if err := o.writer.Start(ctx); err != nil {
			return err
		}
	}

	o.started = true

	return nil
}

// WriteBatch processes a Benthos message batch by grouping messages by event
// type, then processing each group independently.
func (o *xatuClickHouseOutput) WriteBatch(
	ctx context.Context,
	msgs service.MessageBatch,
) error {
	if len(msgs) == 0 {
		return nil
	}

	if ctx.Err() != nil {
		return ctx.Err()
	}

	// Record which trigger caused this batch flush.
	// Each Benthos stream is per-topic, so the first message's topic is
	// representative of the entire batch.
	topic := kafkaMetadata(msgs[0]).Topic

	trigger := "timeout"
	if o.outputBatchCount > 0 && len(msgs) >= o.outputBatchCount {
		trigger = "count"
	}

	o.metrics.BatchFlushTrigger().WithLabelValues(topic, trigger).Inc()

	var batchErr *service.BatchError

	var pooledEvents []*xatu.DecoratedEvent
	defer func() {
		for _, ev := range pooledEvents {
			ev.ReturnToVTPool()
		}
	}()

	groups := make(map[xatu.Event_Name]*eventGroup, 16)

	// Phase 1: decode, route, and group by event type.
	for i, msg := range msgs {
		kafka := kafkaMetadata(msg)
		o.metrics.MessagesConsumed().WithLabelValues(kafka.Topic).Inc()

		raw, err := msg.AsBytes()
		if err != nil {
			o.metrics.DecodeErrors().WithLabelValues(kafka.Topic).Inc()

			if ok, suppressed := o.logSampler.Allow("read_bytes:" + kafka.Topic); ok {
				entry := o.log.WithError(err).
					WithField("topic", kafka.Topic).
					WithField("partition", kafka.Partition).
					WithField("offset", kafka.Offset)
				if suppressed > 0 {
					entry = entry.WithField("suppressed", suppressed)
				}

				entry.Warn("Failed to read message bytes")
			}

			if rejectErr := o.rejectMessage(ctx, &rejectedRecord{
				Reason: rejectReasonDecode,
				Err:    err.Error(),
				Kafka:  kafka,
			}); rejectErr != nil {
				batchErr = addBatchFailure(batchErr, msgs, i, rejectErr)
			}

			continue
		}

		event, err := decodeDecoratedEvent(o.encoding, raw)
		if err != nil {
			o.metrics.DecodeErrors().WithLabelValues(kafka.Topic).Inc()

			if ok, suppressed := o.logSampler.Allow("decode:" + kafka.Topic); ok {
				entry := o.log.WithError(err).
					WithField("topic", kafka.Topic).
					WithField("partition", kafka.Partition).
					WithField("offset", kafka.Offset)
				if suppressed > 0 {
					entry = entry.WithField("suppressed", suppressed)
				}

				entry.Warn("Failed to decode message")
			}

			if rejectErr := o.rejectMessage(ctx, &rejectedRecord{
				Reason:  rejectReasonDecode,
				Err:     err.Error(),
				Payload: raw,
				Kafka:   kafka,
			}); rejectErr != nil {
				batchErr = addBatchFailure(batchErr, msgs, i, rejectErr)
			}

			continue
		}

		pooledEvents = append(pooledEvents, event)

		outcome := o.router.Route(event)

		if outcome.Status == router.StatusRejected {
			if rejectErr := o.rejectMessage(ctx, &rejectedRecord{
				Reason:    rejectReasonRouteRejected,
				Err:       "route rejected message",
				Payload:   raw,
				EventName: event.GetEvent().GetName().String(),
				Kafka:     kafka,
			}); rejectErr != nil {
				batchErr = addBatchFailure(batchErr, msgs, i, rejectErr)
			}

			continue
		}

		if outcome.Status == router.StatusErrored {
			batchErr = addBatchFailure(
				batchErr, msgs, i, fmt.Errorf(
					"no route registered for event %s — offset will not advance until a route is deployed",
					event.GetEvent().GetName(),
				),
			)

			continue
		}

		if len(outcome.Results) == 0 {
			continue
		}

		tables := make([]string, len(outcome.Results))
		for j, result := range outcome.Results {
			tables[j] = result.Table
		}

		eventName := event.GetEvent().GetName()

		g, ok := groups[eventName]
		if !ok {
			g = &eventGroup{
				messages: make([]groupMessage, 0, 8),
			}
			groups[eventName] = g
		}

		g.messages = append(g.messages, groupMessage{
			batchIndex: i,
			raw:        raw,
			event:      event,
			kafka:      kafka,
			tables:     tables,
		})
	}

	// Phase 2: process each event group independently.
	// Pass batchErr through so Phase 1 failures (decode errors) are preserved
	// when a group also fails — otherwise processGroup would create a new
	// BatchError that silently drops the earlier failures.
	for _, g := range groups {
		batchErr = o.processGroup(ctx, msgs, batchErr, g)
	}

	if batchErr != nil {
		return batchErr
	}

	return nil
}

func (o *xatuClickHouseOutput) Close(ctx context.Context) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	if !o.started {
		return nil
	}

	var writerErr error

	if o.ownsWriter {
		writerErr = o.writer.Stop(ctx)
	}

	var rejectErr error

	if o.rejectSink != nil {
		rejectErr = o.rejectSink.Close()
	}

	o.started = false

	if writerErr != nil {
		return writerErr
	}

	return rejectErr
}

// processGroup writes all messages in the group to their target tables, then
// flushes only those tables. On failure the entire group is NAK'd or DLQ'd.
// The caller's accumulated batchErr is threaded through so that failures from
// earlier phases (e.g. decode errors) are preserved.
//
// For events that fan out to multiple tables, partial failures (some tables
// succeed, others fail transiently) are retried with exponential backoff.
// Only the failed tables are retried to prevent duplicate writes on already-
// succeeded tables.
func (o *xatuClickHouseOutput) processGroup(
	ctx context.Context,
	msgs service.MessageBatch,
	batchErr *service.BatchError,
	g *eventGroup,
) *service.BatchError {
	tableEvents := make(map[string][]*xatu.DecoratedEvent, 4)
	for _, gm := range g.messages {
		for _, table := range gm.tables {
			tableEvents[table] = append(tableEvents[table], gm.event)
		}
	}

	var (
		allInvalidEvents []*xatu.DecoratedEvent
		lastResult       *clickhouse.FlushResult
	)

	// Group-level retry loop for partial table failures.
	for attempt := range o.groupRetryMaxAttempts + 1 {
		if attempt > 0 && len(tableEvents) > 0 {
			delay := min(
				o.groupRetryBaseDelay*time.Duration(1<<(attempt-1)),
				o.groupRetryMaxDelay,
			)

			o.log.WithFields(logrus.Fields{
				"attempt":       attempt,
				"max_attempts":  o.groupRetryMaxAttempts,
				"delay":         delay,
				"failed_tables": len(tableEvents),
			}).Warn("Retrying failed tables with backoff")

			o.metrics.GroupRetries().WithLabelValues(
				g.messages[0].event.GetEvent().GetName().String(),
			).Inc()

			select {
			case <-ctx.Done():
				for _, gm := range g.messages {
					batchErr = addBatchFailure(batchErr, msgs, gm.batchIndex, ctx.Err())
				}

				return batchErr
			case <-time.After(delay):
			}
		}

		result := o.writer.FlushTableEvents(ctx, tableEvents)
		lastResult = result

		if len(result.InvalidEvents) > 0 {
			allInvalidEvents = append(allInvalidEvents, result.InvalidEvents...)
		}

		// All tables succeeded.
		if len(result.TableErrors) == 0 {
			break
		}

		// Check if any failures are permanent — don't retry those.
		anyTransient := false

		for _, tErr := range result.TableErrors {
			if !clickhouse.IsPermanentWriteError(tErr) {
				anyTransient = true

				break
			}
		}

		if !anyTransient {
			break
		}

		// Remove succeeded tables from the retry set.
		remaining := make(map[string][]*xatu.DecoratedEvent, len(result.TableErrors))

		for table := range result.TableErrors {
			if events, ok := tableEvents[table]; ok {
				remaining[table] = events
			}
		}

		tableEvents = remaining
	}

	// DLQ invalid events individually. Deduplicate by event pointer since
	// the same event can appear in multiple tables' invalid lists due to fanout.
	batchErr = o.dlqInvalidEvents(ctx, msgs, batchErr, g, allInvalidEvents)

	err := lastResult.Err()
	if err == nil {
		return batchErr
	}

	// Flush failed — attribute to all messages in the group.
	if clickhouse.IsPermanentWriteError(err) {
		for _, gm := range g.messages {
			// Copy raw bytes only when needed for DLQ; the success path
			// avoids the copy entirely by referencing the Benthos-owned slice.
			rejectErr := o.rejectMessage(ctx, &rejectedRecord{
				Reason:    rejectReasonWritePermanent,
				Err:       err.Error(),
				Payload:   append([]byte(nil), gm.raw...),
				EventName: gm.event.GetEvent().GetName().String(),
				Kafka:     gm.kafka,
			})
			if rejectErr != nil {
				batchErr = addBatchFailure(
					batchErr, msgs, gm.batchIndex, rejectErr,
				)
			}
		}

		o.log.WithError(err).
			WithField("dlq_enabled", o.rejectSink != nil && o.rejectSink.Enabled()).
			Warn("Permanent write error during group flush")
	} else {
		for _, gm := range g.messages {
			batchErr = addBatchFailure(
				batchErr, msgs, gm.batchIndex, err,
			)
		}
	}

	return batchErr
}

// dlqInvalidEvents sends invalid events to the DLQ, deduplicating by event
// pointer to handle fanout where the same event appears in multiple tables.
func (o *xatuClickHouseOutput) dlqInvalidEvents(
	ctx context.Context,
	msgs service.MessageBatch,
	batchErr *service.BatchError,
	g *eventGroup,
	invalidEvents []*xatu.DecoratedEvent,
) *service.BatchError {
	if len(invalidEvents) == 0 {
		return batchErr
	}

	seen := make(map[*xatu.DecoratedEvent]struct{}, len(invalidEvents))

	for _, ev := range invalidEvents {
		if _, dup := seen[ev]; dup {
			continue
		}

		seen[ev] = struct{}{}

		for _, gm := range g.messages {
			if gm.event != ev {
				continue
			}

			rejectErr := o.rejectMessage(ctx, &rejectedRecord{
				Reason:    rejectReasonInvalidEvent,
				Err:       "invalid event: permanently unflattenable",
				Payload:   append([]byte(nil), gm.raw...),
				EventName: ev.GetEvent().GetName().String(),
				Kafka:     gm.kafka,
			})
			if rejectErr != nil {
				batchErr = addBatchFailure(batchErr, msgs, gm.batchIndex, rejectErr)
			}

			break
		}
	}

	return batchErr
}

func (o *xatuClickHouseOutput) rejectMessage(
	ctx context.Context,
	record *rejectedRecord,
) error {
	if record == nil {
		return errors.New("nil rejected record")
	}

	if o.rejectSink == nil {
		o.metrics.MessagesRejected().WithLabelValues(record.Reason).Inc()

		// Route rejections are intentional — the event type simply isn't
		// routed to any table. Safe to ack without a DLQ.
		if record.Reason == rejectReasonRouteRejected {
			return nil
		}

		// For all other reasons (decode errors, permanent write failures,
		// invalid events) failing the message forces Kafka to redeliver
		// rather than silently dropping data when no DLQ is configured.
		// Invalid events may become valid after a rolling deploy where a
		// newer version adds support for new fields or event subtypes.
		return fmt.Errorf("no DLQ configured for rejected message (%s): %s", record.Reason, record.Err)
	}

	if err := o.rejectSink.Write(ctx, record); err != nil {
		o.metrics.DLQErrors().WithLabelValues(record.Reason).Inc()

		return fmt.Errorf("writing rejected message to dlq: %w", err)
	}

	o.metrics.MessagesRejected().WithLabelValues(record.Reason).Inc()

	if o.rejectSink.Enabled() {
		o.metrics.DLQWrites().WithLabelValues(record.Reason).Inc()
	}

	return nil
}

func addBatchFailure(
	batchErr *service.BatchError,
	msgs service.MessageBatch,
	index int,
	err error,
) *service.BatchError {
	if batchErr == nil {
		batchErr = service.NewBatchError(msgs, errBatchWriteFailed)
	}

	return batchErr.Failed(index, err)
}
