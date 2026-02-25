package libp2p

import (
	"fmt"
	"time"

	"github.com/ethpandaops/xatu/pkg/consumoor/route"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

var libp2pDuplicateMessageEventNames = []xatu.Event_Name{
	xatu.Event_LIBP2P_TRACE_DUPLICATE_MESSAGE,
}

func init() {
	r, err := route.NewStaticRoute(
		libp2pDuplicateMessageTableName,
		libp2pDuplicateMessageEventNames,
		func() route.ColumnarBatch { return newlibp2pDuplicateMessageBatch() },
	)
	if err != nil {
		route.RecordError(err)

		return
	}

	if err := route.Register(r); err != nil {
		route.RecordError(err)
	}
}

func (b *libp2pDuplicateMessageBatch) FlattenTo(
	event *xatu.DecoratedEvent,
) error {
	if event == nil || event.GetEvent() == nil {
		return nil
	}

	if event.GetLibp2PTraceDuplicateMessage() == nil {
		return fmt.Errorf("nil libp2p_trace_duplicate_message payload: %w", route.ErrInvalidEvent)
	}

	b.appendRuntime(event)
	b.appendMetadata(event)
	b.appendPayload(event)
	b.rows++

	return nil
}

func (b *libp2pDuplicateMessageBatch) appendRuntime(event *xatu.DecoratedEvent) {
	b.UpdatedDateTime.Append(time.Now())

	if ts := event.GetEvent().GetDateTime(); ts != nil {
		b.EventDateTime.Append(ts.AsTime())
	} else {
		b.EventDateTime.Append(time.Time{})
	}
}

func (b *libp2pDuplicateMessageBatch) appendPayload(
	event *xatu.DecoratedEvent,
) {
	payload := event.GetLibp2PTraceDuplicateMessage()
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
		return c.GetLibp2PTraceDuplicateMessage()
	})

	if peerID == "" {
		peerID = wrappedStringValue(payload.GetPeerId())
	}

	networkName := event.GetMeta().GetClient().GetEthereum().GetNetwork().GetName()
	b.PeerIDUniqueKey.Append(computePeerIDUniqueKey(peerID, networkName))
}
