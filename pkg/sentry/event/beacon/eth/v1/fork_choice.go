package event

import (
	"context"
	"fmt"
	"time"

	eth2v1 "github.com/attestantio/go-eth2-client/api/v1"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	xatuethv1 "github.com/ethpandaops/xatu/pkg/proto/eth/v1"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/ethpandaops/xatu/pkg/sentry/ethereum"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type ForkChoice struct {
	log logrus.FieldLogger

	snapshot *ForkChoiceSnapshot

	beacon     *ethereum.BeaconNode
	clientMeta *xatu.ClientMeta
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
		log:        log.WithField("event", "BEACON_API_ETH_V1_DEBUG_FORK_CHOICE"),
		snapshot:   snapshot,
		beacon:     beacon,
		clientMeta: clientMeta,
	}
}

func (f *ForkChoice) Decorate(ctx context.Context) (*xatu.DecoratedEvent, error) {
	ignore, err := f.shouldIgnore(ctx)
	if err != nil {
		return nil, err
	}

	if ignore {
		//nolint:nilnil // Returning nil is intentional.
		return nil, nil
	}

	decoratedEvent := &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_BEACON_API_ETH_V1_DEBUG_FORK_CHOICE,
			DateTime: timestamppb.New(f.snapshot.RequestAt),
		},
		Meta: &xatu.Meta{
			Client: f.clientMeta,
		},
		Data: &xatu.DecoratedEvent_EthV1ForkChoice{
			EthV1ForkChoice: f.GetData(),
		},
	}

	additionalData := f.GetAdditionalData(ctx)

	decoratedEvent.Meta.Client.AdditionalData = &xatu.ClientMeta_EthV1DebugForkChoice{
		EthV1DebugForkChoice: additionalData,
	}

	return decoratedEvent, nil
}

func (f *ForkChoice) shouldIgnore(ctx context.Context) (bool, error) {
	if err := f.beacon.Synced(ctx); err != nil {
		return true, err
	}

	return false, nil
}

func (f *ForkChoice) GetData() *xatuethv1.ForkChoice {
	nodes := []*xatuethv1.ForkChoiceNode{}

	for _, node := range f.snapshot.Event.ForkChoiceNodes {
		nodes = append(nodes, &xatuethv1.ForkChoiceNode{
			Slot:               uint64(node.Slot),
			BlockRoot:          xatuethv1.RootAsString(node.BlockRoot),
			ParentRoot:         xatuethv1.RootAsString(node.ParentRoot),
			JustifiedEpoch:     uint64(node.JustifiedEpoch),
			FinalizedEpoch:     uint64(node.FinalizedEpoch),
			Weight:             node.Weight,
			Validity:           string(node.Validity),
			ExecutionBlockHash: xatuethv1.RootAsString(node.ExecutionBlockHash),
			ExtraData:          fmt.Sprintf("%v", node.ExtraData),
		})
	}

	return &xatuethv1.ForkChoice{
		FinalizedCheckpoint: &xatuethv1.Checkpoint{
			Epoch: uint64(f.snapshot.Event.FinalizedCheckpoint.Epoch),
			Root:  xatuethv1.RootAsString(f.snapshot.Event.FinalizedCheckpoint.Root),
		},
		JustifiedCheckpoint: &xatuethv1.Checkpoint{
			Epoch: uint64(f.snapshot.Event.JustifiedCheckpoint.Epoch),
			Root:  xatuethv1.RootAsString(f.snapshot.Event.JustifiedCheckpoint.Root),
		},
		ForkChoiceNodes: nodes,
	}
}

func (f *ForkChoice) GetAdditionalData(ctx context.Context) *xatu.ClientMeta_AdditionalEthV1DebugForkChoiceData {
	slot := f.beacon.Metadata().Wallclock().Slots().FromNumber(uint64(f.snapshot.RequestSlot))
	epoch := f.beacon.Metadata().Wallclock().Epochs().FromSlot(uint64(f.snapshot.RequestEpoch))

	extra := &xatu.ClientMeta_AdditionalEthV1DebugForkChoiceData{
		Snapshot: &xatu.ClientMeta_ForkChoiceSnapshot{
			RequestedAtSlotStartDiffMs: uint64(f.snapshot.RequestAt.Sub(slot.TimeWindow().Start()).Milliseconds()),
			RequestDurationMs:          uint64(f.snapshot.RequestDuration.Milliseconds()),
		},
	}

	extra.Snapshot.RequestSlot = &xatu.Slot{
		StartDateTime: timestamppb.New(slot.TimeWindow().Start()),
		Number:        slot.Number(),
	}

	extra.Snapshot.RequestEpoch = &xatu.Epoch{
		Number:        epoch.Number(),
		StartDateTime: timestamppb.New(epoch.TimeWindow().Start()),
	}

	return extra
}
