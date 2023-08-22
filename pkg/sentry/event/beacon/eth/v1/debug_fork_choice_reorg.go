package event

import (
	"context"
	"time"

	xatuethv1 "github.com/ethpandaops/xatu/pkg/proto/eth/v1"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/ethpandaops/xatu/pkg/sentry/ethereum"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

type ForkChoiceReOrg struct {
	log logrus.FieldLogger

	snapshot *ForkChoiceReOrgSnapshot

	beacon     *ethereum.BeaconNode
	clientMeta *xatu.ClientMeta
	id         uuid.UUID
}

type ForkChoiceReOrgSnapshot struct {
	ReOrgEventAt time.Time
	Before       *ForkChoice
	After        *ForkChoice
	Event        *xatuethv1.EventChainReorgV2
}

func NewForkChoiceReOrg(log logrus.FieldLogger, snapshot *ForkChoiceReOrgSnapshot, beacon *ethereum.BeaconNode, clientMeta *xatu.ClientMeta) *ForkChoiceReOrg {
	return &ForkChoiceReOrg{
		log:        log.WithField("event", "BEACON_API_ETH_V1_DEBUG_FORK_CHOICE_REORG_V2"),
		snapshot:   snapshot,
		beacon:     beacon,
		clientMeta: clientMeta,
		id:         uuid.New(),
	}
}

func (f *ForkChoiceReOrg) Decorate(ctx context.Context) (*xatu.DecoratedEvent, error) {
	ignore, err := f.ShouldIgnore(ctx)
	if err != nil {
		return nil, err
	}

	if ignore {
		//nolint:nilnil // Returning nil is intentional.
		return nil, nil
	}

	data := &xatu.DebugForkChoiceReorgV2{
		Event: f.snapshot.Event,
	}

	additional := &xatu.ClientMeta_AdditionalEthV1DebugForkChoiceReOrgV2Data{}

	if f.snapshot.Before != nil {
		before, err := f.snapshot.Before.GetData()
		if err == nil {
			data.Before = before
		}

		beforeAdditional := f.snapshot.Before.GetAdditionalData(ctx)

		additional.Before = &xatu.ClientMeta_ForkChoiceSnapshotV2{
			RequestEpoch: beforeAdditional.Snapshot.RequestEpoch,
			RequestSlot:  beforeAdditional.Snapshot.RequestSlot,
			RequestedAtSlotStartDiffMs: &wrapperspb.UInt64Value{
				Value: beforeAdditional.Snapshot.RequestedAtSlotStartDiffMs.Value,
			},
			RequestDurationMs: &wrapperspb.UInt64Value{
				Value: beforeAdditional.Snapshot.RequestDurationMs.Value,
			},
			Timestamp: beforeAdditional.Snapshot.Timestamp,
		}
	}

	if f.snapshot.After != nil {
		after, err := f.snapshot.After.GetData()
		if err == nil {
			data.After = after
		}

		afterAdditional := f.snapshot.After.GetAdditionalData(ctx)

		additional.After = &xatu.ClientMeta_ForkChoiceSnapshotV2{
			RequestEpoch: afterAdditional.Snapshot.RequestEpoch,
			RequestSlot:  afterAdditional.Snapshot.RequestSlot,
			RequestedAtSlotStartDiffMs: &wrapperspb.UInt64Value{
				Value: afterAdditional.Snapshot.RequestedAtSlotStartDiffMs.Value,
			},
			RequestDurationMs: &wrapperspb.UInt64Value{
				Value: afterAdditional.Snapshot.RequestDurationMs.Value,
			},
			Timestamp: afterAdditional.Snapshot.Timestamp,
		}
	}

	decoratedEvent := &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_BEACON_API_ETH_V1_DEBUG_FORK_CHOICE_REORG_V2,
			DateTime: timestamppb.New(f.snapshot.ReOrgEventAt),
			Id:       f.id.String(),
		},
		Meta: &xatu.Meta{
			Client: f.clientMeta,
		},
		Data: &xatu.DecoratedEvent_EthV1ForkChoiceReorgV2{
			EthV1ForkChoiceReorgV2: data,
		},
	}

	decoratedEvent.Meta.Client.AdditionalData = &xatu.ClientMeta_EthV1DebugForkChoiceReorgV2{
		EthV1DebugForkChoiceReorgV2: additional,
	}

	return decoratedEvent, nil
}

func (f *ForkChoiceReOrg) ShouldIgnore(ctx context.Context) (bool, error) {
	if err := f.beacon.Synced(ctx); err != nil {
		return true, err
	}

	return false, nil
}
