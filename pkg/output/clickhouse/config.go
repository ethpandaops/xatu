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
	// for this sink's CH writer/router metrics. Defaults to "cannon".
	// Override only if you run multiple distinct CH targets in the same
	// process and want them reported separately.
	MetricsSubsystem string `yaml:"metricsSubsystem" default:"cannon"`

	// RestrictToTablePrefixes, when non-empty, filters the route catalog
	// to only routes whose target table name begins with any of the listed
	// prefixes. This shrinks the set of tables the writer registers and
	// validates at startup. For cannon (which only writes canonical_*
	// tables), set this to ["canonical_"] to avoid requiring the libp2p,
	// mev, execution, and node table schemas in the target ClickHouse.
	// Empty means use the full route catalog (all ~150 tables).
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
