package libp2p

import (
	"fmt"
	"time"

	"github.com/ClickHouse/ch-go/proto"
	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/flattener"
	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/metadata"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

func init() {
	flattener.MustRegister(flattener.NewStaticRoute(
		libp2pRpcMetaControlPruneTableName,
		[]xatu.Event_Name{xatu.Event_LIBP2P_TRACE_RPC_META_CONTROL_PRUNE},
		func() flattener.ColumnarBatch {
			return newlibp2pRpcMetaControlPruneBatch()
		},
	))
}

func (b *libp2pRpcMetaControlPruneBatch) FlattenTo(
	event *xatu.DecoratedEvent,
	meta *metadata.CommonMetadata,
) error {
	if event == nil || event.GetEvent() == nil {
		return nil
	}

	payload := event.GetLibp2PTraceRpcMetaControlPrune()
	if payload == nil {
		return fmt.Errorf("nil LibP2PTraceRpcMetaControlPrune payload: %w", flattener.ErrInvalidEvent)
	}

	if err := b.validate(event); err != nil {
		return err
	}

	addl := event.GetMeta().GetClient().GetLibp2PTraceRpcMetaControlPrune()

	peerID := addl.GetMetadata().GetPeerId().GetValue()

	b.UniqueKey.Append(flattener.SeaHashInt64(event.GetEvent().GetId()))
	b.UpdatedDateTime.Append(time.Now())
	b.EventDateTime.Append(event.GetEvent().GetDateTime().AsTime())
	b.ControlIndex.Append(int32(payload.GetControlIndex().GetValue())) //nolint:gosec // G115: control index fits int32.
	b.RPCMetaUniqueKey.Append(flattener.SeaHashInt64(payload.GetRootEventId().GetValue()))
	b.PeerIDIndex.Append(int32(payload.GetPeerIndex().GetValue())) //nolint:gosec // G115: peer index fits int32.
	b.PeerIDUniqueKey.Append(flattener.SeaHashInt64(peerID + meta.MetaNetworkName))

	// GraftPeerIDUniqueKey is nullable.
	if graftPeerID := payload.GetGraftPeerId().GetValue(); graftPeerID != "" {
		b.GraftPeerIDUniqueKey.Append(proto.NewNullable(flattener.SeaHashInt64(graftPeerID + meta.MetaNetworkName)))
	} else {
		b.GraftPeerIDUniqueKey.Append(proto.Nullable[int64]{})
	}

	topic := payload.GetTopic().GetValue()
	layer, forkDigest, name, encoding := parseTopic(topic)
	b.TopicLayer.Append(layer)
	b.TopicForkDigestValue.Append(forkDigest)
	b.TopicName.Append(name)
	b.TopicEncoding.Append(encoding)

	b.appendMetadata(meta)
	b.rows++

	return nil
}

func (b *libp2pRpcMetaControlPruneBatch) validate(event *xatu.DecoratedEvent) error {
	addl := event.GetMeta().GetClient().GetLibp2PTraceRpcMetaControlPrune()
	if addl == nil {
		return fmt.Errorf("nil LibP2PTraceRpcMetaControlPrune additional data: %w", flattener.ErrInvalidEvent)
	}

	if addl.GetMetadata() == nil || addl.GetMetadata().GetPeerId() == nil {
		return fmt.Errorf("nil PeerId in metadata: %w", flattener.ErrInvalidEvent)
	}

	return nil
}
