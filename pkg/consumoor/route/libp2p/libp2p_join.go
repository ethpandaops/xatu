package libp2p

import (
	"fmt"
	"time"

	"github.com/ethpandaops/xatu/pkg/consumoor/route"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

var libp2pJoinEventNames = []xatu.Event_Name{
	xatu.Event_LIBP2P_TRACE_JOIN,
}

func init() {
	r, err := route.NewStaticRoute(
		libp2pJoinTableName,
		libp2pJoinEventNames,
		func() route.ColumnarBatch { return newlibp2pJoinBatch() },
	)
	if err != nil {
		route.RecordError(err)

		return
	}

	if err := route.Register(r); err != nil {
		route.RecordError(err)
	}
}

func (b *libp2pJoinBatch) FlattenTo(
	event *xatu.DecoratedEvent,
) error {
	if event == nil || event.GetEvent() == nil {
		return nil
	}

	if event.GetLibp2PTraceJoin() == nil {
		return fmt.Errorf("nil libp2p_trace_join payload: %w", route.ErrInvalidEvent)
	}

	if err := b.validate(event); err != nil {
		return err
	}

	b.appendRuntime(event)
	b.appendMetadata(event)
	b.appendPayload(event)
	b.rows++

	return nil
}

func (b *libp2pJoinBatch) validate(event *xatu.DecoratedEvent) error {
	peerID := peerIDFromMetadata(event, func(c *xatu.ClientMeta) peerIDMetadataProvider {
		return c.GetLibp2PTraceJoin()
	})

	if peerID == "" {
		return fmt.Errorf("nil PeerId: %w", route.ErrInvalidEvent)
	}

	return nil
}

func (b *libp2pJoinBatch) appendRuntime(event *xatu.DecoratedEvent) {
	b.UpdatedDateTime.Append(time.Now())

	if ts := event.GetEvent().GetDateTime(); ts != nil {
		b.EventDateTime.Append(ts.AsTime())
	} else {
		b.EventDateTime.Append(time.Time{})
	}
}

func (b *libp2pJoinBatch) appendPayload(
	event *xatu.DecoratedEvent,
) {
	payload := event.GetLibp2PTraceJoin()
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
		return c.GetLibp2PTraceJoin()
	})

	networkName := event.GetMeta().GetClient().GetEthereum().GetNetwork().GetName()
	b.PeerIDUniqueKey.Append(computePeerIDUniqueKey(peerID, networkName))
}
