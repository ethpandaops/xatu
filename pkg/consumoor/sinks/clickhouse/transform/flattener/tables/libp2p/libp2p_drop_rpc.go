package libp2p

import (
	"time"

	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/flattener"
	catalog "github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/flattener/tables/catalog"
	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/metadata"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

var libp2pDropRpcEventNames = []xatu.Event_Name{
	xatu.Event_LIBP2P_TRACE_DROP_RPC,
}

func init() {
	catalog.MustRegister(flattener.NewStaticRoute(
		libp2pDropRpcTableName,
		libp2pDropRpcEventNames,
		func() flattener.ColumnarBatch { return newlibp2pDropRpcBatch() },
	))
}

func (b *libp2pDropRpcBatch) FlattenTo(
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
	meta *metadata.CommonMetadata,
) {
	payload := event.GetLibp2PTraceDropRpc()
	if payload == nil {
		b.UniqueKey.Append(0)
		b.PeerIDUniqueKey.Append(0)

		return
	}

	peerID := wrappedStringValue(payload.GetPeerId())

	// Compute unique_key from event ID.
	if event.GetEvent() != nil && event.GetEvent().GetId() != "" {
		b.UniqueKey.Append(flattener.SeaHashInt64(event.GetEvent().GetId()))
	} else {
		b.UniqueKey.Append(0)
	}

	networkName := meta.MetaNetworkName
	b.PeerIDUniqueKey.Append(computePeerIDUniqueKey(peerID, networkName))
}
