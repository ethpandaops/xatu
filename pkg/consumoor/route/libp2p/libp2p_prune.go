package libp2p

import (
	"fmt"
	"time"

	"github.com/ethpandaops/xatu/pkg/consumoor/route"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

var libp2pPruneEventNames = []xatu.Event_Name{
	xatu.Event_LIBP2P_TRACE_PRUNE,
}

func init() {
	r, err := route.NewStaticRoute(
		libp2pPruneTableName,
		libp2pPruneEventNames,
		func() route.ColumnarBatch { return newlibp2pPruneBatch() },
	)
	if err != nil {
		route.RecordError(err)

		return
	}

	if err := route.Register(r); err != nil {
		route.RecordError(err)
	}
}

func (b *libp2pPruneBatch) FlattenTo(
	event *xatu.DecoratedEvent,
) error {
	if event == nil || event.GetEvent() == nil {
		return nil
	}

	if event.GetLibp2PTracePrune() == nil {
		return fmt.Errorf("nil libp2p_trace_prune payload: %w", route.ErrInvalidEvent)
	}

	b.appendRuntime(event)
	b.appendMetadata(event)
	b.appendPayload(event)
	b.rows++

	return nil
}

func (b *libp2pPruneBatch) appendRuntime(event *xatu.DecoratedEvent) {
	b.UpdatedDateTime.Append(time.Now())

	if ts := event.GetEvent().GetDateTime(); ts != nil {
		b.EventDateTime.Append(ts.AsTime())
	} else {
		b.EventDateTime.Append(time.Time{})
	}
}

func (b *libp2pPruneBatch) appendPayload(
	event *xatu.DecoratedEvent,
) {
	payload := event.GetLibp2PTracePrune()
	peerID := wrappedStringValue(payload.GetPeerId())

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

	// Prefer client metadata peer ID over payload peer ID.
	metaPeerID := peerIDFromMetadata(event, func(c *xatu.ClientMeta) peerIDMetadataProvider {
		return c.GetLibp2PTracePrune()
	})

	if metaPeerID != "" {
		peerID = metaPeerID
	}

	networkName := event.GetMeta().GetClient().GetEthereum().GetNetwork().GetName()
	b.PeerIDUniqueKey.Append(computePeerIDUniqueKey(peerID, networkName))
}
