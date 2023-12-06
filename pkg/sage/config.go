package sage

import (
	"errors"
	"fmt"

	"github.com/ethpandaops/xatu/pkg/output"
	"github.com/ethpandaops/xatu/pkg/processor"
	"github.com/ethpandaops/xatu/pkg/sage/ethereum"
	"github.com/sirupsen/logrus"
)

type Config struct {
	// LoggingLevel is the logging level to use.
	LoggingLevel string `yaml:"logging" default:"info"`
	// PProfAddr is the address to listen on for pprof.
	PProfAddr *string `yaml:"pprofAddr"`
	// MetricsAddr is the address to server prometheus metrics on.
	MetricsAddr string `yaml:"metricsAddr" default:":9093"`
	// Name is the name of the service.
	Name string `yaml:"name" default:"xatu-sage"`
	// Outputs configuration
	Outputs []output.Config `yaml:"outputs"`
	// ArmiarmaURL configuration
	ArmiarmaURL string `yaml:"armiarmaUrl"`
	// NTP Server to use for clock drift correction
	NTPServer string `yaml:"ntpServer" default:"time.google.com"`
	// Ethereum configuration
	Ethereum ethereum.Config `yaml:"ethereum"`
	// Workers is the number of workers to use for processing events.
	// Warning: Running more than 1 worker may result in more than $DUPLICATE_ATTESTATION_THRESHOLD
	// duplicate events being sent to outputs.
	Workers int `yaml:"workers" default:"1"`
	// DuplicateAttestationThreshold is the number of duplicate attestations to allow before
	// ignoring the event.
	DuplicateAttestationThreshold int `yaml:"duplicateAttestationThreshold" default:"3"`
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

func (c *Config) createSinks(log logrus.FieldLogger) ([]output.Sink, error) {
	sinks := make([]output.Sink, len(c.Outputs))

	for i, out := range c.Outputs {
		sink, err := output.NewSink(out.Name,
			out.SinkType,
			out.Config,
			log,
			out.FilterConfig,
			processor.ShippingMethodAsync,
		)
		if err != nil {
			return nil, err
		}

		sinks[i] = sink
	}

	return sinks, nil
}
