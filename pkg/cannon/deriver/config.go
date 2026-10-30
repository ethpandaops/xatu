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
	Phase0RewardConfig                        v1.Phase0RewardDeriverConfig                        `yaml:"phase0Reward"`
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

// lookAheadSlotsPerEpoch sizes batch look-aheads with the mainnet preset, the largest SLOTS_PER_EPOCH in use.
const lookAheadSlotsPerEpoch = 32

func (c *Config) Validate() error {
	if err := c.Consensus.BlockAccessListConfig.Validate(); err != nil {
		return fmt.Errorf("consensus.blockAccessList: %w", err)
	}

	// These derivers do not derive a step epoch by epoch, or share one beacon state download per epoch.
	singleEpoch := map[*iterator.BackfillingCheckpointConfig]bool{
		&c.Consensus.BeaconSyncCommitteeConfig.Iterator:                     true,
		&c.Consensus.BeaconValidatorsConfig.Iterator:                        true,
		&c.Consensus.BlockAccessListConfig.Iterator:                         true,
		&c.Consensus.BeaconStatePtcMemberConfig.Iterator:                    true,
		&c.Consensus.BeaconStateBuilderConfig.Iterator:                      true,
		&c.Consensus.BeaconStateBuilderPendingPaymentConfig.Iterator:        true,
		&c.Consensus.BeaconStateBuilderPendingWithdrawalConfig.Iterator:     true,
		&c.Consensus.BeaconStateExecutionPayloadAvailabilityConfig.Iterator: true,
		&c.Consensus.Phase0RewardConfig.Iterator:                            true,
	}

	for name, d := range c.Consensus.iterators() {
		if err := d.iterator.Validate(); err != nil {
			return fmt.Errorf("consensus.%s: %w", name, err)
		}

		if singleEpoch[d.iterator] && d.iterator.Backfill.EpochsPerBatch > 1 {
			return fmt.Errorf("consensus.%s: iterator.backfill.epochsPerBatch is not supported, got %d", name, d.iterator.Backfill.EpochsPerBatch)
		}
	}

	return nil
}

// BatchLookAheadSlots is how many blocks the enabled batching backfills preload at once: two batches each.
func (c *Config) BatchLookAheadSlots() uint64 {
	var slots uint64

	for _, d := range c.Consensus.iterators() {
		if !d.enabled || !d.iterator.Backfill.Enabled || d.iterator.Backfill.EpochsPerBatch <= 1 {
			continue
		}

		slots += 2 * d.iterator.Backfill.EpochsPerBatch * lookAheadSlotsPerEpoch
	}

	return slots
}

// consensusIterator is one consensus deriver's enabled flag and iterator config.
type consensusIterator struct {
	enabled  bool
	iterator *iterator.BackfillingCheckpointConfig
}

// iterators returns every consensus deriver's iterator config keyed by its YAML name.
func (c *ConsensusConfig) iterators() map[string]consensusIterator {
	return map[string]consensusIterator{
		"attesterSlashing":                        {enabled: c.AttesterSlashingConfig.Enabled, iterator: &c.AttesterSlashingConfig.Iterator},
		"blsToExecutionChange":                    {enabled: c.BLSToExecutionConfig.Enabled, iterator: &c.BLSToExecutionConfig.Iterator},
		"deposit":                                 {enabled: c.DepositConfig.Enabled, iterator: &c.DepositConfig.Iterator},
		"executionTransaction":                    {enabled: c.ExecutionTransactionConfig.Enabled, iterator: &c.ExecutionTransactionConfig.Iterator},
		"proposerSlashing":                        {enabled: c.ProposerSlashingConfig.Enabled, iterator: &c.ProposerSlashingConfig.Iterator},
		"voluntaryExit":                           {enabled: c.VoluntaryExitConfig.Enabled, iterator: &c.VoluntaryExitConfig.Iterator},
		"withdrawal":                              {enabled: c.WithdrawalConfig.Enabled, iterator: &c.WithdrawalConfig.Iterator},
		"beaconBlock":                             {enabled: c.BeaconBlockConfig.Enabled, iterator: &c.BeaconBlockConfig.Iterator},
		"beaconBlobSidecar":                       {enabled: c.BeaconBlobSidecarConfig.Enabled, iterator: &c.BeaconBlobSidecarConfig.Iterator},
		"proposerDuty":                            {enabled: c.ProposerDutyConfig.Enabled, iterator: &c.ProposerDutyConfig.Iterator},
		"elaboratedAttestation":                   {enabled: c.ElaboratedAttestationConfig.Enabled, iterator: &c.ElaboratedAttestationConfig.Iterator},
		"beaconValidators":                        {enabled: c.BeaconValidatorsConfig.Enabled, iterator: &c.BeaconValidatorsConfig.Iterator},
		"beaconCommittee":                         {enabled: c.BeaconCommitteeConfig.Enabled, iterator: &c.BeaconCommitteeConfig.Iterator},
		"beaconSyncCommittee":                     {enabled: c.BeaconSyncCommitteeConfig.Enabled, iterator: &c.BeaconSyncCommitteeConfig.Iterator},
		"beaconBlockSyncAggregate":                {enabled: c.BeaconBlockSyncAggregateConfig.Enabled, iterator: &c.BeaconBlockSyncAggregateConfig.Iterator},
		"beaconBlockGraffiti":                     {enabled: c.BeaconBlockGraffitiConfig.Enabled, iterator: &c.BeaconBlockGraffitiConfig.Iterator},
		"executionRequestDeposit":                 {enabled: c.ExecutionRequestDepositConfig.Enabled, iterator: &c.ExecutionRequestDepositConfig.Iterator},
		"executionRequestWithdrawal":              {enabled: c.ExecutionRequestWithdrawalConfig.Enabled, iterator: &c.ExecutionRequestWithdrawalConfig.Iterator},
		"executionRequestConsolidation":           {enabled: c.ExecutionRequestConsolidationConfig.Enabled, iterator: &c.ExecutionRequestConsolidationConfig.Iterator},
		"blockReward":                             {enabled: c.BlockRewardConfig.Enabled, iterator: &c.BlockRewardConfig.Iterator},
		"attestationReward":                       {enabled: c.AttestationRewardConfig.Enabled, iterator: &c.AttestationRewardConfig.Iterator},
		"phase0Reward":                            {enabled: c.Phase0RewardConfig.Enabled, iterator: &c.Phase0RewardConfig.Iterator},
		"syncCommitteeReward":                     {enabled: c.SyncCommitteeRewardConfig.Enabled, iterator: &c.SyncCommitteeRewardConfig.Iterator},
		"randao":                                  {enabled: c.RandaoConfig.Enabled, iterator: &c.RandaoConfig.Iterator},
		"finalityCheckpoint":                      {enabled: c.FinalityCheckpointConfig.Enabled, iterator: &c.FinalityCheckpointConfig.Iterator},
		"beaconStatePendingDeposit":               {enabled: c.BeaconStatePendingDepositConfig.Enabled, iterator: &c.BeaconStatePendingDepositConfig.Iterator},
		"beaconStatePendingPartialWithdrawal":     {enabled: c.BeaconStatePendingPartialWithdrawalConfig.Enabled, iterator: &c.BeaconStatePendingPartialWithdrawalConfig.Iterator},
		"beaconStatePendingConsolidation":         {enabled: c.BeaconStatePendingConsolidationConfig.Enabled, iterator: &c.BeaconStatePendingConsolidationConfig.Iterator},
		"beaconStatePtcMember":                    {enabled: c.BeaconStatePtcMemberConfig.Enabled, iterator: &c.BeaconStatePtcMemberConfig.Iterator},
		"beaconStateBuilder":                      {enabled: c.BeaconStateBuilderConfig.Enabled, iterator: &c.BeaconStateBuilderConfig.Iterator},
		"beaconStateBuilderPendingPayment":        {enabled: c.BeaconStateBuilderPendingPaymentConfig.Enabled, iterator: &c.BeaconStateBuilderPendingPaymentConfig.Iterator},
		"beaconStateBuilderPendingWithdrawal":     {enabled: c.BeaconStateBuilderPendingWithdrawalConfig.Enabled, iterator: &c.BeaconStateBuilderPendingWithdrawalConfig.Iterator},
		"beaconStateExecutionPayloadAvailability": {enabled: c.BeaconStateExecutionPayloadAvailabilityConfig.Enabled, iterator: &c.BeaconStateExecutionPayloadAvailabilityConfig.Iterator},
		"blockAccessList":                         {enabled: c.BlockAccessListConfig.Enabled, iterator: &c.BlockAccessListConfig.Iterator},
		"payloadAttestation":                      {enabled: c.PayloadAttestationConfig.Enabled, iterator: &c.PayloadAttestationConfig.Iterator},
		"executionPayloadBid":                     {enabled: c.ExecutionPayloadBidConfig.Enabled, iterator: &c.ExecutionPayloadBidConfig.Iterator},
		"executionRequestBuilderDeposit":          {enabled: c.ExecutionRequestBuilderDepositConfig.Enabled, iterator: &c.ExecutionRequestBuilderDepositConfig.Iterator},
		"executionRequestBuilderExit":             {enabled: c.ExecutionRequestBuilderExitConfig.Enabled, iterator: &c.ExecutionRequestBuilderExitConfig.Iterator},
		"blockAccessListSummary":                  {enabled: c.BlockAccessListSummaryConfig.Enabled, iterator: &c.BlockAccessListSummaryConfig.Iterator},
	}
}
