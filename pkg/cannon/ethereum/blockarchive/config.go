package blockarchive

import (
	"errors"

	"github.com/ethpandaops/beacon/pkg/human"
)

// Config configures serving finalized beacon blocks and Gloas execution
// payload envelopes from block-archiver bundles instead of the beacon node.
// Slots that no published bundle covers, and any archive error, fall back to
// the beacon node.
type Config struct {
	// Enabled turns the archive block source on.
	Enabled bool `yaml:"enabled" default:"false"`
	// Address is the block-archiver HTTP base URL.
	Address string `yaml:"address"`
	// Network is the archive's name for the network. Defaults to the beacon
	// node's network name.
	Network string `yaml:"network"`
	// Headers are sent on every archive request (e.g. authorization).
	Headers map[string]string `yaml:"headers"`
	// Timeout bounds one fetch from the archive, including a full bundle download.
	Timeout human.Duration `yaml:"timeout" default:"2m"`
	// CoverageTTL is how long the archive's bundle coverage, "not bundled"
	// answers and a failed archive are trusted before asking again.
	CoverageTTL human.Duration `yaml:"coverageTtl" default:"1m"`
	// MaxConcurrentFetches bounds parallel block fetches from the archive.
	MaxConcurrentFetches int `yaml:"maxConcurrentFetches" default:"2"`
	// RangeSlots, when non-zero, fetches aligned windows of this many slots of a
	// bundle with HTTP Range requests instead of the whole bundle.
	RangeSlots uint64 `yaml:"rangeSlots" default:"0"`
}

// Validate checks the config when the archive source is enabled.
func (c *Config) Validate() error {
	if !c.Enabled {
		return nil
	}

	if c.Address == "" {
		return errors.New("blockArchive.address is required when blockArchive is enabled")
	}

	if c.MaxConcurrentFetches < 1 {
		return errors.New("blockArchive.maxConcurrentFetches must be at least 1")
	}

	if c.Timeout.Duration <= 0 || c.CoverageTTL.Duration <= 0 {
		return errors.New("blockArchive.timeout and blockArchive.coverageTtl must be positive")
	}

	return nil
}
