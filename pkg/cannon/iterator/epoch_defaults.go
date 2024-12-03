package iterator

import (
	"github.com/attestantio/go-eth2-client/spec"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/ethpandaops/beacon/pkg/beacon/state"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

func NewEpochDefaultsFromForkEpochs(forkEpochs state.ForkEpochs) map[xatu.CannonType]phase0.Epoch {
	var (
		bellatrixEpoch,
		capellaEpoch,
		denebEpoch phase0.Epoch
	)

	bellatrix, err := forkEpochs.GetByName(spec.DataVersionBellatrix.String())
	if err == nil {
		bellatrixEpoch = bellatrix.Epoch
	}

	capella, err := forkEpochs.GetByName(spec.DataVersionCapella.String())
	if err == nil {
		capellaEpoch = capella.Epoch
	}

	deneb, err := forkEpochs.GetByName(spec.DataVersionDeneb.String())
	if err == nil {
		denebEpoch = deneb.Epoch
	}

	return map[xatu.CannonType]phase0.Epoch{
		xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_BLS_TO_EXECUTION_CHANGE: capellaEpoch,
		xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_WITHDRAWAL:              capellaEpoch,
		xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_EXECUTION_TRANSACTION:   bellatrixEpoch,
		xatu.CannonType_BEACON_API_ETH_V1_BEACON_BLOB_SIDECAR:                  denebEpoch,
	}
}
