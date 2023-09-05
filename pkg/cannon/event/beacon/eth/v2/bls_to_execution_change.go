package v2

import (
	"context"
	"fmt"

	"github.com/attestantio/go-eth2-client/spec"
	xatuethv2 "github.com/ethpandaops/xatu/pkg/proto/eth/v2"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

const (
	BLSToExecutionChangeDeriverName = "BEACON_API_ETH_V2_BEACON_BLOCK_BLS_TO_EXECUTION_CHANGE"
)

type BLSToExecutionChangeDeriver struct {
	log logrus.FieldLogger
}

func NewBLSToExecutionChangeDeriver(log logrus.FieldLogger) *BLSToExecutionChangeDeriver {
	return &BLSToExecutionChangeDeriver{
		log: log.WithField("module", "cannon/event/beacon/eth/v2/bls_to_execution_change"),
	}
}

func (b *BLSToExecutionChangeDeriver) Name() string {
	return BLSToExecutionChangeDeriverName
}

func (b *BLSToExecutionChangeDeriver) Filter(ctx context.Context) bool {
	return false
}

func (b *BLSToExecutionChangeDeriver) Process(ctx context.Context, metadata *BeaconBlockMetadata, block *spec.VersionedSignedBeaconBlock) ([]*xatu.DecoratedEvent, error) {
	events := []*xatu.DecoratedEvent{}

	changes, err := b.getBLSToExecutionChanges(ctx, block)
	if err != nil {
		return nil, err
	}

	for _, change := range changes {
		event, err := b.createEvent(ctx, metadata, change)
		if err != nil {
			b.log.WithError(err).Error("Failed to create event")

			return nil, errors.Wrapf(err, "failed to create event for BLS to execution change %s", change.String())
		}

		events = append(events, event)
	}

	return events, nil
}

func (b *BLSToExecutionChangeDeriver) getBLSToExecutionChanges(ctx context.Context, block *spec.VersionedSignedBeaconBlock) ([]*xatuethv2.SignedBLSToExecutionChangeV2, error) {
	changes := []*xatuethv2.SignedBLSToExecutionChangeV2{}

	switch block.Version {
	case spec.DataVersionPhase0:
		return changes, nil
	case spec.DataVersionAltair:
		return changes, nil
	case spec.DataVersionBellatrix:
		return changes, nil
	case spec.DataVersionCapella:
		for _, change := range block.Capella.Message.Body.BLSToExecutionChanges {
			changes = append(changes, &xatuethv2.SignedBLSToExecutionChangeV2{
				Message: &xatuethv2.BLSToExecutionChangeV2{
					ValidatorIndex:     wrapperspb.UInt64(uint64(change.Message.ValidatorIndex)),
					FromBlsPubkey:      change.Message.FromBLSPubkey.String(),
					ToExecutionAddress: change.Message.ToExecutionAddress.String(),
				},
				Signature: change.Signature.String(),
			})
		}
	default:
		return nil, fmt.Errorf("unsupported block version: %s", block.Version.String())
	}

	return changes, nil
}

func (b *BLSToExecutionChangeDeriver) createEvent(ctx context.Context, metadata *BeaconBlockMetadata, change *xatuethv2.SignedBLSToExecutionChangeV2) (*xatu.DecoratedEvent, error) {
	decoratedEvent := &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_BLS_TO_EXECUTION_CHANGE,
			DateTime: timestamppb.New(metadata.Now),
			Id:       uuid.New().String(),
		},
		Meta: &xatu.Meta{
			Client: metadata.ClientMeta,
		},
		Data: &xatu.DecoratedEvent_EthV2BeaconBlockBlsToExecutionChange{
			EthV2BeaconBlockBlsToExecutionChange: change,
		},
	}

	blockIdentifier, err := metadata.BlockIdentifier()
	if err != nil {
		return nil, err
	}

	decoratedEvent.Meta.Client.AdditionalData = &xatu.ClientMeta_EthV2BeaconBlockBlsToExecutionChange{
		EthV2BeaconBlockBlsToExecutionChange: &xatu.ClientMeta_AdditionalEthV2BeaconBlockBLSToExecutionChangeData{
			Block: blockIdentifier,
		},
	}

	return decoratedEvent, nil
}
