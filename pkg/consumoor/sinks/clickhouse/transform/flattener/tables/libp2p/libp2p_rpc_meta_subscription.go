package libp2p

import (
	"fmt"
	"time"

	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/flattener"
	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/metadata"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

func init() {
	flattener.MustRegister(flattener.NewStaticRoute(
		libp2pRpcMetaSubscriptionTableName,
		[]xatu.Event_Name{xatu.Event_LIBP2P_TRACE_RPC_META_SUBSCRIPTION},
		func() flattener.ColumnarBatch {
			return newlibp2pRpcMetaSubscriptionBatch()
		},
	))
}

func (b *libp2pRpcMetaSubscriptionBatch) FlattenTo(
	event *xatu.DecoratedEvent,
	meta *metadata.CommonMetadata,
) error {
	if event == nil || event.GetEvent() == nil {
		return nil
	}

	payload := event.GetLibp2PTraceRpcMetaSubscription()
	if payload == nil {
		return fmt.Errorf("nil LibP2PTraceRpcMetaSubscription payload: %w", flattener.ErrInvalidEvent)
	}

	if err := b.validate(event); err != nil {
		return err
	}

	addl := event.GetMeta().GetClient().GetLibp2PTraceRpcMetaSubscription()

	peerID := addl.GetMetadata().GetPeerId().GetValue()

	b.UniqueKey.Append(flattener.SeaHashInt64(event.GetEvent().GetId()))
	b.UpdatedDateTime.Append(time.Now())
	b.EventDateTime.Append(event.GetEvent().GetDateTime().AsTime())
	b.ControlIndex.Append(int32(payload.GetControlIndex().GetValue())) //nolint:gosec // G115: control index fits int32.
	b.RPCMetaUniqueKey.Append(flattener.SeaHashInt64(payload.GetRootEventId().GetValue()))
	b.Subscribe.Append(payload.GetSubscribe().GetValue())

	topic := payload.GetTopicId().GetValue()
	layer, forkDigest, name, encoding := parseTopic(topic)
	b.TopicLayer.Append(layer)
	b.TopicForkDigestValue.Append(forkDigest)
	b.TopicName.Append(name)
	b.TopicEncoding.Append(encoding)

	b.PeerIDUniqueKey.Append(flattener.SeaHashInt64(peerID + meta.MetaNetworkName))

	b.appendMetadata(meta)
	b.rows++

	return nil
}

func (b *libp2pRpcMetaSubscriptionBatch) validate(event *xatu.DecoratedEvent) error {
	addl := event.GetMeta().GetClient().GetLibp2PTraceRpcMetaSubscription()
	if addl == nil {
		return fmt.Errorf("nil LibP2PTraceRpcMetaSubscription additional data: %w", flattener.ErrInvalidEvent)
	}

	if addl.GetMetadata() == nil || addl.GetMetadata().GetPeerId() == nil {
		return fmt.Errorf("nil PeerId in metadata: %w", flattener.ErrInvalidEvent)
	}

	return nil
}
