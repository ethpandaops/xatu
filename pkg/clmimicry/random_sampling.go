package clmimicry

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

// RandomSamplingConfig represents random sampling configuration for events
// that were not captured by deterministic sharding.
type RandomSamplingConfig struct {
	// Patterns maps event/topic patterns to their sampling chances.
	// Key format: "EVENT_TYPE*:topic_pattern" or "EVENT_TYPE" (for GroupC events without topics)
	Patterns map[string]*RandomSamplingPatternConfig `yaml:"patterns"`

	// compiledPatterns holds compiled regex patterns for performance
	compiledPatterns map[string]*CompiledRandomPattern
}

// RandomSamplingPatternConfig defines chance-based sampling for a pattern.
type RandomSamplingPatternConfig struct {
	// Chance is the probability as a percentage string (e.g., "2%", "0.5%")
	Chance string `yaml:"chance"`

	// parsedChance is the parsed float64 value (0.0 to 1.0)
	parsedChance float64
}

// CompiledRandomPattern holds a compiled pattern with its config.
type CompiledRandomPattern struct {
	// Pattern is the compiled regex for topic matching (nil for GroupC patterns)
	Pattern *regexp.Regexp

	// Config is the associated sampling configuration
	Config *RandomSamplingPatternConfig

	// EventTypeConstraint specifies which event types this pattern applies to.
	// Empty string means it applies to all events.
	// Can be an exact event name (e.g., "LIBP2P_TRACE_RPC_META_CONTROL_IWANT")
	// or a wildcard pattern (e.g., "LIBP2P_TRACE_RPC_META_*")
	EventTypeConstraint string

	// IsGroupCPattern is true if this is an event-type-only pattern (no topic regex)
	IsGroupCPattern bool
}

// parseChancePercentage parses strings like "2%", "0.5%", "100%" to a float64 in range 0.0-1.0.
func parseChancePercentage(chance string) (float64, error) {
	chance = strings.TrimSpace(chance)

	if !strings.HasSuffix(chance, "%") {
		return 0, fmt.Errorf("chance must end with '%%', got '%s'", chance)
	}

	valueStr := strings.TrimSuffix(chance, "%")

	value, err := strconv.ParseFloat(valueStr, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid percentage value '%s': %w", valueStr, err)
	}

	if value < 0 || value > 100 {
		return 0, fmt.Errorf("percentage must be between 0 and 100, got %.2f", value)
	}

	return value / 100.0, nil
}

// isValidEventTypeName checks if a string looks like an event type name.
// Event types start with uppercase letters and contain underscores.
func isValidEventTypeName(s string) bool {
	if s == "" {
		return false
	}

	// Must start with uppercase letter
	if s[0] < 'A' || s[0] > 'Z' {
		return false
	}

	// Must contain underscore
	return strings.Contains(s, "_")
}

// compilePatterns compiles all regex patterns in the configuration.
func (c *RandomSamplingConfig) compilePatterns() error {
	if c.Patterns == nil {
		c.Patterns = make(map[string]*RandomSamplingPatternConfig)

		return nil
	}

	c.compiledPatterns = make(map[string]*CompiledRandomPattern, len(c.Patterns))

	for pattern, config := range c.Patterns {
		// Parse the chance percentage
		parsedChance, err := parseChancePercentage(config.Chance)
		if err != nil {
			return fmt.Errorf("invalid chance '%s' for pattern '%s': %w",
				config.Chance, pattern, err)
		}

		config.parsedChance = parsedChance

		// Check if this is a GroupC event-type-only pattern (no colon separator).
		// GroupC patterns don't have a colon (e.g., "LIBP2P_TRACE_RPC_META_CONTROL_IWANT")
		if !strings.Contains(pattern, ":") && isValidEventTypeName(pattern) {
			c.compiledPatterns[pattern] = &CompiledRandomPattern{
				Pattern:             nil, // No regex needed for event-type-only match
				Config:              config,
				EventTypeConstraint: pattern,
				IsGroupCPattern:     true,
			}

			continue
		}

		// Parse EVENT_TYPE:topic_pattern format (same as existing sharding)
		var (
			eventTypeConstraint string
			topicPattern        string
		)

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

		c.compiledPatterns[pattern] = &CompiledRandomPattern{
			Pattern:             compiled,
			Config:              config,
			EventTypeConstraint: eventTypeConstraint,
			IsGroupCPattern:     false,
		}
	}

	return nil
}

// validate validates the random sampling configuration.
func (c *RandomSamplingConfig) validate() error {
	for pattern, config := range c.Patterns {
		if config.Chance == "" {
			return fmt.Errorf("chance is required for pattern '%s'", pattern)
		}

		if config.parsedChance < 0 || config.parsedChance > 1.0 {
			return fmt.Errorf("invalid parsed chance %.4f for pattern '%s'",
				config.parsedChance, pattern)
		}
	}

	return nil
}

// findMatchingPattern finds the matching random sampling pattern for a given topic and event type.
// Returns nil if no pattern matches.
func (c *RandomSamplingConfig) findMatchingPattern(
	topic string,
	eventType xatu.Event_Name,
) *RandomSamplingPatternConfig {
	if len(c.compiledPatterns) == 0 {
		return nil
	}

	eventTypeName := eventType.String()

	// For GroupC events (no topic), check event-type-only patterns first
	if topic == "" {
		// Try exact match on event type name
		if compiled, exists := c.compiledPatterns[eventTypeName]; exists && compiled.IsGroupCPattern {
			return compiled.Config
		}

		// Check wildcard event type patterns for GroupC
		for _, compiled := range c.compiledPatterns {
			if !compiled.IsGroupCPattern {
				continue
			}

			// Check wildcard match (e.g., "LIBP2P_TRACE_RPC_META_*")
			if strings.HasSuffix(compiled.EventTypeConstraint, "*") {
				prefix := strings.TrimSuffix(compiled.EventTypeConstraint, "*")
				if strings.HasPrefix(eventTypeName, prefix) {
					return compiled.Config
				}
			}
		}
	}

	// Find the best matching pattern (select highest chance when multiple match)
	var bestMatch *RandomSamplingPatternConfig

	var bestChance float64

	for _, compiled := range c.compiledPatterns {
		if compiled.IsGroupCPattern {
			continue // Already handled above for GroupC
		}

		// Check topic pattern match
		if compiled.Pattern != nil && !compiled.Pattern.MatchString(topic) {
			continue
		}

		// Check event type constraint
		if compiled.EventTypeConstraint != "" {
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

		// Select pattern with highest chance (consistent with existing sharding behavior)
		if bestMatch == nil || compiled.Config.parsedChance > bestChance {
			bestMatch = compiled.Config
			bestChance = compiled.Config.parsedChance
		}
	}

	return bestMatch
}

// GetParsedChance returns the parsed chance value (0.0 to 1.0).
func (c *RandomSamplingPatternConfig) GetParsedChance() float64 {
	return c.parsedChance
}

// LogSummary returns a human-readable summary of the random sampling configuration.
func (c *RandomSamplingConfig) LogSummary() string {
	if len(c.Patterns) == 0 {
		return "Random sampling disabled (no patterns configured)"
	}

	summary := fmt.Sprintf("Random sampling enabled with %d patterns:", len(c.Patterns))

	for pattern, config := range c.Patterns {
		summary += fmt.Sprintf("\n  - Pattern '%s': %s chance", pattern, config.Chance)
	}

	return summary
}
