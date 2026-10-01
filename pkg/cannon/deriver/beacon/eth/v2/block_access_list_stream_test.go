package v2

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"testing"

	"github.com/creasty/defaults"
	"github.com/ethpandaops/go-eth2-client/spec/phase0"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var errSendFailed = errors.New("insert timed out")

const testFirstSlot = phase0.Slot(100)

func eventID(slot phase0.Slot, i int) string {
	return fmt.Sprintf("%d/%d", slot, i)
}

func idsOf(events []*xatu.DecoratedEvent) []string {
	ids := make([]string, 0, len(events))
	for _, e := range events {
		ids = append(ids, e.GetEvent().GetId())
	}

	return ids
}

func repeatInt(v, n int) []int {
	out := make([]int, n)
	for i := range out {
		out[i] = v
	}

	return out
}

// streamHarness wires an epochStreamer to a fake slot source and a fake sink.
type streamHarness struct {
	streamer *epochStreamer

	// slotSizes[i] is the number of events slot testFirstSlot+i produces.
	slotSizes []int

	deriveCalls map[phase0.Slot]int
	// failDeriveAt, when set, makes deriving that slot fail once.
	failDeriveAt *phase0.Slot

	// sendCalls counts every call to the sink, including failed ones.
	sendCalls int
	// failSendCalls lists the zero-based call numbers that fail.
	failSendCalls map[int]bool
	// accepted holds the batches the sink accepted, in order.
	accepted [][]string
}

func newStreamHarness(maxRows int, slotSizes []int) *streamHarness {
	h := &streamHarness{
		slotSizes:     slotSizes,
		deriveCalls:   make(map[phase0.Slot]int),
		failSendCalls: make(map[int]bool),
	}

	h.streamer = newEpochStreamer(maxRows, h.derive, h.send)

	return h
}

func (h *streamHarness) derive(_ context.Context, slot phase0.Slot) ([]*xatu.DecoratedEvent, error) {
	h.deriveCalls[slot]++

	if h.failDeriveAt != nil && *h.failDeriveAt == slot {
		h.failDeriveAt = nil

		return nil, errors.New("beacon node unavailable")
	}

	n := h.slotSizes[int(slot-testFirstSlot)]
	events := make([]*xatu.DecoratedEvent, 0, n)

	for i := range n {
		events = append(events, &xatu.DecoratedEvent{
			Event: &xatu.Event{Id: eventID(slot, i)},
		})
	}

	return events, nil
}

func (h *streamHarness) send(_ context.Context, events []*xatu.DecoratedEvent) error {
	call := h.sendCalls
	h.sendCalls++

	if h.failSendCalls[call] {
		return errSendFailed
	}

	h.accepted = append(h.accepted, idsOf(events))

	return nil
}

func (h *streamHarness) stream(ctx context.Context, epoch phase0.Epoch) error {
	return h.streamer.stream(ctx, epoch, testFirstSlot, uint64(len(h.slotSizes)))
}

func (h *streamHarness) allIDs() []string {
	var ids []string
	for slot := range h.slotSizes {
		for i := range h.slotSizes[slot] {
			ids = append(ids, eventID(testFirstSlot+phase0.Slot(slot), i))
		}
	}

	return ids
}

func (h *streamHarness) acceptedIDs() []string {
	return slices.Concat(h.accepted...)
}

func (h *streamHarness) acceptedBatchSizes() []int {
	sizes := make([]int, 0, len(h.accepted))

	for _, b := range h.accepted {
		sizes = append(sizes, len(b))
	}

	return sizes
}

func TestEpochStreamer_Batching(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		maxRows   int
		slotSizes []int
		want      []int
	}{
		{
			name:      "epoch without events sends nothing",
			maxRows:   10,
			slotSizes: []int{0, 0, 0},
			want:      []int{},
		},
		{
			name:      "small slots share one batch",
			maxRows:   10,
			slotSizes: []int{3, 4, 2},
			want:      []int{9},
		},
		{
			name:      "slots filling a batch exactly",
			maxRows:   10,
			slotSizes: []int{5, 5},
			want:      []int{10},
		},
		{
			name:      "next slot that would overflow starts a new batch",
			maxRows:   10,
			slotSizes: []int{4, 4, 4},
			want:      []int{8, 4},
		},
		{
			name:      "oversized slot is split",
			maxRows:   10,
			slotSizes: []int{25},
			want:      []int{10, 10, 5},
		},
		{
			name:      "oversized slot after a small one",
			maxRows:   10,
			slotSizes: []int{3, 25},
			want:      []int{3, 10, 10, 5},
		},
		{
			name:      "oversized slot before a small one",
			maxRows:   10,
			slotSizes: []int{25, 3},
			want:      []int{10, 10, 5, 3},
		},
		{
			name:      "empty slots between busy ones are skipped",
			maxRows:   10,
			slotSizes: []int{6, 0, 0, 6},
			want:      []int{6, 6},
		},
		{
			name:      "one row per batch",
			maxRows:   1,
			slotSizes: []int{2, 1},
			want:      []int{1, 1, 1},
		},
		{
			name:      "epoch of roughly a million rows at the default cap",
			maxRows:   DefaultBlockAccessListMaxRowsPerBatch,
			slotSizes: repeatInt(30_000, 32),
			want:      repeatInt(30_000, 32),
		},
		{
			name:      "mainnet sized slots are packed up to the cap",
			maxRows:   DefaultBlockAccessListMaxRowsPerBatch,
			slotSizes: repeatInt(15_500, 32),
			want:      append(repeatInt(46_500, 10), 31_000),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			h := newStreamHarness(tt.maxRows, tt.slotSizes)

			require.NoError(t, h.stream(t.Context(), 3))

			assert.Equal(t, tt.want, h.acceptedBatchSizes())

			for _, size := range h.acceptedBatchSizes() {
				assert.LessOrEqual(t, size, tt.maxRows, "a batch exceeded the cap")
			}

			assert.Equal(t, h.allIDs(), h.acceptedIDs(), "every event is delivered once, in slot order")

			for slot := range tt.slotSizes {
				assert.Equal(t, 1, h.deriveCalls[testFirstSlot+phase0.Slot(slot)])
			}
		})
	}
}

func TestEpochStreamer_FailedSendResumesWithoutResendingAcceptedSlots(t *testing.T) {
	t.Parallel()

	// Each slot is its own batch, so batch k is exactly slot k.
	slotSizes := repeatInt(6, 6)

	for failAt := range slotSizes {
		t.Run(fmt.Sprintf("send %d fails once", failAt), func(t *testing.T) {
			t.Parallel()

			h := newStreamHarness(10, slotSizes)
			h.failSendCalls[failAt] = true

			err := h.stream(t.Context(), 7)
			require.ErrorIs(t, err, errSendFailed)
			assert.Len(t, h.accepted, failAt, "only batches before the failure are accepted")

			require.NoError(t, h.stream(t.Context(), 7))

			assert.Equal(t, h.allIDs(), h.acceptedIDs(), "every event is accepted exactly once")

			for slot := range failAt {
				assert.Equal(t, 1, h.deriveCalls[testFirstSlot+phase0.Slot(slot)],
					"slot %d was accepted and must not be derived again", slot)
			}
		})
	}
}

func TestEpochStreamer_RetryReplaysOnlyTheFailedFlush(t *testing.T) {
	t.Parallel()

	// A single slot of 25 rows is split into 10, 10, 5. The second chunk
	// fails, so the retry sends the whole slot again.
	h := newStreamHarness(10, []int{25, 3})
	h.failSendCalls[1] = true

	require.ErrorIs(t, h.stream(t.Context(), 7), errSendFailed)
	require.Equal(t, []int{10}, h.acceptedBatchSizes())

	require.NoError(t, h.stream(t.Context(), 7))

	assert.Equal(t, []int{10, 10, 10, 5, 3}, h.acceptedBatchSizes())

	unique := make(map[string]int)
	for _, id := range h.acceptedIDs() {
		unique[id]++
	}

	assert.Len(t, unique, 28, "no event is lost")

	replayed := 0

	for _, n := range unique {
		if n > 1 {
			replayed += n - 1
		}
	}

	assert.Equal(t, 10, replayed, "only the chunk accepted before the failure is replayed")
}

func TestEpochStreamer_ProgressesUnderRepeatedFailures(t *testing.T) {
	t.Parallel()

	// Every other send fails. Without remembering accepted slots the epoch
	// could never complete, since it needs all six sends to succeed in a row.
	h := newStreamHarness(10, repeatInt(6, 6))

	for call := 0; call < 100; call += 2 {
		h.failSendCalls[call] = true
	}

	attempts := 0

	for {
		attempts++
		require.LessOrEqual(t, attempts, 20, "epoch did not converge")

		if err := h.stream(t.Context(), 9); err == nil {
			break
		}
	}

	assert.Equal(t, h.allIDs(), h.acceptedIDs())
}

func TestEpochStreamer_DeriveFailureResumes(t *testing.T) {
	t.Parallel()

	h := newStreamHarness(10, repeatInt(6, 6))

	failSlot := testFirstSlot + 3
	h.failDeriveAt = &failSlot

	err := h.stream(t.Context(), 11)
	require.Error(t, err)
	assert.Contains(t, err.Error(), fmt.Sprintf("failed to process slot %d", failSlot))
	assert.Equal(t, []int{6, 6}, h.acceptedBatchSizes(), "slots before the failing one were already delivered")

	require.NoError(t, h.stream(t.Context(), 11))

	assert.Equal(t, h.allIDs(), h.acceptedIDs())
	assert.Equal(t, 1, h.deriveCalls[testFirstSlot], "accepted slot is not derived again")
	assert.Equal(t, 1, h.deriveCalls[testFirstSlot+1], "accepted slot is not derived again")
}

func TestEpochStreamer_ResumeState(t *testing.T) {
	t.Parallel()

	t.Run("progress from one epoch does not leak into another", func(t *testing.T) {
		t.Parallel()

		h := newStreamHarness(10, repeatInt(6, 4))
		h.failSendCalls[2] = true

		require.ErrorIs(t, h.stream(t.Context(), 20), errSendFailed)

		h.accepted = nil

		require.NoError(t, h.stream(t.Context(), 21))
		assert.Equal(t, h.allIDs(), h.acceptedIDs(), "the new epoch is delivered from its first slot")
	})

	t.Run("a completed epoch is skipped until reset", func(t *testing.T) {
		t.Parallel()

		h := newStreamHarness(10, repeatInt(6, 4))

		require.NoError(t, h.stream(t.Context(), 30))

		// The checkpoint write failed, so the epoch is attempted again.
		h.accepted = nil

		require.NoError(t, h.stream(t.Context(), 30))
		assert.Empty(t, h.accepted, "nothing is sent twice while the checkpoint is pending")

		h.streamer.reset()

		require.NoError(t, h.stream(t.Context(), 30))
		assert.Equal(t, h.allIDs(), h.acceptedIDs(), "after reset the epoch is delivered afresh")
	})
}

func TestEpochStreamer_StopsOnCanceledContext(t *testing.T) {
	t.Parallel()

	h := newStreamHarness(10, repeatInt(6, 4))

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	require.ErrorIs(t, h.stream(ctx, 40), context.Canceled)
	assert.Zero(t, h.sendCalls)
	assert.Empty(t, h.deriveCalls)
}

func TestNewEpochStreamer_DefaultsNonPositiveMaxRows(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		maxRows int
		want    int
	}{
		{name: "zero", maxRows: 0, want: DefaultBlockAccessListMaxRowsPerBatch},
		{name: "negative", maxRows: -5, want: DefaultBlockAccessListMaxRowsPerBatch},
		{name: "positive is kept", maxRows: 123, want: 123},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, newEpochStreamer(tt.maxRows, nil, nil).maxRows)
		})
	}
}

func TestBlockAccessListDeriverConfig(t *testing.T) {
	t.Parallel()

	t.Run("defaults", func(t *testing.T) {
		t.Parallel()

		cfg := &BlockAccessListDeriverConfig{}
		require.NoError(t, defaults.Set(cfg))

		assert.True(t, cfg.Enabled)
		assert.Equal(t, DefaultBlockAccessListMaxRowsPerBatch, cfg.MaxRowsPerBatch)
		require.NoError(t, cfg.Validate())
	})

	tests := []struct {
		name    string
		cfg     BlockAccessListDeriverConfig
		wantErr string
	}{
		{name: "disabled deriver is not validated", cfg: BlockAccessListDeriverConfig{Enabled: false, MaxRowsPerBatch: -1}},
		{name: "positive cap", cfg: BlockAccessListDeriverConfig{Enabled: true, MaxRowsPerBatch: 1}},
		{name: "zero cap", cfg: BlockAccessListDeriverConfig{Enabled: true, MaxRowsPerBatch: 0}, wantErr: "maxRowsPerBatch"},
		{name: "negative cap", cfg: BlockAccessListDeriverConfig{Enabled: true, MaxRowsPerBatch: -10}, wantErr: "maxRowsPerBatch"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.cfg.Validate()
			if tt.wantErr == "" {
				require.NoError(t, err)

				return
			}

			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func TestBlockAccessListDeriver_SendEvents(t *testing.T) {
	t.Parallel()

	deriver := NewBlockAccessListDeriver(
		logrus.New(),
		&BlockAccessListDeriverConfig{Enabled: true, MaxRowsPerBatch: 7},
		nil, nil, nil,
	)

	assert.Equal(t, 7, deriver.streamer.maxRows, "cap comes from the config")

	var calls []string

	deriver.OnEventsDerived(t.Context(), func(_ context.Context, events []*xatu.DecoratedEvent) error {
		calls = append(calls, fmt.Sprintf("first:%d", len(events)))

		return nil
	})

	failSecond := true

	deriver.OnEventsDerived(t.Context(), func(_ context.Context, events []*xatu.DecoratedEvent) error {
		calls = append(calls, fmt.Sprintf("second:%d", len(events)))

		if failSecond {
			return errSendFailed
		}

		return nil
	})

	batch := []*xatu.DecoratedEvent{{}, {}, {}}

	err := deriver.sendEvents(t.Context(), batch)
	require.ErrorIs(t, err, errSendFailed)
	assert.Equal(t, []string{"first:3", "second:3"}, calls)

	failSecond = false
	calls = nil

	require.NoError(t, deriver.sendEvents(t.Context(), batch))
	assert.Equal(t, []string{"first:3", "second:3"}, calls)
}
