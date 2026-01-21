package deriver

// Config holds configuration for all Horizon derivers.
type Config struct {
	// Block-based derivers (real-time processing)
	BeaconBlockConfig           DeriverConfig `yaml:"beaconBlock"`
	AttesterSlashingConfig      DeriverConfig `yaml:"attesterSlashing"`
	ProposerSlashingConfig      DeriverConfig `yaml:"proposerSlashing"`
	DepositConfig               DeriverConfig `yaml:"deposit"`
	WithdrawalConfig            DeriverConfig `yaml:"withdrawal"`
	VoluntaryExitConfig         DeriverConfig `yaml:"voluntaryExit"`
	BLSToExecutionChangeConfig  DeriverConfig `yaml:"blsToExecutionChange"`
	ExecutionTransactionConfig  DeriverConfig `yaml:"executionTransaction"`
	ElaboratedAttestationConfig DeriverConfig `yaml:"elaboratedAttestation"`
}

// DeriverConfig is the common configuration for a deriver.
type DeriverConfig struct {
	Enabled bool `yaml:"enabled" default:"true"`
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		BeaconBlockConfig:           DeriverConfig{Enabled: true},
		AttesterSlashingConfig:      DeriverConfig{Enabled: true},
		ProposerSlashingConfig:      DeriverConfig{Enabled: true},
		DepositConfig:               DeriverConfig{Enabled: true},
		WithdrawalConfig:            DeriverConfig{Enabled: true},
		VoluntaryExitConfig:         DeriverConfig{Enabled: true},
		BLSToExecutionChangeConfig:  DeriverConfig{Enabled: true},
		ExecutionTransactionConfig:  DeriverConfig{Enabled: true},
		ElaboratedAttestationConfig: DeriverConfig{Enabled: true},
	}
}

// Validate validates the config.
func (c *Config) Validate() error {
	return nil
}
