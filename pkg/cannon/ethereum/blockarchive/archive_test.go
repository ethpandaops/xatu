package blockarchive

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ethpandaops/beacon/pkg/human"
	"github.com/ethpandaops/go-eth2-client/spec/phase0"
	dynssz "github.com/pk910/dynamic-ssz"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testNetwork = "testnet"

var testMetrics = sync.OnceValue(func() *Metrics { return NewMetrics("xatu_cannon_test") })

// fakeArchive serves one bundle the way block-archiver does.
type fakeArchive struct {
	start, end, span uint64
	body             []byte
	dataOffset       int64
	ignoreRange      bool
	failLookups      atomic.Bool

	lookups   atomic.Int64
	downloads atomic.Int64
	ranges    atomic.Int64
}

func newFakeArchive(t *testing.T, start, end uint64, blocks []testBlock) *fakeArchive {
	t.Helper()

	body := encodeBundle(t, start, end, blocks)
	_, dataOffset, err := readIndex(bytes.NewReader(body))
	require.NoError(t, err)

	return &fakeArchive{start: start, end: end, span: end - start + 1, body: body, dataOffset: dataOffset}
}

func (f *fakeArchive) url() string {
	return fmt.Sprintf("/bundles/testnet/%d/%010d.bundle", f.span, f.start)
}

func (f *fakeArchive) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/api/v1/bundles":
		f.lookups.Add(1)

		if f.failLookups.Load() {
			http.Error(w, "down", http.StatusBadGateway)

			return
		}

		slot, _ := strconv.ParseUint(r.URL.Query().Get("slot"), 10, 64)
		resp := map[string]any{
			"bundles":  []any{},
			"coverage": map[string]any{"min_slot": f.start, "max_slot": f.end, "span": f.span},
		}

		if r.URL.Query().Get("network") == testNetwork && slot >= f.start && slot <= f.end {
			resp["bundles"] = []any{map[string]any{
				"start_slot": f.start, "end_slot": f.end, "span": f.span, "url": f.url(),
				"size_bytes": len(f.body), "data_offset": f.dataOffset,
			}}
		}

		_ = json.NewEncoder(w).Encode(resp)
	case f.url():
		if r.Header.Get("Range") != "" && !f.ignoreRange {
			f.ranges.Add(1)
			http.ServeContent(w, r, "", time.Time{}, bytes.NewReader(f.body))

			return
		}

		f.downloads.Add(1)
		_, _ = w.Write(f.body)
	default:
		http.NotFound(w, r)
	}
}

func newTestArchive(t *testing.T, srv *httptest.Server, rangeSlots uint64) *Archive {
	t.Helper()

	cfg := &Config{
		Enabled:              true,
		Address:              srv.URL,
		Timeout:              human.Duration{Duration: 10 * time.Second},
		CoverageTTL:          human.Duration{Duration: time.Minute},
		MaxConcurrentFetches: 2,
		RangeSlots:           rangeSlots,
	}
	require.NoError(t, cfg.Validate())

	codec := func(context.Context) (*dynssz.DynSsz, error) { return dynssz.GetGlobalDynSsz(), nil }

	return New(cfg, logrus.New(), func() string { return testNetwork }, codec, testMetrics())
}

func testBlocks(t *testing.T) []testBlock {
	t.Helper()

	return []testBlock{phase0Block(t, 1000), phase0Block(t, 1001), altairBlock(t, 1003), altairBlock(t, 1006)}
}

func TestArchiveWholeBundle(t *testing.T) {
	fake := newFakeArchive(t, 1000, 1007, testBlocks(t))

	srv := httptest.NewServer(fake)
	defer srv.Close()

	a := newTestArchive(t, srv, 0)

	blocks, found, err := a.Blocks(context.Background(), 1003)
	require.NoError(t, err)
	require.True(t, found)
	require.Len(t, blocks, 8)

	for slot := phase0.Slot(1000); slot <= 1007; slot++ {
		switch slot {
		case 1000, 1001, 1003, 1006:
			require.NotNil(t, blocks[slot], "slot %d", slot)

			got, serr := blocks[slot].Slot()
			require.NoError(t, serr)
			assert.Equal(t, slot, got)
		default:
			v, ok := blocks[slot]
			assert.True(t, ok, "missed slot %d present", slot)
			assert.Nil(t, v, "missed slot %d is nil", slot)
		}
	}

	// A second slot of the same bundle reuses the cached descriptor.
	_, found, err = a.Blocks(context.Background(), 1005)
	require.NoError(t, err)
	require.True(t, found)
	assert.Equal(t, int64(1), fake.lookups.Load())
	assert.Equal(t, int64(2), fake.downloads.Load())
	assert.Equal(t, int64(0), fake.ranges.Load())
}

func TestArchiveRangeWindows(t *testing.T) {
	for _, ignoreRange := range []bool{false, true} {
		t.Run(fmt.Sprintf("ignoreRange=%v", ignoreRange), func(t *testing.T) {
			fake := newFakeArchive(t, 1000, 1007, testBlocks(t))
			fake.ignoreRange = ignoreRange

			srv := httptest.NewServer(fake)
			defer srv.Close()

			a := newTestArchive(t, srv, 4)

			blocks, found, err := a.Blocks(context.Background(), 1006)
			require.NoError(t, err)
			require.True(t, found)
			require.Len(t, blocks, 4)
			assert.NotNil(t, blocks[1006])
			assert.Nil(t, blocks[1004])
			assert.Nil(t, blocks[1007])

			blocks, found, err = a.Blocks(context.Background(), 1001)
			require.NoError(t, err)
			require.True(t, found)
			require.Len(t, blocks, 4)
			assert.NotNil(t, blocks[1000])
			assert.NotNil(t, blocks[1003])
			assert.Nil(t, blocks[1002])

			if ignoreRange {
				assert.Equal(t, int64(0), fake.ranges.Load())
			} else {
				// One index fetch, then one range per window.
				assert.Equal(t, int64(3), fake.ranges.Load())
				assert.Equal(t, int64(0), fake.downloads.Load())
			}
		})
	}
}

func TestArchiveNotCovered(t *testing.T) {
	fake := newFakeArchive(t, 1000, 1007, testBlocks(t))

	srv := httptest.NewServer(fake)
	defer srv.Close()

	a := newTestArchive(t, srv, 0)

	_, found, err := a.Blocks(context.Background(), 1003)
	require.NoError(t, err)
	require.True(t, found)

	// Above coverage: answered from cached coverage without asking the archive.
	_, found, err = a.Blocks(context.Background(), 5000)
	require.NoError(t, err)
	assert.False(t, found)
	assert.Equal(t, int64(1), fake.lookups.Load())

	// Below coverage: asked once, then negatively cached.
	_, found, err = a.Blocks(context.Background(), 10)
	require.NoError(t, err)
	assert.False(t, found)

	_, found, err = a.Blocks(context.Background(), 11)
	require.NoError(t, err)
	assert.False(t, found)
	assert.Equal(t, int64(2), fake.lookups.Load())
}

func TestArchiveFailureIsSkippedForCoverageTTL(t *testing.T) {
	fake := newFakeArchive(t, 1000, 1007, testBlocks(t))
	fake.failLookups.Store(true)

	srv := httptest.NewServer(fake)
	defer srv.Close()

	a := newTestArchive(t, srv, 0)

	_, found, err := a.Blocks(context.Background(), 1003)
	require.Error(t, err)
	assert.False(t, found)

	_, found, err = a.Blocks(context.Background(), 1004)
	require.NoError(t, err)
	assert.False(t, found)
	assert.Equal(t, int64(1), fake.lookups.Load())
}

func TestArchiveCorruptBundleErrors(t *testing.T) {
	blocks := testBlocks(t)
	blocks[1].root = "0x" + fmt.Sprintf("%064x", 42)
	fake := newFakeArchive(t, 1000, 1007, blocks)

	srv := httptest.NewServer(fake)
	defer srv.Close()

	a := newTestArchive(t, srv, 0)

	_, found, err := a.Blocks(context.Background(), 1000)
	require.ErrorContains(t, err, "does not match index root")
	assert.False(t, found)
}

func TestConfigValidate(t *testing.T) {
	require.NoError(t, (&Config{}).Validate())
	require.Error(t, (&Config{Enabled: true}).Validate())
	require.Error(t, (&Config{Enabled: true, Address: "http://x"}).Validate())
	require.NoError(t, (&Config{
		Enabled: true, Address: "http://x", MaxConcurrentFetches: 1,
		Timeout: human.Duration{Duration: time.Second}, CoverageTTL: human.Duration{Duration: time.Second},
	}).Validate())
}
