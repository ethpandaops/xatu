package deriver

import (
	"context"

	"github.com/attestantio/go-eth2-client/spec"
	cldataderiver "github.com/ethpandaops/xatu/pkg/cldata/deriver"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

// EventDeriver is the interface that all event derivers must implement.
type EventDeriver interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	Name() string
	CannonType() xatu.CannonType
	// Callbacks
	OnEventsDerived(ctx context.Context, fn func(ctx context.Context, events []*xatu.DecoratedEvent) error)
	// ActivationFork is the fork at which the deriver should start deriving events
	ActivationFork() spec.DataVersion
}

// Ensure that GenericDeriver from cldata package implements the EventDeriver interface.
var _ EventDeriver = (*cldataderiver.GenericDeriver)(nil)
