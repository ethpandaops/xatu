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

// Ensure that shared derivers from cldata package implement the EventDeriver interface.
var (
	_ EventDeriver = (*cldataderiver.BeaconBlockDeriver)(nil)
	_ EventDeriver = (*cldataderiver.AttesterSlashingDeriver)(nil)
	_ EventDeriver = (*cldataderiver.ProposerSlashingDeriver)(nil)
	_ EventDeriver = (*cldataderiver.DepositDeriver)(nil)
	_ EventDeriver = (*cldataderiver.WithdrawalDeriver)(nil)
	_ EventDeriver = (*cldataderiver.VoluntaryExitDeriver)(nil)
	_ EventDeriver = (*cldataderiver.BLSToExecutionChangeDeriver)(nil)
	_ EventDeriver = (*cldataderiver.ExecutionTransactionDeriver)(nil)
	_ EventDeriver = (*cldataderiver.ElaboratedAttestationDeriver)(nil)
	_ EventDeriver = (*cldataderiver.ProposerDutyDeriver)(nil)
	_ EventDeriver = (*cldataderiver.BeaconBlobDeriver)(nil)
	_ EventDeriver = (*cldataderiver.BeaconValidatorsDeriver)(nil)
	_ EventDeriver = (*cldataderiver.BeaconCommitteeDeriver)(nil)
)
