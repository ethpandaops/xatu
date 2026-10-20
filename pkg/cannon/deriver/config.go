package deriver

import (
	"fmt"

	v1 "github.com/ethpandaops/xatu/pkg/cannon/deriver/beacon/eth/v1"
	v2 "github.com/ethpandaops/xatu/pkg/cannon/deriver/beacon/eth/v2"
	"github.com/ethpandaops/xatu/pkg/cannon/deriver/execution"
	"github.com/ethpandaops/xatu/pkg/cannon/iterator"
)

// Config groups cannon's derivers by layer: `consensus` derivers read the
// beacon chain, `execution` derivers extract execution-layer datasets via cryo.
type Config struct {
	// Consensus holds the consensus-layer (beacon) derivers.
	Consensus ConsensusConfig `yaml:"consensus"`
	// Execution holds the execution-layer (cryo) derivers.
	Execution execution.Config `yaml:"execution"`
}

// ConsensusConfig holds the consensus-layer deriver configs.
type ConsensusConfig struct {
	AttesterSlashingConfig         v2.AttesterSlashingDeriverConfig         `yaml:"attesterSlashing"`
	BLSToExecutionConfig           v2.BLSToExecutionChangeDeriverConfig     `yaml:"blsToExecutionChange"`
	DepositConfig                  v2.DepositDeriverConfig                  `yaml:"deposit"`
	ExecutionTransactionConfig     v2.ExecutionTransactionDeriverConfig     `yaml:"executionTransaction"`
	ProposerSlashingConfig         v2.ProposerSlashingDeriverConfig         `yaml:"proposerSlashing"`
	VoluntaryExitConfig            v2.VoluntaryExitDeriverConfig            `yaml:"voluntaryExit"`
	WithdrawalConfig               v2.WithdrawalDeriverConfig               `yaml:"withdrawal"`
	BeaconBlockConfig              v2.BeaconBlockDeriverConfig              `yaml:"beaconBlock"`
	BeaconBlobSidecarConfig        v1.BeaconBlobDeriverConfig               `yaml:"beaconBlobSidecar"`
	ProposerDutyConfig             v1.ProposerDutyDeriverConfig             `yaml:"proposerDuty"`
	ElaboratedAttestationConfig    v2.ElaboratedAttestationDeriverConfig    `yaml:"elaboratedAttestation"`
	BeaconValidatorsConfig         v1.BeaconValidatorsDeriverConfig         `yaml:"beaconValidators"`
	BeaconCommitteeConfig          v1.BeaconCommitteeDeriverConfig          `yaml:"beaconCommittee"`
	BeaconSyncCommitteeConfig      v1.BeaconSyncCommitteeDeriverConfig      `yaml:"beaconSyncCommittee"`
	BeaconBlockSyncAggregateConfig v2.BeaconBlockSyncAggregateDeriverConfig `yaml:"beaconBlockSyncAggregate"`
	BeaconBlockGraffitiConfig      v2.BeaconBlockGraffitiDeriverConfig      `yaml:"beaconBlockGraffiti"`

	ExecutionRequestDepositConfig             v2.ExecutionRequestDepositDeriverConfig             `yaml:"executionRequestDeposit"`
	ExecutionRequestWithdrawalConfig          v2.ExecutionRequestWithdrawalDeriverConfig          `yaml:"executionRequestWithdrawal"`
	ExecutionRequestConsolidationConfig       v2.ExecutionRequestConsolidationDeriverConfig       `yaml:"executionRequestConsolidation"`
	BlockRewardConfig                         v1.BlockRewardDeriverConfig                         `yaml:"blockReward"`
	AttestationRewardConfig                   v1.AttestationRewardDeriverConfig                   `yaml:"attestationReward"`
	SyncCommitteeRewardConfig                 v1.SyncCommitteeRewardDeriverConfig                 `yaml:"syncCommitteeReward"`
	RandaoConfig                              v1.RandaoDeriverConfig                              `yaml:"randao"`
	FinalityCheckpointConfig                  v1.FinalityCheckpointDeriverConfig                  `yaml:"finalityCheckpoint"`
	BeaconStatePendingDepositConfig           v1.BeaconStatePendingDepositDeriverConfig           `yaml:"beaconStatePendingDeposit"`
	BeaconStatePendingPartialWithdrawalConfig v1.BeaconStatePendingPartialWithdrawalDeriverConfig `yaml:"beaconStatePendingPartialWithdrawal"`
	BeaconStatePendingConsolidationConfig     v1.BeaconStatePendingConsolidationDeriverConfig     `yaml:"beaconStatePendingConsolidation"`

	// Gloas beacon state derivers (EIP-7732). They share one state download per epoch.
	BeaconStatePtcMemberConfig                    v1.BeaconStatePtcMemberDeriverConfig                    `yaml:"beaconStatePtcMember"`
	BeaconStateBuilderConfig                      v1.BeaconStateBuilderDeriverConfig                      `yaml:"beaconStateBuilder"`
	BeaconStateBuilderPendingPaymentConfig        v1.BeaconStateBuilderPendingPaymentDeriverConfig        `yaml:"beaconStateBuilderPendingPayment"`
	BeaconStateBuilderPendingWithdrawalConfig     v1.BeaconStateBuilderPendingWithdrawalDeriverConfig     `yaml:"beaconStateBuilderPendingWithdrawal"`
	BeaconStateExecutionPayloadAvailabilityConfig v1.BeaconStateExecutionPayloadAvailabilityDeriverConfig `yaml:"beaconStateExecutionPayloadAvailability"`

	// EIP-7732 ePBS + EIP-7928 BAL derivers (Gloas).
	BlockAccessListConfig     v2.BlockAccessListDeriverConfig     `yaml:"blockAccessList"`
	PayloadAttestationConfig  v2.PayloadAttestationDeriverConfig  `yaml:"payloadAttestation"`
	ExecutionPayloadBidConfig v2.ExecutionPayloadBidDeriverConfig `yaml:"executionPayloadBid"`

	// EIP-8282 builder execution requests and the per-block EIP-7928 BAL
	// summary (Gloas).
	ExecutionRequestBuilderDepositConfig v2.ExecutionRequestBuilderDepositDeriverConfig `yaml:"executionRequestBuilderDeposit"`
	ExecutionRequestBuilderExitConfig    v2.ExecutionRequestBuilderExitDeriverConfig    `yaml:"executionRequestBuilderExit"`
	BlockAccessListSummaryConfig         v2.BlockAccessListSummaryDeriverConfig         `yaml:"blockAccessListSummary"`
}

func (c *Config) Validate() error {
	if err := c.Consensus.BlockAccessListConfig.Validate(); err != nil {
		return fmt.Errorf("consensus.blockAccessList: %w", err)
	}

	// These derivers do not derive a step epoch by epoch, so they only take single-epoch steps.
	singleEpoch := map[*iterator.BackfillingCheckpointConfig]bool{
		&c.Consensus.BeaconSyncCommitteeConfig.Iterator: true,
		&c.Consensus.BeaconValidatorsConfig.Iterator:    true,
		&c.Consensus.BlockAccessListConfig.Iterator:     true,
	}

	for name, it := range c.Consensus.iterators() {
		if err := it.Validate(); err != nil {
			return fmt.Errorf("consensus.%s: %w", name, err)
		}

		if singleEpoch[it] && it.Backfill.EpochsPerBatch > 1 {
			return fmt.Errorf("consensus.%s: iterator.backfill.epochsPerBatch is not supported, got %d", name, it.Backfill.EpochsPerBatch)
		}
	}

	return nil
}

// iterators returns every consensus deriver's iterator config keyed by its YAML name.
func (c *ConsensusConfig) iterators() map[string]*iterator.BackfillingCheckpointConfig {
	return map[string]*iterator.BackfillingCheckpointConfig{
		"attesterSlashing":                        &c.AttesterSlashingConfig.Iterator,
		"blsToExecutionChange":                    &c.BLSToExecutionConfig.Iterator,
		"deposit":                                 &c.DepositConfig.Iterator,
		"executionTransaction":                    &c.ExecutionTransactionConfig.Iterator,
		"proposerSlashing":                        &c.ProposerSlashingConfig.Iterator,
		"voluntaryExit":                           &c.VoluntaryExitConfig.Iterator,
		"withdrawal":                              &c.WithdrawalConfig.Iterator,
		"beaconBlock":                             &c.BeaconBlockConfig.Iterator,
		"beaconBlobSidecar":                       &c.BeaconBlobSidecarConfig.Iterator,
		"proposerDuty":                            &c.ProposerDutyConfig.Iterator,
		"elaboratedAttestation":                   &c.ElaboratedAttestationConfig.Iterator,
		"beaconValidators":                        &c.BeaconValidatorsConfig.Iterator,
		"beaconCommittee":                         &c.BeaconCommitteeConfig.Iterator,
		"beaconSyncCommittee":                     &c.BeaconSyncCommitteeConfig.Iterator,
		"beaconBlockSyncAggregate":                &c.BeaconBlockSyncAggregateConfig.Iterator,
		"beaconBlockGraffiti":                     &c.BeaconBlockGraffitiConfig.Iterator,
		"executionRequestDeposit":                 &c.ExecutionRequestDepositConfig.Iterator,
		"executionRequestWithdrawal":              &c.ExecutionRequestWithdrawalConfig.Iterator,
		"executionRequestConsolidation":           &c.ExecutionRequestConsolidationConfig.Iterator,
		"blockReward":                             &c.BlockRewardConfig.Iterator,
		"attestationReward":                       &c.AttestationRewardConfig.Iterator,
		"syncCommitteeReward":                     &c.SyncCommitteeRewardConfig.Iterator,
		"randao":                                  &c.RandaoConfig.Iterator,
		"finalityCheckpoint":                      &c.FinalityCheckpointConfig.Iterator,
		"beaconStatePendingDeposit":               &c.BeaconStatePendingDepositConfig.Iterator,
		"beaconStatePendingPartialWithdrawal":     &c.BeaconStatePendingPartialWithdrawalConfig.Iterator,
		"beaconStatePendingConsolidation":         &c.BeaconStatePendingConsolidationConfig.Iterator,
		"beaconStatePtcMember":                    &c.BeaconStatePtcMemberConfig.Iterator,
		"beaconStateBuilder":                      &c.BeaconStateBuilderConfig.Iterator,
		"beaconStateBuilderPendingPayment":        &c.BeaconStateBuilderPendingPaymentConfig.Iterator,
		"beaconStateBuilderPendingWithdrawal":     &c.BeaconStateBuilderPendingWithdrawalConfig.Iterator,
		"beaconStateExecutionPayloadAvailability": &c.BeaconStateExecutionPayloadAvailabilityConfig.Iterator,
		"blockAccessList":                         &c.BlockAccessListConfig.Iterator,
		"payloadAttestation":                      &c.PayloadAttestationConfig.Iterator,
		"executionPayloadBid":                     &c.ExecutionPayloadBidConfig.Iterator,
		"executionRequestBuilderDeposit":          &c.ExecutionRequestBuilderDepositConfig.Iterator,
		"executionRequestBuilderExit":             &c.ExecutionRequestBuilderExitConfig.Iterator,
		"blockAccessListSummary":                  &c.BlockAccessListSummaryConfig.Iterator,
	}
}
