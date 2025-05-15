package clmimicry

import (
	"errors"
	"fmt"
	"regexp"
)

// TracesConfig represents the new trace-based configuration.
type TracesConfig struct {
	// Whether or not trace config is globally enabled.
	Enabled bool `yaml:"enabled" default:"false"`
	// Topics allows for per-topic configuration.
	Topics map[string]TopicConfig `yaml:"topics"`
}

// TopicConfig represents configuration for a specific topic pattern.
type TopicConfig struct {
	// Total number of shards for this topic.
	TotalShards uint64 `yaml:"totalShards" default:"64"`
	// List of active shards to process.
	ActiveShards []uint64 `yaml:"activeShards"`
}

// Validate validates the traces config.
func (e *TracesConfig) Validate() error {
	if e.Enabled {
		if err := validateTracesConfig(e); err != nil {
			return fmt.Errorf("invalid traces config: %w", err)
		}
	}

	return nil
}

// FindMatchingTopicConfig finds a matching topic configuration for an event type.
func (e *TracesConfig) FindMatchingTopicConfig(eventType string) (*TopicConfig, bool) {
	if !e.Enabled || len(e.Topics) == 0 {
		return nil, false
	}

	for pattern, config := range e.Topics {
		matched, err := regexp.MatchString(pattern, eventType)
		if err == nil && matched {
			return &config, true
		}
	}

	return nil, false
}

func validateTracesConfig(config *TracesConfig) error {
	if len(config.Topics) == 0 {
		return errors.New("no topics configured")
	}

	for pattern, topicConfig := range config.Topics {
		// Validate regex pattern.
		if _, err := regexp.Compile(pattern); err != nil {
			return fmt.Errorf("invalid regex pattern '%s': %w", pattern, err)
		}

		// Validate total shards.
		if topicConfig.TotalShards == 0 {
			return fmt.Errorf("total_shards must be greater than 0 for pattern '%s'", pattern)
		}

		// Validate active shards.
		if len(topicConfig.ActiveShards) == 0 {
			return fmt.Errorf("active_shards cannot be empty for pattern '%s'", pattern)
		}

		for _, shard := range topicConfig.ActiveShards {
			if shard >= topicConfig.TotalShards {
				return fmt.Errorf(
					"active shard %d is out of range (0-%d) for pattern '%s'",
					shard,
					topicConfig.TotalShards-1,
					pattern,
				)
			}
		}
	}

	return nil
}
