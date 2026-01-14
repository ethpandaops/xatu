package execution

import (
	"context"
	"errors"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/sirupsen/logrus"
)

const (
	ExecutionBlockStateMetricsType = "EXECUTION_BLOCK_STATE_METRICS"
)

type ExecutionBlockStateMetrics struct {
	log   logrus.FieldLogger
	event *xatu.DecoratedEvent
}

func NewExecutionBlockStateMetrics(log logrus.FieldLogger, event *xatu.DecoratedEvent) *ExecutionBlockStateMetrics {
	return &ExecutionBlockStateMetrics{
		log:   log.WithField("event", ExecutionBlockStateMetricsType),
		event: event,
	}
}

func (e *ExecutionBlockStateMetrics) Type() string {
	return ExecutionBlockStateMetricsType
}

func (e *ExecutionBlockStateMetrics) Validate(ctx context.Context) error {
	_, ok := e.event.Data.(*xatu.DecoratedEvent_ExecutionBlockStateMetrics)
	if !ok {
		return errors.New("failed to cast event data to ExecutionBlockStateMetrics")
	}

	return nil
}

func (e *ExecutionBlockStateMetrics) Filter(ctx context.Context) bool {
	return false
}

func (e *ExecutionBlockStateMetrics) AppendServerMeta(ctx context.Context, meta *xatu.ServerMeta) *xatu.ServerMeta {
	return meta
}
