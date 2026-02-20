package consumoor

import (
	"fmt"
	"sort"
	"strings"

	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse"
	"github.com/ethpandaops/xatu/pkg/consumoor/source"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

// Config is the configuration for the consumoor service.
type Config struct {
	// LoggingLevel is the logging level to use.
	LoggingLevel string `yaml:"logging" default:"info"`
	// MetricsAddr is the address to listen on for metrics.
	MetricsAddr string `yaml:"metricsAddr" default:":9090"`
	// PProfAddr is the address to listen on for pprof.
	PProfAddr *string `yaml:"pprofAddr"`

	// Kafka is the Kafka source configuration.
	Kafka source.KafkaConfig `yaml:"kafka"`
	// ClickHouse is the ClickHouse sink configuration.
	ClickHouse clickhouse.Config `yaml:"clickhouse"`

	// DisabledEvents is a list of event names to drop without processing.
	DisabledEvents []string `yaml:"disabledEvents"`
}

// Validate checks the configuration for errors.
func (c *Config) Validate() error {
	if err := c.Kafka.Validate(); err != nil {
		return err
	}

	if err := c.ClickHouse.Validate(); err != nil {
		return err
	}

	if _, err := c.DisabledEventEnums(); err != nil {
		return err
	}

	return nil
}

// DisabledEventEnums parses disabled event names into typed enum values.
// Unknown names are rejected to fail fast during startup.
func (c *Config) DisabledEventEnums() ([]xatu.Event_Name, error) {
	out := make([]xatu.Event_Name, 0, len(c.DisabledEvents))
	invalid := make([]string, 0)

	for _, name := range c.DisabledEvents {
		val, ok := xatu.Event_Name_value[name]
		if !ok {
			invalid = append(invalid, name)

			continue
		}

		out = append(out, xatu.Event_Name(val))
	}

	if len(invalid) > 0 {
		sort.Strings(invalid)

		return nil, fmt.Errorf("unknown disabledEvents: %s", strings.Join(invalid, ", "))
	}

	return out, nil
}

// Override holds values that can be set via CLI flags or environment
// variables, overriding the config file.
type Override struct {
	MetricsAddr struct {
		Enabled bool
		Value   string
	}
}

// ApplyOverrides applies CLI/env overrides to the configuration.
func (c *Config) ApplyOverrides(o *Override) {
	if o == nil {
		return
	}

	if o.MetricsAddr.Enabled {
		c.MetricsAddr = o.MetricsAddr.Value
	}
}
