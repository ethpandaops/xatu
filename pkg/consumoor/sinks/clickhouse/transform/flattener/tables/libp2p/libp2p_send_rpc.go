package libp2p

import (
	"time"

	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/flattener"
	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/metadata"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

var libp2pSendRpcEventNames = []xatu.Event_Name{
	xatu.Event_LIBP2P_TRACE_SEND_RPC,
}

func init() {
	flattener.MustRegister(flattener.NewStaticRoute(
		libp2pSendRpcTableName,
		libp2pSendRpcEventNames,
		func() flattener.ColumnarBatch { return newlibp2pSendRpcBatch() },
	))
}

func (b *libp2pSendRpcBatch) FlattenTo(
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

func (b *libp2pSendRpcBatch) appendRuntime(event *xatu.DecoratedEvent) {
	b.UpdatedDateTime.Append(time.Now())

	if ts := event.GetEvent().GetDateTime(); ts != nil {
		b.EventDateTime.Append(ts.AsTime())
	} else {
		b.EventDateTime.Append(time.Time{})
	}
}

func (b *libp2pSendRpcBatch) appendPayload(
	event *xatu.DecoratedEvent,
	meta *metadata.CommonMetadata,
) {
	payload := event.GetLibp2PTraceSendRpc()
	if payload == nil {
		b.UniqueKey.Append(0)
		b.PeerIDUniqueKey.Append(0)

		return
	}

	// Compute unique_key from event ID.
	if event.GetEvent() != nil && event.GetEvent().GetId() != "" {
		b.UniqueKey.Append(flattener.SeaHashInt64(event.GetEvent().GetId()))
	} else {
		b.UniqueKey.Append(0)
	}

	// Vector uses .data.meta.peer_id (RPCMeta.peer_id) for peer_id_unique_key.
	peerID := wrappedStringValue(payload.GetMeta().GetPeerId())
	networkName := meta.MetaNetworkName
	b.PeerIDUniqueKey.Append(computePeerIDUniqueKey(peerID, networkName))
}
