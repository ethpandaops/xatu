package v1

import (
	"context"
	"errors"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/sirupsen/logrus"
)

const (
	BeaconBlobSidecarType = "BEACON_API_ETH_V1_BEACON_BLOB_SIDECAR"
)

type BeaconBlobSidecar struct {
	log   logrus.FieldLogger
	event *xatu.DecoratedEvent
}

func NewBeaconBlobSidecar(log logrus.FieldLogger, event *xatu.DecoratedEvent) *BeaconBlobSidecar {
	return &BeaconBlobSidecar{
		log:   log.WithField("event", BeaconBlobSidecarType),
		event: event,
	}
}

func (b *BeaconBlobSidecar) Type() string {
	return BeaconBlobSidecarType
}

func (b *BeaconBlobSidecar) Validate(_ context.Context) error {
	_, ok := b.event.GetData().(*xatu.DecoratedEvent_EthV1BeaconBlockBlobSidecar)
	if !ok {
		return errors.New("failed to cast event data")
	}

	return nil
}

func (b *BeaconBlobSidecar) Filter(_ context.Context) bool {
	return false
}
