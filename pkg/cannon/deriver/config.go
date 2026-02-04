package deriver

import (
	v1 "github.com/ethpandaops/xatu/pkg/cannon/deriver/beacon/eth/v1"
	v2 "github.com/ethpandaops/xatu/pkg/cannon/deriver/beacon/eth/v2"
)

type Config struct {
	AttesterSlashingConfig      v2.AttesterSlashingDeriverConfig      `yaml:"attesterSlashing"`
	BLSToExecutionConfig        v2.BLSToExecutionChangeDeriverConfig  `yaml:"blsToExecutionChange"`
	DepositConfig               v2.DepositDeriverConfig               `yaml:"deposit"`
	ExecutionTransactionConfig  v2.ExecutionTransactionDeriverConfig  `yaml:"executionTransaction"`
	ProposerSlashingConfig      v2.ProposerSlashingDeriverConfig      `yaml:"proposerSlashing"`
	VoluntaryExitConfig         v2.VoluntaryExitDeriverConfig         `yaml:"voluntaryExit"`
	WithdrawalConfig            v2.WithdrawalDeriverConfig            `yaml:"withdrawal"`
	BeaconBlockConfig           v2.BeaconBlockDeriverConfig           `yaml:"beaconBlock"`
	BeaconBlobSidecarConfig     v1.BeaconBlobDeriverConfig            `yaml:"beaconBlobSidecar"`
	ProposerDutyConfig          v1.ProposerDutyDeriverConfig          `yaml:"proposerDuty"`
	ElaboratedAttestationConfig v2.ElaboratedAttestationDeriverConfig `yaml:"elaboratedAttestation"`
	BeaconValidatorsConfig      v1.BeaconValidatorsDeriverConfig      `yaml:"beaconValidators"`
	BeaconCommitteeConfig       v1.BeaconCommitteeDeriverConfig       `yaml:"beaconCommittee"`
	BeaconSyncCommitteeConfig   v1.BeaconSyncCommitteeDeriverConfig   `yaml:"beaconSyncCommittee"`
}

func (c *Config) Validate() error {
	return nil
}
