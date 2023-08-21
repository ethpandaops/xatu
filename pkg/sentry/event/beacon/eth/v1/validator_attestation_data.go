package event

import (
	"context"
	"errors"
	"time"

	"github.com/attestantio/go-eth2-client/spec/phase0"
	v1 "github.com/ethpandaops/xatu/pkg/proto/eth/v1"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/ethpandaops/xatu/pkg/sentry/ethereum"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type ValidatorAttestationData struct {
	log logrus.FieldLogger

	snapshot *ValidatorAttestationDataSnapshot

	beacon     *ethereum.BeaconNode
	clientMeta *xatu.ClientMeta
	id         uuid.UUID
}

type ValidatorAttestationDataSnapshot struct {
	Event           *phase0.AttestationData
	RequestAt       time.Time
	RequestDuration time.Duration
}

func NewValidatorAttestationData(log logrus.FieldLogger, snapshot *ValidatorAttestationDataSnapshot, beacon *ethereum.BeaconNode, clientMeta *xatu.ClientMeta) *ValidatorAttestationData {
	return &ValidatorAttestationData{
		log:        log.WithField("event", "BEACON_API_ETH_V1_VALIDATOR_ATTESTATION_DATA"),
		snapshot:   snapshot,
		beacon:     beacon,
		clientMeta: clientMeta,
		id:         uuid.New(),
	}
}

func (e *ValidatorAttestationData) Decorate(ctx context.Context) (*xatu.DecoratedEvent, error) {
	if e.snapshot.Event == nil {
		return nil, errors.New("snapshot event is nil")
	}

	decoratedEvent := &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_BEACON_API_ETH_V1_VALIDATOR_ATTESTATION_DATA,
			DateTime: timestamppb.New(e.snapshot.RequestAt),
			Id:       e.id.String(),
		},
		Meta: &xatu.Meta{
			Client: e.clientMeta,
		},
		Data: &xatu.DecoratedEvent_EthV1ValidatorAttestationData{
			EthV1ValidatorAttestationData: &v1.AttestationData{
				Slot:            uint64(e.snapshot.Event.Slot),
				Index:           uint64(e.snapshot.Event.Index),
				BeaconBlockRoot: v1.RootAsString(e.snapshot.Event.BeaconBlockRoot),
				Source: &v1.Checkpoint{
					Epoch: uint64(e.snapshot.Event.Source.Epoch),
					Root:  v1.RootAsString(e.snapshot.Event.Source.Root),
				},
				Target: &v1.Checkpoint{
					Epoch: uint64(e.snapshot.Event.Target.Epoch),
					Root:  v1.RootAsString(e.snapshot.Event.Target.Root),
				},
			},
		},
	}

	additionalData, err := e.getAdditionalData(ctx)
	if err != nil {
		e.log.WithError(err).Error("Failed to get extra validator attestation data")
	} else {
		decoratedEvent.Meta.Client.AdditionalData = &xatu.ClientMeta_EthV1ValidatorAttestationData{
			EthV1ValidatorAttestationData: additionalData,
		}
	}

	return decoratedEvent, nil
}

func (e *ValidatorAttestationData) ShouldIgnore(ctx context.Context) (bool, error) {
	if err := e.beacon.Synced(ctx); err != nil {
		return true, err
	}

	return false, nil
}

func (e *ValidatorAttestationData) getAdditionalData(_ context.Context) (*xatu.ClientMeta_AdditionalEthV1ValidatorAttestationDataData, error) {
	extra := &xatu.ClientMeta_AdditionalEthV1ValidatorAttestationDataData{}

	attestionSlot := e.beacon.Metadata().Wallclock().Slots().FromNumber(uint64(e.snapshot.Event.Slot))
	epoch := e.beacon.Metadata().Wallclock().Epochs().FromSlot(uint64(e.snapshot.Event.Slot))

	extra.Slot = &xatu.Slot{
		Number:        attestionSlot.Number(),
		StartDateTime: timestamppb.New(attestionSlot.TimeWindow().Start()),
	}

	extra.Epoch = &xatu.Epoch{
		Number:        epoch.Number(),
		StartDateTime: timestamppb.New(epoch.TimeWindow().Start()),
	}

	extra.Snapshot = &xatu.ClientMeta_AttestationDataSnapshot{
		RequestedAtSlotStartDiffMs: uint64(e.snapshot.RequestAt.Sub(attestionSlot.TimeWindow().Start()).Milliseconds()),
		RequestDurationMs:          uint64(e.snapshot.RequestDuration.Milliseconds()),
		Timestamp:                  timestamppb.New(e.snapshot.RequestAt),
	}

	// Build out the target section
	targetEpoch := e.beacon.Metadata().Wallclock().Epochs().FromNumber(uint64(e.snapshot.Event.Target.Epoch))
	extra.Target = &xatu.ClientMeta_AdditionalEthV1AttestationTargetData{
		Epoch: &xatu.Epoch{
			Number:        targetEpoch.Number(),
			StartDateTime: timestamppb.New(targetEpoch.TimeWindow().Start()),
		},
	}

	// Build out the source section
	sourceEpoch := e.beacon.Metadata().Wallclock().Epochs().FromNumber(uint64(e.snapshot.Event.Source.Epoch))
	extra.Source = &xatu.ClientMeta_AdditionalEthV1AttestationSourceData{
		Epoch: &xatu.Epoch{
			Number:        sourceEpoch.Number(),
			StartDateTime: timestamppb.New(sourceEpoch.TimeWindow().Start()),
		},
	}

	return extra, nil
}
