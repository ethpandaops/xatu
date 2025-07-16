package clmimicry

import (
	"context"
	"time"

	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/ethpandaops/ethwallclock"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

// MetadataProvider provides ethereum network metadata
type MetadataProvider interface {
	Wallclock() *ethwallclock.EthereumBeaconChain
	ClockDrift() *time.Duration
	Network() *xatu.ClientMeta_Ethereum_Network
}

// DutiesProvider provides validator duty information
type DutiesProvider interface {
	GetValidatorIndex(epoch phase0.Epoch, slot phase0.Slot, committeeIndex phase0.CommitteeIndex, position uint64) (phase0.ValidatorIndex, error)
}

// OutputHandler handles processed events
type OutputHandler interface {
	HandleDecoratedEvent(ctx context.Context, event *xatu.DecoratedEvent) error
	HandleDecoratedEvents(ctx context.Context, events []*xatu.DecoratedEvent) error
}

// MetricsCollector interface for event processing metrics
type MetricsCollector interface {
	AddEvent(eventType, network string)
	AddProcessedMessage(eventType, network string)
	AddSkippedMessage(eventType, network string)
	AddShardingDecision(eventType, reason, network string)
	AddDecoratedEvent(count float64, eventType, network string)
}
