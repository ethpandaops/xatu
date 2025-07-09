package noderecord

import (
	"context"
	"errors"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/sirupsen/logrus"
)

var (
	ConsensusType = xatu.Event_NODE_RECORD_CONSENSUS.String()
)

type Consensus struct {
	log   logrus.FieldLogger
	event *xatu.DecoratedEvent
}

func NewConsensus(log logrus.FieldLogger, event *xatu.DecoratedEvent) *Consensus {
	return &Consensus{
		log:   log.WithField("event", ConsensusType),
		event: event,
	}
}

func (b *Consensus) Type() string {
	return ConsensusType
}

func (b *Consensus) Validate(ctx context.Context) error {
	_, ok := b.event.Data.(*xatu.DecoratedEvent_NodeRecordConsensus)
	if !ok {
		return errors.New("failed to cast event data")
	}

	return nil
}

func (b *Consensus) Filter(ctx context.Context) bool {
	return false
}

func (b *Consensus) AppendServerMeta(ctx context.Context, meta *xatu.ServerMeta) *xatu.ServerMeta {
	return meta
}
