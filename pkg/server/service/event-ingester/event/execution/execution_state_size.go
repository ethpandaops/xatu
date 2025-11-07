package execution

import (
	"context"
	"errors"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/sirupsen/logrus"
)

const (
	ExecutionStateSizeType = "EXECUTION_STATE_SIZE"
)

type ExecutionStateSize struct {
	log   logrus.FieldLogger
	event *xatu.DecoratedEvent
}

func NewExecutionStateSize(log logrus.FieldLogger, event *xatu.DecoratedEvent) *ExecutionStateSize {
	return &ExecutionStateSize{
		log:   log.WithField("event", ExecutionStateSizeType),
		event: event,
	}
}

func (e *ExecutionStateSize) Type() string {
	return ExecutionStateSizeType
}

func (e *ExecutionStateSize) Validate(ctx context.Context) error {
	_, ok := e.event.Data.(*xatu.DecoratedEvent_ExecutionStateSize)
	if !ok {
		return errors.New("failed to cast event data to ExecutionStateSize")
	}

	return nil
}

func (e *ExecutionStateSize) Filter(ctx context.Context) bool {
	return false
}

func (e *ExecutionStateSize) AppendServerMeta(ctx context.Context, meta *xatu.ServerMeta) *xatu.ServerMeta {
	return meta
}
