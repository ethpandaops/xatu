package sentry

import (
	"context"
	"fmt"

	"github.com/attestantio/go-eth2-client/spec/phase0"
	xatuethv1 "github.com/ethpandaops/xatu/pkg/proto/eth/v1"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (s *Sentry) handleAttestation(ctx context.Context, event *phase0.Attestation) error {
	s.log.Debug("Attestation received")

	meta, err := s.createNewClientMeta(ctx, xatu.ClientMeta_Event_BEACON_API_ETH_V1_EVENTS_ATTESTATION)
	if err != nil {
		return err
	}

	decoratedEvent := &xatu.DecoratedEvent{
		Meta: &xatu.Meta{
			Client: meta,
		},
		Event: &xatu.DecoratedEvent_EthV1Attestation{
			EthV1Attestation: &xatuethv1.Attestation{
				AggregationBits: xatuethv1.BytesToString(event.AggregationBits),
				Data: &xatuethv1.AttestationData{
					Slot:            uint64(event.Data.Slot),
					Index:           uint64(event.Data.Index),
					BeaconBlockRoot: xatuethv1.RootAsString(event.Data.BeaconBlockRoot),
					Source: &xatuethv1.Checkpoint{
						Epoch: uint64(event.Data.Source.Epoch),
						Root:  xatuethv1.RootAsString(event.Data.Source.Root),
					},
					Target: &xatuethv1.Checkpoint{
						Epoch: uint64(event.Data.Target.Epoch),
						Root:  xatuethv1.RootAsString(event.Data.Target.Root),
					},
				},
				Signature: fmt.Sprintf("%#x", event.Signature),
			},
		},
	}

	additionalData, err := s.getAttestationData(ctx, event, meta)
	if err != nil {
		s.log.WithError(err).Error("Failed to get extra attestation data")
	} else {
		decoratedEvent.Meta.Client.AdditionalData = &xatu.ClientMeta_Attestation{
			Attestation: additionalData,
		}
	}

	return s.handleNewDecoratedEvent(ctx, decoratedEvent)
}

func (s *Sentry) getAttestationData(ctx context.Context, event *phase0.Attestation, meta *xatu.ClientMeta) (*xatu.ClientMeta_AdditionalAttestationData, error) {
	extra := &xatu.ClientMeta_AdditionalAttestationData{}

	eventTime := meta.Event.DateTime.AsTime()

	// Get the wallclock time window for when we saw the event
	slot, _, err := s.beacon.Metadata().Wallclock().FromTime(eventTime)
	if err != nil {
		return extra, err
	}

	extra.Slot = &xatu.AdditionalSlotData{
		Number:          slot.Number(),
		StartDateTime:   timestamppb.New(slot.TimeWindow().Start()),
		PropagationDiff: uint64(eventTime.Sub(slot.TimeWindow().Start()).Milliseconds()),
	}

	// Build out the target section
	targetEpoch := s.beacon.Metadata().Wallclock().Epochs().FromNumber(uint64(event.Data.Target.Epoch))
	extra.Target = &xatu.AdditionalEpochData{
		Epoch: &xatu.Epoch{
			Number:        targetEpoch.Number(),
			StartDateTime: timestamppb.New(targetEpoch.TimeWindow().Start()),
		},
	}

	// Build out the source section
	sourceEpoch := s.beacon.Metadata().Wallclock().Epochs().FromNumber(uint64(event.Data.Source.Epoch))
	extra.Source = &xatu.AdditionalEpochData{
		Epoch: &xatu.Epoch{
			Number:        sourceEpoch.Number(),
			StartDateTime: timestamppb.New(sourceEpoch.TimeWindow().Start()),
		},
	}

	return extra, nil
}
