package v2

import (
	"context"
	"fmt"
	"time"

	"github.com/attestantio/go-eth2-client/spec"
	"github.com/ethpandaops/xatu/pkg/cannon/ethereum"
	v1 "github.com/ethpandaops/xatu/pkg/proto/eth/v1"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

type BeaconBlockMetadata struct {
	log logrus.FieldLogger

	Now        time.Time
	ClientMeta *xatu.ClientMeta

	block *spec.VersionedSignedBeaconBlock

	beacon *ethereum.BeaconNode

	processors []BeaconBlockEventDeriver
}

func NewBeaconBlockMetadata(log logrus.FieldLogger, block *spec.VersionedSignedBeaconBlock, now time.Time, beacon *ethereum.BeaconNode, clientMeta *xatu.ClientMeta, processors []BeaconBlockEventDeriver) *BeaconBlockMetadata {
	return &BeaconBlockMetadata{
		log:        log,
		Now:        now,
		block:      block,
		beacon:     beacon,
		ClientMeta: clientMeta,
		processors: processors,
	}
}

func (e *BeaconBlockMetadata) BlockIdentifier() (*xatu.BlockIdentifier, error) {
	if e.block == nil {
		return nil, fmt.Errorf("block is nil")
	}

	slotNum, err := e.block.Slot()
	if err != nil {
		e.log.WithError(err).Error("Failed to get block slot")

		return nil, err
	}

	root, err := e.block.Root()
	if err != nil {
		e.log.WithError(err).Error("Failed to get block root")

		return nil, err
	}

	slot := e.beacon.Metadata().Wallclock().Slots().FromNumber(uint64(slotNum))
	epoch := e.beacon.Metadata().Wallclock().Epochs().FromSlot(uint64(slotNum))

	return &xatu.BlockIdentifier{
		Epoch: &xatu.EpochV2{
			Number:        &wrapperspb.UInt64Value{Value: epoch.Number()},
			StartDateTime: timestamppb.New(epoch.TimeWindow().Start()),
		},
		Slot: &xatu.SlotV2{
			Number:        &wrapperspb.UInt64Value{Value: slot.Number()},
			StartDateTime: timestamppb.New(slot.TimeWindow().Start()),
		},
		Root:    v1.RootAsString(root),
		Version: e.block.Version.String(),
	}, nil
}

func (e *BeaconBlockMetadata) Process(ctx context.Context) ([]*xatu.DecoratedEvent, error) {
	if e.block == nil {
		return nil, fmt.Errorf("block is nil")
	}

	slot, err := e.block.Slot()
	if err != nil {
		e.log.WithError(err).Error("Failed to get block slot")

		return nil, nil
	}

	root, err := e.block.Root()
	if err != nil {
		e.log.WithError(err).Error("Failed to get block root")

		return nil, err
	}

	events := []*xatu.DecoratedEvent{}

	for _, processor := range e.processors {
		evs, err := processor.Process(ctx, e, e.block)
		if err != nil {
			e.log.WithError(err).Error("Failed to process block")

			// Intentionally returning early here as we don't want to continue processing
			// if one processor fails.
			return nil, err
		}

		e.log.WithFields(logrus.Fields{
			"processor": processor.Name(),
			"events":    len(evs),
		}).Info("Processor finished processing block")

		events = append(events, evs...)
	}

	e.log.WithFields(logrus.Fields{
		"events": len(events),
		"slot":   slot,
		"root":   v1.RootAsString(root),
	}).Info("Processed block")

	return events, nil
}
