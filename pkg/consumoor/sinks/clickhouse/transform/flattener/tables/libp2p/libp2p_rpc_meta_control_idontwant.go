package libp2p

import (
	"time"

	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/flattener"
	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/metadata"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

var libp2pRpcMetaControlIdontwantEventNames = []xatu.Event_Name{
	xatu.Event_LIBP2P_TRACE_RPC_META_CONTROL_IDONTWANT,
}

func init() {
	flattener.MustRegister(flattener.NewStaticRoute(
		libp2pRpcMetaControlIdontwantTableName,
		libp2pRpcMetaControlIdontwantEventNames,
		func() flattener.ColumnarBatch { return newlibp2pRpcMetaControlIdontwantBatch() },
	))
}

func (b *libp2pRpcMetaControlIdontwantBatch) FlattenTo(
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

func (b *libp2pRpcMetaControlIdontwantBatch) appendRuntime(event *xatu.DecoratedEvent) {
	b.UpdatedDateTime.Append(time.Now())

	if ts := event.GetEvent().GetDateTime(); ts != nil {
		b.EventDateTime.Append(ts.AsTime())
	} else {
		b.EventDateTime.Append(time.Time{})
	}
}

//nolint:gosec // G115: proto uint32 values are bounded by ClickHouse column schema
func (b *libp2pRpcMetaControlIdontwantBatch) appendPayload(
	event *xatu.DecoratedEvent,
	meta *metadata.CommonMetadata,
) {
	payload := event.GetLibp2PTraceRpcMetaControlIdontwant()
	if payload == nil {
		b.UniqueKey.Append(0)
		b.RPCMetaUniqueKey.Append(0)
		b.ControlIndex.Append(0)
		b.MessageIndex.Append(0)
		b.MessageID.Append("")
		b.PeerIDUniqueKey.Append(0)

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

	if msgIdx := payload.GetMessageIndex(); msgIdx != nil {
		b.MessageIndex.Append(int32(msgIdx.GetValue()))
	} else {
		b.MessageIndex.Append(0)
	}

	b.MessageID.Append(wrappedStringValue(payload.GetMessageId()))

	peerID := wrappedStringValue(payload.GetPeerId())
	networkName := meta.MetaNetworkName
	b.PeerIDUniqueKey.Append(computePeerIDUniqueKey(peerID, networkName))
}
