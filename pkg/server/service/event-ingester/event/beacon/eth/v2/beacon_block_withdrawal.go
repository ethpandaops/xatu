package v2

import (
	"context"
	"errors"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/sirupsen/logrus"
)

const (
	BeaconBlockWithdrawalType = "BEACON_API_ETH_V2_BEACON_BLOCK_WITHDRAWAL"
)

type BeaconBlockWithdrawal struct {
	log   logrus.FieldLogger
	event *xatu.DecoratedEvent
}

func NewBeaconBlockWithdrawal(log logrus.FieldLogger, event *xatu.DecoratedEvent) *BeaconBlockWithdrawal {
	return &BeaconBlockWithdrawal{
		log:   log.WithField("event", BeaconBlockWithdrawalType),
		event: event,
	}
}

func (b *BeaconBlockWithdrawal) Type() string {
	return BeaconBlockWithdrawalType
}

func (b *BeaconBlockWithdrawal) Validate(ctx context.Context) error {
	_, ok := b.event.Data.(*xatu.DecoratedEvent_EthV2BeaconBlockWithdrawal)
	if !ok {
		return errors.New("failed to cast event data")
	}

	return nil
}

func (b *BeaconBlockWithdrawal) Filter(ctx context.Context) bool {
	return false
}

func (b *BeaconBlockWithdrawal) AppendServerMeta(ctx context.Context, meta *xatu.ServerMeta) *xatu.ServerMeta {
	return meta
}
