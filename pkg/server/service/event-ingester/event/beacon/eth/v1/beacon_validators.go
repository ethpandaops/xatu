package v1

import (
	"context"
	"errors"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/sirupsen/logrus"
)

var (
	BeaconValidatorsType = xatu.Event_BEACON_API_ETH_V1_BEACON_VALIDATORS.String()
)

type BeaconValidators struct {
	log   logrus.FieldLogger
	event *xatu.DecoratedEvent
}

func NewBeaconValidators(log logrus.FieldLogger, event *xatu.DecoratedEvent) *BeaconValidators {
	return &BeaconValidators{
		log:   log.WithField("event", BeaconValidatorsType),
		event: event,
	}
}

func (b *BeaconValidators) Type() string {
	return BeaconValidatorsType
}

func (b *BeaconValidators) Validate(_ context.Context) error {
	data, ok := b.event.GetData().(*xatu.DecoratedEvent_EthV1Validators)
	if !ok {
		return errors.New("failed to cast event data")
	}

	if len(data.EthV1Validators.Validators) == 0 {
		return errors.New("no validators found")
	}

	return nil
}

func (b *BeaconValidators) Filter(_ context.Context) bool {
	return false
}

func (b *BeaconValidators) AppendServerMeta(_ context.Context, meta *xatu.ServerMeta) *xatu.ServerMeta {
	return meta
}
