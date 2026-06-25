package v1

import (
	"context"
	"errors"

	"github.com/ethpandaops/xatu/pkg/observability"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

var (
	FinalityCheckpointType = xatu.Event_BEACON_API_ETH_V1_BEACON_STATE_FINALITY_CHECKPOINT.String()
)

type FinalityCheckpoint struct {
	log   observability.ContextualLogger
	event *xatu.DecoratedEvent
}

func NewFinalityCheckpoint(log observability.ContextualLogger, event *xatu.DecoratedEvent) *FinalityCheckpoint {
	return &FinalityCheckpoint{
		log:   log.WithField("event", FinalityCheckpointType),
		event: event,
	}
}

func (f *FinalityCheckpoint) Type() string {
	return FinalityCheckpointType
}

func (f *FinalityCheckpoint) Validate(_ context.Context) error {
	_, ok := f.event.GetData().(*xatu.DecoratedEvent_EthV1BeaconStateFinalityCheckpoint)
	if !ok {
		return errors.New("failed to cast event data")
	}

	return nil
}

func (f *FinalityCheckpoint) Filter(_ context.Context) bool {
	return false
}

func (f *FinalityCheckpoint) AppendServerMeta(_ context.Context, meta *xatu.ServerMeta) *xatu.ServerMeta {
	return meta
}
