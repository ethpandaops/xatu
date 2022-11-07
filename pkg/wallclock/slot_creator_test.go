package wallclock

import (
	"testing"
	"time"
)

func TestSlotCreator(t *testing.T) {
	genesisTime, _ := time.Parse(time.RFC3339, "2020-01-01T12:00:23Z")
	slotCreator := NewDefaultSlotCreator(genesisTime, time.Second*12)

	tests := []struct {
		name string
		slot uint64
		want TimeWindow
	}{
		{
			name: "Test 1",
			slot: 0,
			want: TimeWindow{
				start: genesisTime,
				end:   mustParseTime(time.RFC3339, "2020-01-01T12:00:35Z"),
			},
		},
		{
			name: "Test 2",
			slot: 100,
			want: TimeWindow{
				start: mustParseTime(time.RFC3339, "2020-01-01T12:20:23Z"),
				end:   mustParseTime(time.RFC3339, "2020-01-01T12:20:35Z"),
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := slotCreator.FromNumber(test.slot)
			if got.window.start != test.want.start {
				t.Errorf("incorrect start: got %v, want %v", got.window.start.Format(time.RFC3339), test.want.start.Format(time.RFC3339))
			}

			if got.window.end != test.want.end {
				t.Errorf("incorrect end: got %v, want %v", got.window.end.Format(time.RFC3339), test.want.end.Format(time.RFC3339))
			}
		})
	}
}
