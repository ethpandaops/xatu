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
		libp2pGossipsubDataColumnSidecarTableName,
		[]xatu.Event_Name{xatu.Event_LIBP2P_TRACE_GOSSIPSUB_DATA_COLUMN_SIDECAR},
		func() flattener.ColumnarBatch {
			return newlibp2pGossipsubDataColumnSidecarBatch()
		},
	))
}

func (b *libp2pGossipsubDataColumnSidecarBatch) FlattenTo(
	event *xatu.DecoratedEvent,
	meta *metadata.CommonMetadata,
) error {
	if event == nil || event.GetEvent() == nil {
		return nil
	}

	payload := event.GetLibp2PTraceGossipsubDataColumnSidecar()
	if payload == nil {
		return fmt.Errorf("nil LibP2PTraceGossipsubDataColumnSidecar payload: %w", flattener.ErrInvalidEvent)
	}

	if err := b.validate(event); err != nil {
		return err
	}

	addl := event.GetMeta().GetClient().GetLibp2PTraceGossipsubDataColumnSidecar()

	peerID := addl.GetMetadata().GetPeerId().GetValue()

	b.UpdatedDateTime.Append(time.Now())
	b.Version.Append(gossipsubSchemaVersion)
	b.EventDateTime.Append(event.GetEvent().GetDateTime().AsTime())

	// Slot/epoch from addl data.
	b.Slot.Append(uint32(addl.GetSlot().GetNumber().GetValue())) //nolint:gosec // G115: slot fits uint32.
	b.SlotStartDateTime.Append(addl.GetSlot().GetStartDateTime().AsTime())
	b.Epoch.Append(uint32(addl.GetEpoch().GetNumber().GetValue())) //nolint:gosec // G115: epoch fits uint32.
	b.EpochStartDateTime.Append(addl.GetEpoch().GetStartDateTime().AsTime())

	// Wallclock from addl data.
	b.WallclockSlot.Append(uint32(addl.GetWallclockSlot().GetNumber().GetValue())) //nolint:gosec // G115: wallclock slot fits uint32.
	b.WallclockSlotStartDateTime.Append(addl.GetWallclockSlot().GetStartDateTime().AsTime())
	b.WallclockEpoch.Append(uint32(addl.GetWallclockEpoch().GetNumber().GetValue())) //nolint:gosec // G115: wallclock epoch fits uint32.
	b.WallclockEpochStartDateTime.Append(addl.GetWallclockEpoch().GetStartDateTime().AsTime())

	b.PropagationSlotStartDiff.Append(uint32(addl.GetPropagation().GetSlotStartDiff().GetValue())) //nolint:gosec // G115: propagation diff fits uint32.
	b.ProposerIndex.Append(uint32(payload.GetProposerIndex().GetValue()))                          //nolint:gosec // G115: proposer index fits uint32.
	b.ColumnIndex.Append(payload.GetIndex().GetValue())
	b.KzgCommitmentsCount.Append(payload.GetKzgCommitmentsCount().GetValue())
	b.BeaconBlockRoot.Append([]byte(payload.GetBlockRoot().GetValue()))
	b.ParentRoot.Append([]byte(payload.GetParentRoot().GetValue()))
	b.StateRoot.Append([]byte(payload.GetStateRoot().GetValue()))
	b.PeerIDUniqueKey.Append(flattener.SeaHashInt64(peerID + meta.MetaNetworkName))
	b.MessageID.Append(addl.GetMessageId().GetValue())
	b.MessageSize.Append(addl.GetMessageSize().GetValue())

	topic := addl.GetTopic().GetValue()
	layer, forkDigest, name, encoding := parseTopic(topic)
	b.TopicLayer.Append(layer)
	b.TopicForkDigestValue.Append(forkDigest)
	b.TopicName.Append(name)
	b.TopicEncoding.Append(encoding)

	b.appendMetadata(meta)
	b.rows++

	return nil
}

func (b *libp2pGossipsubDataColumnSidecarBatch) validate(event *xatu.DecoratedEvent) error {
	addl := event.GetMeta().GetClient().GetLibp2PTraceGossipsubDataColumnSidecar()
	if addl == nil {
		return fmt.Errorf("nil LibP2PTraceGossipsubDataColumnSidecar additional data: %w", flattener.ErrInvalidEvent)
	}

	if addl.GetMetadata() == nil || addl.GetMetadata().GetPeerId() == nil {
		return fmt.Errorf("nil PeerId in metadata: %w", flattener.ErrInvalidEvent)
	}

	if addl.GetSlot() == nil || addl.GetSlot().GetNumber() == nil {
		return fmt.Errorf("nil Slot: %w", flattener.ErrInvalidEvent)
	}

	if addl.GetEpoch() == nil || addl.GetEpoch().GetNumber() == nil {
		return fmt.Errorf("nil Epoch: %w", flattener.ErrInvalidEvent)
	}

	if addl.GetWallclockSlot() == nil || addl.GetWallclockSlot().GetNumber() == nil {
		return fmt.Errorf("nil WallclockSlot: %w", flattener.ErrInvalidEvent)
	}

	if addl.GetWallclockEpoch() == nil || addl.GetWallclockEpoch().GetNumber() == nil {
		return fmt.Errorf("nil WallclockEpoch: %w", flattener.ErrInvalidEvent)
	}

	if addl.GetPropagation() == nil || addl.GetPropagation().GetSlotStartDiff() == nil {
		return fmt.Errorf("nil PropagationSlotStartDiff: %w", flattener.ErrInvalidEvent)
	}

	payload := event.GetLibp2PTraceGossipsubDataColumnSidecar()

	if payload.GetProposerIndex() == nil {
		return fmt.Errorf("nil ProposerIndex: %w", flattener.ErrInvalidEvent)
	}

	return nil
}
