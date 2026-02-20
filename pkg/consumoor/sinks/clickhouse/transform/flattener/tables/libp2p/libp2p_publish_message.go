package libp2p

import (
	"time"

	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/flattener"
	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/metadata"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

var libp2pPublishMessageEventNames = []xatu.Event_Name{
	xatu.Event_LIBP2P_TRACE_PUBLISH_MESSAGE,
}

func init() {
	flattener.MustRegister(flattener.NewStaticRoute(
		libp2pPublishMessageTableName,
		libp2pPublishMessageEventNames,
		func() flattener.ColumnarBatch { return newlibp2pPublishMessageBatch() },
	))
}

func (b *libp2pPublishMessageBatch) FlattenTo(
	event *xatu.DecoratedEvent,
	meta *metadata.CommonMetadata,
) error {
	if event == nil || event.GetEvent() == nil {
		return nil
	}

	if meta == nil {
		meta = metadata.Extract(event)
	}

	b.appendRuntime(event)
	b.appendMetadata(meta)
	b.appendPayload(event)
	b.rows++

	return nil
}

func (b *libp2pPublishMessageBatch) appendRuntime(event *xatu.DecoratedEvent) {
	b.UpdatedDateTime.Append(time.Now())

	if ts := event.GetEvent().GetDateTime(); ts != nil {
		b.EventDateTime.Append(ts.AsTime())
	} else {
		b.EventDateTime.Append(time.Time{})
	}
}

func (b *libp2pPublishMessageBatch) appendPayload(event *xatu.DecoratedEvent) {
	payload := event.GetLibp2PTracePublishMessage()
	if payload == nil {
		b.MessageID.Append("")
		b.TopicLayer.Append("")
		b.TopicForkDigestValue.Append("")
		b.TopicName.Append("")
		b.TopicEncoding.Append("")

		return
	}

	b.MessageID.Append(wrappedStringValue(payload.GetMsgId()))

	// Parse topic fields.
	if topic := wrappedStringValue(payload.GetTopic()); topic != "" {
		parsed := parseTopicFields(topic)
		b.TopicLayer.Append(parsed.Layer)
		b.TopicForkDigestValue.Append(parsed.ForkDigestValue)
		b.TopicName.Append(parsed.Name)
		b.TopicEncoding.Append(parsed.Encoding)
	} else {
		b.TopicLayer.Append("")
		b.TopicForkDigestValue.Append("")
		b.TopicName.Append("")
		b.TopicEncoding.Append("")
	}
}
