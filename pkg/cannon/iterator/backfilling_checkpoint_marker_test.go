package iterator

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"strconv"
	"sync"
	"testing"

	apiv1 "github.com/ethpandaops/go-eth2-client/api/v1"
	"github.com/ethpandaops/go-eth2-client/spec"
	"github.com/ethpandaops/go-eth2-client/spec/phase0"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

const testNetworkID = "1"

var (
	errInjected = errors.New("injected failure")
	errCrash    = errors.New("process crashed")
)

func epochID(epoch phase0.Epoch) string {
	return strconv.FormatUint(uint64(epoch), 10)
}

// memoryStore is an in-memory coordinator.
type memoryStore struct {
	mu        sync.Mutex
	locations map[xatu.CannonType]*xatu.CannonLocation
	failNext  bool
}

func newMemoryStore() *memoryStore {
	return &memoryStore{locations: map[xatu.CannonType]*xatu.CannonLocation{}}
}

func (s *memoryStore) GetCannonLocation(_ context.Context, typ xatu.CannonType, networkID string) (*xatu.CannonLocation, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	loc, ok := s.locations[typ]
	if !ok || loc.GetNetworkId() != networkID {
		return nil, nil //nolint:nilnil // the coordinator answers an unseen type with no location
	}

	clone, _ := proto.Clone(loc).(*xatu.CannonLocation)

	return clone, nil
}

func (s *memoryStore) UpsertCannonLocationRequest(_ context.Context, location *xatu.CannonLocation) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.failNext {
		s.failNext = false

		return errInjected
	}

	clone, _ := proto.Clone(location).(*xatu.CannonLocation)
	s.locations[location.GetType()] = clone

	return nil
}

func (s *memoryStore) failNextUpsert() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.failNext = true
}

// staticChain reports a fixed finalized checkpoint and fork schedule.
type staticChain struct {
	mu         sync.Mutex
	finalized  phase0.Epoch
	forkEpochs map[spec.DataVersion]phase0.Epoch
}

func (c *staticChain) Finality() (*apiv1.Finality, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	return &apiv1.Finality{Finalized: &phase0.Checkpoint{Epoch: c.finalized}}, nil
}

func (c *staticChain) ForkEpoch(fork spec.DataVersion) (phase0.Epoch, error) {
	epoch, ok := c.forkEpochs[fork]
	if !ok {
		return 0, fmt.Errorf("unknown fork %s", fork)
	}

	return epoch, nil
}

func (c *staticChain) advance() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.finalized++
}

func (c *staticChain) finalizedEpoch() phase0.Epoch {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.finalized
}

func newTestIterator(
	t *testing.T,
	store locationStore,
	chain checkpointSource,
	metrics *BackfillingCheckpointMetrics,
	cannonType xatu.CannonType,
	fork spec.DataVersion,
	cfg *BackfillingCheckpointConfig,
) *BackfillingCheckpoint {
	t.Helper()

	log := logrus.New()
	log.SetOutput(io.Discard)

	it := &BackfillingCheckpoint{
		log:               log.WithField("cannon_type", cannonType.String()),
		cannonType:        cannonType,
		coordinator:       store,
		networkID:         testNetworkID,
		networkName:       "test",
		metrics:           metrics,
		chain:             chain,
		checkpointName:    "finalized",
		lookAheadDistance: 3,
		config:            cfg,
	}

	require.NoError(t, it.Start(context.Background(), fork))

	return it
}

func backfillConfig(toEpoch phase0.Epoch, epochsPerBatch uint64) *BackfillingCheckpointConfig {
	cfg := &BackfillingCheckpointConfig{}
	cfg.Backfill.Enabled = true
	cfg.Backfill.ToEpoch = toEpoch
	cfg.Backfill.EpochsPerBatch = epochsPerBatch

	return cfg
}

// failurePoint is where a step is interrupted.
type failurePoint int

const (
	failNone failurePoint = iota
	// crashAfterNext loses the process after the step is handed out.
	crashAfterNext
	// failDerive fails deriving one epoch of the step.
	failDerive
	// failWrite writes part of the step's events, then fails the write.
	failWrite
	// crashAfterWrite loses the process after the write, before the marker update.
	crashAfterWrite
	// failMarker fails the coordinator upsert after the write.
	failMarker
)

type stepRecord struct {
	position *BackFillingCheckpointNextResponse
	err      error
}

// harness drives the real iterator through the deriver loop shape: Next, derive every epoch, write, UpdateLocation.
type harness struct {
	t          *testing.T
	store      *memoryStore
	chain      *staticChain
	metrics    BackfillingCheckpointMetrics
	cannonType xatu.CannonType
	fork       spec.DataVersion
	cfg        *BackfillingCheckpointConfig
	it         *BackfillingCheckpoint

	// written counts durable rows per epoch and direction.
	written map[BackfillingCheckpointDirection]map[phase0.Epoch]int
	// rewritable holds epochs that were written without the marker covering them.
	rewritable map[phase0.Epoch]bool
	// deriveFails reports whether deriving an epoch fails.
	deriveFails func(epoch phase0.Epoch) bool
	steps       []stepRecord
}

func newHarness(t *testing.T, cannonType xatu.CannonType, fork spec.DataVersion, chain *staticChain, cfg *BackfillingCheckpointConfig) *harness {
	t.Helper()

	return &harness{
		t:          t,
		store:      newMemoryStore(),
		chain:      chain,
		metrics:    newBackfillingCheckpointMetrics("test"),
		cannonType: cannonType,
		fork:       fork,
		cfg:        cfg,
		written: map[BackfillingCheckpointDirection]map[phase0.Epoch]int{
			BackfillingCheckpointDirectionHead:     {},
			BackfillingCheckpointDirectionBackfill: {},
		},
		rewritable:  map[phase0.Epoch]bool{},
		deriveFails: func(phase0.Epoch) bool { return false },
	}
}

// seed stores a marker as the coordinator would hold it.
func (h *harness) seed(finalized phase0.Epoch, backfill int64) {
	h.t.Helper()

	it := newTestIterator(h.t, h.store, h.chain, &h.metrics, h.cannonType, h.fork, h.cfg)

	loc, err := it.createLocationFromEpochNumber(finalized, phase0.Epoch(backfill))
	require.NoError(h.t, err)
	require.NoError(h.t, h.store.UpsertCannonLocationRequest(context.Background(), loc))
}

// boot starts a fresh process: a new iterator with no memory of earlier steps.
func (h *harness) boot() {
	h.it = newTestIterator(h.t, h.store, h.chain, &h.metrics, h.cannonType, h.fork, h.cfg)
}

func (h *harness) marker() *xatu.BackfillingCheckpointMarker {
	h.t.Helper()

	loc, err := h.store.GetCannonLocation(context.Background(), h.cannonType, testNetworkID)
	require.NoError(h.t, err)
	require.NotNil(h.t, loc)

	marker, err := h.it.GetMarker(loc)
	require.NoError(h.t, err)

	return marker
}

func (h *harness) target() phase0.Epoch {
	h.t.Helper()

	target, err := h.it.getEarliestPossibleBackfillEpoch()
	require.NoError(h.t, err)

	return target
}

// done reports whether both directions are caught up, where Next would sleep.
func (h *harness) done() bool {
	m := h.marker()

	return phase0.Epoch(m.GetFinalizedEpoch()) >= h.chain.finalizedEpoch() &&
		phase0.Epoch(m.GetBackfillEpoch()) <= h.target()
}

func (h *harness) derive(_ context.Context, epoch phase0.Epoch) ([]*xatu.DecoratedEvent, error) {
	if h.deriveFails(epoch) {
		return nil, errInjected
	}

	return []*xatu.DecoratedEvent{epochEvent(epoch)}, nil
}

func (h *harness) write(direction BackfillingCheckpointDirection, events []*xatu.DecoratedEvent) {
	h.t.Helper()

	for _, event := range events {
		epoch, err := strconv.ParseUint(event.GetEvent().GetId(), 10, 64)
		require.NoError(h.t, err)

		h.written[direction][phase0.Epoch(epoch)]++
	}
}

func (h *harness) markRewritable(events []*xatu.DecoratedEvent) {
	for _, event := range events {
		epoch, _ := strconv.ParseUint(event.GetEvent().GetId(), 10, 64)
		h.rewritable[phase0.Epoch(epoch)] = true
	}
}

// step runs one deriver loop iteration, interrupted at fail; crashes reboot the process.
func (h *harness) step(ctx context.Context, fail failurePoint, rng *rand.Rand) error {
	h.t.Helper()

	position, err := h.it.Next(ctx)
	require.NoError(h.t, err)

	record := func(err error) error {
		h.steps = append(h.steps, stepRecord{position: position, err: err})

		if errors.Is(err, errCrash) {
			h.boot()
		}

		return err
	}

	if fail == crashAfterNext {
		return record(errCrash)
	}

	epochs := position.Epochs()

	if fail == failDerive {
		bad := epochs[rng.Intn(len(epochs))]
		previous := h.deriveFails
		h.deriveFails = func(e phase0.Epoch) bool { return e == bad }

		defer func() { h.deriveFails = previous }()
	}

	events, err := DeriveEpochs(ctx, epochs, h.derive)
	if err != nil {
		return record(err)
	}

	if fail == failWrite {
		partial := events[:rng.Intn(len(events)+1)]
		h.write(position.Direction, partial)
		h.markRewritable(partial)

		return record(errInjected)
	}

	h.write(position.Direction, events)

	switch fail {
	case crashAfterWrite:
		h.markRewritable(events)

		return record(errCrash)
	case failMarker:
		h.markRewritable(events)
		h.store.failNextUpsert()
	case failNone, crashAfterNext, failDerive, failWrite:
	}

	return record(h.it.UpdateLocation(ctx, position.LastEpoch(), position.Direction))
}

// assertNoGap checks every epoch the backfill marker covers was durably written.
func (h *harness) assertNoGap(start phase0.Epoch) {
	h.t.Helper()

	backfill := phase0.Epoch(h.marker().GetBackfillEpoch())
	for e := backfill; e < start; e++ {
		require.Positive(h.t, h.written[BackfillingCheckpointDirectionBackfill][e],
			"marker at %d covers epoch %d that was never written", backfill, e)
	}
}

// assertComplete checks the walk covered exactly [target, start) and only re-wrote epochs written without the marker covering them.
func (h *harness) assertComplete(start phase0.Epoch) {
	h.t.Helper()

	target := h.target()
	backfilled := h.written[BackfillingCheckpointDirectionBackfill]

	assert.Equal(h.t, int64(target), h.marker().GetBackfillEpoch())

	for e := target; e < start; e++ {
		require.Positive(h.t, backfilled[e], "epoch %d never written", e)

		if backfilled[e] > 1 {
			assert.True(h.t, h.rewritable[e], "epoch %d written %d times but never written without the marker covering it", e, backfilled[e])
		}
	}

	for e := range backfilled {
		assert.True(h.t, e >= target && e < start, "epoch %d written outside [%d, %d)", e, target, start)
	}
}

// run steps until both directions are caught up.
func (h *harness) run(ctx context.Context, failures func(step int) failurePoint, rng *rand.Rand, start phase0.Epoch) {
	h.t.Helper()

	for i := 0; !h.done(); i++ {
		require.Less(h.t, i, 100000, "walk did not finish")

		before := h.marker().GetBackfillEpoch()

		_ = h.step(ctx, failures(i), rng)

		after := h.marker().GetBackfillEpoch()
		require.LessOrEqual(h.t, after, before, "backfill marker moved up")

		h.assertNoGap(start)
	}
}

func noFailures(int) failurePoint { return failNone }

func TestBackfillBatches_ContiguousAdvance(t *testing.T) {
	for _, size := range []uint64{1, 7, 64} {
		t.Run(fmt.Sprintf("epochsPerBatch=%d", size), func(t *testing.T) {
			const start = phase0.Epoch(1000)

			chain := &staticChain{finalized: start}
			h := newHarness(t, xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_GRAFFITI, spec.DataVersionPhase0, chain, backfillConfig(0, size))
			h.seed(start, int64(start))
			h.boot()

			h.run(context.Background(), noFailures, nil, start)
			h.assertComplete(start)

			expect := start

			for _, s := range h.steps {
				require.NoError(t, s.err)
				require.Equal(t, BackfillingCheckpointDirectionBackfill, s.position.Direction)

				epochs := s.position.Epochs()
				require.Equal(t, expect-1, epochs[0], "step starts right below the marker")
				require.LessOrEqual(t, uint64(len(epochs)), size)
				assert.Equal(t, epochRange(epochs[0], s.position.LastEpoch()), epochs, "step is contiguous")

				expect = s.position.LastEpoch()
			}

			assert.Len(t, h.steps, int((uint64(start)+size-1)/size))

			for e, n := range h.written[BackfillingCheckpointDirectionBackfill] {
				assert.Equal(t, 1, n, "epoch %d written more than once without failures", e)
			}
		})
	}
}

func TestBackfillBatches_HonoursToEpoch(t *testing.T) {
	const start, toEpoch = phase0.Epoch(1000), phase0.Epoch(937)

	chain := &staticChain{finalized: start}
	h := newHarness(t, xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_GRAFFITI, spec.DataVersionPhase0, chain, backfillConfig(toEpoch, 16))
	h.seed(start, int64(start))
	h.boot()

	h.run(context.Background(), noFailures, nil, start)
	h.assertComplete(start)

	last := h.steps[len(h.steps)-1].position
	assert.Equal(t, toEpoch, last.LastEpoch())
	assert.Len(t, last.Epochs(), int((start-toEpoch)%16), "final batch is clamped to toEpoch")

	for _, s := range h.steps {
		for _, e := range s.position.LookAheads {
			assert.GreaterOrEqual(t, e, toEpoch, "look ahead below toEpoch")
		}
	}
}

func TestBackfillBatches_StopsAtActivationFork(t *testing.T) {
	const fork = phase0.Epoch(269568)

	tests := []struct {
		name    string
		toEpoch phase0.Epoch
		want    phase0.Epoch
	}{
		{name: "no toEpoch stops at the fork", toEpoch: 0, want: fork},
		{name: "toEpoch before the fork is clamped", toEpoch: 100, want: fork},
		{name: "toEpoch after the fork wins", toEpoch: fork + 50, want: fork + 50},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start := fork + 300

			chain := &staticChain{finalized: start, forkEpochs: map[spec.DataVersion]phase0.Epoch{spec.DataVersionDeneb: fork}}
			h := newHarness(t, xatu.CannonType_BEACON_API_ETH_V1_BEACON_BLOB_SIDECAR, spec.DataVersionDeneb, chain, backfillConfig(tt.toEpoch, 64))
			h.seed(start, int64(start))
			h.boot()

			h.run(context.Background(), noFailures, nil, start)
			h.assertComplete(start)
			assert.Equal(t, tt.want, h.target())
			assert.Equal(t, tt.want, h.steps[len(h.steps)-1].position.LastEpoch())
		})
	}
}

func TestBackfillBatches_PersistentDeriveErrorStopsBeforeTheBadEpoch(t *testing.T) {
	const start, bad = phase0.Epoch(1000), phase0.Epoch(950)

	chain := &staticChain{finalized: start}
	h := newHarness(t, xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_GRAFFITI, spec.DataVersionPhase0, chain, backfillConfig(0, 32))
	h.deriveFails = func(e phase0.Epoch) bool { return e == bad }
	h.seed(start, int64(start))
	h.boot()

	ctx := context.Background()

	for range 200 {
		_ = h.step(ctx, failNone, nil)
	}

	m := h.marker()
	assert.Equal(t, int64(bad+1), m.GetBackfillEpoch(), "marker stops right above the failing epoch")
	h.assertNoGap(start)

	backfilled := h.written[BackfillingCheckpointDirectionBackfill]
	for e := bad + 1; e < start; e++ {
		assert.Equal(t, 1, backfilled[e], "epoch %d", e)
	}

	assert.Zero(t, backfilled[bad])

	failed := h.steps[len(h.steps)-1]
	require.Error(t, failed.err)
	assert.Equal(t, []phase0.Epoch{bad}, failed.position.Epochs(), "the failing epoch is retried alone")
}

func TestBackfillBatches_TransientDeriveErrorResumesBatching(t *testing.T) {
	const start, size = phase0.Epoch(1000), uint64(32)

	chain := &staticChain{finalized: start}
	h := newHarness(t, xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_GRAFFITI, spec.DataVersionPhase0, chain, backfillConfig(0, size))
	h.seed(start, int64(start))
	h.boot()

	// The second batch (epochs 967..936) fails once at 950.
	failed := false
	h.deriveFails = func(e phase0.Epoch) bool {
		if e == 950 && !failed {
			failed = true

			return true
		}

		return false
	}

	h.run(context.Background(), noFailures, nil, start)
	h.assertComplete(start)

	sizes := make([]int, 0, len(h.steps))
	for _, s := range h.steps {
		sizes = append(sizes, len(s.position.Epochs()))
	}

	want := make([]int, 0, 34)
	want = append(want, 32, 32)

	for range 32 {
		want = append(want, 1)
	}

	assert.Equal(t, want, sizes[:len(want)], "the failed batch is retried one epoch at a time")
	assert.Equal(t, 32, sizes[len(want)], "batching resumes below the failed batch")

	for e, n := range h.written[BackfillingCheckpointDirectionBackfill] {
		assert.Equal(t, 1, n, "epoch %d: a failed derive writes nothing", e)
	}
}

func TestBackfillBatches_WriteFailureRewritesOnlyThatBatch(t *testing.T) {
	const start = phase0.Epoch(500)

	chain := &staticChain{finalized: start}
	h := newHarness(t, xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK, spec.DataVersionPhase0, chain, backfillConfig(0, 50))
	h.seed(start, int64(start))
	h.boot()

	rng := rand.New(rand.NewSource(1))
	failures := func(step int) failurePoint {
		switch step {
		case 2:
			return failWrite
		case 5:
			return failMarker
		default:
			return failNone
		}
	}

	h.run(context.Background(), failures, rng, start)
	h.assertComplete(start)

	for e, n := range h.written[BackfillingCheckpointDirectionBackfill] {
		if n > 1 {
			assert.True(t, h.rewritable[e], "epoch %d", e)
		}
	}
}

func TestBackfillBatches_CrashResumeHasNoGaps(t *testing.T) {
	failurePoints := []failurePoint{crashAfterNext, failDerive, failWrite, crashAfterWrite, failMarker}

	for seed := int64(1); seed <= 40; seed++ {
		t.Run(fmt.Sprintf("seed=%d", seed), func(t *testing.T) {
			rng := rand.New(rand.NewSource(seed))

			start := phase0.Epoch(300 + rng.Intn(400))
			toEpoch := phase0.Epoch(rng.Intn(100))
			size := uint64(1 + rng.Intn(MaxEpochsPerBatch/4))

			chain := &staticChain{finalized: start}
			h := newHarness(t, xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_GRAFFITI, spec.DataVersionPhase0, chain, backfillConfig(toEpoch, size))
			h.seed(start, int64(start))
			h.boot()

			failures := func(int) failurePoint {
				// New finalized epochs interleave head steps with the backfill.
				if rng.Intn(10) == 0 {
					chain.advance()
				}

				if rng.Intn(3) == 0 {
					return failurePoints[rng.Intn(len(failurePoints))]
				}

				return failNone
			}

			h.run(context.Background(), failures, rng, start)
			h.assertComplete(start)

			head := h.written[BackfillingCheckpointDirectionHead]
			for e := start + 1; e <= chain.finalizedEpoch(); e++ {
				require.Positive(t, head[e], "head epoch %d never written", e)
			}

			for _, s := range h.steps {
				if s.position.Direction == BackfillingCheckpointDirectionHead {
					assert.Len(t, s.position.Epochs(), 1, "head steps stay one epoch")
				}
			}
		})
	}
}

func TestBackfillBatches_HeadStepsStayOneEpoch(t *testing.T) {
	const finalized, head = phase0.Epoch(1000), phase0.Epoch(990)

	chain := &staticChain{finalized: finalized}
	h := newHarness(t, xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_GRAFFITI, spec.DataVersionPhase0, chain, backfillConfig(0, 64))
	h.seed(head, int64(head))
	h.boot()

	ctx := context.Background()

	for e := head + 1; e <= finalized; e++ {
		require.NoError(t, h.step(ctx, failNone, nil))

		s := h.steps[len(h.steps)-1].position
		assert.Equal(t, BackfillingCheckpointDirectionHead, s.Direction)
		assert.Equal(t, []phase0.Epoch{e}, s.Epochs())
	}

	require.NoError(t, h.step(ctx, failNone, nil))

	s := h.steps[len(h.steps)-1].position
	assert.Equal(t, BackfillingCheckpointDirectionBackfill, s.Direction)
	assert.Len(t, s.Epochs(), 64)
}

func TestBackfillBatches_SingleEpochMatchesLegacySteps(t *testing.T) {
	const start = phase0.Epoch(100)

	chain := &staticChain{finalized: start}
	h := newHarness(t, xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_GRAFFITI, spec.DataVersionPhase0, chain, backfillConfig(0, 1))
	h.seed(start, int64(start))
	h.boot()

	require.NoError(t, h.step(context.Background(), failNone, nil))

	s := h.steps[0].position
	assert.Equal(t, []phase0.Epoch{99}, s.Epochs())
	assert.Equal(t, []phase0.Epoch{99, 98, 97}, s.LookAheads)
	assert.Equal(t, int64(99), h.marker().GetBackfillEpoch())
}

func TestBackfillBatches_LookAheadsCoverTheNextBatch(t *testing.T) {
	const start = phase0.Epoch(1000)

	chain := &staticChain{finalized: start}
	h := newHarness(t, xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_GRAFFITI, spec.DataVersionPhase0, chain, backfillConfig(0, 16))
	h.seed(start, int64(start))
	h.boot()

	require.NoError(t, h.step(context.Background(), failNone, nil))

	assert.Equal(t, epochRange(999, 968), h.steps[0].position.LookAheads)
}

// TestBackfillBatches_ConcurrentDerivers runs one iterator per cannon type against a shared coordinator and metrics, as cannon does.
func TestBackfillBatches_ConcurrentDerivers(t *testing.T) {
	types := []xatu.CannonType{
		xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_GRAFFITI,
		xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK,
		xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_WITHDRAWAL,
		xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_DEPOSIT,
		xatu.CannonType_BEACON_API_ETH_V1_PROPOSER_DUTY,
		xatu.CannonType_BEACON_API_ETH_V1_BEACON_COMMITTEE,
	}

	const start = phase0.Epoch(400)

	store := newMemoryStore()
	chain := &staticChain{finalized: start}
	metrics := newBackfillingCheckpointMetrics("test")

	harnesses := make([]*harness, len(types))

	for i, typ := range types {
		h := newHarness(t, typ, spec.DataVersionPhase0, chain, backfillConfig(0, uint64(8*(i+1))))
		h.store = store
		h.metrics = metrics
		h.seed(start, int64(start))
		h.boot()

		harnesses[i] = h
	}

	t.Run("walk", func(t *testing.T) {
		for i, h := range harnesses {
			t.Run(h.cannonType.String(), func(t *testing.T) {
				t.Parallel()

				h.t = t

				rng := rand.New(rand.NewSource(int64(i)))
				h.run(context.Background(), func(int) failurePoint {
					if rng.Intn(4) == 0 {
						return failWrite
					}

					return failNone
				}, rng, start)
				h.assertComplete(start)
			})
		}
	})
}
