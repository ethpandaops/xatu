package cannon

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ethpandaops/xatu/pkg/output"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeSink is a minimal output.Sink suitable for benchmarking and
// correctness-testing the fan-out logic without any I/O.
type fakeSink struct {
	name      string
	calls     atomic.Int64
	delay     time.Duration
	returnErr error
}

func (s *fakeSink) Start(_ context.Context) error { return nil }
func (s *fakeSink) Stop(_ context.Context) error  { return nil }
func (s *fakeSink) Type() string                  { return "fake" }
func (s *fakeSink) Name() string                  { return s.name }
func (s *fakeSink) HandleNewDecoratedEvent(_ context.Context, _ *xatu.DecoratedEvent) error {
	return nil
}

func (s *fakeSink) HandleNewDecoratedEvents(_ context.Context, _ []*xatu.DecoratedEvent) error {
	s.calls.Add(1)

	if s.delay > 0 {
		time.Sleep(s.delay)
	}

	return s.returnErr
}

func newFakeSinks(n int) []*fakeSink {
	sinks := make([]*fakeSink, n)
	for i := range sinks {
		sinks[i] = &fakeSink{name: "fake"}
	}

	return sinks
}

func asOutputSinks(fakes []*fakeSink) []output.Sink {
	sinks := make([]output.Sink, len(fakes))
	for i, f := range fakes {
		sinks[i] = f
	}

	return sinks
}

func TestFanOutToSinks_AllSinksCalledOnSuccess(t *testing.T) {
	fakes := newFakeSinks(3)

	err := fanOutToSinks(context.Background(), asOutputSinks(fakes), nil)
	require.NoError(t, err)

	for _, s := range fakes {
		assert.Equal(t, int64(1), s.calls.Load())
	}
}

func TestFanOutToSinks_AllSinksCalledOnFailure(t *testing.T) {
	// One failing sink must NOT short-circuit its siblings — every sink is
	// attempted on every call so dual-write paths each get their chance.
	failErr := errors.New("boom")

	fakes := []*fakeSink{
		{name: "ok-1"},
		{name: "fail", returnErr: failErr},
		{name: "ok-2"},
	}

	err := fanOutToSinks(context.Background(), asOutputSinks(fakes), nil)
	require.Error(t, err)
	assert.ErrorIs(t, err, failErr,
		"joined error must wrap the underlying sink error so the deriver can inspect it")

	for _, s := range fakes {
		assert.Equal(t, int64(1), s.calls.Load(),
			"every sink including siblings of the failing one must be invoked exactly once")
	}
}

func TestFanOutToSinks_JoinsAllErrors(t *testing.T) {
	err1 := errors.New("first")
	err2 := errors.New("second")

	sinks := []output.Sink{
		&fakeSink{name: "fail-1", returnErr: err1},
		&fakeSink{name: "fail-2", returnErr: err2},
	}

	err := fanOutToSinks(context.Background(), sinks, nil)
	require.Error(t, err)
	assert.ErrorIs(t, err, err1)
	assert.ErrorIs(t, err, err2)
}

func TestFanOutToSinks_ConcurrentLatencyEqualsMaxNotSum(t *testing.T) {
	// Two sinks each sleeping 50ms. Sequential would be ~100ms; concurrent
	// fan-out should land closer to 50ms.
	const delay = 50 * time.Millisecond

	sinks := []output.Sink{
		&fakeSink{name: "slow-1", delay: delay},
		&fakeSink{name: "slow-2", delay: delay},
	}

	start := time.Now()
	err := fanOutToSinks(context.Background(), sinks, nil)
	elapsed := time.Since(start)

	require.NoError(t, err)
	assert.Less(t, elapsed, delay+25*time.Millisecond,
		"concurrent fan-out should pay the slowest sink's latency, not the sum")
}

func TestFanOutToSinks_NoSinksIsNoop(t *testing.T) {
	require.NoError(t, fanOutToSinks(context.Background(), nil, nil))
	require.NoError(t, fanOutToSinks(context.Background(), []output.Sink{}, nil))
}

// BenchmarkFanOutToSinks measures the per-call overhead of the fan-out
// path with N noop sinks. The numbers we care about:
//
//   - Single-sink case: should be effectively zero overhead (we hit the
//     fast-path that skips wg/goroutine machinery entirely).
//   - 2/5/10 sinks: a baseline for the goroutine-spawn cost we can point
//     at if "thrash" comes up again. With ~200 ns per goroutine spawn on
//     modern Go, 10 sinks = ~2 µs/call. At cannon's call rate of a few
//     callbacks per second, this is rounding error.
func BenchmarkFanOutToSinks(b *testing.B) {
	for _, n := range []int{1, 2, 5, 10} {
		sinks := asOutputSinks(newFakeSinks(n))
		ctx := context.Background()

		b.Run(name(n), func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				if err := fanOutToSinks(ctx, sinks, nil); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func name(n int) string {
	switch n {
	case 1:
		return "1_sink"
	case 2:
		return "2_sinks"
	case 5:
		return "5_sinks"
	case 10:
		return "10_sinks"
	}

	return "n_sinks"
}
