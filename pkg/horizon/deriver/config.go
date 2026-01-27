package deriver

// Config holds configuration for all Horizon derivers.
type Config struct {
	// Block-based derivers (real-time processing via HEAD iterator)
	BeaconBlockConfig           DeriverConfig `yaml:"beaconBlock"`
	AttesterSlashingConfig      DeriverConfig `yaml:"attesterSlashing"`
	ProposerSlashingConfig      DeriverConfig `yaml:"proposerSlashing"`
	DepositConfig               DeriverConfig `yaml:"deposit"`
	WithdrawalConfig            DeriverConfig `yaml:"withdrawal"`
	VoluntaryExitConfig         DeriverConfig `yaml:"voluntaryExit"`
	BLSToExecutionChangeConfig  DeriverConfig `yaml:"blsToExecutionChange"`
	ExecutionTransactionConfig  DeriverConfig `yaml:"executionTransaction"`
	ElaboratedAttestationConfig DeriverConfig `yaml:"elaboratedAttestation"`

	// Epoch-based derivers (triggered midway through epoch via Epoch iterator)
	ProposerDutyConfig     DeriverConfig          `yaml:"proposerDuty"`
	BeaconBlobConfig       DeriverConfig          `yaml:"beaconBlob"`
	BeaconValidatorsConfig BeaconValidatorsConfig `yaml:"beaconValidators"`
	BeaconCommitteeConfig  DeriverConfig          `yaml:"beaconCommittee"`
}

// DeriverConfig is the common configuration for a deriver.
type DeriverConfig struct {
	Enabled bool `yaml:"enabled" default:"true"`
}

// BeaconValidatorsConfig is the configuration for the beacon validators deriver.
type BeaconValidatorsConfig struct {
	Enabled   bool `yaml:"enabled" default:"true"`
	ChunkSize int  `yaml:"chunkSize" default:"100"`
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		// Block-based derivers.
		BeaconBlockConfig:           DeriverConfig{Enabled: true},
		AttesterSlashingConfig:      DeriverConfig{Enabled: true},
		ProposerSlashingConfig:      DeriverConfig{Enabled: true},
		DepositConfig:               DeriverConfig{Enabled: true},
		WithdrawalConfig:            DeriverConfig{Enabled: true},
		VoluntaryExitConfig:         DeriverConfig{Enabled: true},
		BLSToExecutionChangeConfig:  DeriverConfig{Enabled: true},
		ExecutionTransactionConfig:  DeriverConfig{Enabled: true},
		ElaboratedAttestationConfig: DeriverConfig{Enabled: true},
		// Epoch-based derivers.
		ProposerDutyConfig:     DeriverConfig{Enabled: true},
		BeaconBlobConfig:       DeriverConfig{Enabled: true},
		BeaconValidatorsConfig: BeaconValidatorsConfig{Enabled: true, ChunkSize: 100},
		BeaconCommitteeConfig:  DeriverConfig{Enabled: true},
	}
}

// Validate validates the config.
func (c *Config) Validate() error {
	return nil
}
