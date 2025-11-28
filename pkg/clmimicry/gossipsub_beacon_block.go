package clmimicry

import (
	"context"
	"fmt"

	"github.com/OffchainLabs/prysm/v7/consensus-types/primitives"
	"github.com/ethpandaops/xatu/pkg/proto/libp2p"
	"github.com/ethpandaops/xatu/pkg/proto/libp2p/gossipsub"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/google/uuid"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func (p *Processor) handleGossipBeaconBlock(
	ctx context.Context,
	event *BeaconBlockEvent,
	clientMeta *xatu.ClientMeta,
	traceMeta *libp2p.TraceEventMetadata,
) error {
	var (
		err           error
		root          [32]byte
		slot          primitives.Slot
		proposerIndex primitives.ValidatorIndex
	)

	switch evt := event.Payload.(type) {
	case *TraceEventPhase0Block:
		root, err = evt.Block.GetBlock().HashTreeRoot()
		slot = evt.Block.GetBlock().GetSlot()
		proposerIndex = evt.Block.GetBlock().GetProposerIndex()
	case *TraceEventAltairBlock:
		root, err = evt.Block.GetBlock().HashTreeRoot()
		slot = evt.Block.GetBlock().GetSlot()
		proposerIndex = evt.Block.GetBlock().GetProposerIndex()
	case *TraceEventBellatrixBlock:
		root, err = evt.Block.GetBlock().HashTreeRoot()
		slot = evt.Block.GetBlock().GetSlot()
		proposerIndex = evt.Block.GetBlock().GetProposerIndex()
	case *TraceEventCapellaBlock:
		root, err = evt.Block.GetBlock().HashTreeRoot()
		slot = evt.Block.GetBlock().GetSlot()
		proposerIndex = evt.Block.GetBlock().GetProposerIndex()
	case *TraceEventDenebBlock:
		root, err = evt.Block.GetBlock().HashTreeRoot()
		slot = evt.Block.GetBlock().GetSlot()
		proposerIndex = evt.Block.GetBlock().GetProposerIndex()
	case *TraceEventElectraBlock:
		root, err = evt.Block.GetBlock().HashTreeRoot()
		slot = evt.Block.GetBlock().GetSlot()
		proposerIndex = evt.Block.GetBlock().GetProposerIndex()
	case *TraceEventFuluBlock:
		root, err = evt.Block.GetBlock().HashTreeRoot()
		slot = evt.Block.GetBlock().GetSlot()
		proposerIndex = evt.Block.GetBlock().GetProposerIndex()
	default:
		return fmt.Errorf("handleGossipBeaconBlock(): called with unknown block type")
	}

	if err != nil {
		return fmt.Errorf("failed to determine block hash tree root: %w", err)
	}

	data := &gossipsub.BeaconBlock{
		Slot:          wrapperspb.UInt64(uint64(slot)),
		Block:         wrapperspb.String(fmt.Sprintf("0x%x", root)),
		ProposerIndex: wrapperspb.UInt64(uint64(proposerIndex)),
	}

	metadata, ok := proto.Clone(clientMeta).(*xatu.ClientMeta)
	if !ok {
		return fmt.Errorf("failed to clone client metadata")
	}

	additionalData, err := p.createAdditionalGossipSubBeaconBlockData(event.Payload, slot, &event.TraceEventBase)
	if err != nil {
		return fmt.Errorf("failed to create additional data: %w", err)
	}

	metadata.AdditionalData = &xatu.ClientMeta_Libp2PTraceGossipsubBeaconBlock{
		Libp2PTraceGossipsubBeaconBlock: additionalData,
	}

	decoratedEvent := &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_LIBP2P_TRACE_GOSSIPSUB_BEACON_BLOCK,
			DateTime: timestamppb.New(event.GetTimestamp().Add(p.clockDrift)),
			Id:       uuid.New().String(),
		},
		Meta: &xatu.Meta{
			Client: metadata,
		},
		Data: &xatu.DecoratedEvent_Libp2PTraceGossipsubBeaconBlock{
			Libp2PTraceGossipsubBeaconBlock: data,
		},
	}

	return p.output.HandleDecoratedEvent(ctx, decoratedEvent)
}

//nolint:gosec // int -> uint32 common conversion pattern in xatu.
func (p *Processor) createAdditionalGossipSubBeaconBlockData(
	payload any,
	slotNumber primitives.Slot,
	event *TraceEventBase,
) (*xatu.ClientMeta_AdditionalLibP2PTraceGossipSubBeaconBlockData, error) {
	wallclockSlot, wallclockEpoch, err := p.wallclock.FromTime(event.GetTimestamp())
	if err != nil {
		return nil, fmt.Errorf("failed to get wallclock time: %w", err)
	}

	extra := &xatu.ClientMeta_AdditionalLibP2PTraceGossipSubBeaconBlockData{
		WallclockSlot: &xatu.SlotV2{
			Number:        &wrapperspb.UInt64Value{Value: wallclockSlot.Number()},
			StartDateTime: timestamppb.New(wallclockSlot.TimeWindow().Start()),
		},
		WallclockEpoch: &xatu.EpochV2{
			Number:        &wrapperspb.UInt64Value{Value: wallclockEpoch.Number()},
			StartDateTime: timestamppb.New(wallclockEpoch.TimeWindow().Start()),
		},
	}

	// Add Clock Drift
	timestampAdjusted := event.GetTimestamp().Add(p.clockDrift)

	slot := p.wallclock.Slots().FromNumber(uint64(slotNumber))
	epoch := p.wallclock.Epochs().FromSlot(uint64(slotNumber))

	extra.Slot = &xatu.SlotV2{
		StartDateTime: timestamppb.New(slot.TimeWindow().Start()),
		Number:        &wrapperspb.UInt64Value{Value: uint64(slotNumber)},
	}

	extra.Epoch = &xatu.EpochV2{
		Number:        &wrapperspb.UInt64Value{Value: epoch.Number()},
		StartDateTime: timestamppb.New(epoch.TimeWindow().Start()),
	}

	extra.Propagation = &xatu.PropagationV2{
		SlotStartDiff: &wrapperspb.UInt64Value{
			Value: uint64(timestampAdjusted.Sub(slot.TimeWindow().Start()).Milliseconds()),
		},
	}

	switch evt := payload.(type) {
	case *TraceEventPhase0Block:
		extra.Metadata = &libp2p.TraceEventMetadata{PeerId: wrapperspb.String(evt.PeerID)}
		extra.Topic = wrapperspb.String(evt.Topic)
		extra.MessageId = wrapperspb.String(evt.MsgID)
		extra.MessageSize = wrapperspb.UInt32(uint32(evt.MsgSize))
	case *TraceEventAltairBlock:
		extra.Metadata = &libp2p.TraceEventMetadata{PeerId: wrapperspb.String(evt.PeerID)}
		extra.Topic = wrapperspb.String(evt.Topic)
		extra.MessageId = wrapperspb.String(evt.MsgID)
		extra.MessageSize = wrapperspb.UInt32(uint32(evt.MsgSize))
	case *TraceEventBellatrixBlock:
		extra.Metadata = &libp2p.TraceEventMetadata{PeerId: wrapperspb.String(evt.PeerID)}
		extra.Topic = wrapperspb.String(evt.Topic)
		extra.MessageId = wrapperspb.String(evt.MsgID)
		extra.MessageSize = wrapperspb.UInt32(uint32(evt.MsgSize))
	case *TraceEventCapellaBlock:
		extra.Metadata = &libp2p.TraceEventMetadata{PeerId: wrapperspb.String(evt.PeerID)}
		extra.Topic = wrapperspb.String(evt.Topic)
		extra.MessageId = wrapperspb.String(evt.MsgID)
		extra.MessageSize = wrapperspb.UInt32(uint32(evt.MsgSize))
	case *TraceEventDenebBlock:
		extra.Metadata = &libp2p.TraceEventMetadata{PeerId: wrapperspb.String(evt.PeerID)}
		extra.Topic = wrapperspb.String(evt.Topic)
		extra.MessageId = wrapperspb.String(evt.MsgID)
		extra.MessageSize = wrapperspb.UInt32(uint32(evt.MsgSize))
	case *TraceEventElectraBlock:
		extra.Metadata = &libp2p.TraceEventMetadata{PeerId: wrapperspb.String(evt.PeerID)}
		extra.Topic = wrapperspb.String(evt.Topic)
		extra.MessageId = wrapperspb.String(evt.MsgID)
		extra.MessageSize = wrapperspb.UInt32(uint32(evt.MsgSize))
	case *TraceEventFuluBlock:
		extra.Metadata = &libp2p.TraceEventMetadata{PeerId: wrapperspb.String(evt.PeerID)}
		extra.Topic = wrapperspb.String(evt.Topic)
		extra.MessageId = wrapperspb.String(evt.MsgID)
		extra.MessageSize = wrapperspb.UInt32(uint32(evt.MsgSize))
	default:
		return nil, fmt.Errorf("createAdditionalGossipSubBeaconBlockData(): called with unknown block type")
	}

	return extra, nil
}
