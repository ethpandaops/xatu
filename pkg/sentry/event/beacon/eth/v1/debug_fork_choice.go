package event

import (
	"context"
	"time"

	eth2v1 "github.com/attestantio/go-eth2-client/api/v1"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	xatuethv1 "github.com/ethpandaops/xatu/pkg/proto/eth/v1"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/ethpandaops/xatu/pkg/sentry/ethereum"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

type ForkChoice struct {
	log logrus.FieldLogger

	snapshot *ForkChoiceSnapshot

	beacon     *ethereum.BeaconNode
	clientMeta *xatu.ClientMeta
	id         uuid.UUID
}

type ForkChoiceSnapshot struct {
	Event           *eth2v1.ForkChoice
	RequestSlot     phase0.Slot
	RequestEpoch    phase0.Epoch
	RequestAt       time.Time
	RequestDuration time.Duration
}

func NewForkChoice(log logrus.FieldLogger, snapshot *ForkChoiceSnapshot, beacon *ethereum.BeaconNode, clientMeta *xatu.ClientMeta) *ForkChoice {
	return &ForkChoice{
		log:        log.WithField("event", "BEACON_API_ETH_V1_DEBUG_FORK_CHOICE_V2"),
		snapshot:   snapshot,
		beacon:     beacon,
		clientMeta: clientMeta,
		id:         uuid.New(),
	}
}

func (f *ForkChoice) Decorate(ctx context.Context) (*xatu.DecoratedEvent, error) {
	data, err := f.GetData()
	if err != nil {
		return nil, err
	}

	decoratedEvent := &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_BEACON_API_ETH_V1_DEBUG_FORK_CHOICE_V2,
			DateTime: timestamppb.New(f.snapshot.RequestAt),
			Id:       f.id.String(),
		},
		Meta: &xatu.Meta{
			Client: f.clientMeta,
		},
		Data: &xatu.DecoratedEvent_EthV1ForkChoiceV2{
			EthV1ForkChoiceV2: data,
		},
	}

	additionalData := f.GetAdditionalData(ctx)

	decoratedEvent.Meta.Client.AdditionalData = &xatu.ClientMeta_EthV1DebugForkChoiceV2{
		EthV1DebugForkChoiceV2: additionalData,
	}

	return decoratedEvent, nil
}

func (f *ForkChoice) ShouldIgnore(ctx context.Context) (bool, error) {
	if err := f.beacon.Synced(ctx); err != nil {
		return true, err
	}

	return false, nil
}

func (f *ForkChoice) GetData() (*xatuethv1.ForkChoiceV2, error) {
	return xatuethv1.NewForkChoiceV2FromGoEth2ClientV1(f.snapshot.Event)
}

func (f *ForkChoice) GetAdditionalData(_ context.Context) *xatu.ClientMeta_AdditionalEthV1DebugForkChoiceV2Data {
	slot := f.beacon.Metadata().Wallclock().Slots().FromNumber(uint64(f.snapshot.RequestSlot))
	epoch := f.beacon.Metadata().Wallclock().Epochs().FromNumber(uint64(f.snapshot.RequestEpoch))

	extra := &xatu.ClientMeta_AdditionalEthV1DebugForkChoiceV2Data{
		Snapshot: &xatu.ClientMeta_ForkChoiceSnapshotV2{
			RequestedAtSlotStartDiffMs: &wrapperspb.UInt64Value{
				//nolint:gosec // not concerned in reality
				Value: uint64(f.snapshot.RequestAt.Sub(slot.TimeWindow().Start()).Milliseconds()),
			},
			RequestDurationMs: &wrapperspb.UInt64Value{
				//nolint:gosec // not concerned in reality
				Value: uint64(f.snapshot.RequestDuration.Milliseconds()),
			},
			Timestamp: timestamppb.New(f.snapshot.RequestAt),
		},
	}

	extra.Snapshot.RequestSlot = &xatu.SlotV2{
		StartDateTime: timestamppb.New(slot.TimeWindow().Start()),
		Number:        &wrapperspb.UInt64Value{Value: slot.Number()},
	}

	extra.Snapshot.RequestEpoch = &xatu.EpochV2{
		Number:        &wrapperspb.UInt64Value{Value: epoch.Number()},
		StartDateTime: timestamppb.New(epoch.TimeWindow().Start()),
	}

	return extra
}
