package execution

import (
	"context"
	"errors"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/sirupsen/logrus"
)

const (
	EngineNewPayloadType = "EXECUTION_ENGINE_NEW_PAYLOAD"
)

type EngineNewPayload struct {
	log   logrus.FieldLogger
	event *xatu.DecoratedEvent
}

func NewEngineNewPayload(log logrus.FieldLogger, event *xatu.DecoratedEvent) *EngineNewPayload {
	return &EngineNewPayload{
		log:   log.WithField("event", EngineNewPayloadType),
		event: event,
	}
}

func (e *EngineNewPayload) Type() string {
	return EngineNewPayloadType
}

func (e *EngineNewPayload) Validate(_ context.Context) error {
	_, ok := e.event.Data.(*xatu.DecoratedEvent_ExecutionEngineNewPayload)
	if !ok {
		return errors.New("failed to cast event data to ExecutionEngineNewPayload")
	}

	return nil
}

func (e *EngineNewPayload) Filter(_ context.Context) bool {
	return false
}

func (e *EngineNewPayload) AppendServerMeta(_ context.Context, meta *xatu.ServerMeta) *xatu.ServerMeta {
	return meta
}
