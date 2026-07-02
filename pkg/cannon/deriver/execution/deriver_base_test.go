package execution

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type idxRow struct {
	block uint64
	hash  string
	idx   uint32
}

// TestStampInternalIndex verifies the 1-based cumcount within (block, hash)
// groups, preserving input (parquet) order — matching the legacy
// groupby(block_number, transaction_hash).cumcount()+1.
func TestStampInternalIndex(t *testing.T) {
	rows := []idxRow{
		{block: 100, hash: "0xaa"},
		{block: 100, hash: "0xaa"},
		{block: 100, hash: "0xaa"},
		{block: 100, hash: "0xbb"},
		{block: 101, hash: "0xaa"}, // same hash, different block -> resets
		{block: 100, hash: "0xbb"},
	}

	stampInternalIndex(rows,
		func(r *idxRow) uint64 { return r.block },
		func(r *idxRow) string { return r.hash },
		func(r *idxRow, idx uint32) { r.idx = idx },
	)

	assert.Equal(t, uint32(1), rows[0].idx)
	assert.Equal(t, uint32(2), rows[1].idx)
	assert.Equal(t, uint32(3), rows[2].idx)
	assert.Equal(t, uint32(1), rows[3].idx, "first 0xbb in block 100")
	assert.Equal(t, uint32(1), rows[4].idx, "block 101/0xaa is a distinct group")
	assert.Equal(t, uint32(2), rows[5].idx, "second 0xbb in block 100")
}
