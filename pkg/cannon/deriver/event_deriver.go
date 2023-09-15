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
	// Callbacks
	OnEventsDerived(ctx context.Context, fn func(ctx context.Context, events []*xatu.DecoratedEvent) error)
	OnLocationUpdated(ctx context.Context, fn func(ctx context.Context, loc uint64) error)
}

// Ensure that derivers implements the EventDeriver interface
var _ EventDeriver = &v2.AttesterSlashingDeriver{}
var _ EventDeriver = &v2.ProposerSlashingDeriver{}
var _ EventDeriver = &v2.DepositDeriver{}
var _ EventDeriver = &v2.VoluntaryExitDeriver{}
var _ EventDeriver = &v2.ExecutionTransactionDeriver{}
var _ EventDeriver = &v2.BLSToExecutionChangeDeriver{}
