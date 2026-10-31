package v1

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"

	"github.com/ethpandaops/go-eth2-client/spec/phase0"
	"golang.org/x/sync/errgroup"

	"github.com/ethpandaops/xatu/pkg/cannon/iterator"
)

// phase0BatchRunner derives one iterator step of phase0 epochs with up to workers beacon states and epoch writes in
// flight. Epoch E's rows need the rewards read from the states at the last slots of E-1, E and E+1, so a step of N
// contiguous epochs loads N+2 states, and the states at its ends are kept for the next step in either direction.
type phase0BatchRunner struct {
	workers int
	altair  phase0.Epoch
	// load computes the rewards from the state at the last slot of stateEpoch.
	load func(ctx context.Context, stateEpoch phase0.Epoch) (*phase0EpochRewards, error)
	// emit derives and writes one epoch's rows. previous is nil for genesis, next for the last phase0 epoch.
	emit func(ctx context.Context, epoch phase0.Epoch, previous, current, next *phase0EpochRewards) error

	cache map[phase0.Epoch]*phase0EpochRewards
}

func (r *phase0BatchRunner) run(ctx context.Context, epochs []phase0.Epoch) error {
	if len(epochs) == 0 {
		return errors.New("no epochs to derive")
	}

	lo, hi := slices.Min(epochs), slices.Max(epochs)
	if hi >= r.altair {
		return fmt.Errorf("epoch %d is not before altair (epoch %d)", hi, r.altair)
	}

	states := map[phase0.Epoch]*phase0EpochRewards{}
	missing := []phase0.Epoch{}

	for _, stateEpoch := range r.statesFor(epochs) {
		if cached, ok := r.cache[stateEpoch]; ok {
			states[stateEpoch] = cached
		} else {
			missing = append(missing, stateEpoch)
		}
	}

	var mu sync.Mutex

	loads, loadCtx := errgroup.WithContext(ctx)
	loads.SetLimit(max(1, r.workers))

	for _, stateEpoch := range missing {
		loads.Go(func() error {
			rewards, err := r.load(loadCtx, stateEpoch)
			if err != nil {
				return err
			}

			mu.Lock()
			states[stateEpoch] = rewards
			mu.Unlock()

			return nil
		})
	}

	if err := loads.Wait(); err != nil {
		return err
	}

	r.cache = map[phase0.Epoch]*phase0EpochRewards{}

	for _, keep := range []phase0.Epoch{lo - 1, lo, hi, hi + 1} {
		if rewards, ok := states[keep]; ok {
			r.cache[keep] = rewards
		}
	}

	emits, emitCtx := errgroup.WithContext(ctx)
	emits.SetLimit(max(1, r.workers))

	for _, epoch := range epochs {
		var previous, next *phase0EpochRewards

		if epoch > 0 {
			previous = states[epoch-1]
		}

		// The transition ending the last phase0 epoch upgrades to Altair, whose rules reward that epoch's attestations.
		if epoch+1 < r.altair {
			next = states[epoch+1]
		}

		emits.Go(func() error {
			if err := r.emit(emitCtx, epoch, previous, states[epoch], next); err != nil {
				return &iterator.EpochError{Epoch: epoch, Err: err}
			}

			return nil
		})
	}

	return emits.Wait()
}

// statesFor returns the state epochs the epochs' rows are read from, in ascending order.
func (r *phase0BatchRunner) statesFor(epochs []phase0.Epoch) []phase0.Epoch {
	set := map[phase0.Epoch]bool{}

	for _, epoch := range epochs {
		if epoch > 0 {
			set[epoch-1] = true
		}

		set[epoch] = true

		if epoch+1 < r.altair {
			set[epoch+1] = true
		}
	}

	out := make([]phase0.Epoch, 0, len(set))
	for epoch := range set {
		out = append(out, epoch)
	}

	slices.Sort(out)

	return out
}
