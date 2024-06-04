package clmimicry

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/probe-lab/hermes/host"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/ethpandaops/xatu/pkg/proto/libp2p"
	"github.com/ethpandaops/xatu/pkg/proto/libp2p/gossipsub"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

func (m *Mimicry) handleGossipBlobSidecar(
	ctx context.Context,
	clientMeta *xatu.ClientMeta,
	event *host.TraceEvent,
	payload map[string]any,
) error {
	slot, ok := payload["Slot"].(primitives.Slot)
	if !ok {
		return fmt.Errorf("invalid slot")
	}

	blobIndex, ok := payload["index"].(uint64)
	if !ok {
		return fmt.Errorf("invalid blob index")
	}

	proposerIndex, ok := payload["ValIdx"].(primitives.ValidatorIndex)
	if !ok {
		return fmt.Errorf("invalid proposer index")
	}

	stateRoot, ok := payload["StateRoot"].(string)
	if !ok {
		return fmt.Errorf("invalid state root")
	}

	parentRoot, ok := payload["ParentRoot"].(string)
	if !ok {
		return fmt.Errorf("invalid parent root")
	}

	data := &gossipsub.BlobSidecar{
		Index:         wrapperspb.UInt64(blobIndex),
		Slot:          wrapperspb.UInt64(uint64(slot)),
		ProposerIndex: wrapperspb.UInt64(uint64(proposerIndex)),
		StateRoot:     wrapperspb.String(stateRoot),
		ParentRoot:    wrapperspb.String(parentRoot),
	}

	metadata, ok := proto.Clone(clientMeta).(*xatu.ClientMeta)
	if !ok {
		return fmt.Errorf("failed to clone client metadata")
	}

	additionalData, err := m.createAdditionalGossipSubBlobSidecarData(ctx, payload, event.Timestamp, slot)
	if err != nil {
		return fmt.Errorf("failed to create additional data: %w", err)
	}

	metadata.AdditionalData = &xatu.ClientMeta_Libp2PTraceGossipsubBlobSidecar{
		Libp2PTraceGossipsubBlobSidecar: additionalData,
	}

	decoratedEvent := &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_LIBP2P_TRACE_GOSSIPSUB_BLOB_SIDECAR,
			DateTime: timestamppb.New(event.Timestamp.Add(m.clockDrift)),
			Id:       uuid.New().String(),
		},
		Meta: &xatu.Meta{
			Client: metadata,
		},
		Data: &xatu.DecoratedEvent_Libp2PTraceGossipsubBlobSidecar{
			Libp2PTraceGossipsubBlobSidecar: data,
		},
	}

	return m.handleNewDecoratedEvent(ctx, decoratedEvent)
}

func (m *Mimicry) createAdditionalGossipSubBlobSidecarData(
	ctx context.Context,
	payload map[string]any,
	timestamp time.Time,
	slotNumber primitives.Slot,
) (*xatu.ClientMeta_AdditionalLibP2PTraceGossipSubBlobSidecarData, error) {
	wallclockSlot, wallclockEpoch, err := m.ethereum.Metadata().Wallclock().Now()
	if err != nil {
		return nil, fmt.Errorf("failed to get wallclock time: %w", err)
	}

	slot := m.ethereum.Metadata().Wallclock().Slots().FromNumber(uint64(slotNumber))
	epoch := m.ethereum.Metadata().Wallclock().Epochs().FromSlot(uint64(slotNumber))
	timestampAdjusted := timestamp.Add(m.clockDrift)

	peerID, ok := payload["PeerID"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid peer ID")
	}

	topic, ok := payload["Topic"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid topic")
	}

	msgID, ok := payload["MsgID"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid message ID")
	}

	msgSize, ok := payload["MsgSize"].(int)
	if !ok {
		return nil, fmt.Errorf("invalid message size")
	}

	data := &xatu.ClientMeta_AdditionalLibP2PTraceGossipSubBlobSidecarData{
		WallclockSlot: &xatu.SlotV2{
			Number:        wrapperspb.UInt64(wallclockSlot.Number()),
			StartDateTime: timestamppb.New(wallclockSlot.TimeWindow().Start()),
		},
		WallclockEpoch: &xatu.EpochV2{
			Number:        wrapperspb.UInt64(wallclockEpoch.Number()),
			StartDateTime: timestamppb.New(wallclockEpoch.TimeWindow().Start()),
		},
		Epoch: &xatu.EpochV2{
			Number:        wrapperspb.UInt64(epoch.Number()),
			StartDateTime: timestamppb.New(epoch.TimeWindow().Start()),
		},
		Slot: &xatu.SlotV2{
			Number:        wrapperspb.UInt64(uint64(slotNumber)),
			StartDateTime: timestamppb.New(slot.TimeWindow().Start()),
		},
		Propagation: &xatu.PropagationV2{
			SlotStartDiff: &wrapperspb.UInt64Value{
				Value: uint64(timestampAdjusted.Sub(slot.TimeWindow().Start()).Milliseconds()),
			},
		},
		Metadata: &libp2p.TraceEventMetadata{
			PeerId: wrapperspb.String(peerID),
		},
		Topic:       wrapperspb.String(topic),
		MessageSize: wrapperspb.UInt32(uint32(msgSize)),
		MessageId:   wrapperspb.String(msgID),
	}

	return data, nil
}
