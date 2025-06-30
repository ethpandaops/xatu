package ethereum

import (
	"github.com/sirupsen/logrus"
)

type Config struct {
	// OverrideNetworkName is the name of the network to use for mimicry.
	// If not set, the network name will be automatically detected.
	OverrideNetworkName string `yaml:"overrideNetworkName" default:""`
}

func (c *Config) Validate() error {
	return nil
}

func (c *Config) ApplyOverrides(overrideNetworkName string, log logrus.FieldLogger) {
	if overrideNetworkName != "" {
		log.WithField("network", overrideNetworkName).Info("Overriding network name")
		c.OverrideNetworkName = overrideNetworkName
	}
}
