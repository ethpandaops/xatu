package v1

import (
	"context"
	"errors"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/sirupsen/logrus"
)

const (
	BeaconCommitteeType = "BEACON_API_ETH_V1_BEACON_COMMITTEE"
)

type BeaconCommittee struct {
	log   logrus.FieldLogger
	event *xatu.DecoratedEvent
}

func NewBeaconCommittee(log logrus.FieldLogger, event *xatu.DecoratedEvent) *BeaconCommittee {
	return &BeaconCommittee{
		log:   log.WithField("event", BeaconCommitteeType),
		event: event,
	}
}

func (b *BeaconCommittee) Type() string {
	return BeaconCommitteeType
}

func (b *BeaconCommittee) Validate(_ context.Context) error {
	_, ok := b.event.GetData().(*xatu.DecoratedEvent_EthV1BeaconCommittee)
	if !ok {
		return errors.New("failed to cast event data")
	}

	return nil
}

func (b *BeaconCommittee) Filter(_ context.Context) bool {
	return false
}

func (b *BeaconCommittee) AppendServerMeta(_ context.Context, meta *xatu.ServerMeta) *xatu.ServerMeta {
	return meta
}
