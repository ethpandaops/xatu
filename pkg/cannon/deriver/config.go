package deriver

import (
	v1 "github.com/ethpandaops/xatu/pkg/cannon/deriver/beacon/eth/v1"
	v2 "github.com/ethpandaops/xatu/pkg/cannon/deriver/beacon/eth/v2"
	"github.com/ethpandaops/xatu/pkg/cannon/deriver/blockprint"
	"github.com/pkg/errors"
)

type Config struct {
	AttesterSlashingConfig     v2.AttesterSlashingDeriverConfig            `yaml:"attesterSlashing"`
	BLSToExecutionConfig       v2.BLSToExecutionChangeDeriverConfig        `yaml:"blsToExecutionChange"`
	DepositConfig              v2.DepositDeriverConfig                     `yaml:"deposit"`
	ExecutionTransactionConfig v2.ExecutionTransactionDeriverConfig        `yaml:"executionTransaction"`
	ProposerSlashingConfig     v2.ProposerSlashingDeriverConfig            `yaml:"proposerSlashing"`
	VoluntaryExitConfig        v2.VoluntaryExitDeriverConfig               `yaml:"voluntaryExit"`
	WithdrawalConfig           v2.WithdrawalDeriverConfig                  `yaml:"withdrawal"`
	BeaconBlockConfig          v2.BeaconBlockDeriverConfig                 `yaml:"beaconBlock"`
	BlockClassificationConfig  blockprint.BlockClassificationDeriverConfig `yaml:"blockClassification"`
	BeaconBlobSidecarConfig    v1.BeaconBlobDeriverConfig                  `yaml:"beaconBlobSidecar"`
	ProposerDutyConfig         v1.ProposerDutyDeriverConfig                `yaml:"proposerDuty"`
}

func (c *Config) Validate() error {
	if err := c.BlockClassificationConfig.Validate(); err != nil {
		return errors.Wrap(err, "invalid block classification deriver config")
	}

	return nil
}
