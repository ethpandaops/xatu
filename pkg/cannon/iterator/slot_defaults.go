package iterator

import (
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/ethpandaops/beacon/pkg/beacon/state"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

func NewSlotDefaultsFromForkEpochs(forkEpochs state.ForkEpochs) map[xatu.CannonType]phase0.Slot {
	var (
		bellatrixEpoch,
		capellaEpoch,
		denebEpoch phase0.Epoch
	)

	bellatrix, err := forkEpochs.GetByName("BELLATRIX")
	if err == nil {
		bellatrixEpoch = bellatrix.Epoch
	}

	capella, err := forkEpochs.GetByName("CAPELLA")
	if err == nil {
		capellaEpoch = capella.Epoch
	}

	deneb, err := forkEpochs.GetByName("DENEB")
	if err == nil {
		denebEpoch = deneb.Epoch
	}

	return map[xatu.CannonType]phase0.Slot{
		xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_BLS_TO_EXECUTION_CHANGE: phase0.Slot(capellaEpoch * 32),
		xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_WITHDRAWAL:              phase0.Slot(capellaEpoch * 32),
		xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_EXECUTION_TRANSACTION:   phase0.Slot(bellatrixEpoch * 32),
		xatu.CannonType_BEACON_API_ETH_V1_BEACON_BLOB_SIDECAR:                  phase0.Slot(denebEpoch * 32),
	}
}
