package sentry

import (
	"context"
	"fmt"
	"time"

	"github.com/attestantio/go-eth2-client/spec/phase0"
	xatuethv1 "github.com/ethpandaops/xatu/pkg/proto/eth/v1"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/mitchellh/hashstructure/v2"
	"github.com/savid/ttlcache/v3"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (s *Sentry) handleAttestation(ctx context.Context, event *phase0.Attestation) error {
	s.log.Debug("Attestation received")

	hash, err := hashstructure.Hash(event, hashstructure.FormatV2, nil)
	if err != nil {
		return err
	}

	item, retrieved := s.duplicateCache.Attestation.GetOrSet(fmt.Sprint(hash), time.Now(), ttlcache.DefaultTTL)
	if retrieved {
		s.log.WithFields(logrus.Fields{
			"hash":                   hash,
			"time_since_first_event": time.Since(item.Value()),
			"slot":                   event.Data.Slot,
		}).Debug("Duplicate attestation event received")
		// TODO(savid): add metrics
		return nil
	}

	meta, err := s.createNewClientMeta(ctx, xatu.ClientMeta_Event_BEACON_API_ETH_V1_EVENTS_ATTESTATION)
	if err != nil {
		return err
	}

	decoratedEvent := &xatu.DecoratedEvent{
		Meta: &xatu.Meta{
			Client: meta,
		},
		Data: &xatu.DecoratedEvent_EthV1Attestation{
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
				Signature: xatuethv1.TrimmedString(fmt.Sprintf("%#x", event.Signature)),
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

	attestionSlot := s.beacon.Metadata().Wallclock().Slots().FromNumber(uint64(event.Data.Slot))
	epoch := s.beacon.Metadata().Wallclock().Epochs().FromSlot(uint64(event.Data.Slot))

	extra.Slot = &xatu.Slot{
		Number:        attestionSlot.Number(),
		StartDateTime: timestamppb.New(attestionSlot.TimeWindow().Start()),
	}

	extra.Epoch = &xatu.Epoch{
		Number:        epoch.Number(),
		StartDateTime: timestamppb.New(epoch.TimeWindow().Start()),
	}

	extra.Propagation = &xatu.Propagation{
		SlotStartDiff: uint64(eventTime.Sub(attestionSlot.TimeWindow().Start()).Milliseconds()),
	}

	// Build out the target section
	targetEpoch := s.beacon.Metadata().Wallclock().Epochs().FromNumber(uint64(event.Data.Target.Epoch))
	extra.Target = &xatu.ClientMeta_AdditionalAttestationTargetData{
		Epoch: &xatu.Epoch{
			Number:        targetEpoch.Number(),
			StartDateTime: timestamppb.New(targetEpoch.TimeWindow().Start()),
		},
	}

	// Build out the source section
	sourceEpoch := s.beacon.Metadata().Wallclock().Epochs().FromNumber(uint64(event.Data.Source.Epoch))
	extra.Source = &xatu.ClientMeta_AdditionalAttestationSourceData{
		Epoch: &xatu.Epoch{
			Number:        sourceEpoch.Number(),
			StartDateTime: timestamppb.New(sourceEpoch.TimeWindow().Start()),
		},
	}

	return extra, nil
}
