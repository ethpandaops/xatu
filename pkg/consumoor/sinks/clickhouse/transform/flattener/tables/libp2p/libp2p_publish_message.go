package libp2p

import (
	"fmt"
	"time"

	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/flattener"
	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/metadata"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

func init() {
	flattener.MustRegister(flattener.NewStaticRoute(
		libp2pPublishMessageTableName,
		[]xatu.Event_Name{xatu.Event_LIBP2P_TRACE_PUBLISH_MESSAGE},
		func() flattener.ColumnarBatch {
			return newlibp2pPublishMessageBatch()
		},
	))
}

func (b *libp2pPublishMessageBatch) FlattenTo(
	event *xatu.DecoratedEvent,
	meta *metadata.CommonMetadata,
) error {
	if event == nil || event.GetEvent() == nil {
		return nil
	}

	payload := event.GetLibp2PTracePublishMessage()
	if payload == nil {
		return fmt.Errorf("nil LibP2PTracePublishMessage payload: %w", flattener.ErrInvalidEvent)
	}

	if err := b.validate(event); err != nil {
		return err
	}

	layer, forkDigest, name, encoding := parseTopic(payload.GetTopic().GetValue())

	b.UpdatedDateTime.Append(time.Now())
	b.EventDateTime.Append(event.GetEvent().GetDateTime().AsTime())
	b.TopicLayer.Append(layer)
	b.TopicForkDigestValue.Append(forkDigest)
	b.TopicName.Append(name)
	b.TopicEncoding.Append(encoding)
	b.MessageID.Append(payload.GetMsgId().GetValue())

	b.appendMetadata(meta)
	b.rows++

	return nil
}

func (b *libp2pPublishMessageBatch) validate(event *xatu.DecoratedEvent) error {
	addl := event.GetMeta().GetClient().GetLibp2PTracePublishMessage()
	if addl == nil {
		return fmt.Errorf("nil LibP2PTracePublishMessage additional data: %w", flattener.ErrInvalidEvent)
	}

	return nil
}
