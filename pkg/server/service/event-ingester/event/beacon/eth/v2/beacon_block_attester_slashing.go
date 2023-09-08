package v2

import (
	"context"
	"errors"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/sirupsen/logrus"
)

const (
	BeaconBlockAttesterSlashingType = "BEACON_API_ETH_V2_BEACON_BLOCK_ATTESTER_SLASHING"
)

type BeaconBlockAttesterSlashing struct {
	log   logrus.FieldLogger
	event *xatu.DecoratedEvent
}

func NewBeaconBlockAttesterSlashing(log logrus.FieldLogger, event *xatu.DecoratedEvent) *BeaconBlockAttesterSlashing {
	return &BeaconBlockAttesterSlashing{
		log:   log.WithField("event", BeaconBlockAttesterSlashingType),
		event: event,
	}
}

func (b *BeaconBlockAttesterSlashing) Type() string {
	return BeaconBlockAttesterSlashingType
}

func (b *BeaconBlockAttesterSlashing) Validate(ctx context.Context) error {
	_, ok := b.event.Data.(*xatu.DecoratedEvent_EthV2BeaconBlockAttesterSlashing)
	if !ok {
		return errors.New("failed to cast event data")
	}

	return nil
}

func (b *BeaconBlockAttesterSlashing) Filter(ctx context.Context) bool {
	return false
}
