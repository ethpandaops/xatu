package v1

import (
	"context"
	"errors"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/sirupsen/logrus"
)

const (
	BeaconBlobType = "BEACON_API_ETH_V1_BEACON_BLOB"
)

type BeaconBlob struct {
	log   logrus.FieldLogger
	event *xatu.DecoratedEvent
}

func NewBeaconBlob(log logrus.FieldLogger, event *xatu.DecoratedEvent) *BeaconBlob {
	return &BeaconBlob{
		log:   log.WithField("event", BeaconBlobType),
		event: event,
	}
}

func (b *BeaconBlob) Type() string {
	return BeaconBlobType
}

func (b *BeaconBlob) Validate(ctx context.Context) error {
	_, ok := b.event.GetData().(*xatu.DecoratedEvent_EthV1BeaconBlob)
	if !ok {
		return errors.New("failed to cast event data")
	}

	return nil
}

func (b *BeaconBlob) Filter(ctx context.Context) bool {
	return false
}

func (b *BeaconBlob) AppendServerMeta(ctx context.Context, meta *xatu.ServerMeta) *xatu.ServerMeta {
	return meta
}
