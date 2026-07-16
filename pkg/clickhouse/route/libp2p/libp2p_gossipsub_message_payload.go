package libp2p

import (
	"fmt"
	"time"

	"github.com/ethpandaops/xatu/pkg/clickhouse/route"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

var libp2pGossipsubMessagePayloadEventNames = []xatu.Event_Name{
	xatu.Event_LIBP2P_TRACE_GOSSIPSUB_MESSAGE_PAYLOAD,
}

func init() {
	r, err := route.NewStaticRoute(
		libp2pGossipsubMessagePayloadTableName,
		libp2pGossipsubMessagePayloadEventNames,
		func() route.ColumnarBatch { return newlibp2pGossipsubMessagePayloadBatch() },
	)
	if err != nil {
		route.RecordError(err)

		return
	}

	if err := route.Register(r); err != nil {
		route.RecordError(err)
	}
}

func (b *libp2pGossipsubMessagePayloadBatch) FlattenTo(
	event *xatu.DecoratedEvent,
) error {
	if event == nil || event.GetEvent() == nil {
		return nil
	}

	if event.GetLibp2PTraceGossipsubMessagePayload() == nil {
		return fmt.Errorf("nil libp2p_trace_gossipsub_message_payload payload: %w", route.ErrInvalidEvent)
	}

	if err := b.validate(event); err != nil {
		return err
	}

	b.appendRuntime(event)
	b.appendMetadata(event)
	b.appendPayload(event)
	b.appendClientAdditionalData(event)
	b.rows++

	return nil
}

func (b *libp2pGossipsubMessagePayloadBatch) validate(event *xatu.DecoratedEvent) error {
	additional := event.GetMeta().GetClient().GetLibp2PTraceGossipsubMessagePayload()
	if additional == nil {
		return fmt.Errorf("nil additional data: %w", route.ErrInvalidEvent)
	}

	if traceMeta := additional.GetMetadata(); traceMeta == nil || traceMeta.GetPeerId() == nil {
		return fmt.Errorf("nil PeerId: %w", route.ErrInvalidEvent)
	}

	if additional.GetMessageId() == nil {
		return fmt.Errorf("nil MessageId: %w", route.ErrInvalidEvent)
	}

	if additional.GetWallclockSlot() == nil {
		return fmt.Errorf("nil WallclockSlot: %w", route.ErrInvalidEvent)
	}

	if additional.GetWallclockEpoch() == nil {
		return fmt.Errorf("nil WallclockEpoch: %w", route.ErrInvalidEvent)
	}

	return nil
}

func (b *libp2pGossipsubMessagePayloadBatch) appendRuntime(event *xatu.DecoratedEvent) {
	b.UpdatedDateTime.Append(time.Now())

	if ts := event.GetEvent().GetDateTime(); ts != nil {
		b.EventDateTime.Append(ts.AsTime())
	} else {
		b.EventDateTime.Append(time.Time{})
	}
}

func (b *libp2pGossipsubMessagePayloadBatch) appendPayload(event *xatu.DecoratedEvent) {
	payload := event.GetLibp2PTraceGossipsubMessagePayload()

	b.MessageData.AppendBytes(payload.GetData())
	b.Outcome.Append(wrappedStringValue(payload.GetOutcome()))
	b.RejectReason.Append(wrappedStringValue(payload.GetRejectReason()))
}

func (b *libp2pGossipsubMessagePayloadBatch) appendClientAdditionalData(
	event *xatu.DecoratedEvent,
) {
	additional := event.GetMeta().GetClient().GetLibp2PTraceGossipsubMessagePayload()

	if wallclockSlot := additional.GetWallclockSlot(); wallclockSlot != nil {
		//nolint:gosec // G115: proto uint64 values are bounded by ClickHouse column schema
		b.WallclockSlot.Append(uint32(wallclockSlot.GetNumber().GetValue()))
		b.WallclockSlotStartDateTime.Append(wallclockSlot.GetStartDateTime().AsTime())
	} else {
		b.WallclockSlot.Append(0)
		b.WallclockSlotStartDateTime.Append(time.Time{})
	}

	if wallclockEpoch := additional.GetWallclockEpoch(); wallclockEpoch != nil {
		//nolint:gosec // G115: proto uint64 values are bounded by ClickHouse column schema
		b.WallclockEpoch.Append(uint32(wallclockEpoch.GetNumber().GetValue()))
		b.WallclockEpochStartDateTime.Append(wallclockEpoch.GetStartDateTime().AsTime())
	} else {
		b.WallclockEpoch.Append(0)
		b.WallclockEpochStartDateTime.Append(time.Time{})
	}

	b.MessageID.Append(wrappedStringValue(additional.GetMessageId()))

	if msgSize := additional.GetMessageSize(); msgSize != nil {
		b.MessageSize.Append(msgSize.GetValue())
	} else {
		b.MessageSize.Append(0)
	}

	if topic := wrappedStringValue(additional.GetTopic()); topic != "" {
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

	peerID := peerIDFromMetadata(event, func(c *xatu.ClientMeta) peerIDMetadataProvider {
		return c.GetLibp2PTraceGossipsubMessagePayload()
	})

	networkName := event.GetMeta().GetClient().GetEthereum().GetNetwork().GetName()
	b.PeerIDUniqueKey.Append(computePeerIDUniqueKey(peerID, networkName))
}
