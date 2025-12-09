package consensus

import (
	"context"
	"errors"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/sirupsen/logrus"
)

const (
	EngineAPIGetBlobsType = "CONSENSUS_ENGINE_API_GET_BLOBS"
)

type EngineAPIGetBlobs struct {
	log   logrus.FieldLogger
	event *xatu.DecoratedEvent
}

func NewEngineAPIGetBlobs(log logrus.FieldLogger, event *xatu.DecoratedEvent) *EngineAPIGetBlobs {
	return &EngineAPIGetBlobs{
		log:   log.WithField("event", EngineAPIGetBlobsType),
		event: event,
	}
}

func (e *EngineAPIGetBlobs) Type() string {
	return EngineAPIGetBlobsType
}

func (e *EngineAPIGetBlobs) Validate(_ context.Context) error {
	_, ok := e.event.Data.(*xatu.DecoratedEvent_ConsensusEngineApiGetBlobs)
	if !ok {
		return errors.New("failed to cast event data to ConsensusEngineApiGetBlobs")
	}

	return nil
}

func (e *EngineAPIGetBlobs) Filter(_ context.Context) bool {
	return false
}

func (e *EngineAPIGetBlobs) AppendServerMeta(_ context.Context, meta *xatu.ServerMeta) *xatu.ServerMeta {
	return meta
}
