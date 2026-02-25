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
		libp2pDeliverMessageTableName,
		[]xatu.Event_Name{xatu.Event_LIBP2P_TRACE_DELIVER_MESSAGE},
		func() flattener.ColumnarBatch {
			return newlibp2pDeliverMessageBatch()
		},
	))
}

func (b *libp2pDeliverMessageBatch) FlattenTo(
	event *xatu.DecoratedEvent,
	meta *metadata.CommonMetadata,
) error {
	if event == nil || event.GetEvent() == nil {
		return nil
	}

	payload := event.GetLibp2PTraceDeliverMessage()
	if payload == nil {
		return fmt.Errorf("nil LibP2PTraceDeliverMessage payload: %w", flattener.ErrInvalidEvent)
	}

	if err := b.validate(event); err != nil {
		return err
	}

	addl := event.GetMeta().GetClient().GetLibp2PTraceDeliverMessage()

	peerID := addl.GetMetadata().GetPeerId().GetValue()
	layer, forkDigest, name, encoding := parseTopic(payload.GetTopic().GetValue())

	b.UpdatedDateTime.Append(time.Now())
	b.EventDateTime.Append(event.GetEvent().GetDateTime().AsTime())
	b.TopicLayer.Append(layer)
	b.TopicForkDigestValue.Append(forkDigest)
	b.TopicName.Append(name)
	b.TopicEncoding.Append(encoding)
	b.SeqNumber.Append(payload.GetSeqNumber().GetValue())
	b.LocalDelivery.Append(payload.GetLocal().GetValue())
	b.PeerIDUniqueKey.Append(flattener.SeaHashInt64(peerID + meta.MetaNetworkName))
	b.MessageID.Append(payload.GetMsgId().GetValue())
	b.MessageSize.Append(payload.GetMsgSize().GetValue())

	b.appendMetadata(meta)
	b.rows++

	return nil
}

func (b *libp2pDeliverMessageBatch) validate(event *xatu.DecoratedEvent) error {
	addl := event.GetMeta().GetClient().GetLibp2PTraceDeliverMessage()
	if addl == nil {
		return fmt.Errorf("nil LibP2PTraceDeliverMessage additional data: %w", flattener.ErrInvalidEvent)
	}

	if addl.GetMetadata() == nil || addl.GetMetadata().GetPeerId() == nil {
		return fmt.Errorf("nil PeerId in metadata: %w", flattener.ErrInvalidEvent)
	}

	return nil
}
