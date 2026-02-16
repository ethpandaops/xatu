package consumoor

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTableWriterGetColumnsUsesAllRowsInBatch(t *testing.T) {
	tw := &tableWriter{}

	cols := tw.getColumns([]map[string]any{
		{"validator_id": uint64(1)},
		{"balance": uint64(32_000_000_000)},
		{"slashed": true},
	})

	assert.Equal(t, []string{"balance", "slashed", "validator_id"}, cols)
}
