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
		libp2pPeerTableName,
		[]xatu.Event_Name{
			xatu.Event_LIBP2P_TRACE_CONNECTED,
			xatu.Event_LIBP2P_TRACE_DISCONNECTED,
		},
		func() flattener.ColumnarBatch {
			return newlibp2pPeerBatch()
		},
	))
}

func (b *libp2pPeerBatch) FlattenTo(
	event *xatu.DecoratedEvent,
	meta *metadata.CommonMetadata,
) error {
	if event == nil || event.GetEvent() == nil {
		return nil
	}

	peerID, err := b.extractPeerID(event)
	if err != nil {
		return err
	}

	b.UniqueKey.Append(flattener.SeaHashInt64(peerID + meta.MetaNetworkName))
	b.UpdatedDateTime.Append(time.Now())
	b.PeerID.Append(peerID)

	b.appendMetadata(meta)
	b.rows++

	return nil
}

func (b *libp2pPeerBatch) extractPeerID(event *xatu.DecoratedEvent) (string, error) {
	switch event.GetEvent().GetName() {
	case xatu.Event_LIBP2P_TRACE_CONNECTED:
		payload := event.GetLibp2PTraceConnected()
		if payload == nil {
			return "", fmt.Errorf("nil LibP2PTraceConnected payload: %w", flattener.ErrInvalidEvent)
		}

		peerID := payload.GetRemotePeer().GetValue()
		if peerID == "" {
			return "", fmt.Errorf("empty RemotePeer: %w", flattener.ErrInvalidEvent)
		}

		return peerID, nil

	case xatu.Event_LIBP2P_TRACE_DISCONNECTED:
		payload := event.GetLibp2PTraceDisconnected()
		if payload == nil {
			return "", fmt.Errorf("nil LibP2PTraceDisconnected payload: %w", flattener.ErrInvalidEvent)
		}

		peerID := payload.GetRemotePeer().GetValue()
		if peerID == "" {
			return "", fmt.Errorf("empty RemotePeer: %w", flattener.ErrInvalidEvent)
		}

		return peerID, nil

	default:
		return "", fmt.Errorf("unexpected event type %s: %w", event.GetEvent().GetName(), flattener.ErrInvalidEvent)
	}
}
