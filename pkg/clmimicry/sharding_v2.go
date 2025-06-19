package clmimicry

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"gopkg.in/yaml.v3"
)

const (
	// DefaultTotalShards is the default number of shards if not specified
	DefaultTotalShards = 512
)

// UnifiedSharder provides a single sharding decision point for all events
type UnifiedSharder struct {
	config           *ShardingConfigV2
	eventCategorizer *EventCategorizer
	enabled          bool
}

// ShardingConfigV2 represents the simplified sharding configuration
type ShardingConfigV2 struct {
	// Topic-based patterns with sampling rates
	Topics map[string]*TopicShardingConfig `yaml:"topics"`

	// Events without sharding keys (Group D)
	NoShardingKeyEvents *NoShardingKeyConfig `yaml:"noShardingKeyEvents,omitempty"`

	// Compiled regex patterns for performance
	compiledPatterns map[string]*CompiledPattern
}

// TopicShardingConfig defines sharding for a topic pattern
type TopicShardingConfig struct {
	// Total number of shards for this topic pattern
	TotalShards uint64 `yaml:"totalShards"`
	// Active shards for this topic pattern
	ActiveShards []uint64 `yaml:"activeShards"`
}

// NoShardingKeyConfig defines behavior for events without sharding keys
type NoShardingKeyConfig struct {
	// Whether to record events without sharding keys (default: true)
	Enabled bool `yaml:"enabled" default:"true"`
}

// CompiledPattern holds a compiled regex pattern and its config
type CompiledPattern struct {
	Pattern *regexp.Regexp
	Config  *TopicShardingConfig
}

// NewUnifiedSharder creates a new unified sharder
func NewUnifiedSharder(config *ShardingConfigV2, enabled bool) (*UnifiedSharder, error) {
	if config == nil {
		config = &ShardingConfigV2{
			Topics: make(map[string]*TopicShardingConfig),
			NoShardingKeyEvents: &NoShardingKeyConfig{
				Enabled: true,
			},
		}
	}

	// Compile patterns
	if err := config.compilePatterns(); err != nil {
		return nil, fmt.Errorf("failed to compile patterns: %w", err)
	}

	// Validate configuration
	if err := config.validate(); err != nil {
		return nil, fmt.Errorf("invalid sharding config: %w", err)
	}

	return &UnifiedSharder{
		config:           config,
		eventCategorizer: NewEventCategorizer(),
		enabled:          enabled,
	}, nil
}

// ShouldProcess determines if an event should be processed based on sharding rules
//
//nolint:gocritic // named returns unnecessary.
func (s *UnifiedSharder) ShouldProcess(eventType xatu.Event_Name, msgID, topic string) (bool, string) {
	if !s.enabled {
		return true, "sharding_disabled"
	}

	eventInfo, exists := s.eventCategorizer.GetEventInfo(eventType)
	if !exists {
		// Unknown event type - default to process
		return true, "unknown_event"
	}

	switch eventInfo.ShardingGroup {
	case GroupA: // Topic + MsgID
		// Try topic-based sharding first
		if topic != "" {
			if config := s.findTopicConfig(topic); config != nil {
				shard := s.calculateShard(msgID, config.TotalShards)
				shouldProcess := s.isShardActive(shard, config.ActiveShards)
				reason := fmt.Sprintf("group_a_topic_match_%t", shouldProcess)

				return shouldProcess, reason
			}
		}

		// Fall back to MsgID sharding with default config
		if msgID != "" {
			shard := s.calculateShard(msgID, DefaultTotalShards)
			// Default to sampling a single shard if no config matches
			shouldProcess := shard == 0

			return shouldProcess, "group_a_no_topic_config"
		}

		return false, "group_a_no_keys"
	case GroupB: // Topic only
		if topic == "" {
			return false, "group_b_no_topic"
		}

		if config := s.findTopicConfig(topic); config != nil {
			// Hash the topic itself for sharding
			topicHash := fmt.Sprintf("topic:%s", topic)
			shard := s.calculateShard(topicHash, config.TotalShards)
			shouldProcess := s.isShardActive(shard, config.ActiveShards)
			reason := fmt.Sprintf("group_b_topic_match_%t", shouldProcess)

			return shouldProcess, reason
		}

		return false, "group_b_no_config"
	case GroupC: // MsgID only
		if msgID == "" {
			return false, "group_c_no_msgid"
		}
		// Always use default sharding for Group C
		shard := s.calculateShard(msgID, DefaultTotalShards)
		// Default to sampling a single shard
		shouldProcess := shard == 0

		return shouldProcess, fmt.Sprintf("group_c_shard_%d", shard)
	case GroupD: // No sharding keys
		if s.config.NoShardingKeyEvents != nil && s.config.NoShardingKeyEvents.Enabled {
			return true, "group_d_enabled"
		}

		return false, "group_d_disabled"
	default:
		return true, "unknown_group"
	}
}

// ShouldProcessBatch determines which events in a batch should be processed
// This is used for RPC meta events where we have multiple events to evaluate
func (s *UnifiedSharder) ShouldProcessBatch(eventType xatu.Event_Name, events []ShardableEvent) []bool {
	if !s.enabled {
		// All events should be processed
		results := make([]bool, len(events))
		for i := range results {
			results[i] = true
		}

		return results
	}

	results := make([]bool, len(events))

	for i, event := range events {
		shouldProcess, _ := s.ShouldProcess(eventType, event.MsgID, event.Topic)
		results[i] = shouldProcess
	}

	return results
}

// ShardableEvent represents an event that can be sharded
type ShardableEvent struct {
	MsgID string
	Topic string
}

// findTopicConfig finds the matching topic configuration for a given topic
func (s *UnifiedSharder) findTopicConfig(topic string) *TopicShardingConfig {
	if s.config.compiledPatterns == nil {
		return nil
	}

	// Find the best matching pattern (could prioritize by specificity or sampling rate)
	var bestMatch *TopicShardingConfig

	var bestSamplingRate float64

	for _, compiled := range s.config.compiledPatterns {
		if compiled.Pattern.MatchString(topic) {
			// Calculate sampling rate
			samplingRate := compiled.Config.GetSamplingRate()

			// Use the configuration with the highest sampling rate
			if bestMatch == nil || samplingRate > bestSamplingRate {
				bestMatch = compiled.Config
				bestSamplingRate = samplingRate
			}
		}
	}

	return bestMatch
}

// calculateShard calculates the shard number for a given key
func (s *UnifiedSharder) calculateShard(key string, totalShards uint64) uint64 {
	return GetShard(key, totalShards)
}

// isShardActive checks if a shard is in the active shards list
func (s *UnifiedSharder) isShardActive(shard uint64, activeShards []uint64) bool {
	for _, activeShard := range activeShards {
		if shard == activeShard {
			return true
		}
	}

	return false
}

// GetShardForKey returns the shard number for a given key (for testing/debugging)
func (s *UnifiedSharder) GetShardForKey(key string, totalShards uint64) uint64 {
	return s.calculateShard(key, totalShards)
}

// compilePatterns compiles all regex patterns in the configuration
func (c *ShardingConfigV2) compilePatterns() error {
	c.compiledPatterns = make(map[string]*CompiledPattern)

	for pattern, config := range c.Topics {
		compiled, err := regexp.Compile(pattern)
		if err != nil {
			return fmt.Errorf("invalid regex pattern '%s': %w", pattern, err)
		}

		c.compiledPatterns[pattern] = &CompiledPattern{
			Pattern: compiled,
			Config:  config,
		}
	}

	return nil
}

// validate validates the sharding configuration
func (c *ShardingConfigV2) validate() error {
	// Validate topic configurations
	for pattern, config := range c.Topics {
		if err := config.validate(pattern); err != nil {
			return err
		}
	}

	return nil
}

// validate validates a topic sharding configuration
func (t *TopicShardingConfig) validate(pattern string) error {
	// Validate totalShards
	if t.TotalShards == 0 {
		return fmt.Errorf("totalShards must be greater than 0 for pattern '%s'", pattern)
	}

	if len(t.ActiveShards) == 0 {
		return fmt.Errorf("activeShards cannot be empty for pattern '%s'", pattern)
	}

	// Validate all shards are within range
	for _, shard := range t.ActiveShards {
		if shard >= t.TotalShards {
			return fmt.Errorf("active shard %d is out of range (0-%d) for pattern '%s'", shard, t.TotalShards-1, pattern)
		}
	}

	return nil
}

// GetSamplingRate returns the sampling rate for a topic pattern
func (t *TopicShardingConfig) GetSamplingRate() float64 {
	return float64(len(t.ActiveShards)) / float64(t.TotalShards)
}

// IsFirehose returns true if all shards are active
func (t *TopicShardingConfig) IsFirehose() bool {
	return len(t.ActiveShards) == int(t.TotalShards) //nolint:gosec // conversion fine.
}

// UnmarshalYAML implements custom YAML unmarshaling to support range syntax
func (t *TopicShardingConfig) UnmarshalYAML(node *yaml.Node) error {
	// Define a temporary struct to unmarshal the raw values
	type rawConfig struct {
		TotalShards  uint64        `yaml:"totalShards"`
		ActiveShards []interface{} `yaml:"activeShards"`
	}

	var raw rawConfig
	if err := node.Decode(&raw); err != nil {
		return err
	}

	t.TotalShards = raw.TotalShards

	// Process activeShards which can contain numbers or ranges
	var result []uint64

	seen := make(map[uint64]bool)

	for _, item := range raw.ActiveShards {
		switch v := item.(type) {
		case int:
			shard := uint64(v) //nolint:gosec // conversion fine.
			if !seen[shard] {
				result = append(result, shard)
				seen[shard] = true
			}
		case float64:
			// YAML might parse numbers as float64
			shard := uint64(v)
			if !seen[shard] {
				result = append(result, shard)
				seen[shard] = true
			}
		case string:
			// Handle range syntax like "0-25"
			if strings.Contains(v, "-") {
				parts := strings.Split(v, "-")
				if len(parts) != 2 {
					return fmt.Errorf("invalid range format '%s', expected 'start-end'", v)
				}

				start, err := strconv.ParseUint(strings.TrimSpace(parts[0]), 10, 64)
				if err != nil {
					return fmt.Errorf("invalid range start '%s': %w", parts[0], err)
				}

				end, err := strconv.ParseUint(strings.TrimSpace(parts[1]), 10, 64)
				if err != nil {
					return fmt.Errorf("invalid range end '%s': %w", parts[1], err)
				}

				if start > end {
					return fmt.Errorf("invalid range '%s': start (%d) is greater than end (%d)", v, start, end)
				}

				// Add all numbers in the range
				for i := start; i <= end; i++ {
					if !seen[i] {
						result = append(result, i)
						seen[i] = true
					}
				}
			} else {
				// Handle single number as string
				shard, err := strconv.ParseUint(v, 10, 64)
				if err != nil {
					return fmt.Errorf("invalid shard number '%s': %w", v, err)
				}

				if !seen[shard] {
					result = append(result, shard)
					seen[shard] = true
				}
			}
		default:
			return fmt.Errorf("unsupported type for active shard: %T (value: %v)", v, v)
		}
	}

	// Sort the result for consistency
	sort.Slice(result, func(i, j int) bool {
		return result[i] < result[j]
	})

	t.ActiveShards = result

	return nil
}

// LogSummary returns a human-readable summary of the sharding configuration
func (c *ShardingConfigV2) LogSummary() string {
	if len(c.Topics) == 0 {
		return "Sharding disabled (no topic configurations)"
	}

	summary := fmt.Sprintf("Sharding enabled with %d topic patterns:", len(c.Topics))

	for pattern, config := range c.Topics {
		if config.IsFirehose() {
			summary += fmt.Sprintf("\n  - Pattern '%s': FIREHOSE (all %d shards active)", pattern, config.TotalShards)
		} else {
			samplingRate := config.GetSamplingRate() * 100.0
			summary += fmt.Sprintf("\n  - Pattern '%s': %d/%d shards active (%.1f%%)",
				pattern, len(config.ActiveShards), config.TotalShards, samplingRate)
		}
	}

	if c.NoShardingKeyEvents != nil && c.NoShardingKeyEvents.Enabled {
		summary += "\n  - Events without sharding keys: ENABLED"
	} else {
		summary += "\n  - Events without sharding keys: DISABLED"
	}

	return summary
}
