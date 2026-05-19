package clickhouse

import (
	"testing"

	"github.com/ClickHouse/ch-go/proto"
	"github.com/stretchr/testify/assert"

	"github.com/ethpandaops/xatu/pkg/clickhouse/route"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

// colsStubBatch is a tiny route.ColumnarBatch that reports a fixed column
// list via Input(). All other methods are unused by the validation
// helpers under test.
type colsStubBatch struct {
	cols []string
}

func (b *colsStubBatch) FlattenTo(_ *xatu.DecoratedEvent) error { return nil }
func (b *colsStubBatch) Reset()                                 {}
func (b *colsStubBatch) Rows() int                              { return 0 }

func (b *colsStubBatch) Input() proto.Input {
	out := make(proto.Input, len(b.cols))
	for i, name := range b.cols {
		out[i] = proto.InputColumn{Name: name}
	}

	return out
}

func factoryWithCols(cols ...string) func() route.ColumnarBatch {
	return func() route.ColumnarBatch {
		return &colsStubBatch{cols: cols}
	}
}

func TestBuildExpectedColumns(t *testing.T) {
	t.Run("applies table suffix", func(t *testing.T) {
		factories := map[string]func() route.ColumnarBatch{
			"canonical_beacon_block": factoryWithCols("slot", "block_root", "meta_network_name"),
		}

		got := buildExpectedColumns(factories, "_local")

		assert.Contains(t, got, "canonical_beacon_block_local")
		assert.NotContains(t, got, "canonical_beacon_block")

		cols := got["canonical_beacon_block_local"]
		assert.Contains(t, cols, "slot")
		assert.Contains(t, cols, "block_root")
		assert.Contains(t, cols, "meta_network_name")
		assert.Len(t, cols, 3)
	})

	t.Run("empty suffix is the no-op case", func(t *testing.T) {
		factories := map[string]func() route.ColumnarBatch{
			"canonical_beacon_block": factoryWithCols("slot"),
		}

		got := buildExpectedColumns(factories, "")

		assert.Contains(t, got, "canonical_beacon_block")
	})

	t.Run("nil factories returns empty", func(t *testing.T) {
		got := buildExpectedColumns(nil, "")
		assert.Empty(t, got)
	})
}

func TestFindMissingColumns(t *testing.T) {
	tests := []struct {
		name     string
		expected map[string]map[string]struct{}
		actual   map[string]map[string]struct{}
		want     []columnMismatch
	}{
		{
			name: "all columns exist",
			expected: map[string]map[string]struct{}{
				"canonical_beacon_block": {
					"slot": {}, "block_root": {}, "meta_network_name": {},
				},
			},
			actual: map[string]map[string]struct{}{
				"canonical_beacon_block": {
					"slot": {}, "block_root": {}, "meta_network_name": {}, "extra_col": {},
				},
			},
			want: nil,
		},
		{
			name: "extra columns in target table are NOT a problem",
			expected: map[string]map[string]struct{}{
				"canonical_beacon_block": {
					"slot": {},
				},
			},
			actual: map[string]map[string]struct{}{
				"canonical_beacon_block": {
					"slot":             {},
					"meta_client_geo":  {}, // extra in CH, NOT in route → fine
					"meta_client_impl": {},
				},
			},
			want: nil,
		},
		{
			name: "route columns missing from target IS a problem",
			expected: map[string]map[string]struct{}{
				"canonical_beacon_block": {
					"slot": {}, "meta_consensus_version": {}, "meta_network_name": {},
				},
			},
			actual: map[string]map[string]struct{}{
				"canonical_beacon_block": {
					"slot": {}, "meta_network_name": {},
				},
			},
			want: []columnMismatch{
				{
					table:   "canonical_beacon_block",
					missing: []string{"meta_consensus_version"},
				},
			},
		},
		{
			name: "multiple missing columns are sorted",
			expected: map[string]map[string]struct{}{
				"canonical_beacon_block": {
					"z_col": {}, "a_col": {}, "m_col": {}, "slot": {},
				},
			},
			actual: map[string]map[string]struct{}{
				"canonical_beacon_block": {
					"slot": {},
				},
			},
			want: []columnMismatch{
				{
					table:   "canonical_beacon_block",
					missing: []string{"a_col", "m_col", "z_col"},
				},
			},
		},
		{
			name: "tables not in actual are skipped (ValidateTables surfaces those)",
			expected: map[string]map[string]struct{}{
				"missing_table": {"any": {}},
			},
			actual: map[string]map[string]struct{}{},
			want:   nil,
		},
		{
			name: "results sorted by table name",
			expected: map[string]map[string]struct{}{
				"z_table": {"col_z": {}},
				"a_table": {"col_a": {}},
				"m_table": {"col_m": {}},
			},
			actual: map[string]map[string]struct{}{
				"z_table": {},
				"a_table": {},
				"m_table": {},
			},
			want: []columnMismatch{
				{table: "a_table", missing: []string{"col_a"}},
				{table: "m_table", missing: []string{"col_m"}},
				{table: "z_table", missing: []string{"col_z"}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := findMissingColumns(tt.expected, tt.actual)
			assert.Equal(t, tt.want, got)
		})
	}
}
