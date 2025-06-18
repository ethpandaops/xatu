package noderecord

import (
	"context"
	"errors"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/sirupsen/logrus"
)

var (
	ExecutionType = xatu.Event_NODE_RECORD_EXECUTION.String()
)

type Execution struct {
	log   logrus.FieldLogger
	event *xatu.DecoratedEvent
}

func NewExecution(log logrus.FieldLogger, event *xatu.DecoratedEvent) *Execution {
	return &Execution{
		log:   log.WithField("event", ExecutionType),
		event: event,
	}
}

func (b *Execution) Type() string {
	return ExecutionType
}

func (b *Execution) Validate(ctx context.Context) error {
	_, ok := b.event.Data.(*xatu.DecoratedEvent_NodeRecordExecution)
	if !ok {
		return errors.New("failed to cast event data")
	}

	return nil
}

func (b *Execution) Filter(ctx context.Context) bool {
	return false
}

func (b *Execution) AppendServerMeta(ctx context.Context, meta *xatu.ServerMeta) *xatu.ServerMeta {
	return meta
}
