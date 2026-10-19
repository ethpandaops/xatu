package blockarchive

import (
	"errors"
	"fmt"
	"net/url"

	"github.com/ethpandaops/beacon/pkg/human"
)

// Config configures the era block source; anything it cannot serve falls back to the beacon node.
type Config struct {
	// Enabled turns the era block source on.
	Enabled bool `yaml:"enabled" default:"false"`
	// URL is the directory serving the era files, fetched as <url>/<file>.
	URL string `yaml:"url"`
	// ListingURL is any body naming the era files and sidecars (HTML index, JSON or paged JSON); defaults to URL.
	ListingURL string `yaml:"listingUrl"`
	// Network is the era file name prefix; defaults to the beacon node's network name.
	Network string `yaml:"network"`
	// Headers are sent on every archive request (e.g. authorization).
	Headers map[string]string `yaml:"headers"`
	// Timeout bounds one fetch from the archive, including a whole era download.
	Timeout human.Duration `yaml:"timeout" default:"5m"`
	// ListingTTL is how long the listing, an archive failure and a file that failed to verify are trusted.
	ListingTTL human.Duration `yaml:"listingTtl" default:"5m"`
	// MaxConcurrentFetches bounds parallel fetches from the archive.
	MaxConcurrentFetches int `yaml:"maxConcurrentFetches" default:"2"`
	// RangeSlots is the aligned window fetched with HTTP Range, at most a mainnet-preset era; zero reads whole eras.
	RangeSlots uint64 `yaml:"rangeSlots" default:"256"`
}

// mainnetSlotsPerEra is SLOTS_PER_HISTORICAL_ROOT in the mainnet preset, the largest era in use.
const mainnetSlotsPerEra = 8192

// MaxFetchSlots is the most slots one fetch returns, so callers can size caches to hold a whole fetch.
func (c *Config) MaxFetchSlots() uint64 {
	if c.RangeSlots == 0 {
		return mainnetSlotsPerEra
	}

	return c.RangeSlots
}

// Validate checks the config when the era source is enabled.
func (c *Config) Validate() error {
	if !c.Enabled {
		return nil
	}

	if c.URL == "" {
		return errors.New("blockArchive.url is required when blockArchive is enabled")
	}

	for name, raw := range map[string]string{"url": c.URL, "listingUrl": c.ListingURL} {
		if raw == "" {
			continue
		}

		if u, err := url.Parse(raw); err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
			return fmt.Errorf("blockArchive.%s must be an http(s) URL", name)
		}
	}

	if c.MaxConcurrentFetches < 1 {
		return errors.New("blockArchive.maxConcurrentFetches must be at least 1")
	}

	if c.RangeSlots > mainnetSlotsPerEra {
		return fmt.Errorf("blockArchive.rangeSlots must be at most %d, the slots of a mainnet-preset era", mainnetSlotsPerEra)
	}

	if c.Timeout.Duration <= 0 || c.ListingTTL.Duration <= 0 {
		return errors.New("blockArchive.timeout and blockArchive.listingTtl must be positive")
	}

	return nil
}
