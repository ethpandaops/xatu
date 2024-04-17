package libp2p

import (
	"context"
	"errors"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/sirupsen/logrus"
)

var (
	TracePruneType = xatu.Event_LIBP2P_TRACE_PRUNE.String()
)

type TracePrune struct {
	log   logrus.FieldLogger
	event *xatu.DecoratedEvent
}

func NewTracePrune(log logrus.FieldLogger, event *xatu.DecoratedEvent) *TracePrune {
	return &TracePrune{
		log:   log.WithField("event", TracePruneType),
		event: event,
	}
}

func (tpm *TracePrune) Type() string {
	return TracePruneType
}

func (tpm *TracePrune) Validate(ctx context.Context) error {
	_, ok := tpm.event.Data.(*xatu.DecoratedEvent_Libp2PTracePrune)
	if !ok {
		return errors.New("failed to cast event data to TracePrune")
	}

	return nil
}

func (tpm *TracePrune) Filter(ctx context.Context) bool {
	return false
}

func (tpm *TracePrune) AppendServerMeta(ctx context.Context, meta *xatu.ServerMeta) *xatu.ServerMeta {
	return meta
}
