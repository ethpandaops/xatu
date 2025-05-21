package v2

import (
	"context"
	"errors"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/sirupsen/logrus"
)

const (
	BeaconBlockConsolidationRequestType = "BEACON_API_ETH_V2_BEACON_BLOCK_CONSOLIDATION_REQUEST"
)

type BeaconBlockConsolidationRequest struct {
	log   logrus.FieldLogger
	event *xatu.DecoratedEvent
}

func NewBeaconBlockConsolidationRequest(log logrus.FieldLogger, event *xatu.DecoratedEvent) *BeaconBlockConsolidationRequest {
	return &BeaconBlockConsolidationRequest{
		log:   log.WithField("event", BeaconBlockConsolidationRequestType),
		event: event,
	}
}

func (b *BeaconBlockConsolidationRequest) Type() string {
	return BeaconBlockConsolidationRequestType
}

func (b *BeaconBlockConsolidationRequest) Validate(ctx context.Context) error {
	_, ok := b.event.Data.(*xatu.DecoratedEvent_EthV2BeaconBlockConsolidationRequest)
	if !ok {
		return errors.New("failed to cast event data")
	}

	return nil
}

func (b *BeaconBlockConsolidationRequest) Filter(ctx context.Context) bool {
	return false
}

func (b *BeaconBlockConsolidationRequest) AppendServerMeta(ctx context.Context, meta *xatu.ServerMeta) *xatu.ServerMeta {
	return meta
}