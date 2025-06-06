package clmimicry

import (
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// ActiveShardsConfig represents a list of active shards that can be specified
// as individual numbers or ranges (e.g., "0-255").
type ActiveShardsConfig []interface{}

// TracesConfig represents the new trace-based configuration.
type TracesConfig struct {
	// Whether or not trace config is globally enabled.
	Enabled bool `yaml:"enabled" default:"false"`
	// AlwaysRecordRootRpcEvents is a flag that controls whether or not to record
	// the root rpc events even if there are no rpc meta/control level messages.
	AlwaysRecordRootRpcEvents bool `yaml:"alwaysRecordRootRpcEvents" default:"false"`
	// Topics allows for per-topic configuration.
	Topics map[string]TopicConfig `yaml:"topics"`
	// Compiled regex patterns.
	compiledPatterns map[string]*regexp.Regexp
}

// TopicConfig represents configuration for a specific topic pattern.
type TopicConfig struct {
	// Total number of shards for this topic.
	TotalShards uint64 `yaml:"totalShards" default:"64"`
	// List of active shards to process. Supports individual numbers and ranges (e.g., "0-255").
	ActiveShardsRaw ActiveShardsConfig `yaml:"activeShards"`
	// Processed active shards (populated during validation).
	ActiveShards []uint64 `yaml:"-"`
	// Key to use for sharding (MsgID, PeerID, etc).
	ShardingKey string `yaml:"shardingKey" default:"MsgID"`
}

// Validate validates the traces config.
func (e *TracesConfig) Validate() error {
	if e.Enabled {
		if err := validateTracesConfig(e); err != nil {
			return fmt.Errorf("invalid traces config: %w", err)
		}

		// Pre-compile all regex patterns, otherwise, with each event grok, we'll be
		// compiling the regex.
		if err := e.CompilePatterns(); err != nil {
			return fmt.Errorf("failed to compile regex patterns: %w", err)
		}
	}

	return nil
}

// CompilePatterns pre-compiles all regex patterns for better performance.
func (e *TracesConfig) CompilePatterns() error {
	e.compiledPatterns = make(map[string]*regexp.Regexp)

	for pattern := range e.Topics {
		compiled, err := regexp.Compile(pattern)
		if err != nil {
			return fmt.Errorf("invalid regex pattern '%s': %w", pattern, err)
		}

		e.compiledPatterns[pattern] = compiled
	}

	return nil
}

// FindMatchingTopicConfig finds a matching topic configuration for an event type.
func (e *TracesConfig) FindMatchingTopicConfig(eventType string) (*TopicConfig, bool) {
	if !e.Enabled || len(e.Topics) == 0 {
		return nil, false
	}

	// If patterns haven't been compiled yet, compile them now.
	if e.compiledPatterns == nil {
		if err := e.CompilePatterns(); err != nil {
			return nil, false
		}
	}

	// Use pre-compiled patterns for matching
	for pattern, compiled := range e.compiledPatterns {
		if compiled.MatchString(eventType) {
			config := e.Topics[pattern]

			return &config, true
		}
	}

	return nil, false
}

// LogSummary returns a human-readable summary of the trace configuration.
func (e *TracesConfig) LogSummary() string {
	if !e.Enabled {
		return "Trace-based sampling disabled"
	}

	if len(e.Topics) == 0 {
		return "Trace-based sampling enabled but no topics configured"
	}

	summary := fmt.Sprintf("Trace-based sampling enabled with %d topic patterns:", len(e.Topics))

	// Sort patterns for consistent output.
	patterns := make([]string, 0, len(e.Topics))
	for pattern := range e.Topics {
		patterns = append(patterns, pattern)
	}

	sort.Strings(patterns)

	for _, pattern := range patterns {
		var (
			config           = e.Topics[pattern]
			activePercentage = float64(len(config.ActiveShards)) / float64(config.TotalShards) * 100.0
			isFirehose       = len(config.ActiveShards) == int(config.TotalShards) //nolint:gosec // controlled config, no overflow.
		)

		if isFirehose {
			summary += fmt.Sprintf(
				"\n  - Pattern '%s': FIREHOSE (all %d shards active), sharding on %s",
				pattern,
				config.TotalShards,
				config.ShardingKey,
			)
		} else {
			// Sort active shards for better readability.
			activeShards := make([]string, 0, len(config.ActiveShards))
			for _, shard := range config.ActiveShards {
				activeShards = append(activeShards, fmt.Sprintf("%d", shard))
			}

			// Truncate the list if it's too long.
			shardsDisplay := strings.Join(activeShards, ",")
			if len(activeShards) > 10 {
				shardsDisplay = strings.Join(activeShards[:10], ",") + ",...(" + fmt.Sprintf("%d", len(activeShards)-10) + " more)"
			}

			summary += fmt.Sprintf(
				"\n  - Pattern '%s': %d/%d shards active (%.1f%%) [%s], sharding on %s",
				pattern,
				len(config.ActiveShards),
				config.TotalShards,
				activePercentage,
				shardsDisplay,
				config.ShardingKey,
			)
		}
	}

	return summary
}

// ToUint64Slice converts the ActiveShardsConfig to a slice of uint64.
// It expands any ranges found in the configuration.
func (a ActiveShardsConfig) ToUint64Slice() ([]uint64, error) {
	var (
		result = make([]uint64, 0)
		seen   = make(map[uint64]bool) // To avoid duplicates.
	)

	for _, item := range a {
		switch v := item.(type) {
		case int:
			shard := uint64(v) //nolint:gosec // conversion fine.

			if !seen[shard] {
				result = append(result, shard)
				seen[shard] = true
			}
		case uint64:
			if !seen[v] {
				result = append(result, v)
				seen[v] = true
			}
		case string:
			// Handle range syntax like "0-255".
			if strings.Contains(v, "-") {
				parts := strings.Split(v, "-")
				if len(parts) != 2 {
					return nil, fmt.Errorf("invalid range format '%s', expected 'start-end'", v)
				}

				start, err := strconv.ParseUint(strings.TrimSpace(parts[0]), 10, 64)
				if err != nil {
					return nil, fmt.Errorf("invalid range start '%s': %w", parts[0], err)
				}

				end, err := strconv.ParseUint(strings.TrimSpace(parts[1]), 10, 64)
				if err != nil {
					return nil, fmt.Errorf("invalid range end '%s': %w", parts[1], err)
				}

				if start > end {
					return nil, fmt.Errorf("invalid range '%s': start (%d) is greater than end (%d)", v, start, end)
				}

				// Add all numbers in the range.
				for i := start; i <= end; i++ {
					if !seen[i] {
						result = append(result, i)
						seen[i] = true
					}
				}
			} else {
				// Handle single number as string.
				shard, err := strconv.ParseUint(v, 10, 64)
				if err != nil {
					return nil, fmt.Errorf("invalid shard number '%s': %w", v, err)
				}

				if !seen[shard] {
					result = append(result, shard)
					seen[shard] = true
				}
			}
		default:
			return nil, fmt.Errorf("unsupported type for active shard: %T", v)
		}
	}

	// Sort the result for consistency.
	sort.Slice(result, func(i, j int) bool {
		return result[i] < result[j]
	})

	return result, nil
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

		// Process raw active shards (expand ranges).
		if len(topicConfig.ActiveShardsRaw) == 0 {
			return fmt.Errorf("active_shards cannot be empty for pattern '%s'", pattern)
		}

		activeShards, err := topicConfig.ActiveShardsRaw.ToUint64Slice()
		if err != nil {
			return fmt.Errorf("invalid active_shards for pattern '%s': %w", pattern, err)
		}

		if len(activeShards) == 0 {
			return fmt.Errorf("active_shards cannot be empty for pattern '%s'", pattern)
		}

		// Validate that all active shards are within range.
		for _, shard := range activeShards {
			if shard >= topicConfig.TotalShards {
				return fmt.Errorf(
					"active shard %d is out of range (0-%d) for pattern '%s'",
					shard,
					topicConfig.TotalShards-1,
					pattern,
				)
			}
		}

		// Update the topic config with processed active shards.
		updatedConfig := topicConfig
		updatedConfig.ActiveShards = activeShards
		config.Topics[pattern] = updatedConfig

		// Validate sharding key, if provided (if empty, will default to MsgID).
		switch ShardingKeyType(topicConfig.ShardingKey) {
		case ShardingKeyTypeMsgID, ShardingKeyTypePeerID, "": // Allow empty sharding key (will default to MsgID)
			// Valid sharding key types
		default:
			return fmt.Errorf(
				"invalid sharding key '%s' for pattern '%s', valid values are: %s, %s",
				topicConfig.ShardingKey,
				pattern,
				ShardingKeyTypeMsgID,
				ShardingKeyTypePeerID,
			)
		}
	}

	return nil
}
