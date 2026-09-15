package ethereum

import (
	"testing"
	"time"

	"github.com/creasty/defaults"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSyncWatchdogObserve(t *testing.T) {
	t.Parallel()

	start := time.Date(2026, 7, 21, 16, 36, 0, 0, time.UTC)

	type step struct {
		at      time.Duration
		healthy bool
		want    syncVerdict
	}

	tests := []struct {
		name         string
		logEvery     time.Duration
		restartAfter time.Duration
		steps        []step
	}{
		{
			name:         "healthy checks never log or exit",
			logEvery:     5 * time.Minute,
			restartAfter: time.Hour,
			steps: []step{
				{at: 0, healthy: true},
				{at: 2 * time.Hour, healthy: true},
			},
		},
		{
			name:         "logs once per interval and exits at the limit",
			logEvery:     5 * time.Minute,
			restartAfter: 15 * time.Minute,
			steps: []step{
				{at: 0, healthy: false},
				{at: 4 * time.Minute, healthy: false, want: syncVerdict{unhealthyFor: 4 * time.Minute}},
				{at: 5 * time.Minute, healthy: false, want: syncVerdict{unhealthyFor: 5 * time.Minute, shouldLog: true}},
				{at: 7 * time.Minute, healthy: false, want: syncVerdict{unhealthyFor: 7 * time.Minute}},
				{at: 10 * time.Minute, healthy: false, want: syncVerdict{unhealthyFor: 10 * time.Minute, shouldLog: true}},
				{at: 15 * time.Minute, healthy: false, want: syncVerdict{unhealthyFor: 15 * time.Minute, shouldLog: true, shouldExit: true}},
			},
		},
		{
			name:         "recovery resets the clock",
			logEvery:     5 * time.Minute,
			restartAfter: 10 * time.Minute,
			steps: []step{
				{at: 0, healthy: false},
				{at: 6 * time.Minute, healthy: false, want: syncVerdict{unhealthyFor: 6 * time.Minute, shouldLog: true}},
				{at: 7 * time.Minute, healthy: true},
				{at: 8 * time.Minute, healthy: false},
				{at: 12 * time.Minute, healthy: false, want: syncVerdict{unhealthyFor: 4 * time.Minute}},
			},
		},
		{
			name:         "zero restartAfter never exits",
			logEvery:     time.Minute,
			restartAfter: 0,
			steps: []step{
				{at: 0, healthy: false},
				{at: 48 * time.Hour, healthy: false, want: syncVerdict{unhealthyFor: 48 * time.Hour, shouldLog: true}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			watchdog := newSyncWatchdog(tt.logEvery, tt.restartAfter)

			for _, s := range tt.steps {
				got := watchdog.observe(start.Add(s.at), s.healthy)
				assert.Equal(t, s.want, got, "step at %s", s.at)
			}
		})
	}
}

func TestBeaconConfigDefaultUnhealthyRestartAfter(t *testing.T) {
	t.Parallel()

	cfg := &BeaconConfig{}
	require.NoError(t, defaults.Set(cfg))

	assert.Equal(t, time.Hour, cfg.UnhealthyRestartAfter.Duration)
}
