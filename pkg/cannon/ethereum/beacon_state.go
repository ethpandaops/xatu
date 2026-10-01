package ethereum

import (
	"context"
	"fmt"
	"io"
	"runtime/debug"
	"time"

	client "github.com/ethpandaops/go-eth2-client"
	"github.com/ethpandaops/go-eth2-client/api"
	"github.com/ethpandaops/go-eth2-client/spec"
	"github.com/ethpandaops/go-eth2-client/spec/gloas"
	"github.com/ethpandaops/go-eth2-client/spec/phase0"
	dynssz "github.com/pk910/dynamic-ssz"
	"github.com/pkg/errors"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/ethpandaops/xatu/pkg/observability"
)

const (
	gloasEpochStateCacheTTL  = 5 * time.Minute
	gloasEpochStateCacheSize = 8
	maxBeaconStateSSZSize    = 1 << 30
	sszContentType           = "application/octet-stream"
)

// GloasEpochState is the slice of a Gloas beacon state the state derivers read, small enough to cache.
type GloasEpochState struct {
	Slot                  phase0.Slot
	LatestBlockHeaderSlot phase0.Slot
	FinalizedEpoch        phase0.Epoch

	// PTCWindow holds the previous, current and next epoch's committees, in that order.
	PTCWindow                 [][]phase0.ValidatorIndex
	Builders                  []*gloas.Builder
	BuilderPendingPayments    []*gloas.BuilderPendingPayment
	BuilderPendingWithdrawals []*gloas.BuilderPendingWithdrawal
	// ExecutionPayloadAvailability is the SSZ bitvector indexed by slot modulo its bit length.
	ExecutionPayloadAvailability []byte
}

func newGloasEpochState(state *gloas.BeaconState) (*GloasEpochState, error) {
	if state.Fork == nil || state.LatestBlockHeader == nil || state.FinalizedCheckpoint == nil {
		return nil, errors.New("beacon state is missing fork, latest block header or finalized checkpoint")
	}

	window := make([][]phase0.ValidatorIndex, len(state.PTCWindow))
	for i, committee := range state.PTCWindow {
		window[i] = append([]phase0.ValidatorIndex(nil), committee...)
	}

	return &GloasEpochState{
		Slot:                         state.Slot,
		LatestBlockHeaderSlot:        state.LatestBlockHeader.Slot,
		FinalizedEpoch:               state.FinalizedCheckpoint.Epoch,
		PTCWindow:                    window,
		Builders:                     state.Builders,
		BuilderPendingPayments:       state.BuilderPendingPayments,
		BuilderPendingWithdrawals:    state.BuilderPendingWithdrawals,
		ExecutionPayloadAvailability: append([]byte(nil), state.ExecutionPayloadAvailability...),
	}, nil
}

// PTC returns slot's committee in order, so index i is bit i of a payload attestation's aggregation bits (spec get_ptc).
func (s *GloasEpochState) PTC(slot phase0.Slot, slotsPerEpoch uint64) ([]phase0.ValidatorIndex, error) {
	if slotsPerEpoch == 0 || len(s.PTCWindow) < 2*int(slotsPerEpoch) || len(s.PTCWindow)%int(slotsPerEpoch) != 0 { //nolint:gosec // slots per epoch is small
		return nil, fmt.Errorf("unexpected ptc window of %d committees for %d slots per epoch", len(s.PTCWindow), slotsPerEpoch)
	}

	epoch := uint64(slot) / slotsPerEpoch
	stateEpoch := uint64(s.Slot) / slotsPerEpoch
	lookahead := uint64(len(s.PTCWindow))/slotsPerEpoch - 2

	var index uint64

	switch {
	case epoch < stateEpoch:
		if epoch+1 != stateEpoch {
			return nil, fmt.Errorf("slot %d is more than one epoch behind state slot %d", slot, s.Slot)
		}

		index = uint64(slot) % slotsPerEpoch
	case epoch <= stateEpoch+lookahead:
		index = (epoch-stateEpoch+1)*slotsPerEpoch + uint64(slot)%slotsPerEpoch
	default:
		return nil, fmt.Errorf("slot %d is beyond the ptc lookahead of state slot %d", slot, s.Slot)
	}

	return s.PTCWindow[index], nil
}

// PayloadAvailable returns the execution_payload_availability bit of slot.
func (s *GloasEpochState) PayloadAvailable(slot phase0.Slot) (bool, error) {
	bits := uint64(len(s.ExecutionPayloadAvailability)) * 8
	if bits == 0 {
		return false, errors.New("state has no execution payload availability bits")
	}

	if slot > s.Slot || uint64(s.Slot-slot) >= bits {
		return false, fmt.Errorf("slot %d is outside the availability window of state slot %d", slot, s.Slot)
	}

	bit := uint64(slot) % bits

	return s.ExecutionPayloadAvailability[bit/8]>>(bit%8)&1 == 1, nil
}

// PayloadAvailabilitySettled reports whether a later block exists, which is what applies a slot's payload and sets its bit.
func (s *GloasEpochState) PayloadAvailabilitySettled(slot phase0.Slot) bool {
	return slot < s.LatestBlockHeaderSlot
}

// GetGloasEpochState returns the reduced Gloas state of stateID, sharing one download between callers and holding one full state at a time.
func (b *BeaconNode) GetGloasEpochState(ctx context.Context, stateID string) (*GloasEpochState, error) {
	ctx, span := observability.Tracer().Start(ctx, "ethereum.beacon.GetGloasEpochState",
		trace.WithAttributes(attribute.String("state_id", stateID)))
	defer span.End()

	network := string(b.Metadata().Network.Name)

	if item := b.stateCache.Get(stateID); item != nil {
		b.metrics.IncBeaconStateCacheHit(network)
		span.SetAttributes(attribute.Bool("cached", true))

		return item.Value(), nil
	}

	span.SetAttributes(attribute.Bool("cached", false))

	x, err, shared := b.stateSfGroup.Do(stateID, func() (any, error) {
		// The previous flight may have cached the result after our first lookup.
		if item := b.stateCache.Get(stateID); item != nil {
			return item.Value(), nil
		}

		select {
		case b.stateSem <- struct{}{}:
		case <-ctx.Done():
			return nil, ctx.Err()
		}

		defer func() { <-b.stateSem }()

		start := time.Now()

		state, err := b.fetchGloasEpochState(ctx, stateID)
		if err != nil {
			return nil, err
		}

		b.metrics.ObserveBeaconStateFetch(network, time.Since(start))
		b.stateCache.Set(stateID, state, gloasEpochStateCacheTTL)

		// Return the decoded state's heap to the OS now instead of waiting for the scavenger.
		debug.FreeOSMemory()

		return state, nil
	})
	if err != nil {
		span.SetStatus(codes.Error, err.Error())

		return nil, err
	}

	span.AddEvent("Beacon state fetch complete.", trace.WithAttributes(attribute.Bool("shared", shared)))

	state, ok := x.(*GloasEpochState)
	if !ok {
		return nil, fmt.Errorf("beacon state singleflight returned unexpected type %T", x)
	}

	return state, nil
}

func (b *BeaconNode) fetchGloasEpochState(ctx context.Context, stateID string) (*GloasEpochState, error) {
	ds, err := b.sszCodec(ctx)
	if err != nil {
		return nil, err
	}

	rsp, err := b.beacon.OpenRawBeaconState(ctx, stateID, sszContentType)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to open beacon state %s", stateID)
	}

	defer rsp.Close()

	if version := rsp.Header.Get("Eth-Consensus-Version"); version != spec.DataVersionGloas.String() {
		return nil, fmt.Errorf("beacon state %s has consensus version %q, expected %s", stateID, version, spec.DataVersionGloas)
	}

	projected, err := decodeGloasEpochState(ds, rsp.Body, int(rsp.ContentLength))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to decode beacon state %s", stateID)
	}

	return projected, nil
}

// decodeGloasEpochState streams a Gloas state from r (size is negative when unknown) and reduces it.
func decodeGloasEpochState(ds *dynssz.DynSsz, r io.Reader, size int) (*GloasEpochState, error) {
	state := &gloas.BeaconState{}

	if err := ds.UnmarshalSSZReader(state, r, size); err != nil {
		return nil, err
	}

	return newGloasEpochState(state)
}

// sszCodec returns an SSZ codec built from the node's spec so non-mainnet presets decode.
func (b *BeaconNode) sszCodec(ctx context.Context) (*dynssz.DynSsz, error) {
	b.stateCodecMu.Lock()
	defer b.stateCodecMu.Unlock()

	if b.stateCodec != nil {
		return b.stateCodec, nil
	}

	provider, ok := b.beacon.Service().(client.SpecProvider)
	if !ok {
		return nil, errors.New("spec provider not available")
	}

	specs, err := provider.Spec(ctx, &api.SpecOpts{})
	if err != nil {
		return nil, errors.Wrap(err, "failed to fetch spec")
	}

	b.stateCodec = dynssz.NewDynSsz(specs.Data, dynssz.WithMaxStreamSize(maxBeaconStateSSZSize))

	return b.stateCodec, nil
}
