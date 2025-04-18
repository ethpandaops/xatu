package sentry

import (
	"errors"
	"fmt"
	"time"

	"github.com/ethpandaops/beacon/pkg/human"
	"github.com/ethpandaops/xatu/pkg/observability"
	"github.com/ethpandaops/xatu/pkg/output"
	"github.com/ethpandaops/xatu/pkg/processor"
	"github.com/ethpandaops/xatu/pkg/sentry/ethereum"
	"github.com/ethpandaops/xatu/pkg/sentry/execution"
	"github.com/sirupsen/logrus"
)

type Config struct {
	// Preset is the name of the preset to use
	Preset string `yaml:"preset"`

	LoggingLevel string  `yaml:"logging" default:"info"`
	MetricsAddr  string  `yaml:"metricsAddr" default:":9090"`
	PProfAddr    *string `yaml:"pprofAddr"`

	// The name of the sentry
	Name string `yaml:"name"`

	// Ethereum configuration
	Ethereum ethereum.Config `yaml:"ethereum"`

	// Execution client configuration
	Execution *execution.Config `yaml:"execution" default:"{'enabled': false}"`

	// Outputs configuration
	Outputs []output.Config `yaml:"outputs"`

	// Labels configures the sentry with labels
	Labels map[string]string `yaml:"labels"`

	// NTP Server to use for clock drift correction
	NTPServer string `yaml:"ntpServer" default:"time.google.com"`

	// ForkChoice configuration
	ForkChoice *ForkChoiceConfig `yaml:"forkChoice" default:"{'enabled': false}"`

	// BeaconCommittees configuration
	BeaconCommittees *BeaconCommitteesConfig `yaml:"beaconCommittees" default:"{'enabled': false}"`

	// AttestationData configuration
	AttestationData *AttestationDataConfig `yaml:"attestationData" default:"{'enabled': false}"`

	// ProposerDuty configuration
	ProposerDuty *ProposerDutyConfig `yaml:"proposerDuty" default:"{'enabled': true}"`

	// ValidatorBlock configuration
	ValidatorBlock *ValidatorBlockConfig `yaml:"validatorBlock" default:"{'enabled': false}"`

	// Tracing configuration
	Tracing observability.TracingConfig `yaml:"tracing"`
}

func (c *Config) Validate() error {
	if err := c.Ethereum.Validate(); err != nil {
		return err
	}

	for _, output := range c.Outputs {
		if err := output.Validate(); err != nil {
			return fmt.Errorf("output %s: %w", output.Name, err)
		}
	}

	if err := c.ForkChoice.Validate(); err != nil {
		return fmt.Errorf("invalid forkChoice config: %w", err)
	}

	if err := c.Tracing.Validate(); err != nil {
		return fmt.Errorf("invalid tracing config: %w", err)
	}

	return nil
}

func (c *Config) CreateSinks(log logrus.FieldLogger) ([]output.Sink, error) {
	sinks := make([]output.Sink, len(c.Outputs))

	for i, out := range c.Outputs {
		if out.ShippingMethod == nil {
			shippingMethod := processor.ShippingMethodAsync
			out.ShippingMethod = &shippingMethod
		}

		sink, err := output.NewSink(out.Name,
			out.SinkType,
			out.Config,
			log,
			out.FilterConfig,
			*out.ShippingMethod,
		)
		if err != nil {
			return nil, err
		}

		sinks[i] = sink
	}

	return sinks, nil
}

type ForkChoiceConfig struct {
	Enabled bool `yaml:"enabled" default:"false"`

	OnReOrgEvent struct {
		Enabled bool `yaml:"enabled" default:"false"`
	} `yaml:"onReOrgEvent"`

	Interval struct {
		Enabled bool           `yaml:"enabled" default:"false"`
		Every   human.Duration `yaml:"every" default:"12s"`
	} `yaml:"interval"`

	At struct {
		Enabled   bool             `yaml:"enabled" default:"false"`
		SlotTimes []human.Duration `yaml:"slotTimes"`
	} `yaml:"at"`
}

func (f *ForkChoiceConfig) Validate() error {
	if f.At.Enabled {
		if len(f.At.SlotTimes) == 0 {
			return errors.New("at.slotTimes must be provided when at.enabled is true")
		}

		for _, slotTime := range f.At.SlotTimes {
			if slotTime.Duration > 12*time.Second {
				return errors.New("at.slotTimes must be less than 12s")
			}
		}
	}

	return nil
}

type AttestationDataConfig struct {
	Enabled bool `yaml:"enabled" default:"false"`

	AllCommittees bool `yaml:"allCommittees" default:"false"`

	Interval struct {
		Enabled bool           `yaml:"enabled" default:"false"`
		Every   human.Duration `yaml:"every" default:"12s"`
	} `yaml:"interval"`

	At struct {
		Enabled   bool             `yaml:"enabled" default:"false"`
		SlotTimes []human.Duration `yaml:"slotTimes"`
	} `yaml:"at"`
}

func (f *AttestationDataConfig) Validate() error {
	if f.At.Enabled {
		if len(f.At.SlotTimes) == 0 {
			return errors.New("at.slotTimes must be provided when at.enabled is true")
		}

		for _, slotTime := range f.At.SlotTimes {
			if slotTime.Duration > 12*time.Second {
				return errors.New("at.slotTimes must be less than 12s")
			}
		}
	}

	return nil
}

type ValidatorBlockConfig struct {
	Enabled bool `yaml:"enabled" default:"false"`

	Interval struct {
		Enabled bool           `yaml:"enabled" default:"false"`
		Every   human.Duration `yaml:"every" default:"12s"`
	} `yaml:"interval"`

	At struct {
		Enabled   bool             `yaml:"enabled" default:"false"`
		SlotTimes []human.Duration `yaml:"slotTimes"`
	} `yaml:"at"`
}

func (f *ValidatorBlockConfig) Validate() error {
	if f.At.Enabled {
		if len(f.At.SlotTimes) == 0 {
			return errors.New("at.slotTimes must be provided when at.enabled is true")
		}

		for _, slotTime := range f.At.SlotTimes {
			if slotTime.Duration > 12*time.Second {
				return errors.New("at.slotTimes must be less than 12s")
			}
		}
	}

	return nil
}

type BeaconCommitteesConfig struct {
	Enabled bool `yaml:"enabled" default:"false"`
}

type ProposerDutyConfig struct {
	Enabled bool `yaml:"enabled" default:"true"`
}
