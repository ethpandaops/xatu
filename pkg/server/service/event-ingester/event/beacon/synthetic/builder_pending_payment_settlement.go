package synthetic

import (
	"context"
	"errors"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/sirupsen/logrus"
)

const (
	BuilderPendingPaymentSettlementType = "BEACON_SYNTHETIC_BUILDER_PENDING_PAYMENT_SETTLEMENT"
)

// BuilderPendingPaymentSettlement handles BEACON_SYNTHETIC_BUILDER_PENDING_PAYMENT_SETTLEMENT
// events — epoch-boundary builder pending payment settle/drop decisions synthesized
// from beacon-node internals (TYSM-instrumented, EIP-7732 ePBS).
type BuilderPendingPaymentSettlement struct {
	log   logrus.FieldLogger
	event *xatu.DecoratedEvent
}

func NewBuilderPendingPaymentSettlement(log logrus.FieldLogger, event *xatu.DecoratedEvent) *BuilderPendingPaymentSettlement {
	return &BuilderPendingPaymentSettlement{
		log:   log.WithField("event", BuilderPendingPaymentSettlementType),
		event: event,
	}
}

func (e *BuilderPendingPaymentSettlement) Type() string {
	return BuilderPendingPaymentSettlementType
}

func (e *BuilderPendingPaymentSettlement) Validate(_ context.Context) error {
	_, ok := e.event.GetData().(*xatu.DecoratedEvent_BeaconSyntheticBuilderPendingPaymentSettlement)
	if !ok {
		return errors.New("failed to cast event data")
	}

	return nil
}

func (e *BuilderPendingPaymentSettlement) Filter(_ context.Context) bool {
	return false
}

func (e *BuilderPendingPaymentSettlement) AppendServerMeta(_ context.Context, meta *xatu.ServerMeta) *xatu.ServerMeta {
	return meta
}
