package v1

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/ethpandaops/go-eth2-client/spec/phase0"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ethpandaops/xatu/pkg/cannon/iterator"
)

type fakePhase0Chain struct {
	mu      sync.Mutex
	loads   []phase0.Epoch
	emitted map[phase0.Epoch][3]phase0.Epoch
	// gate, when set, holds every load until it is closed.
	gate     chan struct{}
	inFlight int
	peak     int
	failEmit phase0.Epoch
}

const noState = phase0.Epoch(1 << 62)

func (f *fakePhase0Chain) load(ctx context.Context, stateEpoch phase0.Epoch) (*phase0EpochRewards, error) {
	f.mu.Lock()
	f.loads = append(f.loads, stateEpoch)
	f.inFlight++
	f.peak = max(f.peak, f.inFlight)
	f.mu.Unlock()

	if f.gate != nil {
		select {
		case <-f.gate:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	f.mu.Lock()
	f.inFlight--
	f.mu.Unlock()

	return &phase0EpochRewards{StateEpoch: stateEpoch}, nil
}

func (f *fakePhase0Chain) emit(_ context.Context, epoch phase0.Epoch, previous, current, next *phase0EpochRewards) error {
	if epoch == f.failEmit {
		return errors.New("write failed")
	}

	epochOf := func(r *phase0EpochRewards) phase0.Epoch {
		if r == nil {
			return noState
		}

		return r.StateEpoch
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	f.emitted[epoch] = [3]phase0.Epoch{epochOf(previous), epochOf(current), epochOf(next)}

	return nil
}

func newFakeRunner(workers int, altair phase0.Epoch) (*phase0BatchRunner, *fakePhase0Chain) {
	chain := &fakePhase0Chain{emitted: map[phase0.Epoch][3]phase0.Epoch{}, failEmit: noState}

	return &phase0BatchRunner{workers: workers, altair: altair, load: chain.load, emit: chain.emit}, chain
}

// TestPhase0BatchRunnerWalksDown runs two consecutive backfill steps: every state is loaded once, each epoch gets the
// states around it, and the second step reuses the two states it shares with the first.
func TestPhase0BatchRunnerWalksDown(t *testing.T) {
	runner, chain := newFakeRunner(3, 100)

	require.NoError(t, runner.run(context.Background(), []phase0.Epoch{10, 9, 8, 7}))
	assert.ElementsMatch(t, []phase0.Epoch{6, 7, 8, 9, 10, 11}, chain.loads)

	chain.loads = nil

	require.NoError(t, runner.run(context.Background(), []phase0.Epoch{6, 5, 4, 3}))
	assert.ElementsMatch(t, []phase0.Epoch{2, 3, 4, 5}, chain.loads)

	for epoch := phase0.Epoch(3); epoch <= 10; epoch++ {
		assert.Equal(t, [3]phase0.Epoch{epoch - 1, epoch, epoch + 1}, chain.emitted[epoch], "epoch %d", epoch)
	}

	assert.LessOrEqual(t, chain.peak, 3)
}

func TestPhase0BatchRunnerBounds(t *testing.T) {
	runner, chain := newFakeRunner(2, 5)

	require.NoError(t, runner.run(context.Background(), []phase0.Epoch{4, 1, 0}))
	assert.ElementsMatch(t, []phase0.Epoch{0, 1, 2, 3, 4}, chain.loads, "no state before genesis or at altair")
	assert.Equal(t, [3]phase0.Epoch{noState, 0, 1}, chain.emitted[0])
	assert.Equal(t, [3]phase0.Epoch{3, 4, noState}, chain.emitted[4], "the last phase0 epoch has no attestation rows")
}

// TestPhase0BatchRunnerLoadsInParallel holds every load until workers of them are in flight at once.
func TestPhase0BatchRunnerLoadsInParallel(t *testing.T) {
	runner, chain := newFakeRunner(3, 100)
	chain.gate = make(chan struct{})

	go func() {
		for {
			chain.mu.Lock()
			inFlight := chain.inFlight
			chain.mu.Unlock()

			if inFlight == 3 {
				close(chain.gate)

				return
			}

			time.Sleep(time.Millisecond)
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	require.NoError(t, runner.run(ctx, []phase0.Epoch{20, 19, 18, 17, 16}))
	assert.Equal(t, 3, chain.peak)
}

func TestPhase0BatchRunnerReportsTheFailedEpoch(t *testing.T) {
	runner, chain := newFakeRunner(4, 100)
	chain.failEmit = 8

	err := runner.run(context.Background(), []phase0.Epoch{10, 9, 8, 7})
	require.Error(t, err)
	assert.Equal(t, phase0.Epoch(8), iterator.FailedEpoch(err, 10))

	require.Error(t, runner.run(context.Background(), []phase0.Epoch{100}), "epochs from altair on are rejected")
}
