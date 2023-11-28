package armiarma

import (
	"errors"
	"fmt"

	"github.com/ethpandaops/xatu/pkg/output"
)

type Config struct {
	// LoggingLevel is the logging level to use.
	LoggingLevel string `yaml:"logging" default:"info"`
	// PProfAddr is the address to listen on for pprof.
	PProfAddr *string `yaml:"pprofAddr"`
	// Outputs configuration
	Outputs []output.Config `yaml:"outputs"`
	// ArmiarmaURL configuration
	ArmiarmaURL string `yaml:"armiarmaURL"`
	// NTP Server to use for clock drift correction
	NTPServer string `yaml:"ntpServer" default:"time.google.com"`
}

func (c *Config) Validate() error {
	if c.ArmiarmaURL == "" {
		return errors.New("armiarmaURL is required")
	}

	for _, output := range c.Outputs {
		if err := output.Validate(); err != nil {
			return fmt.Errorf("invalid output config (%s): %w", output.Name, err)
		}
	}

	return nil
}
