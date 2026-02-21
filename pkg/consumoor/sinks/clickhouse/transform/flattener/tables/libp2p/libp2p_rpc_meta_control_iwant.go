package libp2p

import (
	"strconv"
	"time"

	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/flattener"
	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/metadata"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

var libp2pRpcMetaControlIwantEventNames = []xatu.Event_Name{
	xatu.Event_LIBP2P_TRACE_RPC_META_CONTROL_IWANT,
}

func init() {
	flattener.MustRegister(flattener.NewStaticRoute(
		libp2pRpcMetaControlIwantTableName,
		libp2pRpcMetaControlIwantEventNames,
		func() flattener.ColumnarBatch { return newlibp2pRpcMetaControlIwantBatch() },
	))
}

func (b *libp2pRpcMetaControlIwantBatch) FlattenTo(
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

func (b *libp2pRpcMetaControlIwantBatch) appendRuntime(event *xatu.DecoratedEvent) {
	b.UpdatedDateTime.Append(time.Now())

	if ts := event.GetEvent().GetDateTime(); ts != nil {
		b.EventDateTime.Append(ts.AsTime())
	} else {
		b.EventDateTime.Append(time.Time{})
	}
}

//nolint:gosec // G115: proto uint32 values are bounded by ClickHouse column schema
func (b *libp2pRpcMetaControlIwantBatch) appendPayload(
	event *xatu.DecoratedEvent,
	meta *metadata.CommonMetadata,
) {
	payload := event.GetLibp2PTraceRpcMetaControlIwant()
	if payload == nil {
		b.UniqueKey.Append(0)
		b.RPCMetaUniqueKey.Append(0)
		b.ControlIndex.Append(0)
		b.MessageIndex.Append(0)
		b.MessageID.Append("")
		b.PeerIDUniqueKey.Append(0)

		return
	}

	// Extract values needed for key computations.
	rootEventID := wrappedStringValue(payload.GetRootEventId())

	var controlIdxVal, msgIdxVal uint32
	if idx := payload.GetControlIndex(); idx != nil {
		controlIdxVal = idx.GetValue()
	}

	if idx := payload.GetMessageIndex(); idx != nil {
		msgIdxVal = idx.GetValue()
	}

	// Compute unique_key as composite key matching Vector's VRL format.
	if rootEventID != "" {
		b.UniqueKey.Append(computeRPCMetaChildUniqueKey(
			rootEventID, "rpc_meta_control_iwant",
			strconv.FormatUint(uint64(controlIdxVal), 10),
			strconv.FormatUint(uint64(msgIdxVal), 10)))
	} else {
		b.UniqueKey.Append(0)
	}

	// Compute rpc_meta_unique_key from root_event_id.
	if rootEventID != "" {
		b.RPCMetaUniqueKey.Append(computeRPCMetaUniqueKey(rootEventID))
	} else {
		b.RPCMetaUniqueKey.Append(0)
	}

	b.ControlIndex.Append(int32(controlIdxVal))
	b.MessageIndex.Append(int32(msgIdxVal))

	b.MessageID.Append(wrappedStringValue(payload.GetMessageId()))

	peerID := wrappedStringValue(payload.GetPeerId())
	networkName := meta.MetaNetworkName
	b.PeerIDUniqueKey.Append(computePeerIDUniqueKey(peerID, networkName))
}
