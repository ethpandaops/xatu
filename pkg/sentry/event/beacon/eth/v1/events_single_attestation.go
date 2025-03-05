package event

import (
	"context"
	"fmt"
	"time"

	"github.com/attestantio/go-eth2-client/spec/electra"
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

type EventsSingleAttestation struct {
	log logrus.FieldLogger

	now time.Time

	event          *electra.SingleAttestation
	beacon         *ethereum.BeaconNode
	duplicateCache *ttlcache.Cache[string, time.Time]
	clientMeta     *xatu.ClientMeta
	id             uuid.UUID
}

func NewEventsSingleAttestation(log logrus.FieldLogger, event *electra.SingleAttestation, now time.Time, beacon *ethereum.BeaconNode, duplicateCache *ttlcache.Cache[string, time.Time], clientMeta *xatu.ClientMeta) (*EventsSingleAttestation, error) {
	if event == nil {
		return nil, fmt.Errorf("event is nil")
	}

	return &EventsSingleAttestation{
		log:            log.WithField("event", "BEACON_API_ETH_V1_EVENTS_ATTESTATION_V2/SINGLE"),
		now:            now,
		event:          event,
		beacon:         beacon,
		duplicateCache: duplicateCache,
		clientMeta:     clientMeta,
		id:             uuid.New(),
	}, nil
}

func (e *EventsSingleAttestation) AttestationData() *phase0.AttestationData {
	return e.event.Data
}

func (e *EventsSingleAttestation) getData() (*xatuethv1.AttestationV2, error) {

	attestation := e.event
	if attestation == nil {
		return nil, fmt.Errorf("electra attestation is nil")
	}

	return &xatuethv1.AttestationV2{
		AggregationBits: "",
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
}

func (e *EventsSingleAttestation) Decorate(ctx context.Context) (*xatu.DecoratedEvent, error) {
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

func (e *EventsSingleAttestation) ShouldIgnore(ctx context.Context) (bool, error) {
	if err := e.beacon.Synced(ctx); err != nil {
		return true, err
	}

	hash, err := hashstructure.Hash(e.event, hashstructure.FormatV2, nil)
	if err != nil {
		return true, err
	}

	item, retrieved := e.duplicateCache.GetOrSet(fmt.Sprint(hash), e.now, ttlcache.WithTTL[string, time.Time](ttlcache.DefaultTTL))
	if retrieved {
		e.log.WithFields(logrus.Fields{
			"hash":                  hash,
			"time_since_first_item": time.Since(item.Value()),
		}).Debug("Duplicate attestation event received")

		return true, nil
	}

	return false, nil
}

func (e *EventsSingleAttestation) getAdditionalData(_ context.Context) (*xatu.ClientMeta_AdditionalEthV1EventsAttestationV2Data, error) {
	extra := &xatu.ClientMeta_AdditionalEthV1EventsAttestationV2Data{}

	attestationData := e.AttestationData()

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

	extra.AttestingValidator = &xatu.AttestingValidatorV2{
		CommitteeIndex: &wrapperspb.UInt64Value{Value: uint64(e.event.CommitteeIndex)},
		Index:          &wrapperspb.UInt64Value{Value: uint64(e.event.AttesterIndex)},
	}

	return extra, nil
}
