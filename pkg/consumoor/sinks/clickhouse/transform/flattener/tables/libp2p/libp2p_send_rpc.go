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
		libp2pSendRpcTableName,
		[]xatu.Event_Name{xatu.Event_LIBP2P_TRACE_SEND_RPC},
		func() flattener.ColumnarBatch {
			return newlibp2pSendRpcBatch()
		},
	))
}

func (b *libp2pSendRpcBatch) FlattenTo(
	event *xatu.DecoratedEvent,
	meta *metadata.CommonMetadata,
) error {
	if event == nil || event.GetEvent() == nil {
		return nil
	}

	payload := event.GetLibp2PTraceSendRpc()
	if payload == nil {
		return fmt.Errorf("nil LibP2PTraceSendRpc payload: %w", flattener.ErrInvalidEvent)
	}

	if err := b.validate(event); err != nil {
		return err
	}

	addl := event.GetMeta().GetClient().GetLibp2PTraceSendRpc()

	peerID := addl.GetMetadata().GetPeerId().GetValue()

	b.UniqueKey.Append(flattener.SeaHashInt64(event.GetEvent().GetId()))
	b.UpdatedDateTime.Append(time.Now())
	b.EventDateTime.Append(event.GetEvent().GetDateTime().AsTime())
	b.PeerIDUniqueKey.Append(flattener.SeaHashInt64(peerID + meta.MetaNetworkName))

	b.appendMetadata(meta)
	b.rows++

	return nil
}

func (b *libp2pSendRpcBatch) validate(event *xatu.DecoratedEvent) error {
	addl := event.GetMeta().GetClient().GetLibp2PTraceSendRpc()
	if addl == nil {
		return fmt.Errorf("nil LibP2PTraceSendRpc additional data: %w", flattener.ErrInvalidEvent)
	}

	if addl.GetMetadata() == nil || addl.GetMetadata().GetPeerId() == nil {
		return fmt.Errorf("nil PeerId in metadata: %w", flattener.ErrInvalidEvent)
	}

	return nil
}
