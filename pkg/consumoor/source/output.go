//nolint:wsl_v5 // Pipeline branches are clearer when kept compact.
package source

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/redpanda-data/benthos/v4/public/service"

	chtransform "github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform"
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
	log          logrus.FieldLogger
	encoding     string
	deliveryMode string
	router       *chtransform.Engine
	writer       Writer
	metrics      *telemetry.Metrics
	classifier   WriteErrorClassifier
	rejectSink   rejectSink

	mu      sync.Mutex
	started bool
}

func (o *xatuClickHouseOutput) Connect(ctx context.Context) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	if o.started {
		return nil
	}

	if err := o.writer.Start(ctx); err != nil {
		return err
	}

	o.started = true

	return nil
}

func (o *xatuClickHouseOutput) WriteBatch(ctx context.Context, msgs service.MessageBatch) error {
	if len(msgs) == 0 {
		return nil
	}

	if strings.EqualFold(o.deliveryMode, DeliveryModeMessage) {
		return o.writeMessageMode(ctx, msgs)
	}

	return o.writeBatchMode(ctx, msgs)
}

func (o *xatuClickHouseOutput) Close(ctx context.Context) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	if !o.started {
		return nil
	}

	writerErr := o.writer.Stop(ctx)

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

func (o *xatuClickHouseOutput) writeMessageMode(ctx context.Context, msgs service.MessageBatch) error {
	var batchErr *service.BatchError

	for i, msg := range msgs {
		mctx := o.buildMessageContext(msg)
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

		mctx.raw = append(mctx.raw[:0], raw...)

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

		mctx.event = event
		outcome := o.router.Route(event)

		if outcome.Status == chtransform.StatusRejected {
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

		if outcome.Status == chtransform.StatusErrored {
			batchErr = addBatchFailure(batchErr, msgs, i, errors.New("route errored"))

			continue
		}

		if len(outcome.Results) == 0 {
			continue
		}

		for _, result := range outcome.Results {
			o.writer.Write(result.Table, event, outcome.Meta)
		}

		if err := o.writer.FlushAll(ctx); err != nil {
			if o.isPermanentWriteError(err) {
				if rejectErr := o.rejectMessage(ctx, &rejectedRecord{
					Reason:    rejectReasonWritePermanent,
					Err:       err.Error(),
					Payload:   raw,
					EventName: event.GetEvent().GetName().String(),
					Kafka:     mctx.kafka,
				}); rejectErr != nil {
					batchErr = addBatchFailure(batchErr, msgs, i, rejectErr)
				}

				o.log.WithError(err).WithField("topic", mctx.kafka.Topic).Warn("Dropped permanently invalid message")

				continue
			}

			batchErr = addBatchFailure(batchErr, msgs, i, err)
			for j := i + 1; j < len(msgs); j++ {
				batchErr = addBatchFailure(batchErr, msgs, j, err)
			}

			return batchErr
		}
	}

	if batchErr != nil {
		return batchErr
	}

	return nil
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

		if outcome.Status == chtransform.StatusRejected {
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

		if outcome.Status == chtransform.StatusErrored {
			batchErr = addBatchFailure(batchErr, msgs, i, errors.New("route errored"))

			continue
		}

		for _, result := range outcome.Results {
			hasResults = true

			o.writer.Write(result.Table, event, outcome.Meta)
			addTableMessageIndex(tableToMessageIndexes, result.Table, i)
		}
	}

	if !hasResults {
		if batchErr != nil {
			return batchErr
		}

		return nil
	}

	if err := o.writer.FlushAll(ctx); err != nil {
		failedIndexes := failedIndexesForWriteError(tableToMessageIndexes, err, o.classifier)
		if len(failedIndexes) == 0 {
			failedIndexes = allTableMessageIndexes(tableToMessageIndexes)
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

	o.metrics.MessagesRejected().WithLabelValues(record.Reason).Inc()

	if o.rejectSink == nil {
		return nil
	}

	if err := o.rejectSink.Write(ctx, record); err != nil {
		o.metrics.DLQErrors().WithLabelValues(record.Reason).Inc()

		return fmt.Errorf("writing rejected message to dlq: %w", err)
	}

	if o.rejectSink.Enabled() {
		o.metrics.DLQWrites().WithLabelValues(record.Reason).Inc()
	}

	return nil
}

func (o *xatuClickHouseOutput) isPermanentWriteError(err error) bool {
	if o.classifier == nil {
		return false
	}

	return o.classifier.IsPermanent(err)
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

func failedIndexesForWriteError(
	tableToMessageIndexes map[string]map[int]struct{},
	err error,
	classifier WriteErrorClassifier,
) []int {
	if classifier == nil {
		return nil
	}

	table := classifier.Table(err)
	if table == "" {
		return nil
	}

	perTable, ok := tableToMessageIndexes[table]
	if !ok {
		return nil
	}

	out := make([]int, 0, len(perTable))
	for idx := range perTable {
		out = append(out, idx)
	}

	sort.Ints(out)

	return out
}

func allTableMessageIndexes(tableToMessageIndexes map[string]map[int]struct{}) []int {
	seen := make(map[int]struct{}, 32)
	for _, perTable := range tableToMessageIndexes {
		for idx := range perTable {
			seen[idx] = struct{}{}
		}
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
