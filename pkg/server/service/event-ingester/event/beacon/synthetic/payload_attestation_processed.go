package synthetic

import (
	"context"
	"errors"

	"github.com/ethpandaops/xatu/pkg/observability"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

const (
	PayloadAttestationProcessedType = "BEACON_SYNTHETIC_PAYLOAD_ATTESTATION_PROCESSED"
)

// PayloadAttestationProcessed handles BEACON_SYNTHETIC_PAYLOAD_ATTESTATION_PROCESSED events
// — per-PTC-vote observation synthesized from beacon-node internals
// (TYSM-instrumented, EIP-7732 ePBS).
type PayloadAttestationProcessed struct {
	log   observability.ContextualLogger
	event *xatu.DecoratedEvent
}

func NewPayloadAttestationProcessed(log observability.ContextualLogger, event *xatu.DecoratedEvent) *PayloadAttestationProcessed {
	return &PayloadAttestationProcessed{
		log:   log.WithField("event", PayloadAttestationProcessedType),
		event: event,
	}
}

func (e *PayloadAttestationProcessed) Type() string {
	return PayloadAttestationProcessedType
}

func (e *PayloadAttestationProcessed) Validate(_ context.Context) error {
	_, ok := e.event.GetData().(*xatu.DecoratedEvent_BeaconSyntheticPayloadAttestationProcessed)
	if !ok {
		return errors.New("failed to cast event data")
	}

	return nil
}

func (e *PayloadAttestationProcessed) Filter(_ context.Context) bool {
	return false
}

func (e *PayloadAttestationProcessed) AppendServerMeta(_ context.Context, meta *xatu.ServerMeta) *xatu.ServerMeta {
	return meta
}
