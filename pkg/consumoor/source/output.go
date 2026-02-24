//nolint:wsl_v5 // Pipeline branches are clearer when kept compact.
package source

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"

	"github.com/redpanda-data/benthos/v4/public/service"

	"github.com/ethpandaops/xatu/pkg/consumoor/router"
	"github.com/ethpandaops/xatu/pkg/consumoor/telemetry"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/sirupsen/logrus"
)

const (
	rejectReasonDecode         = "decode_error"
	rejectReasonRouteRejected  = "route_rejected"
	rejectReasonWritePermanent = "write_permanent"
)

var errBatchWriteFailed = errors.New("clickhouse batch write failed")

type messageContext struct {
	raw   []byte
	event *xatu.DecoratedEvent
	kafka kafkaMessageMetadata
}

type xatuClickHouseOutput struct {
	log        logrus.FieldLogger
	encoding   string
	router     *router.Engine
	writer     Writer
	metrics    *telemetry.Metrics
	classifier WriteErrorClassifier
	rejectSink rejectSink
	ownsWriter bool

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

func (o *xatuClickHouseOutput) WriteBatch(ctx context.Context, msgs service.MessageBatch) error {
	if len(msgs) == 0 {
		return nil
	}

	return o.writeBatchMode(ctx, msgs)
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

func (o *xatuClickHouseOutput) writeBatchMode(ctx context.Context, msgs service.MessageBatch) error {
	msgContexts := make([]messageContext, len(msgs))
	tableToMessageIndexes := make(map[string]map[int]struct{}, 32)
	hasResults := false

	var batchErr *service.BatchError

	for i, msg := range msgs {
		mctx := o.buildMessageContext(msg)
		msgContexts[i] = mctx
		o.metrics.MessagesConsumed().WithLabelValues(mctx.kafka.Topic).Inc()

		raw, err := msg.AsBytes()
		if err != nil {
			o.metrics.DecodeErrors().WithLabelValues(mctx.kafka.Topic).Inc()
			o.log.WithError(err).WithField("topic", mctx.kafka.Topic).Warn("Failed to read message bytes")

			if rejectErr := o.rejectMessage(ctx, &rejectedRecord{
				Reason: rejectReasonDecode,
				Err:    err.Error(),
				Kafka:  mctx.kafka,
			}); rejectErr != nil {
				batchErr = addBatchFailure(batchErr, msgs, i, rejectErr)
			}

			continue
		}

		msgContexts[i].raw = append(msgContexts[i].raw[:0], raw...)

		event, err := decodeDecoratedEvent(o.encoding, raw)
		if err != nil {
			o.metrics.DecodeErrors().WithLabelValues(mctx.kafka.Topic).Inc()
			o.log.WithError(err).WithField("topic", mctx.kafka.Topic).Warn("Failed to decode message")

			if rejectErr := o.rejectMessage(ctx, &rejectedRecord{
				Reason:  rejectReasonDecode,
				Err:     err.Error(),
				Payload: raw,
				Kafka:   mctx.kafka,
			}); rejectErr != nil {
				batchErr = addBatchFailure(batchErr, msgs, i, rejectErr)
			}

			continue
		}

		msgContexts[i].event = event
		outcome := o.router.Route(event)

		if outcome.Status == router.StatusRejected {
			if rejectErr := o.rejectMessage(ctx, &rejectedRecord{
				Reason:    rejectReasonRouteRejected,
				Err:       "route rejected message",
				Payload:   raw,
				EventName: event.GetEvent().GetName().String(),
				Kafka:     mctx.kafka,
			}); rejectErr != nil {
				batchErr = addBatchFailure(batchErr, msgs, i, rejectErr)
			}

			continue
		}

		if outcome.Status == router.StatusErrored {
			batchErr = addBatchFailure(batchErr, msgs, i, errors.New("route errored"))

			continue
		}

		for _, result := range outcome.Results {
			hasResults = true

			o.writer.Write(result.Table, event)
			addTableMessageIndex(tableToMessageIndexes, result.Table, i)
		}
	}

	if !hasResults {
		if batchErr != nil {
			return batchErr
		}

		return nil
	}

	flushedTables := make([]string, 0, len(tableToMessageIndexes))
	for table := range tableToMessageIndexes {
		flushedTables = append(flushedTables, table)
	}

	if err := o.writer.FlushTables(ctx, flushedTables); err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		failedIndexes := failedIndexesForWriteError(tableToMessageIndexes, err, o.classifier)
		if len(failedIndexes) == 0 {
			return fmt.Errorf("flush failed (table unidentified): %w", err)
		}

		if o.isPermanentWriteError(err) {
			for _, idx := range failedIndexes {
				rejectErr := o.rejectMessage(ctx, &rejectedRecord{
					Reason:    rejectReasonWritePermanent,
					Err:       err.Error(),
					Payload:   msgContexts[idx].raw,
					EventName: eventNameFromContext(msgContexts[idx]),
					Kafka:     msgContexts[idx].kafka,
				})
				if rejectErr != nil {
					batchErr = addBatchFailure(batchErr, msgs, idx, rejectErr)

					continue
				}
			}

			o.log.WithError(err).Warn("Dropped permanently invalid rows during batch flush")
		} else {
			for _, idx := range failedIndexes {
				batchErr = addBatchFailure(batchErr, msgs, idx, err)
			}
		}
	}

	if batchErr != nil {
		return batchErr
	}

	return nil
}

func (o *xatuClickHouseOutput) buildMessageContext(msg *service.Message) messageContext {
	return messageContext{
		kafka: kafkaMetadata(msg),
	}
}

func (o *xatuClickHouseOutput) rejectMessage(ctx context.Context, record *rejectedRecord) error {
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

		// For all other reasons (decode errors, permanent write failures)
		// failing the message forces Kafka to redeliver rather than
		// silently dropping data when no DLQ is configured.
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

// isPermanentWriteError returns true if any constituent error in a
// (possibly joined) error is classified as permanent. A single
// permanent sub-error means the affected rows will never succeed on
// retry, so the entire set of failed indexes should be rejected.
func (o *xatuClickHouseOutput) isPermanentWriteError(err error) bool {
	if o.classifier == nil {
		return false
	}

	if o.classifier.IsPermanent(err) {
		return true
	}

	// Check sub-errors inside a joined error.
	if joined, ok := err.(interface{ Unwrap() []error }); ok {
		for _, subErr := range joined.Unwrap() {
			if o.classifier.IsPermanent(subErr) {
				return true
			}
		}
	}

	return false
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

func addTableMessageIndex(indexes map[string]map[int]struct{}, table string, idx int) {
	perTable, ok := indexes[table]
	if !ok {
		perTable = make(map[int]struct{}, 8)
		indexes[table] = perTable
	}

	perTable[idx] = struct{}{}
}

// failedIndexesForWriteError extracts message indexes for all tables
// that failed. The error may be a single tableWriteError or a joined
// error (from errors.Join) containing multiple tableWriteErrors when
// several tables fail in the same flush.
func failedIndexesForWriteError(
	tableToMessageIndexes map[string]map[int]struct{},
	err error,
	classifier WriteErrorClassifier,
) []int {
	if classifier == nil {
		return nil
	}

	// Collect table names from all constituent errors.
	tables := make(map[string]struct{}, 4)

	// Check if this is a joined error containing multiple sub-errors.
	if joined, ok := err.(interface{ Unwrap() []error }); ok {
		for _, subErr := range joined.Unwrap() {
			if t := classifier.Table(subErr); t != "" {
				tables[t] = struct{}{}
			}
		}
	}

	// Also try the top-level error itself (handles single-error case
	// and any wrapper that embeds a table name directly).
	if t := classifier.Table(err); t != "" {
		tables[t] = struct{}{}
	}

	if len(tables) == 0 {
		return nil
	}

	// Collect unique message indexes across all failed tables.
	seen := make(map[int]struct{}, 16)

	for table := range tables {
		perTable, ok := tableToMessageIndexes[table]
		if !ok {
			continue
		}

		for idx := range perTable {
			seen[idx] = struct{}{}
		}
	}

	if len(seen) == 0 {
		return nil
	}

	out := make([]int, 0, len(seen))
	for idx := range seen {
		out = append(out, idx)
	}

	sort.Ints(out)

	return out
}

func eventNameFromContext(mctx messageContext) string {
	if mctx.event == nil || mctx.event.GetEvent() == nil {
		return ""
	}

	return mctx.event.GetEvent().GetName().String()
}
