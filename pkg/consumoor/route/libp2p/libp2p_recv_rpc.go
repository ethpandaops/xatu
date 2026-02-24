package libp2p

import (
	"time"

	"github.com/ethpandaops/xatu/pkg/consumoor/route"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

var libp2pRecvRpcEventNames = []xatu.Event_Name{
	xatu.Event_LIBP2P_TRACE_RECV_RPC,
}

func init() {
	r, err := route.NewStaticRoute(
		libp2pRecvRpcTableName,
		libp2pRecvRpcEventNames,
		func() route.ColumnarBatch { return newlibp2pRecvRpcBatch() },
	)
	if err != nil {
		route.RecordError(err)

		return
	}

	if err := route.Register(r); err != nil {
		route.RecordError(err)
	}
}

func (b *libp2pRecvRpcBatch) FlattenTo(
	event *xatu.DecoratedEvent,
) error {
	if event == nil || event.GetEvent() == nil {
		return nil
	}

	b.appendRuntime(event)
	b.appendMetadata(event)
	b.appendPayload(event)
	b.rows++

	return nil
}

func (b *libp2pRecvRpcBatch) appendRuntime(event *xatu.DecoratedEvent) {
	b.UpdatedDateTime.Append(time.Now())

	if ts := event.GetEvent().GetDateTime(); ts != nil {
		b.EventDateTime.Append(ts.AsTime())
	} else {
		b.EventDateTime.Append(time.Time{})
	}
}

func (b *libp2pRecvRpcBatch) appendPayload(
	event *xatu.DecoratedEvent,
) {
	payload := event.GetLibp2PTraceRecvRpc()
	if payload == nil {
		b.UniqueKey.Append(0)
		b.PeerIDUniqueKey.Append(0)

		return
	}

	// Compute unique_key from event ID.
	if event.GetEvent() != nil && event.GetEvent().GetId() != "" {
		b.UniqueKey.Append(route.SeaHashInt64(event.GetEvent().GetId()))
	} else {
		b.UniqueKey.Append(0)
	}

	// Vector uses .data.meta.peer_id (RPCMeta.peer_id) for peer_id_unique_key.
	peerID := wrappedStringValue(payload.GetMeta().GetPeerId())
	networkName := event.GetMeta().GetClient().GetEthereum().GetNetwork().GetName()
	b.PeerIDUniqueKey.Append(computePeerIDUniqueKey(peerID, networkName))
}
