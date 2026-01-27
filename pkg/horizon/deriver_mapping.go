package horizon

import (
	"github.com/ethpandaops/xatu/pkg/cldata/deriver"
	horizonderiver "github.com/ethpandaops/xatu/pkg/horizon/deriver"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

// cannonToHorizonType maps CannonType to HorizonType for iterator creation.
var cannonToHorizonType = map[xatu.CannonType]xatu.HorizonType{
	xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK:                         xatu.HorizonType_HORIZON_TYPE_BEACON_API_ETH_V2_BEACON_BLOCK,
	xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_ATTESTER_SLASHING:       xatu.HorizonType_HORIZON_TYPE_BEACON_API_ETH_V2_BEACON_BLOCK_ATTESTER_SLASHING,
	xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_PROPOSER_SLASHING:       xatu.HorizonType_HORIZON_TYPE_BEACON_API_ETH_V2_BEACON_BLOCK_PROPOSER_SLASHING,
	xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_DEPOSIT:                 xatu.HorizonType_HORIZON_TYPE_BEACON_API_ETH_V2_BEACON_BLOCK_DEPOSIT,
	xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_WITHDRAWAL:              xatu.HorizonType_HORIZON_TYPE_BEACON_API_ETH_V2_BEACON_BLOCK_WITHDRAWAL,
	xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_VOLUNTARY_EXIT:          xatu.HorizonType_HORIZON_TYPE_BEACON_API_ETH_V2_BEACON_BLOCK_VOLUNTARY_EXIT,
	xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_BLS_TO_EXECUTION_CHANGE: xatu.HorizonType_HORIZON_TYPE_BEACON_API_ETH_V2_BEACON_BLOCK_BLS_TO_EXECUTION_CHANGE,
	xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_EXECUTION_TRANSACTION:   xatu.HorizonType_HORIZON_TYPE_BEACON_API_ETH_V2_BEACON_BLOCK_EXECUTION_TRANSACTION,
	xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_ELABORATED_ATTESTATION:  xatu.HorizonType_HORIZON_TYPE_BEACON_API_ETH_V2_BEACON_BLOCK_ELABORATED_ATTESTATION,
	xatu.CannonType_BEACON_API_ETH_V1_PROPOSER_DUTY:                        xatu.HorizonType_HORIZON_TYPE_BEACON_API_ETH_V1_PROPOSER_DUTY,
	xatu.CannonType_BEACON_API_ETH_V1_BEACON_BLOB_SIDECAR:                  xatu.HorizonType_HORIZON_TYPE_BEACON_API_ETH_V1_BEACON_BLOB_SIDECAR,
	xatu.CannonType_BEACON_API_ETH_V1_BEACON_VALIDATORS:                    xatu.HorizonType_HORIZON_TYPE_BEACON_API_ETH_V1_BEACON_VALIDATORS,
	xatu.CannonType_BEACON_API_ETH_V1_BEACON_COMMITTEE:                     xatu.HorizonType_HORIZON_TYPE_BEACON_API_ETH_V1_BEACON_COMMITTEE,
}

// GetHorizonType returns the HorizonType for a given CannonType.
func GetHorizonType(cannonType xatu.CannonType) (xatu.HorizonType, bool) {
	horizonType, ok := cannonToHorizonType[cannonType]

	return horizonType, ok
}

// IsDeriverEnabled returns whether a deriver is enabled based on config.
func IsDeriverEnabled(config *horizonderiver.Config, cannonType xatu.CannonType) bool {
	switch cannonType {
	case xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK:
		return config.BeaconBlockConfig.Enabled
	case xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_ATTESTER_SLASHING:
		return config.AttesterSlashingConfig.Enabled
	case xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_PROPOSER_SLASHING:
		return config.ProposerSlashingConfig.Enabled
	case xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_DEPOSIT:
		return config.DepositConfig.Enabled
	case xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_WITHDRAWAL:
		return config.WithdrawalConfig.Enabled
	case xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_VOLUNTARY_EXIT:
		return config.VoluntaryExitConfig.Enabled
	case xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_BLS_TO_EXECUTION_CHANGE:
		return config.BLSToExecutionChangeConfig.Enabled
	case xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_EXECUTION_TRANSACTION:
		return config.ExecutionTransactionConfig.Enabled
	case xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_ELABORATED_ATTESTATION:
		return config.ElaboratedAttestationConfig.Enabled
	case xatu.CannonType_BEACON_API_ETH_V1_PROPOSER_DUTY:
		return config.ProposerDutyConfig.Enabled
	case xatu.CannonType_BEACON_API_ETH_V1_BEACON_BLOB_SIDECAR:
		return config.BeaconBlobConfig.Enabled
	case xatu.CannonType_BEACON_API_ETH_V1_BEACON_VALIDATORS:
		return config.BeaconValidatorsConfig.Enabled
	case xatu.CannonType_BEACON_API_ETH_V1_BEACON_COMMITTEE:
		return config.BeaconCommitteeConfig.Enabled
	default:
		return false
	}
}

// IsEpochBased returns whether a deriver spec is epoch-based (vs slot-based).
func IsEpochBased(spec *deriver.DeriverSpec) bool {
	return spec.Mode == deriver.ProcessingModeEpoch
}
