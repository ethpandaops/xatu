package deriver

import (
	"github.com/ethpandaops/xatu/pkg/cannon/iterator"
)

// DeriverConfig is the base configuration for all Cannon derivers.
// It combines the Enabled flag with iterator-specific configuration.
type DeriverConfig struct {
	Enabled  bool                                 `yaml:"enabled" default:"true"`
	Iterator iterator.BackfillingCheckpointConfig `yaml:"iterator"`
}

// BeaconValidatorsDeriverConfig extends DeriverConfig with validator-specific settings.
type BeaconValidatorsDeriverConfig struct {
	Enabled   bool                                 `yaml:"enabled" default:"true"`
	ChunkSize int                                  `yaml:"chunkSize" default:"100"`
	Iterator  iterator.BackfillingCheckpointConfig `yaml:"iterator"`
}

// Config holds configuration for all Cannon derivers.
type Config struct {
	AttesterSlashingConfig      DeriverConfig                 `yaml:"attesterSlashing"`
	BLSToExecutionConfig        DeriverConfig                 `yaml:"blsToExecutionChange"`
	DepositConfig               DeriverConfig                 `yaml:"deposit"`
	ExecutionTransactionConfig  DeriverConfig                 `yaml:"executionTransaction"`
	ProposerSlashingConfig      DeriverConfig                 `yaml:"proposerSlashing"`
	VoluntaryExitConfig         DeriverConfig                 `yaml:"voluntaryExit"`
	WithdrawalConfig            DeriverConfig                 `yaml:"withdrawal"`
	BeaconBlockConfig           DeriverConfig                 `yaml:"beaconBlock"`
	BeaconBlobSidecarConfig     DeriverConfig                 `yaml:"beaconBlobSidecar"`
	ProposerDutyConfig          DeriverConfig                 `yaml:"proposerDuty"`
	ElaboratedAttestationConfig DeriverConfig                 `yaml:"elaboratedAttestation"`
	BeaconValidatorsConfig      BeaconValidatorsDeriverConfig `yaml:"beaconValidators"`
	BeaconCommitteeConfig       DeriverConfig                 `yaml:"beaconCommittee"`
}

// Validate validates the deriver configuration.
func (c *Config) Validate() error {
	return nil
}
