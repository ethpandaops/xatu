package libp2p

import (
	"fmt"
	"strconv"
	"time"

	"github.com/ethpandaops/xatu/pkg/consumoor/route"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

var libp2pRpcMetaControlIdontwantEventNames = []xatu.Event_Name{
	xatu.Event_LIBP2P_TRACE_RPC_META_CONTROL_IDONTWANT,
}

func init() {
	r, err := route.NewStaticRoute(
		libp2pRpcMetaControlIdontwantTableName,
		libp2pRpcMetaControlIdontwantEventNames,
		func() route.ColumnarBatch { return newlibp2pRpcMetaControlIdontwantBatch() },
	)
	if err != nil {
		route.RecordError(err)

		return
	}

	if err := route.Register(r); err != nil {
		route.RecordError(err)
	}
}

func (b *libp2pRpcMetaControlIdontwantBatch) FlattenTo(
	event *xatu.DecoratedEvent,
) error {
	if event == nil || event.GetEvent() == nil {
		return nil
	}

	if event.GetLibp2PTraceRpcMetaControlIdontwant() == nil {
		return fmt.Errorf("nil libp2p_trace_rpc_meta_control_idontwant payload: %w", route.ErrInvalidEvent)
	}

	b.appendRuntime(event)
	b.appendMetadata(event)
	b.appendPayload(event)
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
) {
	payload := event.GetLibp2PTraceRpcMetaControlIdontwant()
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
			rootEventID, "rpc_meta_control_idontwant",
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
	networkName := event.GetMeta().GetClient().GetEthereum().GetNetwork().GetName()
	b.PeerIDUniqueKey.Append(computePeerIDUniqueKey(peerID, networkName))
}
