package libp2p

import (
	"time"

	"github.com/ethpandaops/xatu/pkg/consumoor/route"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

var libp2pPeerEventNames = []xatu.Event_Name{
	xatu.Event_LIBP2P_TRACE_CONNECTED,
	xatu.Event_LIBP2P_TRACE_DISCONNECTED,
	xatu.Event_LIBP2P_TRACE_ADD_PEER,
	xatu.Event_LIBP2P_TRACE_REMOVE_PEER,
	xatu.Event_LIBP2P_TRACE_RECV_RPC,
	xatu.Event_LIBP2P_TRACE_SEND_RPC,
	xatu.Event_LIBP2P_TRACE_DROP_RPC,
	xatu.Event_LIBP2P_TRACE_GRAFT,
	xatu.Event_LIBP2P_TRACE_PRUNE,
}

func init() {
	r, err := route.NewStaticRoute(
		libp2pPeerTableName,
		libp2pPeerEventNames,
		func() route.ColumnarBatch { return newlibp2pPeerBatch() },
	)
	if err != nil {
		route.RecordError(err)

		return
	}

	if err := route.Register(r); err != nil {
		route.RecordError(err)
	}
}

//nolint:gosec // G115: SeaHash64 returns uint64, stored as int64 for ClickHouse column schema
func (b *libp2pPeerBatch) FlattenTo(
	event *xatu.DecoratedEvent,
) error {
	if event == nil || event.GetEvent() == nil {
		return nil
	}

	peerID := upsertPeerIDFromEvent(event)
	if peerID == "" {
		return nil
	}

	networkName := event.GetMeta().GetClient().GetEthereum().GetNetwork().GetName()

	b.UniqueKey.Append(int64(route.SeaHash64(peerID + networkName)))
	b.UpdatedDateTime.Append(time.Now())
	b.PeerID.Append(peerID)
	b.appendMetadata(event)
	b.rows++

	return nil
}

func upsertPeerIDFromEvent(event *xatu.DecoratedEvent) string {
	if event == nil || event.GetEvent() == nil {
		return ""
	}

	switch event.GetEvent().GetName() {
	case xatu.Event_LIBP2P_TRACE_CONNECTED:
		return wrappedStringValue(event.GetLibp2PTraceConnected().GetRemotePeer())
	case xatu.Event_LIBP2P_TRACE_DISCONNECTED:
		return wrappedStringValue(event.GetLibp2PTraceDisconnected().GetRemotePeer())
	case xatu.Event_LIBP2P_TRACE_ADD_PEER:
		return wrappedStringValue(event.GetLibp2PTraceAddPeer().GetPeerId())
	case xatu.Event_LIBP2P_TRACE_REMOVE_PEER:
		return wrappedStringValue(event.GetLibp2PTraceRemovePeer().GetPeerId())
	case xatu.Event_LIBP2P_TRACE_RECV_RPC:
		return wrappedStringValue(event.GetLibp2PTraceRecvRpc().GetPeerId())
	case xatu.Event_LIBP2P_TRACE_SEND_RPC:
		return wrappedStringValue(event.GetLibp2PTraceSendRpc().GetPeerId())
	case xatu.Event_LIBP2P_TRACE_DROP_RPC:
		return wrappedStringValue(event.GetLibp2PTraceDropRpc().GetPeerId())
	case xatu.Event_LIBP2P_TRACE_GRAFT:
		return wrappedStringValue(event.GetLibp2PTraceGraft().GetPeerId())
	case xatu.Event_LIBP2P_TRACE_PRUNE:
		return wrappedStringValue(event.GetLibp2PTracePrune().GetPeerId())
	default:
		return ""
	}
}
