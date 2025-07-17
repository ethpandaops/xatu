package clmimicry

import (
	"time"

	"github.com/ethpandaops/ethwallclock"
	"github.com/sirupsen/logrus"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

// Processor encapsulates all event processing logic for Hermes events
type Processor struct {
	// Interface dependencies (for other projects to implement)
	duties       DutiesProvider
	output       OutputHandler
	metrics      MetricsCollector
	metaProvider MetaProvider

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
		log:              log,
	}
}
