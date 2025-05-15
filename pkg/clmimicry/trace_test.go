package clmimicry

import (
	"fmt"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
)

func TestShouldTraceMessage(t *testing.T) {
	registry := prometheus.NewRegistry()
	prometheus.DefaultRegisterer = registry

	// Find message IDs that map to specific shards for our tests
	msgIDForShard2 := findMsgIDForShard(2, 64)
	msgIDForShard4 := findMsgIDForShard(4, 64)

	tests := []struct {
		name      string
		mimicry   *Mimicry
		msgID     string
		eventType string
		network   string
		expected  bool
	}{
		{
			name: "empty msgID always returns true",
			mimicry: &Mimicry{
				Config: &Config{
					Traces: TracesConfig{
						Enabled: true,
						Topics: map[string]TopicConfig{
							"test_event": {
								TotalShards:  64,
								ActiveShards: []uint64{1, 2, 3},
							},
						},
					},
				},
				metrics: NewMetrics("test1"),
			},
			msgID:     "",
			eventType: "test_event",
			network:   "mainnet",
			expected:  true,
		},
		{
			name: "traces disabled returns true",
			mimicry: &Mimicry{
				Config: &Config{
					Traces: TracesConfig{
						Enabled: false,
					},
				},
				metrics: NewMetrics("test2"),
			},
			msgID:     "0x1234",
			eventType: "test_event",
			network:   "mainnet",
			expected:  true,
		},
		{
			name: "no matching topic returns true",
			mimicry: &Mimicry{
				Config: &Config{
					Traces: TracesConfig{
						Enabled: true,
						Topics: map[string]TopicConfig{
							"other_event": {
								TotalShards:  64,
								ActiveShards: []uint64{1, 2, 3},
							},
						},
					},
				},
				metrics: NewMetrics("test3"),
			},
			msgID:     "0x1234",
			eventType: "test_event",
			network:   "mainnet",
			expected:  true,
		},
		{
			name: "shard in active shards returns true",
			mimicry: &Mimicry{
				Config: &Config{
					Traces: TracesConfig{
						Enabled: true,
						Topics: map[string]TopicConfig{
							"test_event": {
								TotalShards:  64,
								ActiveShards: []uint64{1, 2, 3},
							},
						},
					},
				},
				metrics: NewMetrics("test4"),
			},
			msgID:     msgIDForShard2,
			eventType: "test_event",
			network:   "mainnet",
			expected:  true,
		},
		{
			name: "shard not in active shards returns false",
			mimicry: &Mimicry{
				Config: &Config{
					Traces: TracesConfig{
						Enabled: true,
						Topics: map[string]TopicConfig{
							"test_event": {
								TotalShards:  64,
								ActiveShards: []uint64{1, 2, 3},
							},
						},
					},
				},
				metrics: NewMetrics("test5"),
			},
			msgID:     msgIDForShard4,
			eventType: "test_event",
			network:   "mainnet",
			expected:  false,
		},
		{
			name: "empty network uses unknown",
			mimicry: &Mimicry{
				Config: &Config{
					Traces: TracesConfig{
						Enabled: true,
						Topics: map[string]TopicConfig{
							"test_event": {
								TotalShards:  64,
								ActiveShards: []uint64{1, 2, 3},
							},
						},
					},
				},
				metrics: NewMetrics("test6"),
			},
			msgID:     msgIDForShard2,
			eventType: "test_event",
			network:   "",
			expected:  true,
		},
		{
			name: "regex pattern matching",
			mimicry: &Mimicry{
				Config: &Config{
					Traces: TracesConfig{
						Enabled: true,
						Topics: map[string]TopicConfig{
							"test_.*": {
								TotalShards:  64,
								ActiveShards: []uint64{1, 2, 3},
							},
						},
					},
				},
				metrics: NewMetrics("test7"),
			},
			msgID:     msgIDForShard2,
			eventType: "test_something",
			network:   "mainnet",
			expected:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.mimicry.ShouldTraceMessage(tt.msgID, tt.eventType, tt.network)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestGetShard verifies that the GetShard function consistently maps message IDs to shards.
func TestGetShard(t *testing.T) {
	tests := []struct {
		msgID       string
		totalShards uint64
		expected    uint64
	}{
		{
			msgID:       "0x1234",
			totalShards: 64,
			expected:    GetShard("0x1234", 64),
		},
		{
			msgID:       "0xabcd",
			totalShards: 64,
			expected:    GetShard("0xabcd", 64),
		},
		{
			msgID:       "0x1234",
			totalShards: 128,
			expected:    GetShard("0x1234", 128),
		},
	}

	for _, tt := range tests {
		t.Run(tt.msgID, func(t *testing.T) {
			// Call it multiple times to verify consistency
			for i := 0; i < 5; i++ {
				result := GetShard(tt.msgID, tt.totalShards)
				assert.Equal(t, tt.expected, result)
				assert.Less(t, result, tt.totalShards)
			}
		})
	}
}

// TestIsShardActive tests the IsShardActive function
func TestIsShardActive(t *testing.T) {
	tests := []struct {
		name         string
		shard        uint64
		activeShards []uint64
		expected     bool
	}{
		{
			name:         "shard is active",
			shard:        2,
			activeShards: []uint64{1, 2, 3},
			expected:     true,
		},
		{
			name:         "shard is not active",
			shard:        4,
			activeShards: []uint64{1, 2, 3},
			expected:     false,
		},
		{
			name:         "empty active shards",
			shard:        1,
			activeShards: []uint64{},
			expected:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsShardActive(tt.shard, tt.activeShards)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// findMsgIDForShard finds a message ID that maps to a specific shard.
func findMsgIDForShard(shard, totalShards uint64) string {
	// Start with a base message ID and increment until we find one that maps to the desired shard.
	for i := 0; i < 1000; i++ {
		msgID := fmt.Sprintf("0xtest%d", i)
		if GetShard(msgID, totalShards) == shard {
			return msgID
		}
	}

	return fmt.Sprintf("0xshard%d", shard) // Fallback, though this might not map to the desired shard.
}
