package v2

import (
	"context"
	"errors"

	"github.com/attestantio/go-eth2-client/spec"
	xatuethv1 "github.com/ethpandaops/xatu/pkg/proto/eth/v1"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	BeaconBlockProposerSlashingType = "BEACON_API_ETH_V2_BEACON_BLOCK_PROPOSER_SLASHING"
)

type BeaconBlockProposerSlashing struct {
	log   logrus.FieldLogger
	event *xatu.DecoratedEvent
}

func NewBeaconBlockProposerSlashing(log logrus.FieldLogger, event *xatu.DecoratedEvent) *BeaconBlockProposerSlashing {
	return &BeaconBlockProposerSlashing{
		log:   log.WithField("event", BeaconBlockProposerSlashingType),
		event: event,
	}
}

func (b *BeaconBlockProposerSlashing) Type() string {
	return BeaconBlockProposerSlashingType
}

func (b *BeaconBlockProposerSlashing) Validate(ctx context.Context) error {
	_, ok := b.event.Data.(*xatu.DecoratedEvent_EthV2BeaconBlockProposerSlashing)
	if !ok {
		return errors.New("failed to cast event data")
	}

	return nil
}

func (b *BeaconBlockProposerSlashing) Filter(ctx context.Context) bool {
	return false
}

func (b *BeaconBlockProposerSlashing) Process(ctx context.Context, metadata *BeaconBlockMetadata, block *spec.VersionedSignedBeaconBlock) ([]*xatu.DecoratedEvent, error) {
	events := []*xatu.DecoratedEvent{}

	slashings, err := b.getProposerSlashings(ctx, block)
	if err != nil {
		return nil, err
	}

	for _, slashing := range slashings {
		event, err := b.createEvent(ctx, metadata, slashing)
		if err != nil {
			b.log.WithError(err).Error("Failed to create event")

			continue
		}

		events = append(events, event)
	}

	return events, nil
}

func (b *BeaconBlockProposerSlashing) getProposerSlashings(ctx context.Context, block *spec.VersionedSignedBeaconBlock) ([]*xatuethv1.ProposerSlashing, error) {
	slashings := []*xatuethv1.ProposerSlashing{}

	blockSlashings, err := block.ProposerSlashings()
	if err != nil {
		return nil, err
	}

	for _, slashing := range blockSlashings {
		slashings = append(slashings, &xatuethv1.ProposerSlashing{
			SignedHeader_1: &xatuethv1.SignedBeaconBlockHeader{
				Message: &xatuethv1.BeaconBlockHeader{
					Slot:          uint64(slashing.SignedHeader1.Message.Slot),
					ProposerIndex: uint64(slashing.SignedHeader1.Message.ProposerIndex),
					ParentRoot:    slashing.SignedHeader1.Message.ParentRoot.String(),
					StateRoot:     slashing.SignedHeader1.Message.StateRoot.String(),
					BodyRoot:      slashing.SignedHeader1.Message.BodyRoot.String(),
				},
				Signature: slashing.SignedHeader1.Signature.String(),
			},
			SignedHeader_2: &xatuethv1.SignedBeaconBlockHeader{
				Message: &xatuethv1.BeaconBlockHeader{
					Slot:          uint64(slashing.SignedHeader2.Message.Slot),
					ProposerIndex: uint64(slashing.SignedHeader2.Message.ProposerIndex),
					ParentRoot:    slashing.SignedHeader2.Message.ParentRoot.String(),
					StateRoot:     slashing.SignedHeader2.Message.StateRoot.String(),
					BodyRoot:      slashing.SignedHeader2.Message.BodyRoot.String(),
				},
				Signature: slashing.SignedHeader2.Signature.String(),
			},
		})
	}

	return slashings, nil
}

func (b *BeaconBlockProposerSlashing) createEvent(ctx context.Context, metadata *BeaconBlockMetadata, slashing *xatuethv1.ProposerSlashing) (*xatu.DecoratedEvent, error) {
	decoratedEvent := &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_PROPOSER_SLASHING,
			DateTime: timestamppb.New(metadata.Now),
			Id:       uuid.New().String(),
		},
		Meta: &xatu.Meta{
			Client: metadata.ClientMeta,
		},
		Data: &xatu.DecoratedEvent_EthV2BeaconBlockProposerSlashing{
			EthV2BeaconBlockProposerSlashing: slashing,
		},
	}

	blockIdentifier, err := metadata.BlockIdentifier()
	if err != nil {
		return nil, err
	}

	decoratedEvent.Meta.Client.AdditionalData = &xatu.ClientMeta_EthV2BeaconBlockProposerSlashing{
		EthV2BeaconBlockProposerSlashing: &xatu.ClientMeta_AdditionalEthV2BeaconBlockProposerSlashingData{
			Block: blockIdentifier,
		},
	}

	return decoratedEvent, nil
}
