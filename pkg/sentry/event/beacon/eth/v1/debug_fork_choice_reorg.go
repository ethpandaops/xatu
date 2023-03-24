package event

import (
	"context"
	"time"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/ethpandaops/xatu/pkg/sentry/ethereum"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type ForkChoiceReOrg struct {
	log logrus.FieldLogger

	snapshot *ForkChoiceReOrgSnapshot

	beacon     *ethereum.BeaconNode
	clientMeta *xatu.ClientMeta
}

type ForkChoiceReOrgSnapshot struct {
	ReOrgEventAt time.Time
	Before       *ForkChoice
	After        *ForkChoice
}

func NewForkChoiceReOrg(log logrus.FieldLogger, snapshot *ForkChoiceReOrgSnapshot, beacon *ethereum.BeaconNode, clientMeta *xatu.ClientMeta) *ForkChoiceReOrg {
	return &ForkChoiceReOrg{
		log:        log.WithField("event", "BEACON_API_ETH_V1_DEBUG_FORK_CHOICE_REORG"),
		snapshot:   snapshot,
		beacon:     beacon,
		clientMeta: clientMeta,
	}
}

func (f *ForkChoiceReOrg) Decorate(ctx context.Context) (*xatu.DecoratedEvent, error) {
	ignore, err := f.shouldIgnore(ctx)
	if err != nil {
		return nil, err
	}

	if ignore {
		//nolint:nilnil // Returning nil is intentional.
		return nil, nil
	}

	data := &xatu.DebugForkChoiceReorg{}
	additional := &xatu.ClientMeta_AdditionalEthV1DebugForkChoiceReOrgData{}

	if f.snapshot.Before != nil {
		data.Before = f.snapshot.Before.GetData()

		beforeAdditional := f.snapshot.Before.GetAdditionalData(ctx)

		additional.Before = &xatu.ClientMeta_ForkChoiceSnapshot{
			RequestEpoch:               beforeAdditional.Snapshot.RequestEpoch,
			RequestSlot:                beforeAdditional.Snapshot.RequestSlot,
			RequestedAtSlotStartDiffMs: beforeAdditional.Snapshot.RequestedAtSlotStartDiffMs,
			RequestDurationMs:          beforeAdditional.Snapshot.RequestDurationMs,
		}
	}

	if f.snapshot.After != nil {
		data.After = f.snapshot.Before.GetData()

		afterAdditional := f.snapshot.Before.GetAdditionalData(ctx)

		additional.Before = &xatu.ClientMeta_ForkChoiceSnapshot{
			RequestEpoch:               afterAdditional.Snapshot.RequestEpoch,
			RequestSlot:                afterAdditional.Snapshot.RequestSlot,
			RequestedAtSlotStartDiffMs: afterAdditional.Snapshot.RequestedAtSlotStartDiffMs,
			RequestDurationMs:          afterAdditional.Snapshot.RequestDurationMs,
		}
	}

	decoratedEvent := &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_BEACON_API_ETH_V1_DEBUG_FORK_CHOICE_REORG,
			DateTime: timestamppb.New(f.snapshot.ReOrgEventAt),
		},
		Meta: &xatu.Meta{
			Client: f.clientMeta,
		},
		Data: &xatu.DecoratedEvent_EthV1ForkChoiceReorg{
			EthV1ForkChoiceReorg: data,
		},
	}

	decoratedEvent.Meta.Client.AdditionalData = &xatu.ClientMeta_EthV1DebugForkChoiceReorg{
		EthV1DebugForkChoiceReorg: additional,
	}

	return decoratedEvent, nil
}

func (f *ForkChoiceReOrg) shouldIgnore(ctx context.Context) (bool, error) {
	if err := f.beacon.Synced(ctx); err != nil {
		return true, err
	}

	return false, nil
}
