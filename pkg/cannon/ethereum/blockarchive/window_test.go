package blockarchive

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPlanWindow(t *testing.T) {
	// Era 2 is slots 16..31 with blocks at 17, 20, 22 and 27.
	file := encodeEra(t, 2, testBlocksAt(t, 17, 20, 22, 27), eraOptions{nimbusMissed: true})
	idx, err := eraIndexOf(t, file, 2)
	require.NoError(t, err)

	tests := map[string]struct {
		first, last uint64
		want        windowPlan
		ok          bool
	}{
		"between blocks":       {20, 23, windowPlan{readFirst: 17, readLast: 27, lo: 20, hi: 23, anchor: 27, succ: true}, true},
		"era start":            {16, 19, windowPlan{readFirst: 17, readLast: 20, lo: 17, hi: 19, anchor: 20, succ: true}, true},
		"era end":              {24, 31, windowPlan{readFirst: 22, readLast: 27, lo: 24, hi: 27, anchor: 27}, true},
		"whole era":            {16, 31, windowPlan{readFirst: 17, readLast: 27, lo: 17, hi: 27, anchor: 27}, true},
		"empty between blocks": {23, 26, windowPlan{readFirst: 22, readLast: 27, lo: 23, hi: 26, anchor: 27, succ: true}, true},
		"empty at era end":     {28, 31, windowPlan{}, false},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			p, ok := planWindow(idx.slots, tc.first, tc.last)
			assert.Equal(t, tc.ok, ok)
			assert.Equal(t, tc.want, p)
		})
	}
}

func TestSlotSet(t *testing.T) {
	s := newSlotSet(100, 300)
	for _, slot := range []uint64{100, 163, 164, 300, 99, 301} {
		s.add(slot)
	}

	for slot, want := range map[uint64]bool{100: true, 163: true, 164: true, 300: true, 99: false, 301: false, 101: false, 299: false} {
		assert.Equal(t, want, s.has(slot), "slot %d", slot)
	}
}
