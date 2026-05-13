package clmimicry

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"

	ethv1 "github.com/ethpandaops/xatu/pkg/proto/eth/v1"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

// handleBeaconSyntheticPayloadStatusResolvedEvent handles a fork-choice payload status
// transition observed from beacon node internals (TYSM-instrumented). EIP-7732 ePBS.
func (p *Processor) handleBeaconSyntheticPayloadStatusResolvedEvent(
	ctx context.Context,
	clientMeta *xatu.ClientMeta,
	event *TraceEvent,
) error {
	data, err := TraceEventToBeaconSyntheticPayloadStatusResolved(event)
	if err != nil {
		return errors.Wrapf(err, "failed to convert event to beacon synthetic payload status resolved event")
	}

	metadata, ok := proto.Clone(clientMeta).(*xatu.ClientMeta)
	if !ok {
		return fmt.Errorf("failed to clone client metadata")
	}

	extra := p.deriveAdditionalDataForBeaconSyntheticPayloadStatusResolved(data)

	metadata.AdditionalData = &xatu.ClientMeta_BeaconSyntheticPayloadStatusResolved{
		BeaconSyntheticPayloadStatusResolved: extra,
	}

	decoratedEvent := &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_BEACON_SYNTHETIC_PAYLOAD_STATUS_RESOLVED,
			DateTime: timestamppb.New(event.Timestamp.Add(p.clockDrift)),
			Id:       uuid.New().String(),
		},
		Meta: &xatu.Meta{
			Client: metadata,
		},
		Data: &xatu.DecoratedEvent_BeaconSyntheticPayloadStatusResolved{
			BeaconSyntheticPayloadStatusResolved: data,
		},
	}

	return p.output.HandleDecoratedEvent(ctx, decoratedEvent)
}

func (p *Processor) deriveAdditionalDataForBeaconSyntheticPayloadStatusResolved(
	data *ethv1.PayloadStatusResolved,
) *xatu.ClientMeta_AdditionalBeaconSyntheticPayloadStatusResolvedData {
	extra := &xatu.ClientMeta_AdditionalBeaconSyntheticPayloadStatusResolvedData{}

	slot := p.wallclock.Slots().FromNumber(data.GetSlot().GetValue())
	epoch := p.wallclock.Epochs().FromSlot(data.GetSlot().GetValue())

	slotStart := slot.TimeWindow().Start()

	extra.Slot = &xatu.SlotV2{
		Number:        wrapperspb.UInt64(slot.Number()),
		StartDateTime: timestamppb.New(slotStart),
	}
	extra.Epoch = &xatu.EpochV2{
		Number:        wrapperspb.UInt64(epoch.Number()),
		StartDateTime: timestamppb.New(epoch.TimeWindow().Start()),
	}

	// PropagationSlotStartDiff = ResolvedAt - slot_start, in milliseconds.
	var slotStartDiffMs uint64

	if resolvedAt := data.GetResolvedAt(); resolvedAt != nil {
		diff := resolvedAt.AsTime().Sub(slotStart).Milliseconds()
		if diff > 0 {
			slotStartDiffMs = uint64(diff)
		}
	}

	extra.Propagation = &xatu.PropagationV2{
		SlotStartDiff: wrapperspb.UInt64(slotStartDiffMs),
	}

	return extra
}

// handleBeaconSyntheticBuilderPendingPaymentSettlementEvent handles an epoch-boundary
// builder pending payment settle/drop decision observed from beacon node internals
// (TYSM-instrumented). EIP-7732 ePBS.
func (p *Processor) handleBeaconSyntheticBuilderPendingPaymentSettlementEvent(
	ctx context.Context,
	clientMeta *xatu.ClientMeta,
	event *TraceEvent,
) error {
	data, err := TraceEventToBeaconSyntheticBuilderPendingPaymentSettlement(event)
	if err != nil {
		return errors.Wrapf(err, "failed to convert event to beacon synthetic builder pending payment settlement event")
	}

	metadata, ok := proto.Clone(clientMeta).(*xatu.ClientMeta)
	if !ok {
		return fmt.Errorf("failed to clone client metadata")
	}

	extra := p.deriveAdditionalDataForBeaconSyntheticBuilderPendingPaymentSettlement(data)

	metadata.AdditionalData = &xatu.ClientMeta_BeaconSyntheticBuilderPendingPaymentSettlement{
		BeaconSyntheticBuilderPendingPaymentSettlement: extra,
	}

	decoratedEvent := &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_BEACON_SYNTHETIC_BUILDER_PENDING_PAYMENT_SETTLEMENT,
			DateTime: timestamppb.New(event.Timestamp.Add(p.clockDrift)),
			Id:       uuid.New().String(),
		},
		Meta: &xatu.Meta{
			Client: metadata,
		},
		Data: &xatu.DecoratedEvent_BeaconSyntheticBuilderPendingPaymentSettlement{
			BeaconSyntheticBuilderPendingPaymentSettlement: data,
		},
	}

	return p.output.HandleDecoratedEvent(ctx, decoratedEvent)
}

func (p *Processor) deriveAdditionalDataForBeaconSyntheticBuilderPendingPaymentSettlement(
	data *ethv1.BuilderPendingPaymentSettlement,
) *xatu.ClientMeta_AdditionalBeaconSyntheticBuilderPendingPaymentSettlementData {
	extra := &xatu.ClientMeta_AdditionalBeaconSyntheticBuilderPendingPaymentSettlementData{}

	epoch := p.wallclock.Epochs().FromNumber(data.GetEpoch().GetValue())

	extra.Epoch = &xatu.EpochV2{
		Number:        wrapperspb.UInt64(epoch.Number()),
		StartDateTime: timestamppb.New(epoch.TimeWindow().Start()),
	}

	return extra
}
