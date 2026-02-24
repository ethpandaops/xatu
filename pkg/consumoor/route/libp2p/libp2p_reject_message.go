package libp2p

import (
	"fmt"
	"time"

	"github.com/ethpandaops/xatu/pkg/consumoor/route"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

var libp2pRejectMessageEventNames = []xatu.Event_Name{
	xatu.Event_LIBP2P_TRACE_REJECT_MESSAGE,
}

func init() {
	r, err := route.NewStaticRoute(
		libp2pRejectMessageTableName,
		libp2pRejectMessageEventNames,
		func() route.ColumnarBatch { return newlibp2pRejectMessageBatch() },
	)
	if err != nil {
		route.RecordError(err)

		return
	}

	if err := route.Register(r); err != nil {
		route.RecordError(err)
	}
}

func (b *libp2pRejectMessageBatch) FlattenTo(
	event *xatu.DecoratedEvent,
) error {
	if event == nil || event.GetEvent() == nil {
		return nil
	}

	if event.GetLibp2PTraceRejectMessage() == nil {
		return fmt.Errorf("nil libp2p_trace_reject_message payload: %w", route.ErrInvalidEvent)
	}

	b.appendRuntime(event)
	b.appendMetadata(event)
	b.appendPayload(event)
	b.rows++

	return nil
}

func (b *libp2pRejectMessageBatch) appendRuntime(event *xatu.DecoratedEvent) {
	b.UpdatedDateTime.Append(time.Now())

	if ts := event.GetEvent().GetDateTime(); ts != nil {
		b.EventDateTime.Append(ts.AsTime())
	} else {
		b.EventDateTime.Append(time.Time{})
	}
}

func (b *libp2pRejectMessageBatch) appendPayload(
	event *xatu.DecoratedEvent,
) {
	payload := event.GetLibp2PTraceRejectMessage()
	b.MessageID.Append(wrappedStringValue(payload.GetMsgId()))
	b.Reason.Append(wrappedStringValue(payload.GetReason()))

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
		return c.GetLibp2PTraceRejectMessage()
	})

	if peerID == "" {
		peerID = wrappedStringValue(payload.GetPeerId())
	}

	networkName := event.GetMeta().GetClient().GetEthereum().GetNetwork().GetName()
	b.PeerIDUniqueKey.Append(computePeerIDUniqueKey(peerID, networkName))
}
