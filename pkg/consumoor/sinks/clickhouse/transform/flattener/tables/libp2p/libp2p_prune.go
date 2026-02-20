package libp2p

import (
	"time"

	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/flattener"
	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/metadata"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

var libp2pPruneEventNames = []xatu.Event_Name{
	xatu.Event_LIBP2P_TRACE_PRUNE,
}

func init() {
	flattener.MustRegister(flattener.NewStaticRoute(
		libp2pPruneTableName,
		libp2pPruneEventNames,
		func() flattener.ColumnarBatch { return newlibp2pPruneBatch() },
	))
}

func (b *libp2pPruneBatch) FlattenTo(
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
	meta *metadata.CommonMetadata,
) {
	payload := event.GetLibp2PTracePrune()
	if payload == nil {
		b.TopicLayer.Append("")
		b.TopicForkDigestValue.Append("")
		b.TopicName.Append("")
		b.TopicEncoding.Append("")
		b.PeerIDUniqueKey.Append(0)

		return
	}

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

	networkName := meta.MetaNetworkName
	b.PeerIDUniqueKey.Append(computePeerIDUniqueKey(peerID, networkName))
}
