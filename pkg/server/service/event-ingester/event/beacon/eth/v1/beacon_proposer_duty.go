package v1

import (
	"context"
	"errors"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/sirupsen/logrus"
)

const (
	BeaconProposerDutyType = "BEACON_API_ETH_V1_PROPOSER_DUTY"
)

type BeaconProposerDuty struct {
	log   logrus.FieldLogger
	event *xatu.DecoratedEvent
}

func NewBeaconProposerDuty(log logrus.FieldLogger, event *xatu.DecoratedEvent) *BeaconProposerDuty {
	return &BeaconProposerDuty{
		log:   log.WithField("event", BeaconProposerDutyType),
		event: event,
	}
}

func (b *BeaconProposerDuty) Type() string {
	return BeaconProposerDutyType
}

func (b *BeaconProposerDuty) Validate(_ context.Context) error {
	_, ok := b.event.GetData().(*xatu.DecoratedEvent_EthV1ProposerDuty)
	if !ok {
		return errors.New("failed to cast event data")
	}

	return nil
}

func (b *BeaconProposerDuty) Filter(_ context.Context) bool {
	return false
}

func (b *BeaconProposerDuty) AppendServerMeta(_ context.Context, meta *xatu.ServerMeta) *xatu.ServerMeta {
	return meta
}
