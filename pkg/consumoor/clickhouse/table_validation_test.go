package clickhouse

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFindMissingTables(t *testing.T) {
	tests := []struct {
		name     string
		expected []string
		existing map[string]struct{}
		want     []string
	}{
		{
			name:     "no tables expected",
			expected: nil,
			existing: map[string]struct{}{"foo": {}},
			want:     nil,
		},
		{
			name:     "all tables exist",
			expected: []string{"beacon_api_eth_v1_events_block", "canonical_beacon_block"},
			existing: map[string]struct{}{
				"beacon_api_eth_v1_events_block": {},
				"canonical_beacon_block":         {},
				"other_table":                    {},
			},
			want: nil,
		},
		{
			name:     "some tables missing",
			expected: []string{"table_a", "table_b", "table_c"},
			existing: map[string]struct{}{
				"table_b": {},
			},
			want: []string{"table_a", "table_c"},
		},
		{
			name:     "all tables missing",
			expected: []string{"table_x", "table_y"},
			existing: map[string]struct{}{},
			want:     []string{"table_x", "table_y"},
		},
		{
			name:     "existing set is nil",
			expected: []string{"table_a"},
			existing: nil,
			want:     []string{"table_a"},
		},
		{
			name: "suffix applied tables missing",
			expected: []string{
				"beacon_api_eth_v1_events_block_local",
				"canonical_beacon_block_local",
			},
			existing: map[string]struct{}{
				"beacon_api_eth_v1_events_block": {},
				"canonical_beacon_block":         {},
			},
			want: []string{
				"beacon_api_eth_v1_events_block_local",
				"canonical_beacon_block_local",
			},
		},
		{
			name: "results are sorted",
			expected: []string{
				"z_table",
				"a_table",
				"m_table",
			},
			existing: map[string]struct{}{},
			want:     []string{"a_table", "m_table", "z_table"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := findMissingTables(tt.expected, tt.existing)
			assert.Equal(t, tt.want, got)
		})
	}
}
