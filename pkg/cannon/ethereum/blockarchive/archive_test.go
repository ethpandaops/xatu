package blockarchive

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/creasty/defaults"
	"github.com/ethpandaops/beacon/pkg/human"
	"github.com/ethpandaops/go-eth2-client/spec/phase0"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	yaml "gopkg.in/yaml.v3"
)

var testMetrics = sync.OnceValue(func() *Metrics { return NewMetrics("xatu_cannon_test") })

// fakeServer serves era files and sidecars under /era/ with a directory listing, like a Nimbus era host.
type fakeServer struct {
	files       map[string][]byte
	status      map[string]int
	listing     string
	ignoreRange bool
	failListing atomic.Bool
	// gate, when set, holds every file request until it is closed; started reports each held request.
	gate    chan struct{}
	started chan struct{}

	listings  atomic.Int64
	downloads atomic.Int64
	ranges    atomic.Int64
	requested sync.Map
}

func newFakeServer(files map[string][]byte) *fakeServer {
	f := &fakeServer{files: files, status: map[string]int{}}

	var b strings.Builder

	b.WriteString("<html><body><pre>\n")

	for name := range files {
		fmt.Fprintf(&b, "<a href=%q>%s</a>\n", name, name)
	}

	b.WriteString("</pre></body></html>\n")
	f.listing = b.String()

	return f
}

func (f *fakeServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/era/" {
		f.listings.Add(1)

		if f.failListing.Load() {
			http.Error(w, "down", http.StatusBadGateway)

			return
		}

		_, _ = w.Write([]byte(f.listing))

		return
	}

	name := strings.TrimPrefix(r.URL.Path, "/era/")
	f.requested.Store(name, true)

	if f.gate != nil {
		f.started <- struct{}{}

		<-f.gate
	}

	if status, ok := f.status[name]; ok {
		http.Error(w, http.StatusText(status), status)

		return
	}

	body, ok := f.files[name]
	if !ok {
		http.NotFound(w, r)

		return
	}

	if r.Header.Get("Range") != "" && !f.ignoreRange {
		f.ranges.Add(1)
		http.ServeContent(w, r, "", time.Time{}, bytes.NewReader(body))

		return
	}

	f.downloads.Add(1)
	w.Header().Set("Content-Length", fmt.Sprint(len(body)))
	_, _ = w.Write(body)
}

func (f *fakeServer) wasRequested(name string) bool {
	_, ok := f.requested.Load(name)

	return ok
}

func testConfig(srvURL string, rangeSlots uint64) *Config {
	return &Config{
		Enabled:              true,
		URL:                  srvURL + "/era",
		Timeout:              human.Duration{Duration: 10 * time.Second},
		ListingTTL:           human.Duration{Duration: time.Minute},
		MaxConcurrentFetches: 2,
		RangeSlots:           rangeSlots,
	}
}

func newTestArchive(t *testing.T, cfg *Config, beacon *fakeBeacon) *Archive {
	t.Helper()

	require.NoError(t, cfg.Validate())

	chain := func(context.Context) (*Chain, error) { return testChain(), nil }

	return New(cfg, logrus.New(), func() string { return testNetwork }, chain, beacon.root, testMetrics())
}

func eraName(era uint64) string {
	return fmt.Sprintf("%s-%05d-%08x.era", testNetwork, era, era)
}

func sidecarName(era uint64) string {
	return strings.TrimSuffix(eraName(era), ".era") + ".envelopes.e2s"
}

// assertSlots checks slots holds exactly first..last, with blocks at present and missed slots elsewhere.
func assertSlots(t *testing.T, slots Slots, first, last uint64, present ...uint64) {
	t.Helper()

	require.Len(t, slots, int(last-first+1))

	want := make(map[uint64]bool, len(present))
	for _, s := range present {
		want[s] = true
	}

	for s := first; s <= last; s++ {
		content, ok := slots[phase0.Slot(s)]
		require.True(t, ok, "slot %d present in range", s)

		if !want[s] {
			assert.Nil(t, content.Block, "slot %d is missed", s)

			continue
		}

		require.NotNil(t, content.Block, "slot %d", s)

		got, err := content.Block.Slot()
		require.NoError(t, err)
		assert.Equal(t, phase0.Slot(s), got)
	}
}

func serve(t *testing.T, fake http.Handler) string {
	t.Helper()

	srv := httptest.NewServer(fake)
	t.Cleanup(srv.Close)

	return srv.URL
}

func TestArchiveWholeEra(t *testing.T) {
	blocks := testBlocksAt(t, 16, 17, 23, 24, 30)
	fake := newFakeServer(map[string][]byte{eraName(2): encodeEra(t, 2, blocks, eraOptions{nimbusMissed: true})})
	beacon := newFakeBeacon(blocks)
	a := newTestArchive(t, testConfig(serve(t, fake), 0), beacon)

	slots, found, err := a.Slots(context.Background(), 23)
	require.NoError(t, err)
	require.True(t, found)
	assertSlots(t, slots, 16, 30, 16, 17, 23, 24, 30)

	// Nothing in the file proves slot 31, after the era's last block, missed; the beacon node answers it.
	_, found, err = a.Slots(context.Background(), 31)
	require.NoError(t, err)
	require.False(t, found)

	assert.Equal(t, int64(1), fake.listings.Load())
	assert.Equal(t, int64(2), fake.ranges.Load(), "one index read and one block-section read")
	assert.Equal(t, int64(0), fake.downloads.Load())
	assert.Equal(t, int64(1), beacon.requests.Load(), "one beacon root per window")
}

func TestArchiveRangeWindows(t *testing.T) {
	for _, ignoreRange := range []bool{false, true} {
		t.Run(fmt.Sprintf("ignoreRange=%v", ignoreRange), func(t *testing.T) {
			blocks := testBlocksAt(t, 16, 17, 20, 22, 27)
			fake := newFakeServer(map[string][]byte{eraName(2): encodeEra(t, 2, blocks, eraOptions{unknownEntries: true})})
			fake.ignoreRange = ignoreRange
			beacon := newFakeBeacon(blocks)
			a := newTestArchive(t, testConfig(serve(t, fake), 4), beacon)

			// Blocks 17 and 27 either side of the window prove its missed slots.
			slots, found, err := a.Slots(context.Background(), 22)
			require.NoError(t, err)
			require.True(t, found)
			assertSlots(t, slots, 20, 23, 20, 22)

			slots, found, err = a.Slots(context.Background(), 17)
			require.NoError(t, err)
			require.True(t, found)
			assertSlots(t, slots, 16, 19, 16, 17)

			// Nothing after the window's missed slots proves them missed at the era's end.
			slots, found, err = a.Slots(context.Background(), 29)
			require.NoError(t, err)
			require.True(t, found)
			assert.Empty(t, slots)

			_, found, err = a.Slots(context.Background(), 30)
			require.NoError(t, err)
			assert.False(t, found, "an unprovable window is not fetched again")

			if ignoreRange {
				assert.Equal(t, int64(0), fake.ranges.Load())
				assert.Equal(t, int64(3), fake.downloads.Load())
			} else {
				assert.Equal(t, int64(3), fake.ranges.Load(), "one index read, then one range per provable window")
				assert.Equal(t, int64(0), fake.downloads.Load())
			}

			assert.Equal(t, int64(2), beacon.requests.Load())
		})
	}
}

func TestArchiveEmptyWindowBetweenBlocks(t *testing.T) {
	blocks := testBlocksAt(t, 17, 24)
	fake := newFakeServer(map[string][]byte{eraName(2): encodeEra(t, 2, blocks, eraOptions{})})
	a := newTestArchive(t, testConfig(serve(t, fake), 4), newFakeBeacon(blocks))

	slots, found, err := a.Slots(context.Background(), 21)
	require.NoError(t, err)
	require.True(t, found)
	assertSlots(t, slots, 20, 23)
}

func TestArchiveNotCovered(t *testing.T) {
	blocks := testBlocksAt(t, 16)
	fake := newFakeServer(map[string][]byte{eraName(2): encodeEra(t, 2, blocks, eraOptions{})})
	a := newTestArchive(t, testConfig(serve(t, fake), 0), newFakeBeacon(blocks))

	// The current era has no file yet, and neither do eras missing from the listing.
	for _, slot := range []phase0.Slot{40, 5, 100} {
		_, found, err := a.Slots(context.Background(), slot)
		require.NoError(t, err)
		assert.False(t, found, "slot %d", slot)
	}

	_, found, err := a.Slots(context.Background(), 16)
	require.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, int64(1), fake.listings.Load(), "the listing is trusted for ListingTTL")
}

func TestArchiveListingFailureIsSkippedForListingTTL(t *testing.T) {
	blocks := testBlocksAt(t, 16)
	fake := newFakeServer(map[string][]byte{eraName(2): encodeEra(t, 2, blocks, eraOptions{})})
	fake.failListing.Store(true)
	a := newTestArchive(t, testConfig(serve(t, fake), 0), newFakeBeacon(blocks))

	_, found, err := a.Slots(context.Background(), 16)
	require.Error(t, err)
	assert.False(t, found)

	_, found, err = a.Slots(context.Background(), 17)
	require.NoError(t, err)
	assert.False(t, found)
	assert.Equal(t, int64(1), fake.listings.Load())
}

func TestArchiveCorruptEraIsSkipped(t *testing.T) {
	// Slot 17's block sits where the index expects slot 18's.
	blocks := testBlocksAt(t, 16, 17)
	blocks[1].slot = 18
	era3 := testBlocksAt(t, 32)

	fake := newFakeServer(map[string][]byte{
		eraName(2): encodeEra(t, 2, blocks, eraOptions{}),
		eraName(3): encodeEra(t, 3, era3, eraOptions{}),
	})
	a := newTestArchive(t, testConfig(serve(t, fake), 0), newFakeBeacon(blocks, era3))

	_, found, err := a.Slots(context.Background(), 16)
	require.ErrorContains(t, err, "decodes to slot 17")
	require.ErrorIs(t, err, errCorrupt)
	assert.False(t, found)

	// Only the corrupt era is skipped, and its index forgotten; the archive stays available.
	assert.False(t, a.isUnavailable())
	assert.Nil(t, a.indexes.Get(eraName(2)))

	_, found, err = a.Slots(context.Background(), 20)
	require.NoError(t, err)
	assert.False(t, found)

	_, found, err = a.Slots(context.Background(), 32)
	require.NoError(t, err)
	assert.True(t, found)
}

func TestArchiveBrokenLinksAreCorrupt(t *testing.T) {
	blocks := testBlocksAt(t, 16, 17, 18)
	blocks[1] = linkedBlocks(t, phase0.Root{0x02}, []uint64{17}, nil)[0]

	fake := newFakeServer(map[string][]byte{eraName(2): encodeEra(t, 2, blocks, eraOptions{})})
	a := newTestArchive(t, testConfig(serve(t, fake), 0), newFakeBeacon(blocks))

	_, found, err := a.Slots(context.Background(), 16)
	require.ErrorIs(t, err, errCorrupt)
	require.ErrorContains(t, err, "slot 17 builds on")
	assert.False(t, found)
}

func TestArchiveMissingEraFileFallsBackAlone(t *testing.T) {
	era2, era3 := testBlocksAt(t, 16), testBlocksAt(t, 32)
	fake := newFakeServer(map[string][]byte{
		eraName(2): encodeEra(t, 2, era2, eraOptions{}),
		eraName(3): encodeEra(t, 3, era3, eraOptions{}),
	})
	fake.status[eraName(2)] = http.StatusNotFound
	a := newTestArchive(t, testConfig(serve(t, fake), 0), newFakeBeacon(era2, era3))

	_, found, err := a.Slots(context.Background(), 16)
	require.ErrorIs(t, err, errMissing)
	assert.False(t, found)
	assert.False(t, a.isUnavailable(), "a missing file does not take the archive down")

	_, found, err = a.Slots(context.Background(), 32)
	require.NoError(t, err)
	assert.True(t, found)
}

func TestArchiveServerErrorSkipsArchive(t *testing.T) {
	blocks := testBlocksAt(t, 16)
	fake := newFakeServer(map[string][]byte{eraName(2): encodeEra(t, 2, blocks, eraOptions{})})
	fake.status[eraName(2)] = http.StatusBadGateway
	a := newTestArchive(t, testConfig(serve(t, fake), 0), newFakeBeacon(blocks))

	_, found, err := a.Slots(context.Background(), 16)
	require.ErrorIs(t, err, errUnreachable)
	assert.False(t, found)
	assert.True(t, a.isUnavailable())
}

func TestArchiveNotCanonicalWindowIsSkipped(t *testing.T) {
	blocks := testBlocksAt(t, 16, 17, 20, 21)
	fork := linkedBlocks(t, phase0.Root{0x02}, []uint64{16, 17, 20, 21}, nil)

	for name, beacon := range map[string]*fakeBeacon{
		"other root": newFakeBeacon(blocks[:2], fork[2:]),
		"no block":   newFakeBeacon(blocks[:2]),
	} {
		t.Run(name, func(t *testing.T) {
			fake := newFakeServer(map[string][]byte{eraName(2): encodeEra(t, 2, blocks, eraOptions{})})
			a := newTestArchive(t, testConfig(serve(t, fake), 4), beacon)

			// Window 16..19 is tied to the beacon node by block 20, which it disagrees on.
			slots, found, err := a.Slots(context.Background(), 17)
			require.NoError(t, err)
			assert.False(t, found)
			assert.Nil(t, slots)

			ranges, requests := fake.ranges.Load(), beacon.requests.Load()

			_, found, err = a.Slots(context.Background(), 16)
			require.NoError(t, err)
			assert.False(t, found)
			assert.Equal(t, ranges, fake.ranges.Load(), "the window is skipped, not fetched again")
			assert.Equal(t, requests, beacon.requests.Load())
			assert.False(t, a.isUnavailable())
		})
	}
}

func TestArchiveBeaconErrorIsNotCached(t *testing.T) {
	blocks := testBlocksAt(t, 16)
	fake := newFakeServer(map[string][]byte{eraName(2): encodeEra(t, 2, blocks, eraOptions{})})
	beacon := newFakeBeacon(blocks)
	beacon.err = errors.New("beacon node down")
	a := newTestArchive(t, testConfig(serve(t, fake), 0), beacon)

	_, found, err := a.Slots(context.Background(), 16)
	require.ErrorContains(t, err, "beacon node down")
	assert.False(t, found)
	assert.False(t, a.isUnavailable(), "a beacon failure is not the archive's")
	assert.Equal(t, int64(1), fake.ranges.Load(), "the window is not downloaded before the beacon answers")

	beacon.err = nil

	_, found, err = a.Slots(context.Background(), 16)
	require.NoError(t, err)
	assert.True(t, found)
}

func TestArchiveCallerCancelDoesNotFailSharedFetch(t *testing.T) {
	blocks := testBlocksAt(t, 16)
	fake := newFakeServer(map[string][]byte{eraName(2): encodeEra(t, 2, blocks, eraOptions{})})
	fake.gate, fake.started = make(chan struct{}), make(chan struct{}, 8)
	a := newTestArchive(t, testConfig(serve(t, fake), 0), newFakeBeacon(blocks))

	ctx, cancel := context.WithCancel(context.Background())
	first := make(chan error, 1)

	go func() {
		_, _, err := a.Slots(ctx, 16)
		first <- err
	}()

	<-fake.started

	type result struct {
		found bool
		err   error
	}

	second := make(chan result, 1)

	go func() {
		_, found, err := a.Slots(context.Background(), 16)
		second <- result{found, err}
	}()

	cancel()
	require.ErrorIs(t, <-first, context.Canceled)

	close(fake.gate)

	res := <-second
	require.NoError(t, res.err)
	assert.True(t, res.found, "the shared fetch outlives the cancelled caller")
	assert.False(t, a.isUnavailable())
}

func TestArchiveEnvelopes(t *testing.T) {
	for _, rangeSlots := range []uint64{0, 4} {
		t.Run(fmt.Sprintf("rangeSlots=%d", rangeSlots), func(t *testing.T) {
			// Era 3 is slots 32..47 with gloas from slot 36; slot 38's payload is withheld and slot 39 is missed.
			blocks := gloasChain(t, []uint64{32, 35, 36, 37, 38, 40, 41}, map[uint64]bool{38: true})

			fake := newFakeServer(map[string][]byte{
				eraName(3):     encodeEra(t, 3, blocks, eraOptions{}),
				sidecarName(3): encodeSidecar(t, 3, blocks),
			})
			a := newTestArchive(t, testConfig(serve(t, fake), rangeSlots), newFakeBeacon(blocks))

			for slot, want := range map[phase0.Slot]bool{32: false, 36: true, 47: true} {
				has, err := a.HasEnvelopes(context.Background(), slot)
				require.NoError(t, err)
				assert.Equal(t, want, has, "slot %d", slot)
			}

			slots, found, err := a.Slots(context.Background(), 37)
			require.NoError(t, err)
			require.True(t, found)

			require.True(t, slots[37].EnvelopeKnown)
			require.NotNil(t, slots[37].Envelope)
			assert.Equal(t, testHash(37), slots[37].Envelope.Message.Payload.BlockHash)
			assert.True(t, slots[36].EnvelopeKnown)

			slots, found, err = a.Slots(context.Background(), 38)
			require.NoError(t, err)
			require.True(t, found)

			assert.True(t, slots[38].EnvelopeKnown, "block 40's bid proves the payload withheld")
			assert.Nil(t, slots[38].Envelope, "a withheld payload has no envelope")
			assert.NotNil(t, slots[38].Block)
			assert.False(t, slots[39].EnvelopeKnown, "a missed slot has no envelope")

			slots, found, err = a.Slots(context.Background(), 35)
			require.NoError(t, err)
			require.True(t, found)
			assert.False(t, slots[35].EnvelopeKnown, "pre-gloas blocks have no envelope")
			assert.NotNil(t, slots[35].Block)
		})
	}
}

func TestArchiveWithheldAtWindowAndEraEnds(t *testing.T) {
	for _, rangeSlots := range []uint64{0, 4} {
		t.Run(fmt.Sprintf("rangeSlots=%d", rangeSlots), func(t *testing.T) {
			// Slot 39 ends window 36..39 and slot 47 ends era 3, both with their payloads withheld.
			blocks := gloasChain(t, []uint64{36, 39, 40, 47}, map[uint64]bool{39: true, 47: true})

			fake := newFakeServer(map[string][]byte{
				eraName(3):     encodeEra(t, 3, blocks, eraOptions{}),
				sidecarName(3): encodeSidecar(t, 3, blocks),
			})
			a := newTestArchive(t, testConfig(serve(t, fake), rangeSlots), newFakeBeacon(blocks))

			slots, found, err := a.Slots(context.Background(), 39)
			require.NoError(t, err)
			require.True(t, found)
			assert.True(t, slots[39].EnvelopeKnown, "block 40, after the window, proves slot 39 withheld")
			assert.Nil(t, slots[39].Envelope)

			slots, found, err = a.Slots(context.Background(), 47)
			require.NoError(t, err)
			require.True(t, found)
			require.NotNil(t, slots[47].Block)
			assert.False(t, slots[47].EnvelopeKnown, "no block in the era proves slot 47 withheld")

			has, err := a.HasEnvelopes(context.Background(), 47)
			require.NoError(t, err)
			assert.False(t, has, "the beacon node answers for slot 47's envelope without another fetch")
		})
	}
}

func TestArchiveSidecarMissingAppliedEnvelopeAtWindowEnd(t *testing.T) {
	// The sidecar leaves out slot 38's envelope, so claims its payload withheld, which block 40 contradicts.
	blocks := gloasChain(t, []uint64{36, 37, 38, 40}, nil)
	sidecarBlocks := append([]testBlock(nil), blocks...)
	sidecarBlocks[2].envelope = nil

	fake := newFakeServer(map[string][]byte{
		eraName(3):     encodeEra(t, 3, blocks, eraOptions{}),
		sidecarName(3): encodeSidecar(t, 3, sidecarBlocks),
	})
	a := newTestArchive(t, testConfig(serve(t, fake), 4), newFakeBeacon(blocks))

	slots, found, err := a.Slots(context.Background(), 38)
	require.NoError(t, err)
	require.True(t, found, "blocks are still served")
	assert.NotNil(t, slots[38].Block)
	assert.False(t, slots[38].EnvelopeKnown)
	assert.False(t, slots[36].EnvelopeKnown, "the whole sidecar is skipped")
}

func TestArchiveWithoutSidecarServesBlocksOnly(t *testing.T) {
	blocks := gloasChain(t, []uint64{36, 37}, nil)
	fake := newFakeServer(map[string][]byte{eraName(3): encodeEra(t, 3, blocks, eraOptions{})})
	a := newTestArchive(t, testConfig(serve(t, fake), 0), newFakeBeacon(blocks))

	has, err := a.HasEnvelopes(context.Background(), 36)
	require.NoError(t, err)
	assert.False(t, has)

	slots, found, err := a.Slots(context.Background(), 36)
	require.NoError(t, err)
	require.True(t, found)
	assert.NotNil(t, slots[36].Block)
	assert.False(t, slots[36].EnvelopeKnown)
	assert.False(t, slots[37].EnvelopeKnown)
}

func TestArchiveCorruptSidecarIsSkipped(t *testing.T) {
	blocks := gloasChain(t, []uint64{36, 37, 38}, nil)
	// The sidecar claims slot 37's payload was withheld, which slot 38's bid contradicts.
	sidecarBlocks := append([]testBlock(nil), blocks...)
	sidecarBlocks[1].envelope = nil

	fake := newFakeServer(map[string][]byte{
		eraName(3):     encodeEra(t, 3, blocks, eraOptions{}),
		sidecarName(3): encodeSidecar(t, 3, sidecarBlocks),
	})
	a := newTestArchive(t, testConfig(serve(t, fake), 0), newFakeBeacon(blocks))

	slots, found, err := a.Slots(context.Background(), 36)
	require.NoError(t, err)
	require.True(t, found, "blocks are still served")
	assert.NotNil(t, slots[37].Block)
	assert.False(t, slots[37].EnvelopeKnown)
	assert.True(t, fake.wasRequested(sidecarName(3)))
	assert.Nil(t, a.indexes.Get(sidecarName(3)), "a skipped sidecar's index is forgotten")

	has, err := a.HasEnvelopes(context.Background(), 36)
	require.NoError(t, err)
	assert.False(t, has, "the sidecar is skipped after failing to verify")
}

func TestArchiveSidecarFetchErrorServesBlocks(t *testing.T) {
	for _, status := range []int{http.StatusNotFound, http.StatusInternalServerError} {
		t.Run(strconv.Itoa(status), func(t *testing.T) {
			blocks := gloasChain(t, []uint64{36, 37}, nil)
			fake := newFakeServer(map[string][]byte{
				eraName(3):     encodeEra(t, 3, blocks, eraOptions{}),
				sidecarName(3): encodeSidecar(t, 3, blocks),
			})
			fake.status[sidecarName(3)] = status
			a := newTestArchive(t, testConfig(serve(t, fake), 0), newFakeBeacon(blocks))

			slots, found, err := a.Slots(context.Background(), 36)
			require.NoError(t, err)
			require.True(t, found)
			assert.NotNil(t, slots[36].Block)
			assert.False(t, slots[36].EnvelopeKnown)
			assert.False(t, a.isUnavailable(), "a sidecar failure does not stop block serving")

			file, err := a.lookup(context.Background(), testNetwork, 3)
			require.NoError(t, err)
			assert.Equal(t, eraFile{name: eraName(3)}, file, "the sidecar is skipped")
		})
	}
}

// blockArchiverAPI lists eras 1..lastEra like block-archiver's /api/v1/era?network=: all of them, or a page with limit.
func blockArchiverAPI(lastEra uint64, sidecars map[uint64]bool, requests *atomic.Int64) http.HandlerFunc {
	type file struct {
		File string `json:"file"`
		URL  string `json:"url"`
	}

	type era struct {
		Era   uint64           `json:"era"`
		Files map[string]*file `json:"files"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)

		var limit *int

		offset := 0

		if v, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil {
			limit = new(int)
			*limit = max(1, v)
		}

		if v, err := strconv.Atoi(r.URL.Query().Get("offset")); err == nil {
			offset = v
		}

		eras := []era{}

		for n := uint64(offset) + 1; r.URL.Query().Get("network") == testNetwork && n <= lastEra && (limit == nil || len(eras) < *limit); n++ {
			e := era{Era: n, Files: map[string]*file{"era": {eraName(n), "/era/" + testNetwork + "/" + eraName(n)}, "envelopes": nil}}
			if sidecars[n] {
				e.Files["envelopes"] = &file{sidecarName(n), "/era/" + testNetwork + "/" + sidecarName(n)}
			}

			eras = append(eras, e)
		}

		_ = json.NewEncoder(w).Encode(map[string]any{"network": testNetwork, "eras": eras, "limit": limit, "offset": offset})
	}
}

func TestArchiveBlockArchiverListing(t *testing.T) {
	era2 := testBlocksAt(t, 16)
	blocks := gloasChain(t, []uint64{36, 37, 38, 40}, map[uint64]bool{38: true})
	fake := newFakeServer(map[string][]byte{
		eraName(2):     encodeEra(t, 2, era2, eraOptions{}),
		eraName(3):     encodeEra(t, 3, blocks, eraOptions{}),
		sidecarName(3): encodeSidecar(t, 3, blocks),
	})

	for query, pages := range map[string]int64{"": 1, "&limit=100": 3} {
		t.Run("listing"+query, func(t *testing.T) {
			var listings atomic.Int64

			mux := http.NewServeMux()
			mux.HandleFunc("/api/v1/era", blockArchiverAPI(250, map[uint64]bool{3: true}, &listings))
			mux.Handle("/era/"+testNetwork+"/", http.StripPrefix("/era/"+testNetwork, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				r.URL.Path = "/era" + r.URL.Path
				fake.ServeHTTP(w, r)
			})))

			srvURL := serve(t, mux)
			cfg := testConfig(srvURL, 0)
			cfg.URL = srvURL + "/era/" + testNetwork
			cfg.ListingURL = srvURL + "/api/v1/era?network=" + testNetwork + query
			a := newTestArchive(t, cfg, newFakeBeacon(era2, blocks))

			slots, found, err := a.Slots(context.Background(), 36)
			require.NoError(t, err)
			require.True(t, found)
			assert.Equal(t, pages, listings.Load(), "listing requests for eras 1..250")

			require.NotNil(t, slots[36].Envelope)
			assert.True(t, slots[38].EnvelopeKnown)
			assert.Nil(t, slots[38].Envelope, "withheld payload")

			file, err := a.lookup(context.Background(), testNetwork, 250)
			require.NoError(t, err)
			assert.Equal(t, eraFile{name: eraName(250)}, file, "the newest era is listed")

			has, err := a.HasEnvelopes(context.Background(), 16)
			require.NoError(t, err)
			assert.False(t, has, "era 2 has no sidecar")
		})
	}

	assert.Equal(t, int64(0), fake.listings.Load(), "the era directory is never listed")
}

func TestArchiveListingPagesAreBounded(t *testing.T) {
	var listings atomic.Int64

	srvURL := serve(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		listings.Add(1)

		_, _ = fmt.Fprintf(w, `{"eras": [{"era": 2, "files": {"era": {"file": %q}}}], "limit": 1, "offset": 0}`, eraName(2))
	}))

	cfg := testConfig(srvURL, 0)
	cfg.ListingURL = srvURL + "/api/v1/era"
	a := newTestArchive(t, cfg, newFakeBeacon())

	_, found, err := a.Slots(context.Background(), 16)
	require.ErrorContains(t, err, "pages")
	assert.False(t, found)
	assert.Equal(t, int64(maxListingPages), listings.Load())
}

func TestArchiveOversizedListingIsAnError(t *testing.T) {
	srvURL := serve(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = fmt.Fprintf(w, "<a href=%q></a>", eraName(2))
		_, _ = io.CopyN(w, zeros{}, maxListingSize)
	}))

	a := newTestArchive(t, testConfig(srvURL, 0), newFakeBeacon())

	_, found, err := a.Slots(context.Background(), 16)
	require.ErrorContains(t, err, "larger than")
	assert.False(t, found)
}

type zeros struct{}

func (zeros) Read(p []byte) (int, error) {
	clear(p)

	return len(p), nil
}

func TestArchiveChainErrorFallsBack(t *testing.T) {
	cfg := testConfig("http://127.0.0.1:1", 0)
	require.NoError(t, cfg.Validate())

	chain := func(context.Context) (*Chain, error) { return nil, errors.New("beacon not ready") }
	a := New(cfg, logrus.New(), func() string { return testNetwork }, chain, newFakeBeacon().root, testMetrics())

	_, found, err := a.Slots(context.Background(), 16)
	require.ErrorContains(t, err, "beacon not ready")
	assert.False(t, found)

	has, err := a.HasEnvelopes(context.Background(), 16)
	require.Error(t, err)
	assert.False(t, has)
}

func TestConfigDefaultsAndYAML(t *testing.T) {
	cfg := &Config{}
	require.NoError(t, defaults.Set(cfg))

	assert.False(t, cfg.Enabled)
	assert.Equal(t, 5*time.Minute, cfg.Timeout.Duration)
	assert.Equal(t, 5*time.Minute, cfg.ListingTTL.Duration)
	assert.Equal(t, 2, cfg.MaxConcurrentFetches)
	assert.Equal(t, uint64(256), cfg.RangeSlots)
	assert.Equal(t, uint64(256), cfg.MaxFetchSlots())

	require.NoError(t, yaml.Unmarshal([]byte("enabled: true\nurl: https://mainnet.era.nimbus.team\nrangeSlots: 0\ntimeout: 30s\n"), cfg))
	require.NoError(t, cfg.Validate())
	assert.Equal(t, uint64(0), cfg.RangeSlots)
	assert.Equal(t, uint64(8192), cfg.MaxFetchSlots(), "whole eras are at most a mainnet-preset era")
	assert.Equal(t, 30*time.Second, cfg.Timeout.Duration)
	assert.Equal(t, 5*time.Minute, cfg.ListingTTL.Duration)
}

func TestConfigValidate(t *testing.T) {
	valid := func() *Config {
		return &Config{
			Enabled: true, URL: "http://x", MaxConcurrentFetches: 1, RangeSlots: 8192,
			Timeout: human.Duration{Duration: time.Second}, ListingTTL: human.Duration{Duration: time.Second},
		}
	}

	require.NoError(t, (&Config{}).Validate(), "a disabled config needs nothing")
	require.NoError(t, valid().Validate())

	tests := map[string]func(*Config){
		"missing url":        func(c *Config) { c.URL = "" },
		"relative url":       func(c *Config) { c.URL = "/era" },
		"bad listing url":    func(c *Config) { c.ListingURL = "ftp://x/list" },
		"no fetches":         func(c *Config) { c.MaxConcurrentFetches = 0 },
		"no timeout":         func(c *Config) { c.Timeout.Duration = 0 },
		"no listing ttl":     func(c *Config) { c.ListingTTL.Duration = 0 },
		"negative timeout":   func(c *Config) { c.Timeout.Duration = -time.Second },
		"listing url scheme": func(c *Config) { c.ListingURL = "archive/list" },
		"range over an era":  func(c *Config) { c.RangeSlots = 8193 },
		"huge range":         func(c *Config) { c.RangeSlots = 1<<64 - 1 },
	}

	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			cfg := valid()
			mutate(cfg)
			require.Error(t, cfg.Validate())
		})
	}
}
