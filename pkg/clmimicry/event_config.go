package clmimicry

import (
	"errors"
	"fmt"
)

// EventConfig represents configuration for all event types.
type EventConfig struct {
	RecvRPC              EventTypeConfig `yaml:"recvRpc" default:"{'enabled': true}"`
	SendRPC              EventTypeConfig `yaml:"sendRpc" default:"{'enabled': true}"`
	AddPeer              EventTypeConfig `yaml:"addPeer" default:"{'enabled': true}"`
	RemovePeer           EventTypeConfig `yaml:"removePeer" default:"{'enabled': true}"`
	Connected            EventTypeConfig `yaml:"connected" default:"{'enabled': true}"`
	Disconnected         EventTypeConfig `yaml:"disconnected" default:"{'enabled': true}"`
	Join                 EventTypeConfig `yaml:"join" default:"{'enabled': true}"`
	HandleMetadata       EventTypeConfig `yaml:"handleMetadata" default:"{'enabled': true}"`
	HandleStatus         EventTypeConfig `yaml:"handleStatus" default:"{'enabled': true}"`
	GossipSubBeaconBlock EventTypeConfig `yaml:"gossipSubBeaconBlock" default:"{'enabled': true}"`
	GossipSubAttestation EventTypeConfig `yaml:"gossipSubAttestation" default:"{'enabled': true}"`
	GossipSubBlobSidecar EventTypeConfig `yaml:"gossipSubBlobSidecar" default:"{'enabled': true}"`

	// The mode of sampling ("prefix", "regex", "modulo").
	SamplingMode string `yaml:"samplingMode" default:"prefix"`
}

// EventTypeConfig represents configuration for a specific event type.
type EventTypeConfig struct {
	// Whether this event type is enabled.
	Enabled bool `yaml:"enabled"`
	// Sampling configuration.
	Sampling *SamplingConfig `yaml:"sampling,omitempty"`
}

// SamplingConfig is the configuration for sampling a specific event type.
type SamplingConfig struct {
	// Whether sampling is enabled for this event type.
	Enabled bool `yaml:"enabled" default:"false"`
	// The pattern to match against (interpretation depends on Mode).
	Pattern string `yaml:"pattern" default:""`
	// Sample rate (0.0-1.0) for matching messages.
	SampleRate float64 `yaml:"sampleRate" default:"1.0"`
}

func (e *EventConfig) Validate() error {
	// Check if sampling mode is valid when any event has sampling enabled
	if hasSamplingEnabled(e) {
		if e.SamplingMode != "prefix" && e.SamplingMode != "regex" && e.SamplingMode != "modulo" {
			return fmt.Errorf("invalid sampling mode: %s. Must be one of: prefix, regex, modulo", e.SamplingMode)
		}
	}

	// Validate sampling configs for events that support sampling
	if e.GossipSubBeaconBlock.Sampling != nil && e.GossipSubBeaconBlock.Sampling.Enabled {
		if err := validateSamplingConfig(e.GossipSubBeaconBlock.Sampling, e.SamplingMode); err != nil {
			return fmt.Errorf("invalid gossipSubBeaconBlock sampling config: %w", err)
		}
	}

	if e.GossipSubAttestation.Sampling != nil && e.GossipSubAttestation.Sampling.Enabled {
		if err := validateSamplingConfig(e.GossipSubAttestation.Sampling, e.SamplingMode); err != nil {
			return fmt.Errorf("invalid gossipSubAttestation sampling config: %w", err)
		}
	}

	if e.GossipSubBlobSidecar.Sampling != nil && e.GossipSubBlobSidecar.Sampling.Enabled {
		if err := validateSamplingConfig(e.GossipSubBlobSidecar.Sampling, e.SamplingMode); err != nil {
			return fmt.Errorf("invalid gossipSubBlobSidecar sampling config: %w", err)
		}
	}

	return nil
}

// hasSamplingEnabled checks if any event has sampling enabled
func hasSamplingEnabled(e *EventConfig) bool {
	return (e.GossipSubBeaconBlock.Sampling != nil && e.GossipSubBeaconBlock.Sampling.Enabled) ||
		(e.GossipSubAttestation.Sampling != nil && e.GossipSubAttestation.Sampling.Enabled) ||
		(e.GossipSubBlobSidecar.Sampling != nil && e.GossipSubBlobSidecar.Sampling.Enabled)
}

func validateSamplingConfig(config *SamplingConfig, mode string) error {
	if config.Pattern == "" {
		return errors.New("pattern cannot be empty")
	}

	if config.SampleRate < 0.0 || config.SampleRate > 1.0 {
		return fmt.Errorf("sampleRate must be between 0.0 and 1.0, got %f", config.SampleRate)
	}

	if mode == "regex" {
		// In real implementation, would validate regex here
	}

	return nil
}
