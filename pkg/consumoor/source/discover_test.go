package source

import (
	"context"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMatchTopics(t *testing.T) {
	tests := []struct {
		name     string
		patterns []string
		topics   []string
		want     []string
	}{
		{
			name:     "single pattern matches multiple topics",
			patterns: []string{`^xatu\.cannon\.`},
			topics: []string{
				"xatu.cannon.beacon_block",
				"xatu.cannon.beacon_blob_sidecar",
				"xatu.sentry.beacon_head",
			},
			want: []string{
				"xatu.cannon.beacon_blob_sidecar",
				"xatu.cannon.beacon_block",
			},
		},
		{
			name:     "multiple patterns with overlap deduplicates",
			patterns: []string{`^xatu\.cannon\.`, `beacon_block`},
			topics: []string{
				"xatu.cannon.beacon_block",
				"xatu.cannon.beacon_blob_sidecar",
				"xatu.sentry.beacon_block",
			},
			want: []string{
				"xatu.cannon.beacon_blob_sidecar",
				"xatu.cannon.beacon_block",
				"xatu.sentry.beacon_block",
			},
		},
		{
			name:     "pattern matches nothing",
			patterns: []string{`^nonexistent\.`},
			topics: []string{
				"xatu.cannon.beacon_block",
				"xatu.sentry.beacon_head",
			},
			want: []string{},
		},
		{
			name:     "empty patterns match nothing",
			patterns: []string{},
			topics: []string{
				"xatu.cannon.beacon_block",
			},
			want: []string{},
		},
		{
			name:     "empty topics returns empty",
			patterns: []string{`.*`},
			topics:   []string{},
			want:     []string{},
		},
		{
			name:     "results are sorted lexicographically",
			patterns: []string{`.*`},
			topics:   []string{"z_topic", "a_topic", "m_topic"},
			want:     []string{"a_topic", "m_topic", "z_topic"},
		},
		{
			name:     "exact match pattern",
			patterns: []string{`^xatu\.cannon\.beacon_block$`},
			topics: []string{
				"xatu.cannon.beacon_block",
				"xatu.cannon.beacon_block_v2",
			},
			want: []string{"xatu.cannon.beacon_block"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compiled := make([]*regexp.Regexp, 0, len(tt.patterns))
			for _, p := range tt.patterns {
				compiled = append(compiled, regexp.MustCompile(p))
			}

			got := matchTopics(compiled, tt.topics)

			if len(tt.want) == 0 {
				assert.Empty(t, got)
			} else {
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestDiscoverTopics_NilConfig(t *testing.T) {
	_, err := DiscoverTopics(context.TODO(), nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "nil kafka config")
}

func TestDiscoverTopics_InvalidRegex(t *testing.T) {
	cfg := &KafkaConfig{
		Brokers: []string{"localhost:9092"},
		Topics:  []string{"[invalid"},
	}

	_, err := DiscoverTopics(context.TODO(), cfg)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "compiling topic pattern")
}
