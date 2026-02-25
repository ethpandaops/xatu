package libp2p

import (
	"fmt"
	"time"

	"github.com/ethpandaops/xatu/pkg/consumoor/route"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

var libp2pDropRpcEventNames = []xatu.Event_Name{
	xatu.Event_LIBP2P_TRACE_DROP_RPC,
}

func init() {
	r, err := route.NewStaticRoute(
		libp2pDropRpcTableName,
		libp2pDropRpcEventNames,
		func() route.ColumnarBatch { return newlibp2pDropRpcBatch() },
	)
	if err != nil {
		route.RecordError(err)

		return
	}

	if err := route.Register(r); err != nil {
		route.RecordError(err)
	}
}

func (b *libp2pDropRpcBatch) FlattenTo(
	event *xatu.DecoratedEvent,
) error {
	if event == nil || event.GetEvent() == nil {
		return nil
	}

	if event.GetLibp2PTraceDropRpc() == nil {
		return fmt.Errorf("nil libp2p_trace_drop_rpc payload: %w", route.ErrInvalidEvent)
	}

	b.appendRuntime(event)
	b.appendMetadata(event)
	b.appendPayload(event)
	b.rows++

	return nil
}

func (b *libp2pDropRpcBatch) appendRuntime(event *xatu.DecoratedEvent) {
	b.UpdatedDateTime.Append(time.Now())

	if ts := event.GetEvent().GetDateTime(); ts != nil {
		b.EventDateTime.Append(ts.AsTime())
	} else {
		b.EventDateTime.Append(time.Time{})
	}
}

func (b *libp2pDropRpcBatch) appendPayload(
	event *xatu.DecoratedEvent,
) {
	payload := event.GetLibp2PTraceDropRpc()
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
