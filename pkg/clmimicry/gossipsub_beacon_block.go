package clmimicry

import (
	"context"
	"fmt"

	"github.com/ethpandaops/xatu/pkg/proto/libp2p"
	"github.com/ethpandaops/xatu/pkg/proto/libp2p/gossipsub"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/google/uuid"
	"github.com/probe-lab/hermes/host"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func (m *Mimicry) handleGossipBeaconBlock(ctx context.Context,
	clientMeta *xatu.ClientMeta,
	event *host.TraceEvent, payload map[string]any) error {
	slot, ok := payload["Slot"].(primitives.Slot)
	if !ok {
		return fmt.Errorf("invalid slot")
	}

	blockRoot, ok := payload["Root"].([32]byte)
	if !ok {
		return fmt.Errorf("invalid block root")
	}

	proposerIndex, ok := payload["ValIdx"].(primitives.ValidatorIndex)
	if !ok {
		return fmt.Errorf("invalid proposer index")
	}

	data := &gossipsub.BeaconBlock{
		Slot:          wrapperspb.UInt64(uint64(slot)),
		Block:         wrapperspb.String(fmt.Sprintf("0x%x", blockRoot)),
		ProposerIndex: wrapperspb.UInt64(uint64(proposerIndex)),
	}

	metadata, ok := proto.Clone(clientMeta).(*xatu.ClientMeta)
	if !ok {
		return fmt.Errorf("failed to clone client metadata")
	}

	additionalData, err := m.createAdditionalGossipSubBeaconBlockData(ctx, payload, slot, event)
	if err != nil {
		return fmt.Errorf("failed to create additional data: %w", err)
	}

	metadata.AdditionalData = &xatu.ClientMeta_Libp2PTraceGossipsubBeaconBlock{
		Libp2PTraceGossipsubBeaconBlock: additionalData,
	}

	decoratedEvent := &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_LIBP2P_TRACE_GOSSIPSUB_BEACON_BLOCK,
			DateTime: timestamppb.New(event.Timestamp.Add(m.clockDrift)),
			Id:       uuid.New().String(),
		},
		Meta: &xatu.Meta{
			Client: metadata,
		},
		Data: &xatu.DecoratedEvent_Libp2PTraceGossipsubBeaconBlock{
			Libp2PTraceGossipsubBeaconBlock: data,
		},
	}

	return m.handleNewDecoratedEvent(ctx, decoratedEvent)
}

func (m *Mimicry) createAdditionalGossipSubBeaconBlockData(ctx context.Context,
	payload map[string]any,
	slotNumber primitives.Slot,
	event *host.TraceEvent,
) (*xatu.ClientMeta_AdditionalLibP2PTraceGossipSubBeaconBlockData, error) {
	wallclockSlot, wallclockEpoch, err := m.ethereum.Metadata().Wallclock().Now()
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
	timestampAdjusted := event.Timestamp.Add(m.clockDrift)

	slot := m.ethereum.Metadata().Wallclock().Slots().FromNumber(uint64(slotNumber))
	epoch := m.ethereum.Metadata().Wallclock().Epochs().FromSlot(uint64(slotNumber))

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

	peerID, ok := payload["PeerID"].(string)
	if ok {
		extra.Metadata = &libp2p.TraceEventMetadata{
			PeerId: wrapperspb.String(peerID),
		}
	}

	topic, ok := payload["Topic"].(string)
	if ok {
		extra.Topic = wrapperspb.String(topic)
	}

	msgID, ok := payload["MsgID"].(string)
	if ok {
		extra.MessageId = wrapperspb.String(msgID)
	}

	msgSize, ok := payload["MsgSize"].(int)
	if ok {
		extra.MessageSize = wrapperspb.UInt32(uint32(msgSize))
	}

	return extra, nil
}
