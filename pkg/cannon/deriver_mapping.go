package cannon

import (
	"github.com/ethpandaops/xatu/pkg/cannon/deriver"
	"github.com/ethpandaops/xatu/pkg/cannon/iterator"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

// GetDeriverConfig returns the deriver config for a given cannon type.
func GetDeriverConfig(config *deriver.Config, cannonType xatu.CannonType) *deriver.DeriverConfig {
	switch cannonType {
	case xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK:
		return &config.BeaconBlockConfig
	case xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_ATTESTER_SLASHING:
		return &config.AttesterSlashingConfig
	case xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_PROPOSER_SLASHING:
		return &config.ProposerSlashingConfig
	case xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_DEPOSIT:
		return &config.DepositConfig
	case xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_WITHDRAWAL:
		return &config.WithdrawalConfig
	case xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_VOLUNTARY_EXIT:
		return &config.VoluntaryExitConfig
	case xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_BLS_TO_EXECUTION_CHANGE:
		return &config.BLSToExecutionConfig
	case xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_EXECUTION_TRANSACTION:
		return &config.ExecutionTransactionConfig
	case xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_ELABORATED_ATTESTATION:
		return &config.ElaboratedAttestationConfig
	case xatu.CannonType_BEACON_API_ETH_V1_PROPOSER_DUTY:
		return &config.ProposerDutyConfig
	case xatu.CannonType_BEACON_API_ETH_V1_BEACON_BLOB_SIDECAR:
		return &config.BeaconBlobSidecarConfig
	case xatu.CannonType_BEACON_API_ETH_V1_BEACON_COMMITTEE:
		return &config.BeaconCommitteeConfig
	default:
		return nil
	}
}

// GetIteratorConfig returns the iterator config for a given cannon type.
func GetIteratorConfig(config *deriver.Config, cannonType xatu.CannonType) *iterator.BackfillingCheckpointConfig {
	switch cannonType {
	case xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK:
		return &config.BeaconBlockConfig.Iterator
	case xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_ATTESTER_SLASHING:
		return &config.AttesterSlashingConfig.Iterator
	case xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_PROPOSER_SLASHING:
		return &config.ProposerSlashingConfig.Iterator
	case xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_DEPOSIT:
		return &config.DepositConfig.Iterator
	case xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_WITHDRAWAL:
		return &config.WithdrawalConfig.Iterator
	case xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_VOLUNTARY_EXIT:
		return &config.VoluntaryExitConfig.Iterator
	case xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_BLS_TO_EXECUTION_CHANGE:
		return &config.BLSToExecutionConfig.Iterator
	case xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_EXECUTION_TRANSACTION:
		return &config.ExecutionTransactionConfig.Iterator
	case xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_ELABORATED_ATTESTATION:
		return &config.ElaboratedAttestationConfig.Iterator
	case xatu.CannonType_BEACON_API_ETH_V1_PROPOSER_DUTY:
		return &config.ProposerDutyConfig.Iterator
	case xatu.CannonType_BEACON_API_ETH_V1_BEACON_BLOB_SIDECAR:
		return &config.BeaconBlobSidecarConfig.Iterator
	case xatu.CannonType_BEACON_API_ETH_V1_BEACON_VALIDATORS:
		return &config.BeaconValidatorsConfig.Iterator
	case xatu.CannonType_BEACON_API_ETH_V1_BEACON_COMMITTEE:
		return &config.BeaconCommitteeConfig.Iterator
	default:
		return nil
	}
}

// IsDeriverEnabled returns whether a deriver is enabled based on config.
func IsDeriverEnabled(config *deriver.Config, cannonType xatu.CannonType) bool {
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
		return config.BLSToExecutionConfig.Enabled
	case xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_EXECUTION_TRANSACTION:
		return config.ExecutionTransactionConfig.Enabled
	case xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_ELABORATED_ATTESTATION:
		return config.ElaboratedAttestationConfig.Enabled
	case xatu.CannonType_BEACON_API_ETH_V1_PROPOSER_DUTY:
		return config.ProposerDutyConfig.Enabled
	case xatu.CannonType_BEACON_API_ETH_V1_BEACON_BLOB_SIDECAR:
		return config.BeaconBlobSidecarConfig.Enabled
	case xatu.CannonType_BEACON_API_ETH_V1_BEACON_VALIDATORS:
		return config.BeaconValidatorsConfig.Enabled
	case xatu.CannonType_BEACON_API_ETH_V1_BEACON_COMMITTEE:
		return config.BeaconCommitteeConfig.Enabled
	default:
		return false
	}
}
