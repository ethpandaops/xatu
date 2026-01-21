package deriver

import (
	"context"

	"github.com/attestantio/go-eth2-client/spec"
	v1 "github.com/ethpandaops/xatu/pkg/cannon/deriver/beacon/eth/v1"
	v2 "github.com/ethpandaops/xatu/pkg/cannon/deriver/beacon/eth/v2"
	cldataderiver "github.com/ethpandaops/xatu/pkg/cldata/deriver"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

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

// Ensure that derivers implements the EventDeriver interface
var _ EventDeriver = &v2.AttesterSlashingDeriver{}
var _ EventDeriver = &v2.ProposerSlashingDeriver{}
var _ EventDeriver = &v2.DepositDeriver{}
var _ EventDeriver = &v2.VoluntaryExitDeriver{}
var _ EventDeriver = &v2.ExecutionTransactionDeriver{}
var _ EventDeriver = &v2.BLSToExecutionChangeDeriver{}
var _ EventDeriver = &v2.WithdrawalDeriver{}
var _ EventDeriver = &v2.BeaconBlockDeriver{}
var _ EventDeriver = &v2.ElaboratedAttestationDeriver{}
var _ EventDeriver = &v1.ProposerDutyDeriver{}
var _ EventDeriver = &v1.BeaconBlobDeriver{}
var _ EventDeriver = &v1.BeaconValidatorsDeriver{}
var _ EventDeriver = &v1.BeaconCommitteeDeriver{}

// Shared derivers from cldata package
var _ EventDeriver = &cldataderiver.BeaconBlockDeriver{}
var _ EventDeriver = &cldataderiver.AttesterSlashingDeriver{}
var _ EventDeriver = &cldataderiver.ProposerSlashingDeriver{}
var _ EventDeriver = &cldataderiver.DepositDeriver{}
var _ EventDeriver = &cldataderiver.WithdrawalDeriver{}
var _ EventDeriver = &cldataderiver.VoluntaryExitDeriver{}
var _ EventDeriver = &cldataderiver.BLSToExecutionChangeDeriver{}
var _ EventDeriver = &cldataderiver.ExecutionTransactionDeriver{}
var _ EventDeriver = &cldataderiver.ElaboratedAttestationDeriver{}
var _ EventDeriver = &cldataderiver.ProposerDutyDeriver{}
var _ EventDeriver = &cldataderiver.BeaconBlobDeriver{}
var _ EventDeriver = &cldataderiver.BeaconValidatorsDeriver{}
