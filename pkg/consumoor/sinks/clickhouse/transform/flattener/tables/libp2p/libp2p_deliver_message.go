package libp2p

import (
	"time"

	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/flattener"
	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/metadata"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

var libp2pDeliverMessageEventNames = []xatu.Event_Name{
	xatu.Event_LIBP2P_TRACE_DELIVER_MESSAGE,
}

func init() {
	flattener.MustRegister(flattener.NewStaticRoute(
		libp2pDeliverMessageTableName,
		libp2pDeliverMessageEventNames,
		func() flattener.ColumnarBatch { return newlibp2pDeliverMessageBatch() },
	))
}

func (b *libp2pDeliverMessageBatch) FlattenTo(
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
	b.appendPayload(event, meta)
	b.rows++

	return nil
}

func (b *libp2pDeliverMessageBatch) appendRuntime(event *xatu.DecoratedEvent) {
	b.UpdatedDateTime.Append(time.Now())

	if ts := event.GetEvent().GetDateTime(); ts != nil {
		b.EventDateTime.Append(ts.AsTime())
	} else {
		b.EventDateTime.Append(time.Time{})
	}
}

func (b *libp2pDeliverMessageBatch) appendPayload(
	event *xatu.DecoratedEvent,
	meta *metadata.CommonMetadata,
) {
	payload := event.GetLibp2PTraceDeliverMessage()
	if payload == nil {
		b.MessageID.Append("")
		b.MessageSize.Append(0)
		b.SeqNumber.Append(0)
		b.LocalDelivery.Append(false)
		b.TopicLayer.Append("")
		b.TopicForkDigestValue.Append("")
		b.TopicName.Append("")
		b.TopicEncoding.Append("")
		b.PeerIDUniqueKey.Append(0)

		return
	}

	b.MessageID.Append(wrappedStringValue(payload.GetMsgId()))

	if msgSize := payload.GetMsgSize(); msgSize != nil {
		b.MessageSize.Append(msgSize.GetValue())
	} else {
		b.MessageSize.Append(0)
	}

	if seqNo := payload.GetSeqNumber(); seqNo != nil {
		b.SeqNumber.Append(seqNo.GetValue())
	} else {
		b.SeqNumber.Append(0)
	}

	if local := payload.GetLocal(); local != nil {
		b.LocalDelivery.Append(local.GetValue())
	} else {
		b.LocalDelivery.Append(false)
	}

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

	// Compute peer_id_unique_key from client metadata peer ID.
	peerID := peerIDFromMetadata(event, func(c *xatu.ClientMeta) peerIDMetadataProvider {
		return c.GetLibp2PTraceDeliverMessage()
	})

	if peerID == "" {
		peerID = wrappedStringValue(payload.GetPeerId())
	}

	networkName := meta.MetaNetworkName
	b.PeerIDUniqueKey.Append(computePeerIDUniqueKey(peerID, networkName))
}
