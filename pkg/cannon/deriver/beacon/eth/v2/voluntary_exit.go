package v2

import (
	"context"
	"fmt"

	"github.com/attestantio/go-eth2-client/spec"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	xatuethv1 "github.com/ethpandaops/xatu/pkg/proto/eth/v1"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

const (
	VoluntaryExitDeriverName = xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_VOLUNTARY_EXIT
)

type VoluntaryExitDeriver struct {
	log logrus.FieldLogger
	cfg *VoluntaryExitDeriverConfig
}

type VoluntaryExitDeriverConfig struct {
	Enabled     *bool   `yaml:"enabled" default:"true"`
	HeadSlotLag *uint64 `yaml:"headSlotLag" default:"1"`
}

func NewVoluntaryExitDeriver(log logrus.FieldLogger, config *VoluntaryExitDeriverConfig) *VoluntaryExitDeriver {
	return &VoluntaryExitDeriver{
		log: log.WithField("module", "cannon/event/beacon/eth/v2/voluntary_exit"),
		cfg: config,
	}
}

func (b *VoluntaryExitDeriver) CannonType() xatu.CannonType {
	return VoluntaryExitDeriverName
}

func (b *VoluntaryExitDeriver) Name() string {
	return VoluntaryExitDeriverName.String()
}

func (b *VoluntaryExitDeriver) Process(ctx context.Context, metadata *BeaconBlockMetadata, block *spec.VersionedSignedBeaconBlock) ([]*xatu.DecoratedEvent, error) {
	events := []*xatu.DecoratedEvent{}

	exits, err := b.getVoluntaryExits(ctx, block)
	if err != nil {
		return nil, err
	}

	for _, exit := range exits {
		event, err := b.createEvent(ctx, metadata, exit)
		if err != nil {
			b.log.WithError(err).Error("Failed to create event")

			return nil, errors.Wrapf(err, "failed to create event for voluntary exit %s", exit.String())
		}

		events = append(events, event)
	}

	return events, nil
}

func (b *VoluntaryExitDeriver) getVoluntaryExits(ctx context.Context, block *spec.VersionedSignedBeaconBlock) ([]*xatuethv1.SignedVoluntaryExitV2, error) {
	exits := []*xatuethv1.SignedVoluntaryExitV2{}

	var voluntaryExits []*phase0.SignedVoluntaryExit

	switch block.Version {
	case spec.DataVersionPhase0:
		voluntaryExits = block.Phase0.Message.Body.VoluntaryExits
	case spec.DataVersionAltair:
		voluntaryExits = block.Altair.Message.Body.VoluntaryExits
	case spec.DataVersionBellatrix:
		voluntaryExits = block.Bellatrix.Message.Body.VoluntaryExits
	case spec.DataVersionCapella:
		voluntaryExits = block.Capella.Message.Body.VoluntaryExits
	default:
		return nil, fmt.Errorf("unsupported block version: %s", block.Version.String())
	}

	for _, exit := range voluntaryExits {
		exits = append(exits, &xatuethv1.SignedVoluntaryExitV2{
			Message: &xatuethv1.VoluntaryExitV2{
				Epoch:          wrapperspb.UInt64(uint64(exit.Message.Epoch)),
				ValidatorIndex: wrapperspb.UInt64(uint64(exit.Message.ValidatorIndex)),
			},
			Signature: exit.Signature.String(),
		})
	}

	return exits, nil
}

func (b *VoluntaryExitDeriver) createEvent(ctx context.Context, metadata *BeaconBlockMetadata, exit *xatuethv1.SignedVoluntaryExitV2) (*xatu.DecoratedEvent, error) {
	decoratedEvent := &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_VOLUNTARY_EXIT,
			DateTime: timestamppb.New(metadata.Now),
			Id:       uuid.New().String(),
		},
		Meta: &xatu.Meta{
			Client: metadata.ClientMeta,
		},
		Data: &xatu.DecoratedEvent_EthV2BeaconBlockVoluntaryExit{
			EthV2BeaconBlockVoluntaryExit: exit,
		},
	}

	blockIdentifier, err := metadata.BlockIdentifier()
	if err != nil {
		return nil, err
	}

	decoratedEvent.Meta.Client.AdditionalData = &xatu.ClientMeta_EthV2BeaconBlockVoluntaryExit{
		EthV2BeaconBlockVoluntaryExit: &xatu.ClientMeta_AdditionalEthV2BeaconBlockVoluntaryExitData{
			Block: blockIdentifier,
		},
	}

	return decoratedEvent, nil
}
