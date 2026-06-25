package v1

import (
	"context"
	"errors"

	"github.com/ethpandaops/xatu/pkg/observability"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

var (
	RandaoType = xatu.Event_BEACON_API_ETH_V1_BEACON_STATE_RANDAO.String()
)

type Randao struct {
	log   observability.ContextualLogger
	event *xatu.DecoratedEvent
}

func NewRandao(log observability.ContextualLogger, event *xatu.DecoratedEvent) *Randao {
	return &Randao{
		log:   log.WithField("event", RandaoType),
		event: event,
	}
}

func (r *Randao) Type() string {
	return RandaoType
}

func (r *Randao) Validate(_ context.Context) error {
	data, ok := r.event.GetData().(*xatu.DecoratedEvent_EthV1BeaconStateRandao)
	if !ok {
		return errors.New("failed to cast event data")
	}

	if data.EthV1BeaconStateRandao.GetRandao() == "" {
		return errors.New("randao is empty")
	}

	return nil
}

func (r *Randao) Filter(_ context.Context) bool {
	return false
}

func (r *Randao) AppendServerMeta(_ context.Context, meta *xatu.ServerMeta) *xatu.ServerMeta {
	return meta
}
