package libp2p

import (
	"context"
	"errors"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/sirupsen/logrus"
)

var (
	TraceGossipSubBeaconBlockType = xatu.Event_LIBP2P_TRACE_GOSSIPSUB_BEACON_BLOCK.String()
)

type TraceGossipSubBeaconBlock struct {
	log   logrus.FieldLogger
	event *xatu.DecoratedEvent
}

func NewTraceGossipSubBeaconBlock(log logrus.FieldLogger, event *xatu.DecoratedEvent) *TraceGossipSubBeaconBlock {

	return &TraceGossipSubBeaconBlock{
		log:   log.WithField("event", TraceGossipSubBeaconBlockType),
		event: event,
	}
}

func (gsb *TraceGossipSubBeaconBlock) Type() string {
	return TraceGossipSubBeaconBlockType
}

func (gsb *TraceGossipSubBeaconBlock) Validate(ctx context.Context) error {
	_, ok := gsb.event.Data.(*xatu.DecoratedEvent_Libp2PTraceGossipsubBeaconBlock)
	if !ok {
		return errors.New("failed to cast event data to TraceGossipSubBeaconBlock")
	}

	return nil
}

func (gsb *TraceGossipSubBeaconBlock) Filter(ctx context.Context) bool {
	return false
}

func (gsb *TraceGossipSubBeaconBlock) AppendServerMeta(ctx context.Context, meta *xatu.ServerMeta) *xatu.ServerMeta {
	return meta
}
