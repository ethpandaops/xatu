package s3blobstore

import (
	"errors"

	"github.com/ethpandaops/xatu/pkg/output/internal/s3"
)

// Config configures the s3blobstore sink. The s3.Config fields are
// inlined so YAML keys land directly under `config:` (matching the
// layout used by the other sinks).
type Config struct {
	s3.Config `yaml:",inline"`

	// KeyPrefix is prepended to every object key. The full key shape is
	//
	//   <KeyPrefix><network>/<versioned_hash><KeySuffix>
	//
	// Defaults to empty, matching the existing Vector-based blob archive
	// convention. Use a prefix when sharing a bucket with other data.
	KeyPrefix string `yaml:"keyPrefix"`

	// KeySuffix is appended after the versioned hash. Defaults to ".gz"
	// because blob payloads are uploaded gzip-compressed; downstream
	// consumers identify gzipped objects by extension.
	KeySuffix string `yaml:"keySuffix" default:".gz"`

	// Concurrency caps the number of in-flight uploads when handling a
	// batch. The deriver typically delivers ~6 blobs/slot × 32 slots per
	// epoch ≈ ~192 blobs; serial uploads over a 50ms-RTT link would gate
	// checkpoint advance on ~10s of latency.
	Concurrency int `yaml:"concurrency" default:"8"`
}

// Validate checks the config.
func (c *Config) Validate() error {
	if c == nil {
		return errors.New("config is required")
	}

	if err := c.Config.Validate(); err != nil {
		return err
	}

	if c.Concurrency <= 0 {
		return errors.New("concurrency must be > 0")
	}

	return nil
}
