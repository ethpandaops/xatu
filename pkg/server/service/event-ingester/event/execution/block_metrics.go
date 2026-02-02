package execution

import (
	"context"
	"errors"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/sirupsen/logrus"
)

const (
	BlockMetricsType = "EXECUTION_BLOCK_METRICS"
)

type BlockMetrics struct {
	log   logrus.FieldLogger
	event *xatu.DecoratedEvent
}

func NewBlockMetrics(log logrus.FieldLogger, event *xatu.DecoratedEvent) *BlockMetrics {
	return &BlockMetrics{
		log:   log.WithField("event", BlockMetricsType),
		event: event,
	}
}

func (e *BlockMetrics) Type() string {
	return BlockMetricsType
}

func (e *BlockMetrics) Validate(_ context.Context) error {
	_, ok := e.event.Data.(*xatu.DecoratedEvent_ExecutionBlockMetrics)
	if !ok {
		e.log.WithField("data_type", e.event.GetData()).WithField("data_nil", e.event.Data == nil).Warn("Event data type mismatch for ExecutionBlockMetrics")

		return errors.New("failed to cast event data to ExecutionBlockMetrics")
	}

	return nil
}

func (e *BlockMetrics) Filter(_ context.Context) bool {
	return false
}

func (e *BlockMetrics) AppendServerMeta(_ context.Context, meta *xatu.ServerMeta) *xatu.ServerMeta {
	return meta
}
