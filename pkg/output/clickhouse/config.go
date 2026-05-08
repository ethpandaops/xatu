package clickhouse

import (
	"errors"
	"fmt"

	chwriter "github.com/ethpandaops/xatu/pkg/clickhouse"
)

// Config configures the clickhouse output sink. The Writer fields are
// inlined so YAML keys land directly under `config:` (matching the layout
// used by the other sinks).
type Config struct {
	chwriter.Config `yaml:",inline"`

	// MetricsSubsystem is the Prometheus subsystem (under namespace "xatu")
	// for this sink's CH writer/router metrics. The sink itself is
	// binary-agnostic, so there is no built-in default — the binary that
	// instantiates this sink is responsible for setting a stable subsystem
	// name (cannon's wiring fills in "cannon", consumoor uses "consumoor",
	// etc.). When empty, falls back to "clickhouse" inside the metrics
	// layer; that fallback exists for safety only and shouldn't be relied
	// on by production binaries.
	MetricsSubsystem string `yaml:"metricsSubsystem"`

	// RestrictToTablePrefixes, when non-empty, filters the route catalog
	// to only routes whose target table name begins with any of the listed
	// prefixes. This shrinks the set of tables the writer registers and
	// validates at startup. The field is here for binaries whose event set
	// is narrower than the full route catalog (cannon emits only
	// canonical_*, sentries emit beacon_api_* / libp2p_*, etc.). The
	// binary's wiring is the right place to set this — user YAML should
	// not have to know the prefix taxonomy. Empty means use the full
	// catalog.
	RestrictToTablePrefixes []string `yaml:"restrictToTablePrefixes"`
}

// Validate checks the sink configuration for errors.
func (c *Config) Validate() error {
	if c == nil {
		return errors.New("config is required")
	}

	if err := c.Config.Validate(); err != nil {
		return fmt.Errorf("clickhouse sink: %w", err)
	}

	return nil
}
