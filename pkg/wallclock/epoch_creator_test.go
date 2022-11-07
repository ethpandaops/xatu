package wallclock

import (
	"testing"
	"time"
)

func mustParseTime(format, t string) time.Time {
	tm, err := time.Parse(format, t)
	if err != nil {
		panic(err)
	}

	return tm
}

func TestEpochCreator(t *testing.T) {
	genesisTime, _ := time.Parse(time.RFC3339, "2020-01-01T12:00:23Z")
	epochCreator := NewDefaultEpochCreator(genesisTime, time.Second*12, 32)

	tests := []struct {
		name  string
		epoch uint64
		want  TimeWindow
	}{
		{
			name:  "Test 1",
			epoch: 0,
			want: TimeWindow{
				start: mustParseTime(time.RFC3339, "2020-01-01T12:00:23Z"),
				end:   mustParseTime(time.RFC3339, "2020-01-01T12:06:47Z"),
			},
		},
		{
			name:  "Test 2",
			epoch: 1,
			want: TimeWindow{
				start: mustParseTime(time.RFC3339, "2020-01-01T12:06:47Z"),
				end:   mustParseTime(time.RFC3339, "2020-01-01T12:13:11Z"),
			},
		},
		{
			name:  "Test 3",
			epoch: 2,
			want: TimeWindow{
				start: mustParseTime(time.RFC3339, "2020-01-01T12:13:11Z"),
				end:   mustParseTime(time.RFC3339, "2020-01-01T12:19:35Z"),
			},
		},
		{
			name:  "Test 4",
			epoch: 3,
			want: TimeWindow{
				start: mustParseTime(time.RFC3339, "2020-01-01T12:19:35Z"),
				end:   mustParseTime(time.RFC3339, "2020-01-01T12:25:59Z"),
			},
		},
		{
			name:  "Test 5",
			epoch: 4,
			want: TimeWindow{
				start: mustParseTime(time.RFC3339, "2020-01-01T12:25:59Z"),
				end:   mustParseTime(time.RFC3339, "2020-01-01T12:32:23Z"),
			},
		},
		{
			name:  "Test 6",
			epoch: 100,
			want: TimeWindow{
				start: mustParseTime(time.RFC3339, "2020-01-01T22:40:23Z"),
				end:   mustParseTime(time.RFC3339, "2020-01-01T22:46:47Z"),
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := epochCreator.FromNumber(test.epoch)
			if got.window.start != test.want.start {
				t.Errorf("incorrect start: got %v, want %v", got.window.start.Format(time.RFC3339), test.want.start.Format(time.RFC3339))
			}

			if got.window.end != test.want.end {
				t.Errorf("incorrect end: got %v, want %v", got.window.end.Format(time.RFC3339), test.want.end.Format(time.RFC3339))
			}
		})
	}
}
