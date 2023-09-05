package deriver

import (
	"context"

	v2 "github.com/ethpandaops/xatu/pkg/cannon/deriver/beacon/eth/v2"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

type EventDeriver interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	Name() string
	CannonType() xatu.CannonType
	Location() uint64
	// Callbacks
	OnEventDerived(ctx context.Context, fn func(ctx context.Context, event *xatu.DecoratedEvent) error)
}

// Ensure that derivers implements the EventDeriver interface
var _ EventDeriver = &v2.AttesterSlashingDeriver{}
