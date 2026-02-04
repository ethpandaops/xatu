package v1

import (
	"context"
	"errors"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/sirupsen/logrus"
)

const (
	BeaconSyncCommitteeType = "BEACON_API_ETH_V1_BEACON_SYNC_COMMITTEE"
)

type BeaconSyncCommittee struct {
	log   logrus.FieldLogger
	event *xatu.DecoratedEvent
}

func NewBeaconSyncCommittee(log logrus.FieldLogger, event *xatu.DecoratedEvent) *BeaconSyncCommittee {
	return &BeaconSyncCommittee{
		log:   log.WithField("event", BeaconSyncCommitteeType),
		event: event,
	}
}

func (b *BeaconSyncCommittee) Type() string {
	return BeaconSyncCommitteeType
}

func (b *BeaconSyncCommittee) Validate(_ context.Context) error {
	_, ok := b.event.GetData().(*xatu.DecoratedEvent_EthV1BeaconSyncCommittee)
	if !ok {
		return errors.New("failed to cast event data")
	}

	return nil
}

func (b *BeaconSyncCommittee) Filter(_ context.Context) bool {
	return false
}

func (b *BeaconSyncCommittee) AppendServerMeta(_ context.Context, meta *xatu.ServerMeta) *xatu.ServerMeta {
	return meta
}
