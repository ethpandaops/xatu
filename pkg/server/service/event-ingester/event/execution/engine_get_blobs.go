package execution

import (
	"context"
	"errors"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/sirupsen/logrus"
)

const (
	EngineGetBlobsType = "EXECUTION_ENGINE_GET_BLOBS"
)

type EngineGetBlobs struct {
	log   logrus.FieldLogger
	event *xatu.DecoratedEvent
}

func NewEngineGetBlobs(log logrus.FieldLogger, event *xatu.DecoratedEvent) *EngineGetBlobs {
	return &EngineGetBlobs{
		log:   log.WithField("event", EngineGetBlobsType),
		event: event,
	}
}

func (e *EngineGetBlobs) Type() string {
	return EngineGetBlobsType
}

func (e *EngineGetBlobs) Validate(_ context.Context) error {
	_, ok := e.event.Data.(*xatu.DecoratedEvent_ExecutionEngineGetBlobs)
	if !ok {
		return errors.New("failed to cast event data to ExecutionEngineGetBlobs")
	}

	return nil
}

func (e *EngineGetBlobs) Filter(_ context.Context) bool {
	return false
}

func (e *EngineGetBlobs) AppendServerMeta(_ context.Context, meta *xatu.ServerMeta) *xatu.ServerMeta {
	return meta
}
