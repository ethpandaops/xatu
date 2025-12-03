package consensus

import (
	"context"
	"errors"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/sirupsen/logrus"
)

const (
	EngineAPINewPayloadType = "CONSENSUS_ENGINE_API_NEW_PAYLOAD"
)

type EngineAPINewPayload struct {
	log   logrus.FieldLogger
	event *xatu.DecoratedEvent
}

func NewEngineAPINewPayload(log logrus.FieldLogger, event *xatu.DecoratedEvent) *EngineAPINewPayload {
	return &EngineAPINewPayload{
		log:   log.WithField("event", EngineAPINewPayloadType),
		event: event,
	}
}

func (e *EngineAPINewPayload) Type() string {
	return EngineAPINewPayloadType
}

func (e *EngineAPINewPayload) Validate(_ context.Context) error {
	_, ok := e.event.Data.(*xatu.DecoratedEvent_ConsensusEngineApiNewPayload)
	if !ok {
		return errors.New("failed to cast event data to ConsensusEngineApiNewPayload")
	}

	return nil
}

func (e *EngineAPINewPayload) Filter(_ context.Context) bool {
	return false
}

func (e *EngineAPINewPayload) AppendServerMeta(_ context.Context, meta *xatu.ServerMeta) *xatu.ServerMeta {
	return meta
}
