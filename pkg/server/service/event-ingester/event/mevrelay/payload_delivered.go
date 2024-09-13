package mevrelay

import (
	"context"
	"errors"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/sirupsen/logrus"
)

var (
	ProposerPayloadDeliveredType = xatu.Event_MEV_RELAY_PROPOSER_PAYLOAD_DELIVERED.String()
)

type ProposerPayloadDelivered struct {
	log   logrus.FieldLogger
	event *xatu.DecoratedEvent
}

func NewProposerPayloadDelivered(log logrus.FieldLogger, event *xatu.DecoratedEvent) *ProposerPayloadDelivered {
	return &ProposerPayloadDelivered{
		log:   log.WithField("event", ProposerPayloadDeliveredType),
		event: event,
	}
}

func (p *ProposerPayloadDelivered) Type() string {
	return ProposerPayloadDeliveredType
}

func (p *ProposerPayloadDelivered) Validate(ctx context.Context) error {
	_, ok := p.event.Data.(*xatu.DecoratedEvent_MevRelayPayloadDelivered)
	if !ok {
		return errors.New("failed to cast event data to proposer_payload_delivered")
	}

	return nil
}

func (p *ProposerPayloadDelivered) Filter(ctx context.Context) bool {
	return false
}

func (p *ProposerPayloadDelivered) AppendServerMeta(ctx context.Context, meta *xatu.ServerMeta) *xatu.ServerMeta {
	return meta
}
