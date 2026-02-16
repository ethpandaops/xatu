package clmimicry

import (
	"fmt"
	"math/rand/v2"
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
	config                *ShardingConfig
	randomSamplingConfig  *RandomSamplingConfig
	eventCategorizer      *EventCategorizer
	enabled               bool
	randomSamplingEnabled bool
}

// ShardingConfig represents the sharding configuration
type ShardingConfig struct {
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
	// EventTypeConstraint specifies which event types this pattern applies to.
	// Empty string means it applies to all events (backward compatibility).
	// Can be an exact event name (e.g., "LIBP2P_TRACE_GOSSIPSUB_BEACON_ATTESTATION")
	// or a wildcard pattern (e.g., "LIBP2P_TRACE_RPC_META_*")
	EventTypeConstraint string
}

// NewUnifiedSharder creates a new unified sharder
func NewUnifiedSharder(
	config *ShardingConfig,
	randomConfig *RandomSamplingConfig,
	enabled bool,
) (*UnifiedSharder, error) {
	if config == nil {
		config = &ShardingConfig{
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

	// Check if random sampling is enabled
	randomSamplingEnabled := randomConfig != nil && len(randomConfig.Patterns) > 0

	return &UnifiedSharder{
		config:                config,
		randomSamplingConfig:  randomConfig,
		eventCategorizer:      NewEventCategorizer(),
		enabled:               enabled,
		randomSamplingEnabled: randomSamplingEnabled,
	}, nil
}

// ShouldProcess determines if an event should be processed based on sharding rules.
// It first tries deterministic sharding, and if that rejects the event, it tries
// random sampling as a "second chance" mechanism.
//
//nolint:gocritic // named returns unnecessary.
func (s *UnifiedSharder) ShouldProcess(eventType xatu.Event_Name, msgID, topic string) (bool, string) {
	if !s.enabled {
		return true, "sharding_disabled"
	}

	// Try deterministic sharding first
	deterministicResult, deterministicReason := s.deterministicProcess(eventType, msgID, topic)

	// If deterministic sharding accepted, return immediately (no duplicates)
	if deterministicResult {
		return true, deterministicReason
	}

	// Try random sampling as "second chance" if deterministic rejected
	if s.randomSamplingEnabled {
		randomResult, randomReason := s.randomSamplingProcess(eventType, topic)
		if randomResult {
			return true, randomReason
		}
	}

	// Neither deterministic nor random sampling accepted
	return false, deterministicReason
}

// deterministicProcess contains the deterministic sharding logic based on SipHash.
//
//nolint:gocritic // named returns unnecessary.
func (s *UnifiedSharder) deterministicProcess(
	eventType xatu.Event_Name,
	msgID, topic string,
) (bool, string) {
	eventInfo, exists := s.eventCategorizer.GetEventInfo(eventType)
	if !exists {
		// Unknown event type - default to process
		return true, "unknown_event"
	}

	switch eventInfo.ShardingGroup {
	case GroupA: // Topic + MsgID
		// Try topic-based sharding first
		if topic != "" {
			if config := s.findTopicConfig(topic, eventType); config != nil {
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

		if config := s.findTopicConfig(topic, eventType); config != nil {
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

		// For GroupC events, check if there's a configuration matching the event type
		// (since they have no topic, we use empty string for topic)
		if config := s.findTopicConfig("", eventType); config != nil {
			shard := s.calculateShard(msgID, config.TotalShards)
			shouldProcess := s.isShardActive(shard, config.ActiveShards)
			reason := fmt.Sprintf("group_c_configured_%t_shard_%d", shouldProcess, shard)

			return shouldProcess, reason
		}

		// Fall back to default sharding for Group C
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

// randomSamplingProcess checks if random sampling should accept this event.
// Returns true if the event should be processed based on random chance.
//
//nolint:gocritic // named returns unnecessary.
func (s *UnifiedSharder) randomSamplingProcess(
	eventType xatu.Event_Name,
	topic string,
) (bool, string) {
	if s.randomSamplingConfig == nil {
		return false, "random_sampling_not_configured"
	}

	// Find matching random sampling pattern
	config := s.randomSamplingConfig.findMatchingPattern(topic, eventType)
	if config == nil {
		return false, "random_no_pattern_match"
	}

	// Roll the dice - rand.Float64() is thread-safe in math/rand/v2
	//nolint:gosec // G404: crypto strength not needed for sampling
	if rand.Float64() < config.parsedChance {
		return true, fmt.Sprintf("random_sampled_%.2f_pct", config.parsedChance*100)
	}

	return false, fmt.Sprintf("random_not_sampled_%.2f_pct", config.parsedChance*100)
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

// findTopicConfig finds the matching topic configuration for a given topic and event type
func (s *UnifiedSharder) findTopicConfig(topic string, eventType xatu.Event_Name) *TopicShardingConfig {
	if s.config.compiledPatterns == nil {
		return nil
	}

	// Special handling for GroupC events with no topic (e.g., IDONTWANT, IWANT)
	// These events only have MsgID and need to match by event type name directly
	if topic == "" {
		eventTypeName := eventType.String()
		// Try exact match on event type name in the Topics map
		if config, exists := s.config.Topics[eventTypeName]; exists {
			return config
		}
	}

	// Find the best matching pattern (could prioritize by specificity or sampling rate)
	var bestMatch *TopicShardingConfig
	var bestSamplingRate float64

	for _, compiled := range s.config.compiledPatterns {
		// Check if topic matches the pattern
		if !compiled.Pattern.MatchString(topic) {
			continue
		}

		// Check if event type matches the constraint
		if compiled.EventTypeConstraint != "" {
			eventTypeName := eventType.String()

			// Check for wildcard match (e.g., "LIBP2P_TRACE_RPC_META_*")
			if strings.HasSuffix(compiled.EventTypeConstraint, "*") {
				prefix := strings.TrimSuffix(compiled.EventTypeConstraint, "*")
				if !strings.HasPrefix(eventTypeName, prefix) {
					continue
				}
			} else if eventTypeName != compiled.EventTypeConstraint {
				// Exact match required
				continue
			}
		}

		// Calculate sampling rate
		samplingRate := compiled.Config.GetSamplingRate()

		// Use the configuration with the highest sampling rate
		if bestMatch == nil || samplingRate > bestSamplingRate {
			bestMatch = compiled.Config
			bestSamplingRate = samplingRate
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
func (c *ShardingConfig) compilePatterns() error {
	c.compiledPatterns = make(map[string]*CompiledPattern)

	for pattern, config := range c.Topics {
		var (
			eventTypeConstraint string
			topicPattern        string
		)

		// Check if pattern contains event type prefix
		// Event types always start with uppercase letters and contain underscores
		if colonIdx := strings.Index(pattern, ":"); colonIdx != -1 {
			potentialEventType := pattern[:colonIdx]
			// Check if it looks like an event type (starts with uppercase letter and contains underscore)
			if potentialEventType != "" &&
				potentialEventType[0] >= 'A' && potentialEventType[0] <= 'Z' &&
				strings.Contains(potentialEventType, "_") {
				// Extract event type constraint and topic pattern
				eventTypeConstraint = potentialEventType
				topicPattern = pattern[colonIdx+1:]
			} else {
				// Colon is part of the regex pattern, not an event type separator
				eventTypeConstraint = ""
				topicPattern = pattern
			}
		} else {
			// No event type prefix, pattern applies to all events
			eventTypeConstraint = ""
			topicPattern = pattern
		}

		// Compile the topic pattern as regex
		compiled, err := regexp.Compile(topicPattern)
		if err != nil {
			return fmt.Errorf("invalid regex pattern '%s': %w", topicPattern, err)
		}

		c.compiledPatterns[pattern] = &CompiledPattern{
			Pattern:             compiled,
			Config:              config,
			EventTypeConstraint: eventTypeConstraint,
		}
	}

	return nil
}

// validate validates the sharding configuration
func (c *ShardingConfig) validate() error {
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
func (c *ShardingConfig) LogSummary() string {
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
