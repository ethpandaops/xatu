package execution

import (
	"context"
	"errors"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/sirupsen/logrus"
)

const (
	ExecutionDebugStateSizeType = "EXECUTION_DEBUG_STATE_SIZE"
)

type ExecutionDebugStateSize struct {
	log   logrus.FieldLogger
	event *xatu.DecoratedEvent
}

func NewExecutionDebugStateSize(log logrus.FieldLogger, event *xatu.DecoratedEvent) *ExecutionDebugStateSize {
	return &ExecutionDebugStateSize{
		log:   log.WithField("event", ExecutionDebugStateSizeType),
		event: event,
	}
}

func (e *ExecutionDebugStateSize) Type() string {
	return ExecutionDebugStateSizeType
}

func (e *ExecutionDebugStateSize) Validate(ctx context.Context) error {
	_, ok := e.event.Data.(*xatu.DecoratedEvent_ExecutionDebugStateSize)
	if !ok {
		return errors.New("failed to cast event data to ExecutionDebugStateSize")
	}

	return nil
}

func (e *ExecutionDebugStateSize) Filter(ctx context.Context) bool {
	return false
}

func (e *ExecutionDebugStateSize) AppendServerMeta(ctx context.Context, meta *xatu.ServerMeta) *xatu.ServerMeta {
	return meta
}
