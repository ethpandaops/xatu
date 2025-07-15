package clmimicry

import (
	"context"
	"time"

	"github.com/ethpandaops/ethwallclock"
	"github.com/pkg/errors"
	"github.com/probe-lab/hermes/host"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/ethpandaops/xatu/pkg/proto/libp2p"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

// Processor encapsulates all event processing logic for Hermes events
type Processor struct {
	// Interface dependencies (for other projects to implement)
	duties  DutiesProvider
	output  OutputHandler
	metrics MetricsCollector

	// Shared dependencies
	unifiedSharder   *UnifiedSharder
	eventCategorizer *EventCategorizer

	// Ethereum wallclock (simplified from complex interface hierarchy)
	wallclock *ethwallclock.EthereumBeaconChain

	// Clock drift for timestamp adjustments
	clockDrift time.Duration

	// Configuration
	events EventConfig

	// Client metadata
	clientMeta *xatu.ClientMeta

	// Logging
	log logrus.FieldLogger
}

// NewProcessor creates a new Processor instance
func NewProcessor(
	duties DutiesProvider,
	output OutputHandler,
	metrics MetricsCollector,
	unifiedSharder *UnifiedSharder,
	eventCategorizer *EventCategorizer,
	wallclock *ethwallclock.EthereumBeaconChain,
	clockDrift time.Duration,
	events EventConfig,
	clientMeta *xatu.ClientMeta,
	log logrus.FieldLogger,
) *Processor {
	return &Processor{
		duties:           duties,
		output:           output,
		metrics:          metrics,
		unifiedSharder:   unifiedSharder,
		eventCategorizer: eventCategorizer,
		wallclock:        wallclock,
		clockDrift:       clockDrift,
		events:           events,
		clientMeta:       clientMeta,
		log:              log,
	}
}

// HandleHermesEvent processes a Hermes trace event and routes it to the appropriate handler
func (p *Processor) HandleHermesEvent(ctx context.Context, event *host.TraceEvent) error {
	if event == nil {
		return errors.New("event is nil")
	}

	p.log.WithField("type", event.Type).Trace("Received Hermes event")

	traceMeta := &libp2p.TraceEventMetadata{
		PeerId: wrapperspb.String(event.PeerID.String()),
	}

	// Route the event to the appropriate handler based on its category.
	switch {
	// GossipSub protocol events.
	case isGossipSubEvent(event):
		return p.handleHermesGossipSubEvent(ctx, event, p.clientMeta, traceMeta)

	// libp2p pubsub protocol level events.
	case isLibp2pEvent(event):
		return p.handleHermesLibp2pEvent(ctx, event, p.clientMeta, traceMeta)

	// libp2p core networking events.
	case isLibp2pCoreEvent(event):
		return p.handleHermesLibp2pCoreEvent(ctx, event, p.clientMeta, traceMeta)

	// Request/Response (RPC) protocol events.
	case isRpcEvent(event):
		return p.handleHermesRPCEvent(ctx, event, p.clientMeta, traceMeta)

	default:
		p.log.WithField("type", event.Type).Trace("unsupported Hermes event")

		return nil
	}
}
