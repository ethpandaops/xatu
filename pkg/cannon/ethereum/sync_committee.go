package ethereum

import (
	"github.com/ethpandaops/go-eth2-client/spec"
	"github.com/ethpandaops/go-eth2-client/spec/phase0"
	"github.com/pkg/errors"
)

// SyncCommitteeStateEpoch returns the epoch whose state serves the sync committee active at epoch:
// the start of its sync committee period, or the Altair fork epoch when the period began before Altair.
func (b *BeaconNode) SyncCommitteeStateEpoch(epoch phase0.Epoch) (phase0.Epoch, error) {
	sp, err := b.beacon.Spec()
	if err != nil {
		return 0, errors.Wrap(err, "failed to obtain spec")
	}

	altair, err := sp.ForkEpochs.GetByName(spec.DataVersionAltair.String())
	if err != nil {
		return 0, errors.Wrap(err, "failed to get altair fork epoch")
	}

	return syncCommitteeStateEpoch(epoch, sp.EpochsPerSyncCommitteePeriod, altair.Epoch), nil
}

func syncCommitteeStateEpoch(epoch, epochsPerPeriod, altairForkEpoch phase0.Epoch) phase0.Epoch {
	periodStart := epoch / epochsPerPeriod * epochsPerPeriod
	if periodStart < altairForkEpoch {
		return altairForkEpoch
	}

	return periodStart
}
