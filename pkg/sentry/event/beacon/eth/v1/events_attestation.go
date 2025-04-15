package event

import (
	"context"
	"fmt"
	"time"

	"github.com/attestantio/go-eth2-client/spec"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	xatuethv1 "github.com/ethpandaops/xatu/pkg/proto/eth/v1"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/ethpandaops/xatu/pkg/sentry/ethereum"
	"github.com/google/uuid"
	ttlcache "github.com/jellydator/ttlcache/v3"
	hashstructure "github.com/mitchellh/hashstructure/v2"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

type EventsAttestation struct {
	log logrus.FieldLogger

	now time.Time

	event          *spec.VersionedAttestation
	beacon         *ethereum.BeaconNode
	duplicateCache *ttlcache.Cache[string, time.Time]
	clientMeta     *xatu.ClientMeta
	id             uuid.UUID
}

func NewEventsAttestation(log logrus.FieldLogger, event *spec.VersionedAttestation, now time.Time, beacon *ethereum.BeaconNode, duplicateCache *ttlcache.Cache[string, time.Time], clientMeta *xatu.ClientMeta) (*EventsAttestation, error) {
	if event == nil {
		return nil, fmt.Errorf("event is nil")
	}

	return &EventsAttestation{
		log:            log.WithField("event", "BEACON_API_ETH_V1_EVENTS_ATTESTATION_V2"),
		now:            now,
		event:          event,
		beacon:         beacon,
		duplicateCache: duplicateCache,
		clientMeta:     clientMeta,
		id:             uuid.New(),
	}, nil
}

func (e *EventsAttestation) AttestationData() (*phase0.AttestationData, error) {
	data, err := e.event.Data()
	if err != nil {
		return nil, fmt.Errorf("failed to get attestation data: %w", err)
	}

	return data, nil
}

func (e *EventsAttestation) getData() (*xatuethv1.AttestationV2, error) {
	switch e.event.Version {
	case spec.DataVersionPhase0:
		attestation := e.event.Phase0
		if attestation == nil {
			return nil, fmt.Errorf("phase0 attestation is nil")
		}

		return &xatuethv1.AttestationV2{
			AggregationBits: xatuethv1.BytesToString(attestation.AggregationBits),
			Data: &xatuethv1.AttestationDataV2{
				Slot:            &wrapperspb.UInt64Value{Value: uint64(attestation.Data.Slot)},
				Index:           &wrapperspb.UInt64Value{Value: uint64(attestation.Data.Index)},
				BeaconBlockRoot: xatuethv1.RootAsString(attestation.Data.BeaconBlockRoot),
				Source: &xatuethv1.CheckpointV2{
					Epoch: &wrapperspb.UInt64Value{Value: uint64(attestation.Data.Source.Epoch)},
					Root:  xatuethv1.RootAsString(attestation.Data.Source.Root),
				},
				Target: &xatuethv1.CheckpointV2{
					Epoch: &wrapperspb.UInt64Value{Value: uint64(attestation.Data.Target.Epoch)},
					Root:  xatuethv1.RootAsString(attestation.Data.Target.Root),
				},
			},
			Signature: xatuethv1.TrimmedString(fmt.Sprintf("%#x", attestation.Signature)),
		}, nil
	case spec.DataVersionElectra:
		attestation := e.event.Electra
		if attestation == nil {
			return nil, fmt.Errorf("electra attestation is nil")
		}

		return &xatuethv1.AttestationV2{
			AggregationBits: xatuethv1.BytesToString(attestation.AggregationBits),
			Data: &xatuethv1.AttestationDataV2{
				Slot:            &wrapperspb.UInt64Value{Value: uint64(attestation.Data.Slot)},
				Index:           &wrapperspb.UInt64Value{Value: uint64(attestation.Data.Index)},
				BeaconBlockRoot: xatuethv1.RootAsString(attestation.Data.BeaconBlockRoot),
				Source: &xatuethv1.CheckpointV2{
					Epoch: &wrapperspb.UInt64Value{Value: uint64(attestation.Data.Source.Epoch)},
					Root:  xatuethv1.RootAsString(attestation.Data.Source.Root),
				},
				Target: &xatuethv1.CheckpointV2{
					Epoch: &wrapperspb.UInt64Value{Value: uint64(attestation.Data.Target.Epoch)},
					Root:  xatuethv1.RootAsString(attestation.Data.Target.Root),
				},
			},
		}, nil
	default:
		return nil, fmt.Errorf("unsupported attestation version: %s", e.event.Version)
	}
}

func (e *EventsAttestation) Decorate(ctx context.Context) (*xatu.DecoratedEvent, error) {
	data, err := e.getData()
	if err != nil {
		return nil, err
	}

	decoratedEvent := &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_BEACON_API_ETH_V1_EVENTS_ATTESTATION_V2,
			DateTime: timestamppb.New(e.now),
			Id:       e.id.String(),
		},
		Meta: &xatu.Meta{
			Client: e.clientMeta,
		},
		Data: &xatu.DecoratedEvent_EthV1EventsAttestationV2{
			EthV1EventsAttestationV2: data,
		},
	}

	additionalData, err := e.getAdditionalData(ctx)
	if err != nil {
		e.log.WithError(err).Error("Failed to get extra attestation data")
	} else {
		decoratedEvent.Meta.Client.AdditionalData = &xatu.ClientMeta_EthV1EventsAttestationV2{
			EthV1EventsAttestationV2: additionalData,
		}
	}

	return decoratedEvent, nil
}

func (e *EventsAttestation) ShouldIgnore(ctx context.Context) (bool, error) {
	if err := e.beacon.Synced(ctx); err != nil {
		return true, err
	}

	hash, err := hashstructure.Hash(e.event, hashstructure.FormatV2, nil)
	if err != nil {
		return true, err
	}

	_, retrieved := e.duplicateCache.GetOrSet(fmt.Sprint(hash), e.now, ttlcache.WithTTL[string, time.Time](ttlcache.DefaultTTL))
	if retrieved {
		// e.log.WithFields(logrus.Fields{
		// 	"hash":                  hash,
		// 	"time_since_first_item": time.Since(item.Value()),
		// }).Debug("Duplicate attestation event received")

		return true, nil
	}

	return false, nil
}

func (e *EventsAttestation) getAdditionalData(_ context.Context) (*xatu.ClientMeta_AdditionalEthV1EventsAttestationV2Data, error) {
	extra := &xatu.ClientMeta_AdditionalEthV1EventsAttestationV2Data{}

	attestationData, err := e.AttestationData()
	if err != nil {
		return nil, err
	}

	slot := attestationData.Slot

	attestionSlot := e.beacon.Metadata().Wallclock().Slots().FromNumber(uint64(slot))
	epoch := e.beacon.Metadata().Wallclock().Epochs().FromSlot(uint64(slot))

	extra.Slot = &xatu.SlotV2{
		Number:        &wrapperspb.UInt64Value{Value: attestionSlot.Number()},
		StartDateTime: timestamppb.New(attestionSlot.TimeWindow().Start()),
	}

	extra.Epoch = &xatu.EpochV2{
		Number:        &wrapperspb.UInt64Value{Value: epoch.Number()},
		StartDateTime: timestamppb.New(epoch.TimeWindow().Start()),
	}

	extra.Propagation = &xatu.PropagationV2{
		SlotStartDiff: &wrapperspb.UInt64Value{
			//nolint:gosec // not concerned in reality
			Value: uint64(e.now.Sub(attestionSlot.TimeWindow().Start()).Milliseconds()),
		},
	}

	target := attestationData.Target

	// Build out the target section
	targetEpoch := e.beacon.Metadata().Wallclock().Epochs().FromNumber(uint64(target.Epoch))
	extra.Target = &xatu.ClientMeta_AdditionalEthV1AttestationTargetV2Data{
		Epoch: &xatu.EpochV2{
			Number:        &wrapperspb.UInt64Value{Value: targetEpoch.Number()},
			StartDateTime: timestamppb.New(targetEpoch.TimeWindow().Start()),
		},
	}

	source := attestationData.Source

	// Build out the source section
	sourceEpoch := e.beacon.Metadata().Wallclock().Epochs().FromNumber(uint64(source.Epoch))
	extra.Source = &xatu.ClientMeta_AdditionalEthV1AttestationSourceV2Data{
		Epoch: &xatu.EpochV2{
			Number:        &wrapperspb.UInt64Value{Value: sourceEpoch.Number()},
			StartDateTime: timestamppb.New(sourceEpoch.TimeWindow().Start()),
		},
	}

	// If the attestation is unaggreated, we can append the validator position within the committee
	aggregationBits, err := e.event.AggregationBits()
	if err != nil {
		return nil, err
	}

	// Append the validator index if the attestation is unaggreated
	if aggregationBits.Count() == 1 {
		//nolint:gosec // not concerned in reality
		position := uint64(aggregationBits.BitIndices()[0])

		validatorIndex, err := e.beacon.Duties().GetValidatorIndex(
			phase0.Epoch(epoch.Number()),
			attestationData.Slot,
			attestationData.Index,
			position,
		)
		if err == nil {
			extra.AttestingValidator = &xatu.AttestingValidatorV2{
				CommitteeIndex: &wrapperspb.UInt64Value{Value: position},
				Index:          &wrapperspb.UInt64Value{Value: uint64(validatorIndex)},
			}
		}
	}

	return extra, nil
}
