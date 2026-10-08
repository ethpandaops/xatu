package event

import (
	"testing"

	hashstructure "github.com/mitchellh/hashstructure/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testFastConfirmationBlock = "0xblock"

// fast_confirmation fires on every run of the algorithm, so repeated runs that
// confirm the same block must hash identically regardless of current_slot.
func TestFastConfirmationDataHashIgnoresCurrentSlot(t *testing.T) {
	first, second := uint64(101), uint64(102)

	a, err := hashstructure.Hash(&FastConfirmationData{Slot: 100, Block: testFastConfirmationBlock, CurrentSlot: &first}, hashstructure.FormatV2, nil)
	require.NoError(t, err)

	b, err := hashstructure.Hash(&FastConfirmationData{Slot: 100, Block: testFastConfirmationBlock, CurrentSlot: &second}, hashstructure.FormatV2, nil)
	require.NoError(t, err)

	c, err := hashstructure.Hash(&FastConfirmationData{Slot: 100, Block: testFastConfirmationBlock}, hashstructure.FormatV2, nil)
	require.NoError(t, err)

	assert.Equal(t, a, b)
	assert.Equal(t, a, c)

	other, err := hashstructure.Hash(&FastConfirmationData{Slot: 101, Block: "0xother", CurrentSlot: &first}, hashstructure.FormatV2, nil)
	require.NoError(t, err)
	assert.NotEqual(t, a, other)
}
