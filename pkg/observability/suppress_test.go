package observability

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestSuppressor_FirstCallAllowed(t *testing.T) {
	s := Suppressor{Window: time.Second}

	ok, suppressed := s.Allow(time.Unix(100, 0))
	require.True(t, ok)
	require.Zero(t, suppressed)
}

func TestSuppressor_SuppressesWithinWindow(t *testing.T) {
	start := time.Unix(100, 0)
	s := Suppressor{Window: 10 * time.Second}

	ok, _ := s.Allow(start)
	require.True(t, ok)

	for i := 1; i <= 5; i++ {
		var suppressed uint64

		ok, suppressed = s.Allow(start.Add(time.Duration(i) * time.Second))
		require.False(t, ok, "call %d inside the window must be suppressed", i)
		require.Zero(t, suppressed, "suppressed calls report no count")
	}

	ok, _ = s.Allow(start.Add(10*time.Second - time.Nanosecond))
	require.False(t, ok, "the window boundary is exclusive")

	ok, suppressed := s.Allow(start.Add(10 * time.Second))
	require.True(t, ok, "the first call after the window must be allowed")
	require.Equal(t, uint64(6), suppressed)
}

func TestSuppressor_ResetsCountAfterEmit(t *testing.T) {
	start := time.Unix(100, 0)
	s := Suppressor{Window: time.Second}

	_, _ = s.Allow(start)
	_, _ = s.Allow(start)
	_, _ = s.Allow(start)

	ok, suppressed := s.Allow(start.Add(time.Second))
	require.True(t, ok)
	require.Equal(t, uint64(2), suppressed)

	ok, suppressed = s.Allow(start.Add(2 * time.Second))
	require.True(t, ok)
	require.Zero(t, suppressed, "the count must reset once reported")
}

func TestSuppressor_Concurrent(t *testing.T) {
	const (
		goroutines = 16
		calls      = 1000
	)

	start := time.Unix(100, 0)
	s := Suppressor{Window: time.Second}

	var (
		wg      sync.WaitGroup
		allowed atomic.Uint64
		swept   atomic.Uint64
	)

	for range goroutines {
		wg.Add(1)

		go func() {
			defer wg.Done()

			for range calls {
				if ok, suppressed := s.Allow(start); ok {
					allowed.Add(1)
					swept.Add(suppressed)
				}
			}
		}()
	}

	wg.Wait()

	require.Equal(t, uint64(1), allowed.Load(), "exactly one call per window may pass")

	ok, suppressed := s.Allow(start.Add(time.Second))
	require.True(t, ok)
	require.Equal(t, uint64(goroutines*calls-1), swept.Load()+suppressed,
		"every suppressed call must be counted exactly once")
}

func TestNewSuppressors(t *testing.T) {
	suppressors := NewSuppressors(3, time.Minute)
	require.Len(t, suppressors, 3)

	for i := range suppressors {
		require.Equal(t, time.Minute, suppressors[i].Window)
	}

	now := time.Unix(100, 0)

	ok, _ := suppressors[0].Allow(now)
	require.True(t, ok)

	ok, _ = suppressors[1].Allow(now)
	require.True(t, ok, "entries must key independently")

	ok, _ = suppressors[0].Allow(now)
	require.False(t, ok)
}
