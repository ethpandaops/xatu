package synthetic

import (
	"context"
	"errors"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/sirupsen/logrus"
)

const (
	PayloadStatusResolvedType = "BEACON_SYNTHETIC_PAYLOAD_STATUS_RESOLVED"
)

// PayloadStatusResolved handles BEACON_SYNTHETIC_PAYLOAD_STATUS_RESOLVED events
// — fork-choice slot payload status transitions synthesized from beacon-node
// internals (TYSM-instrumented, EIP-7732 ePBS).
type PayloadStatusResolved struct {
	log   logrus.FieldLogger
	event *xatu.DecoratedEvent
}

func NewPayloadStatusResolved(log logrus.FieldLogger, event *xatu.DecoratedEvent) *PayloadStatusResolved {
	return &PayloadStatusResolved{
		log:   log.WithField("event", PayloadStatusResolvedType),
		event: event,
	}
}

func (e *PayloadStatusResolved) Type() string {
	return PayloadStatusResolvedType
}

func (e *PayloadStatusResolved) Validate(_ context.Context) error {
	_, ok := e.event.GetData().(*xatu.DecoratedEvent_BeaconSyntheticPayloadStatusResolved)
	if !ok {
		return errors.New("failed to cast event data")
	}

	return nil
}

func (e *PayloadStatusResolved) Filter(_ context.Context) bool {
	return false
}

func (e *PayloadStatusResolved) AppendServerMeta(_ context.Context, meta *xatu.ServerMeta) *xatu.ServerMeta {
	return meta
}
