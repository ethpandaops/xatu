package libp2p

import (
	"time"

	"github.com/ClickHouse/ch-go/proto"
	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/flattener"
	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/metadata"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

var libp2pRpcMetaControlPruneEventNames = []xatu.Event_Name{
	xatu.Event_LIBP2P_TRACE_RPC_META_CONTROL_PRUNE,
}

func init() {
	flattener.MustRegister(flattener.NewStaticRoute(
		libp2pRpcMetaControlPruneTableName,
		libp2pRpcMetaControlPruneEventNames,
		func() flattener.ColumnarBatch { return newlibp2pRpcMetaControlPruneBatch() },
	))
}

func (b *libp2pRpcMetaControlPruneBatch) FlattenTo(
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

func (b *libp2pRpcMetaControlPruneBatch) appendRuntime(event *xatu.DecoratedEvent) {
	b.UpdatedDateTime.Append(time.Now())

	if ts := event.GetEvent().GetDateTime(); ts != nil {
		b.EventDateTime.Append(ts.AsTime())
	} else {
		b.EventDateTime.Append(time.Time{})
	}
}

//nolint:gosec // G115: proto uint32 values are bounded by ClickHouse column schema
func (b *libp2pRpcMetaControlPruneBatch) appendPayload(
	event *xatu.DecoratedEvent,
	meta *metadata.CommonMetadata,
) {
	payload := event.GetLibp2PTraceRpcMetaControlPrune()
	if payload == nil {
		b.UniqueKey.Append(0)
		b.RPCMetaUniqueKey.Append(0)
		b.ControlIndex.Append(0)
		b.PeerIDIndex.Append(0)
		b.PeerIDUniqueKey.Append(0)
		b.GraftPeerIDUniqueKey.Append(proto.Nullable[int64]{})
		b.TopicLayer.Append("")
		b.TopicForkDigestValue.Append("")
		b.TopicName.Append("")
		b.TopicEncoding.Append("")

		return
	}

	// Compute unique_key from the event ID.
	if eventID := event.GetEvent().GetId(); eventID != "" {
		b.UniqueKey.Append(flattener.SeaHashInt64(eventID))
	} else {
		b.UniqueKey.Append(0)
	}

	// Compute rpc_meta_unique_key from root_event_id.
	rootEventID := wrappedStringValue(payload.GetRootEventId())
	if rootEventID != "" {
		b.RPCMetaUniqueKey.Append(computeRPCMetaUniqueKey(rootEventID))
	} else {
		b.RPCMetaUniqueKey.Append(0)
	}

	if controlIdx := payload.GetControlIndex(); controlIdx != nil {
		b.ControlIndex.Append(int32(controlIdx.GetValue()))
	} else {
		b.ControlIndex.Append(0)
	}

	if peerIdx := payload.GetPeerIndex(); peerIdx != nil {
		b.PeerIDIndex.Append(int32(peerIdx.GetValue()))
	} else {
		b.PeerIDIndex.Append(0)
	}

	networkName := meta.MetaNetworkName

	peerID := wrappedStringValue(payload.GetPeerId())
	b.PeerIDUniqueKey.Append(computePeerIDUniqueKey(peerID, networkName))

	// Compute graft_peer_id_unique_key (nullable).
	graftPeerID := wrappedStringValue(payload.GetGraftPeerId())
	if graftPeerID != "" {
		b.GraftPeerIDUniqueKey.Append(proto.NewNullable[int64](computePeerIDUniqueKey(graftPeerID, networkName)))
	} else {
		b.GraftPeerIDUniqueKey.Append(proto.NewNullable[int64](flattener.SeaHashInt64(networkName)))
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
}
