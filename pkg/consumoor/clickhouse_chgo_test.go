package consumoor

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToTimeNumericValuesUseUnixSecondsByDefault(t *testing.T) {
	expected := time.Unix(1_700_000_000, 0).UTC()

	cases := []any{
		int64(1_700_000_000),
		int(1_700_000_000),
		uint64(1_700_000_000),
		float64(1_700_000_000),
		"1700000000",
	}

	for _, value := range cases {
		got, err := toTime(value, "updated_date_time")
		require.NoError(t, err)
		assert.True(t, got.Equal(expected), "value=%T %v", value, value)
	}
}

func TestToTimeEventDateTimeUsesUnixMilliseconds(t *testing.T) {
	expected := time.UnixMilli(1_700_000_000_000).UTC()

	cases := []any{
		int64(1_700_000_000_000),
		int(1_700_000_000_000),
		uint64(1_700_000_000_000),
		float64(1_700_000_000_000),
		"1700000000000",
	}

	for _, value := range cases {
		got, err := toTime(value, "event_date_time")
		require.NoError(t, err)
		assert.True(t, got.Equal(expected), "value=%T %v", value, value)
	}
}

func TestSortedColumnsUsesAllRows(t *testing.T) {
	cols := sortedColumns([]map[string]any{
		{"validator_id": uint64(1)},
		{"balance": uint64(32_000_000_000)},
	})

	assert.Equal(t, []string{"balance", "validator_id"}, cols)
}
