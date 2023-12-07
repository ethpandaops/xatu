package v2

import (
	"context"
	"errors"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/sirupsen/logrus"
)

const (
	BeaconBlockDepositType = "BEACON_API_ETH_V2_BEACON_BLOCK_DEPOSIT"
)

type BeaconBlockDeposit struct {
	log   logrus.FieldLogger
	event *xatu.DecoratedEvent
}

func NewBeaconBlockDeposit(log logrus.FieldLogger, event *xatu.DecoratedEvent) *BeaconBlockDeposit {
	return &BeaconBlockDeposit{
		log:   log.WithField("event", BeaconBlockDepositType),
		event: event,
	}
}

func (b *BeaconBlockDeposit) Type() string {
	return BeaconBlockDepositType
}

func (b *BeaconBlockDeposit) Validate(ctx context.Context) error {
	_, ok := b.event.Data.(*xatu.DecoratedEvent_EthV2BeaconBlockDeposit)
	if !ok {
		return errors.New("failed to cast event data")
	}

	return nil
}

func (b *BeaconBlockDeposit) Filter(ctx context.Context) bool {
	return false
}

func (b *BeaconBlockDeposit) AppendServerMeta(ctx context.Context, meta *xatu.ServerMeta) *xatu.ServerMeta {
	return meta
}
