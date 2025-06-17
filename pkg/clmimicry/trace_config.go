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
	// Compiled regex patterns for event types.
	compiledPatterns map[string]*regexp.Regexp
	// Compiled topic configurations with gossip topic patterns.
	compiledTopics map[string]*CompiledTopicConfig
}

// CompiledTopicConfig represents a compiled topic configuration with pre-compiled gossip patterns.
type CompiledTopicConfig struct {
	// Original topic configuration.
	Original *TopicConfig
	// Compiled gossip topic patterns for hierarchical matching.
	GossipPatterns map[*regexp.Regexp]*GossipTopicConfig
}

// TopicConfig represents configuration for a specific topic pattern.
// Supports both simple sharding (backward compatible) and hierarchical gossip topic sharding.
type TopicConfig struct {
	// Simple sharding configuration (backward compatible)
	// Total number of shards for this topic.
	TotalShards *uint64 `yaml:"totalShards,omitempty"`
	// List of active shards to process. Supports individual numbers and ranges (e.g., "0-255").
	ActiveShardsRaw *ActiveShardsConfig `yaml:"activeShards,omitempty"`
	// Processed active shards (populated during validation).
	ActiveShards []uint64 `yaml:"-"`
	// Key to use for sharding (MsgID, PeerID, etc).
	ShardingKey string `yaml:"shardingKey,omitempty"`

	// Hierarchical gossip topic configuration
	Topics *TopicsConfig `yaml:"topics,omitempty"`
}

// TopicsConfig represents hierarchical gossip topic-based sharding configuration.
type TopicsConfig struct {
	// GossipTopics maps gossip topic patterns to their specific sharding configuration.
	GossipTopics map[string]GossipTopicConfig `yaml:"gossipTopics,omitempty"`
	// Fallback configuration for gossip topics that don't match any patterns.
	Fallback *GossipTopicConfig `yaml:"fallback,omitempty"`
}

// GossipTopicConfig represents sharding configuration for a specific gossip topic pattern.
type GossipTopicConfig struct {
	// Total number of shards for this gossip topic.
	TotalShards uint64 `yaml:"totalShards"`
	// List of active shards to process. Supports individual numbers and ranges (e.g., "0-255").
	ActiveShardsRaw ActiveShardsConfig `yaml:"activeShards"`
	// Processed active shards (populated during validation).
	ActiveShards []uint64 `yaml:"-"`
	// Key to use for sharding (MsgID, PeerID, etc).
	ShardingKey string `yaml:"shardingKey,omitempty"`
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
	e.compiledTopics = make(map[string]*CompiledTopicConfig)

	for pattern, config := range e.Topics {
		// Compile event type pattern
		compiled, err := regexp.Compile(pattern)
		if err != nil {
			return fmt.Errorf("invalid regex pattern '%s': %w", pattern, err)
		}
		e.compiledPatterns[pattern] = compiled

		// Create compiled topic config
		compiledConfig := CompiledTopicConfig{
			Original:       &config,
			GossipPatterns: make(map[*regexp.Regexp]*GossipTopicConfig),
		}

		// Compile gossip topic patterns if hierarchical config is used
		if config.Topics != nil && config.Topics.GossipTopics != nil {
			for gossipPattern, gossipConfig := range config.Topics.GossipTopics {
				gossipRegex, err := regexp.Compile(gossipPattern)
				if err != nil {
					return fmt.Errorf("invalid gossip topic pattern '%s' in event pattern '%s': %w", gossipPattern, pattern, err)
				}
				// Store a copy of the gossip config
				gossipConfigCopy := gossipConfig
				compiledConfig.GossipPatterns[gossipRegex] = &gossipConfigCopy
			}
		}

		e.compiledTopics[pattern] = &compiledConfig
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
		config := e.Topics[pattern]

		// Check if this is hierarchical or simple configuration
		if config.Topics != nil {
			summary += e.formatHierarchicalConfig(pattern, &config)
		} else {
			summary += e.formatSimpleConfig(pattern, &config)
		}
	}

	return summary
}

// formatSimpleConfig formats a simple (backward compatible) configuration for logging.
func (e *TracesConfig) formatSimpleConfig(pattern string, config *TopicConfig) string {
	if config.TotalShards == nil {
		return fmt.Sprintf("\n  - Pattern '%s': INVALID (no totalShards)", pattern)
	}

	activePercentage := float64(len(config.ActiveShards)) / float64(*config.TotalShards) * 100.0
	isFirehose := len(config.ActiveShards) == int(*config.TotalShards) //nolint:gosec // controlled config, no overflow.

	shardingKey := config.ShardingKey
	if shardingKey == "" {
		shardingKey = "MsgID" // default
	}

	if isFirehose {
		return fmt.Sprintf(
			"\n  - Pattern '%s': FIREHOSE (all %d shards active), sharding on %s",
			pattern,
			*config.TotalShards,
			shardingKey,
		)
	}

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

	return fmt.Sprintf(
		"\n  - Pattern '%s': %d/%d shards active (%.1f%%) [%s], sharding on %s",
		pattern,
		len(config.ActiveShards),
		*config.TotalShards,
		activePercentage,
		shardsDisplay,
		shardingKey,
	)
}

// formatHierarchicalConfig formats a hierarchical configuration for logging.
func (e *TracesConfig) formatHierarchicalConfig(pattern string, config *TopicConfig) string {
	result := fmt.Sprintf("\n  - Pattern '%s': HIERARCHICAL", pattern)

	if len(config.Topics.GossipTopics) > 0 {
		result += "\n    Gossip Topics:"

		// Sort gossip patterns for consistent output
		gossipPatterns := make([]string, 0, len(config.Topics.GossipTopics))

		for gossipPattern := range config.Topics.GossipTopics {
			gossipPatterns = append(gossipPatterns, gossipPattern)
		}

		sort.Strings(gossipPatterns)

		for _, gossipPattern := range gossipPatterns {
			gossipConfig := config.Topics.GossipTopics[gossipPattern]
			result += e.formatGossipTopicConfig(gossipPattern, &gossipConfig, "      ")
		}
	}

	if config.Topics.Fallback != nil {
		result += "\n    Fallback:"
		result += e.formatGossipTopicConfig("*", config.Topics.Fallback, "      ")
	}

	return result
}

// formatGossipTopicConfig formats a single gossip topic configuration for logging.
func (e *TracesConfig) formatGossipTopicConfig(gossipPattern string, config *GossipTopicConfig, indent string) string {
	activePercentage := float64(len(config.ActiveShards)) / float64(config.TotalShards) * 100.0
	isFirehose := len(config.ActiveShards) == int(config.TotalShards) //nolint:gosec // controlled config, no overflow.

	shardingKey := config.ShardingKey
	if shardingKey == "" {
		shardingKey = "MsgID" // default
	}

	if isFirehose {
		return fmt.Sprintf(
			"\n%s- '%s': FIREHOSE (all %d shards active), sharding on %s",
			indent,
			gossipPattern,
			config.TotalShards,
			shardingKey,
		)
	}

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

	return fmt.Sprintf(
		"\n%s- '%s': %d/%d shards active (%.1f%%) [%s], sharding on %s",
		indent,
		gossipPattern,
		len(config.ActiveShards),
		config.TotalShards,
		activePercentage,
		shardsDisplay,
		shardingKey,
	)
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

		// Validate that configuration is either simple or hierarchical, not both
		if err := validateTopicConfig(pattern, &topicConfig); err != nil {
			return err
		}

		// Update the topic config with processed active shards.
		config.Topics[pattern] = topicConfig
	}

	return nil
}

// validateTopicConfig validates a single topic configuration.
func validateTopicConfig(pattern string, config *TopicConfig) error {
	hasSimpleConfig := config.TotalShards != nil || config.ActiveShardsRaw != nil
	hasHierarchicalConfig := config.Topics != nil

	if hasSimpleConfig && hasHierarchicalConfig {
		return fmt.Errorf("cannot use both simple and hierarchical configuration for pattern '%s'", pattern)
	}

	if !hasSimpleConfig && !hasHierarchicalConfig {
		return fmt.Errorf("must specify either simple or hierarchical configuration for pattern '%s'", pattern)
	}

	if hasSimpleConfig {
		return validateSimpleTopicConfig(pattern, config)
	}

	return validateHierarchicalTopicConfig(pattern, config)
}

// validateSimpleTopicConfig validates simple (backward compatible) topic configuration.
func validateSimpleTopicConfig(pattern string, config *TopicConfig) error {
	// Validate total shards.
	if config.TotalShards == nil {
		return fmt.Errorf("totalShards must be specified for simple configuration of pattern '%s'", pattern)
	}

	if *config.TotalShards == 0 {
		return fmt.Errorf("totalShards must be greater than 0 for pattern '%s'", pattern)
	}

	// Process raw active shards (expand ranges).
	if config.ActiveShardsRaw == nil || len(*config.ActiveShardsRaw) == 0 {
		return fmt.Errorf("activeShards cannot be empty for pattern '%s'", pattern)
	}

	activeShards, err := config.ActiveShardsRaw.ToUint64Slice()
	if err != nil {
		return fmt.Errorf("invalid activeShards for pattern '%s': %w", pattern, err)
	}

	if len(activeShards) == 0 {
		return fmt.Errorf("activeShards cannot be empty for pattern '%s'", pattern)
	}

	// Validate that all active shards are within range.
	for _, shard := range activeShards {
		if shard >= *config.TotalShards {
			return fmt.Errorf(
				"active shard %d is out of range (0-%d) for pattern '%s'",
				shard,
				*config.TotalShards-1,
				pattern,
			)
		}
	}

	// Update the config with processed active shards.
	config.ActiveShards = activeShards

	// Validate sharding key.
	return validateShardingKey(config.ShardingKey, pattern)
}

// validateHierarchicalTopicConfig validates hierarchical gossip topic configuration.
func validateHierarchicalTopicConfig(pattern string, config *TopicConfig) error {
	if config.Topics == nil {
		return fmt.Errorf("topics configuration is nil for pattern '%s'", pattern)
	}

	// Must have either gossip topics or fallback (or both)
	if (len(config.Topics.GossipTopics) == 0) && config.Topics.Fallback == nil {
		return fmt.Errorf("hierarchical configuration for pattern '%s' must have either gossipTopics or fallback", pattern)
	}

	// Validate gossip topic patterns and their configurations
	if config.Topics.GossipTopics != nil {
		for gossipPattern, gossipConfig := range config.Topics.GossipTopics {
			// Validate gossip topic regex pattern
			if _, err := regexp.Compile(gossipPattern); err != nil {
				return fmt.Errorf("invalid gossip topic pattern '%s' in event pattern '%s': %w", gossipPattern, pattern, err)
			}

			// Validate gossip topic configuration
			if err := validateGossipTopicConfig(gossipPattern, pattern, &gossipConfig); err != nil {
				return err
			}

			// Update the gossip config with processed active shards
			config.Topics.GossipTopics[gossipPattern] = gossipConfig
		}
	}

	// Validate fallback configuration if present
	if config.Topics.Fallback != nil {
		if err := validateGossipTopicConfig("fallback", pattern, config.Topics.Fallback); err != nil {
			return err
		}
	}

	return nil
}

// validateGossipTopicConfig validates a single gossip topic configuration.
func validateGossipTopicConfig(gossipPattern, eventPattern string, config *GossipTopicConfig) error {
	// Validate total shards.
	if config.TotalShards == 0 {
		return fmt.Errorf("totalShards must be greater than 0 for gossip pattern '%s' in event pattern '%s'", gossipPattern, eventPattern)
	}

	// Process raw active shards (expand ranges).
	if len(config.ActiveShardsRaw) == 0 {
		return fmt.Errorf("activeShards cannot be empty for gossip pattern '%s' in event pattern '%s'", gossipPattern, eventPattern)
	}

	activeShards, err := config.ActiveShardsRaw.ToUint64Slice()
	if err != nil {
		return fmt.Errorf("invalid activeShards for gossip pattern '%s' in event pattern '%s': %w", gossipPattern, eventPattern, err)
	}

	if len(activeShards) == 0 {
		return fmt.Errorf("activeShards cannot be empty for gossip pattern '%s' in event pattern '%s'", gossipPattern, eventPattern)
	}

	// Validate that all active shards are within range.
	for _, shard := range activeShards {
		if shard >= config.TotalShards {
			return fmt.Errorf(
				"active shard %d is out of range (0-%d) for gossip pattern '%s' in event pattern '%s'",
				shard,
				config.TotalShards-1,
				gossipPattern,
				eventPattern,
			)
		}
	}

	// Update the config with processed active shards.
	config.ActiveShards = activeShards

	// Validate sharding key.
	return validateShardingKey(config.ShardingKey, fmt.Sprintf("gossip pattern '%s' in event pattern '%s'", gossipPattern, eventPattern))
}

// validateShardingKey validates a sharding key.
func validateShardingKey(shardingKey, context string) error {
	switch ShardingKeyType(shardingKey) {
	case ShardingKeyTypeMsgID, ShardingKeyTypePeerID, "": // Allow empty sharding key (will default to MsgID)
		// Valid sharding key types
		return nil
	default:
		return fmt.Errorf(
			"invalid sharding key '%s' for %s, valid values are: %s, %s",
			shardingKey,
			context,
			ShardingKeyTypeMsgID,
			ShardingKeyTypePeerID,
		)
	}
}
