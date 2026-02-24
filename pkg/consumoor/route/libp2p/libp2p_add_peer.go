package libp2p

import (
	"fmt"
	"time"

	"github.com/ethpandaops/xatu/pkg/consumoor/route"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

var libp2pAddPeerEventNames = []xatu.Event_Name{
	xatu.Event_LIBP2P_TRACE_ADD_PEER,
}

func init() {
	r, err := route.NewStaticRoute(
		libp2pAddPeerTableName,
		libp2pAddPeerEventNames,
		func() route.ColumnarBatch { return newlibp2pAddPeerBatch() },
	)
	if err != nil {
		route.RecordError(err)

		return
	}

	if err := route.Register(r); err != nil {
		route.RecordError(err)
	}
}

func (b *libp2pAddPeerBatch) FlattenTo(
	event *xatu.DecoratedEvent,
) error {
	if event == nil || event.GetEvent() == nil {
		return nil
	}

	if event.GetLibp2PTraceAddPeer() == nil {
		return fmt.Errorf("nil libp2p_trace_add_peer payload: %w", route.ErrInvalidEvent)
	}

	b.appendRuntime(event)
	b.appendMetadata(event)
	b.appendPayload(event)
	b.rows++

	return nil
}

func (b *libp2pAddPeerBatch) appendRuntime(event *xatu.DecoratedEvent) {
	b.UpdatedDateTime.Append(time.Now())

	if ts := event.GetEvent().GetDateTime(); ts != nil {
		b.EventDateTime.Append(ts.AsTime())
	} else {
		b.EventDateTime.Append(time.Time{})
	}
}

func (b *libp2pAddPeerBatch) appendPayload(
	event *xatu.DecoratedEvent,
) {
	payload := event.GetLibp2PTraceAddPeer()
	peerID := wrappedStringValue(payload.GetPeerId())
	b.Protocol.Append(wrappedStringValue(payload.GetProtocol()))

	networkName := event.GetMeta().GetClient().GetEthereum().GetNetwork().GetName()
	b.PeerIDUniqueKey.Append(computePeerIDUniqueKey(peerID, networkName))
}
