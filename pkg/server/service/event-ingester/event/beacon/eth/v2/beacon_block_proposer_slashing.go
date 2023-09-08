package v2

import (
	"context"
	"errors"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/sirupsen/logrus"
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
