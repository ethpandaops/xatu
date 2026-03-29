package v2

import (
	"context"
	"errors"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/sirupsen/logrus"
)

const (
	BeaconBlockExecutionPayloadBidType = "BEACON_API_ETH_V2_BEACON_BLOCK_EXECUTION_PAYLOAD_BID"
)

type BeaconBlockExecutionPayloadBid struct {
	log   logrus.FieldLogger
	event *xatu.DecoratedEvent
}

func NewBeaconBlockExecutionPayloadBid(log logrus.FieldLogger, event *xatu.DecoratedEvent) *BeaconBlockExecutionPayloadBid {
	return &BeaconBlockExecutionPayloadBid{
		log:   log.WithField("event", BeaconBlockExecutionPayloadBidType),
		event: event,
	}
}

func (b *BeaconBlockExecutionPayloadBid) Type() string {
	return BeaconBlockExecutionPayloadBidType
}

func (b *BeaconBlockExecutionPayloadBid) Validate(_ context.Context) error {
	_, ok := b.event.GetData().(*xatu.DecoratedEvent_EthV2BeaconBlockExecutionPayloadBid)
	if !ok {
		return errors.New("failed to cast event data")
	}

	return nil
}

func (b *BeaconBlockExecutionPayloadBid) Filter(_ context.Context) bool {
	return false
}

func (b *BeaconBlockExecutionPayloadBid) AppendServerMeta(_ context.Context, meta *xatu.ServerMeta) *xatu.ServerMeta {
	return meta
}
