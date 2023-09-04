package v2

import (
	"context"

	"github.com/attestantio/go-eth2-client/spec"
	xatuethv1 "github.com/ethpandaops/xatu/pkg/proto/eth/v1"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

const (
	ProposerSlashingDeriverName = "BEACON_API_ETH_V2_BEACON_BLOCK_PROPOSER_SLASHING"
)

type ProposerSlashingDeriver struct {
	log logrus.FieldLogger
}

func NewProposerSlashingDeriver(log logrus.FieldLogger) *ProposerSlashingDeriver {
	return &ProposerSlashingDeriver{
		log: log.WithField("module", "cannon/event/beacon/eth/v2/proposer_slashing"),
	}
}

func (b *ProposerSlashingDeriver) Name() string {
	return ProposerSlashingDeriverName
}

func (b *ProposerSlashingDeriver) Filter(ctx context.Context) bool {
	return false
}

func (b *ProposerSlashingDeriver) Process(ctx context.Context, metadata *BeaconBlockMetadata, block *spec.VersionedSignedBeaconBlock) ([]*xatu.DecoratedEvent, error) {
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

func (b *ProposerSlashingDeriver) getProposerSlashings(ctx context.Context, block *spec.VersionedSignedBeaconBlock) ([]*xatuethv1.ProposerSlashingV2, error) {
	slashings := []*xatuethv1.ProposerSlashingV2{}

	blockSlashings, err := block.ProposerSlashings()
	if err != nil {
		return nil, err
	}

	for _, slashing := range blockSlashings {
		slashings = append(slashings, &xatuethv1.ProposerSlashingV2{
			SignedHeader_1: &xatuethv1.SignedBeaconBlockHeaderV2{
				Message: &xatuethv1.BeaconBlockHeaderV2{
					Slot:          wrapperspb.UInt64(uint64(slashing.SignedHeader1.Message.Slot)),
					ProposerIndex: wrapperspb.UInt64(uint64(slashing.SignedHeader1.Message.ProposerIndex)),
					ParentRoot:    slashing.SignedHeader1.Message.ParentRoot.String(),
					StateRoot:     slashing.SignedHeader1.Message.StateRoot.String(),
					BodyRoot:      slashing.SignedHeader1.Message.BodyRoot.String(),
				},
				Signature: slashing.SignedHeader1.Signature.String(),
			},
			SignedHeader_2: &xatuethv1.SignedBeaconBlockHeaderV2{
				Message: &xatuethv1.BeaconBlockHeaderV2{
					Slot:          wrapperspb.UInt64(uint64(slashing.SignedHeader2.Message.Slot)),
					ProposerIndex: wrapperspb.UInt64(uint64(slashing.SignedHeader2.Message.ProposerIndex)),
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

func (b *ProposerSlashingDeriver) createEvent(ctx context.Context, metadata *BeaconBlockMetadata, slashing *xatuethv1.ProposerSlashingV2) (*xatu.DecoratedEvent, error) {
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
