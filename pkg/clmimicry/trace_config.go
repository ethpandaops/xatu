package clmimicry

import (
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// TracesConfig represents the new trace-based configuration.
type TracesConfig struct {
	// Whether or not trace config is globally enabled.
	Enabled bool `yaml:"enabled" default:"false"`
	// Topics allows for per-topic configuration.
	Topics map[string]TopicConfig `yaml:"topics"`
	// Compiled regex patterns.
	compiledPatterns map[string]*regexp.Regexp
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
				"\n  - Pattern '%s': FIREHOSE (all %d shards active)",
				pattern,
				config.TotalShards,
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
				"\n  - Pattern '%s': %d/%d shards active (%.1f%%) [%s]",
				pattern,
				len(config.ActiveShards),
				config.TotalShards,
				activePercentage,
				shardsDisplay,
			)
		}
	}

	return summary
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
