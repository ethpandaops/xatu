package clmimicry

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/ethpandaops/xatu/pkg/proto/libp2p"
	"github.com/ethpandaops/xatu/pkg/proto/libp2p/gossipsub"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

func (p *Processor) handleGossipProposerPreferences(
	ctx context.Context,
	clientMeta *xatu.ClientMeta,
	event *TraceEvent,
	payload *TraceEventProposerPreferences,
) error {
	if payload.ProposerPreferences == nil {
		return fmt.Errorf("handleGossipProposerPreferences() called with nil preferences")
	}

	prefs := payload.ProposerPreferences.GetMessage()
	if prefs == nil {
		return fmt.Errorf("handleGossipProposerPreferences() called with nil preferences message")
	}

	data := &gossipsub.ProposerPreferences{
		Slot:           wrapperspb.UInt64(uint64(prefs.GetProposalSlot())),
		ValidatorIndex: wrapperspb.UInt64(uint64(prefs.GetValidatorIndex())),
		FeeRecipient:   wrapperspb.String(fmt.Sprintf("0x%x", prefs.GetFeeRecipient())),
		TargetGasLimit: wrapperspb.UInt64(prefs.GetTargetGasLimit()),
	}

	metadata, ok := proto.Clone(clientMeta).(*xatu.ClientMeta)
	if !ok {
		return fmt.Errorf("failed to clone client metadata")
	}

	additionalData, err := p.createAdditionalGossipSubProposerPreferencesData(payload, event.Timestamp)
	if err != nil {
		return fmt.Errorf("failed to create additional data: %w", err)
	}

	metadata.AdditionalData = &xatu.ClientMeta_Libp2PTraceGossipsubProposerPreferences{
		Libp2PTraceGossipsubProposerPreferences: additionalData,
	}

	decoratedEvent := &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_LIBP2P_TRACE_GOSSIPSUB_PROPOSER_PREFERENCES,
			DateTime: timestamppb.New(event.Timestamp.Add(p.clockDrift)),
			Id:       uuid.New().String(),
		},
		Meta: &xatu.Meta{
			Client: metadata,
		},
		Data: &xatu.DecoratedEvent_Libp2PTraceGossipsubProposerPreferences{
			Libp2PTraceGossipsubProposerPreferences: data,
		},
	}

	return p.output.HandleDecoratedEvent(ctx, decoratedEvent)
}

//nolint:gosec // int -> uint32 common conversion pattern in xatu.
func (p *Processor) createAdditionalGossipSubProposerPreferencesData(
	payload *TraceEventProposerPreferences,
	timestamp time.Time,
) (*xatu.ClientMeta_AdditionalLibP2PTraceGossipSubProposerPreferencesData, error) {
	wallclockSlot, wallclockEpoch, err := p.wallclock.FromTime(timestamp)
	if err != nil {
		return nil, fmt.Errorf("failed to get wallclock time: %w", err)
	}

	slotNumber := uint64(payload.ProposerPreferences.GetMessage().GetProposalSlot())
	slot := p.wallclock.Slots().FromNumber(slotNumber)
	epoch := p.wallclock.Epochs().FromSlot(slotNumber)
	timestampAdjusted := timestamp.Add(p.clockDrift)

	return &xatu.ClientMeta_AdditionalLibP2PTraceGossipSubProposerPreferencesData{
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
			Number:        wrapperspb.UInt64(slotNumber),
			StartDateTime: timestamppb.New(slot.TimeWindow().Start()),
		},
		Propagation: &xatu.PropagationV2{
			SlotStartDiff: &wrapperspb.UInt64Value{
				Value: uint64(timestampAdjusted.Sub(slot.TimeWindow().Start()).Milliseconds()),
			},
		},
		Metadata:    &libp2p.TraceEventMetadata{PeerId: wrapperspb.String(payload.PeerID)},
		Topic:       wrapperspb.String(payload.Topic),
		MessageSize: wrapperspb.UInt32(uint32(payload.MsgSize)),
		MessageId:   wrapperspb.String(payload.MsgID),
	}, nil
}
