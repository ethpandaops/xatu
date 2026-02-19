package execution

import (
	"context"
	"errors"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/sirupsen/logrus"
)

const (
	MPTDepthType = "EXECUTION_MPT_DEPTH"
)

type MPTDepth struct {
	log   logrus.FieldLogger
	event *xatu.DecoratedEvent
}

func NewMPTDepth(log logrus.FieldLogger, event *xatu.DecoratedEvent) *MPTDepth {
	return &MPTDepth{
		log:   log.WithField("event", MPTDepthType),
		event: event,
	}
}

func (e *MPTDepth) Type() string {
	return MPTDepthType
}

func (e *MPTDepth) Validate(_ context.Context) error {
	_, ok := e.event.Data.(*xatu.DecoratedEvent_ExecutionMptDepth)
	if !ok {
		e.log.WithField("data_type", e.event.GetData()).WithField("data_nil", e.event.Data == nil).Warn("Event data type mismatch for ExecutionMPTDepth")

		return errors.New("failed to cast event data to ExecutionMPTDepth")
	}

	return nil
}

func (e *MPTDepth) Filter(_ context.Context) bool {
	return false
}

func (e *MPTDepth) AppendServerMeta(_ context.Context, meta *xatu.ServerMeta) *xatu.ServerMeta {
	return meta
}
