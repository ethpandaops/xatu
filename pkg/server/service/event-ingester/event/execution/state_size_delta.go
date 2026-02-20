package execution

import (
	"context"
	"errors"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/sirupsen/logrus"
)

const (
	StateSizeDeltaType = "EXECUTION_STATE_SIZE_DELTA"
)

type StateSizeDelta struct {
	log   logrus.FieldLogger
	event *xatu.DecoratedEvent
}

func NewStateSizeDelta(log logrus.FieldLogger, event *xatu.DecoratedEvent) *StateSizeDelta {
	return &StateSizeDelta{
		log:   log.WithField("event", StateSizeDeltaType),
		event: event,
	}
}

func (e *StateSizeDelta) Type() string {
	return StateSizeDeltaType
}

func (e *StateSizeDelta) Validate(_ context.Context) error {
	_, ok := e.event.Data.(*xatu.DecoratedEvent_ExecutionStateSizeDelta)
	if !ok {
		e.log.WithField("data_type", e.event.GetData()).WithField("data_nil", e.event.Data == nil).Warn("Event data type mismatch for ExecutionStateSizeDelta")

		return errors.New("failed to cast event data to ExecutionStateSizeDelta")
	}

	return nil
}

func (e *StateSizeDelta) Filter(_ context.Context) bool {
	return false
}

func (e *StateSizeDelta) AppendServerMeta(_ context.Context, meta *xatu.ServerMeta) *xatu.ServerMeta {
	return meta
}
