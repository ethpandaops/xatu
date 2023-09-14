package deriver

import v2 "github.com/ethpandaops/xatu/pkg/cannon/deriver/beacon/eth/v2"

type Config struct {
	AttesterSlashingConfig     v2.AttesterSlashingDeriverConfig     `yaml:"attesterSlashing"`
	BLSToExecutionConfig       v2.BLSToExecutionChangeDeriverConfig `yaml:"blsToExecutionChange"`
	DepositConfig              v2.DepositDeriverConfig              `yaml:"deposit"`
	ExecutionTransactionConfig v2.ExecutionTransactionDeriverConfig `yaml:"executionTransaction"`
	ProposerSlashingConfig     v2.ProposerSlashingDeriverConfig     `yaml:"proposerSlashing"`
	VoluntaryExitConfig        v2.VoluntaryExitDeriverConfig        `yaml:"voluntaryExit"`
	WithdrawalConfig           v2.WithdrawalDeriverConfig           `yaml:"withdrawal"`
}

func (c *Config) Validate() error {
	return nil
}
