package blockarchive

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
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
	listing     string
	ignoreRange bool
	failListing atomic.Bool

	listings  atomic.Int64
	downloads atomic.Int64
	ranges    atomic.Int64
	requested sync.Map
}

func newFakeServer(files map[string][]byte) *fakeServer {
	f := &fakeServer{files: files}

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

	body, ok := f.files[name]
	if !ok {
		http.NotFound(w, r)

		return
	}

	f.requested.Store(name, true)

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

func newTestArchive(t *testing.T, cfg *Config) *Archive {
	t.Helper()

	require.NoError(t, cfg.Validate())

	chain := func(context.Context) (*Chain, error) { return testChain(), nil }

	return New(cfg, logrus.New(), func() string { return testNetwork }, chain, testMetrics())
}

func eraName(era uint64) string {
	return fmt.Sprintf("%s-%05d-%08x.era", testNetwork, era, era)
}

func sidecarName(era uint64) string {
	return strings.TrimSuffix(eraName(era), ".era") + ".envelopes.e2s"
}

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

func TestArchiveWholeEra(t *testing.T) {
	fake := newFakeServer(map[string][]byte{
		eraName(2): encodeEra(t, 2, testBlocksAt(t, 16, 17, 23, 24, 30), eraOptions{nimbusMissed: true}),
	})

	srv := httptest.NewServer(fake)
	defer srv.Close()

	a := newTestArchive(t, testConfig(srv.URL, 0))

	slots, found, err := a.Slots(context.Background(), 23)
	require.NoError(t, err)
	require.True(t, found)
	assertSlots(t, slots, 16, 31, 16, 17, 23, 24, 30)

	// Another slot of the era reuses the listing and the cached index.
	_, found, err = a.Slots(context.Background(), 31)
	require.NoError(t, err)
	require.True(t, found)

	assert.Equal(t, int64(1), fake.listings.Load())
	assert.Equal(t, int64(3), fake.ranges.Load(), "one index read, then one block-section read per fetch")
	assert.Equal(t, int64(0), fake.downloads.Load())
}

func TestArchiveRangeWindows(t *testing.T) {
	for _, ignoreRange := range []bool{false, true} {
		t.Run(fmt.Sprintf("ignoreRange=%v", ignoreRange), func(t *testing.T) {
			fake := newFakeServer(map[string][]byte{
				eraName(2): encodeEra(t, 2, testBlocksAt(t, 16, 17, 20, 22, 27), eraOptions{unknownEntries: true}),
			})
			fake.ignoreRange = ignoreRange

			srv := httptest.NewServer(fake)
			defer srv.Close()

			a := newTestArchive(t, testConfig(srv.URL, 4))

			slots, found, err := a.Slots(context.Background(), 22)
			require.NoError(t, err)
			require.True(t, found)
			assertSlots(t, slots, 20, 23, 20, 22)

			slots, found, err = a.Slots(context.Background(), 17)
			require.NoError(t, err)
			require.True(t, found)
			assertSlots(t, slots, 16, 19, 16, 17)

			// A window of only missed slots needs no block read.
			slots, found, err = a.Slots(context.Background(), 29)
			require.NoError(t, err)
			require.True(t, found)
			assertSlots(t, slots, 28, 31)

			if ignoreRange {
				assert.Equal(t, int64(0), fake.ranges.Load())
				assert.Equal(t, int64(3), fake.downloads.Load())
			} else {
				assert.Equal(t, int64(3), fake.ranges.Load(), "one index read, then one range per window with blocks")
				assert.Equal(t, int64(0), fake.downloads.Load())
			}
		})
	}
}

func TestArchiveNotCovered(t *testing.T) {
	fake := newFakeServer(map[string][]byte{
		eraName(2): encodeEra(t, 2, testBlocksAt(t, 16), eraOptions{}),
	})

	srv := httptest.NewServer(fake)
	defer srv.Close()

	a := newTestArchive(t, testConfig(srv.URL, 0))

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
	fake := newFakeServer(map[string][]byte{
		eraName(2): encodeEra(t, 2, testBlocksAt(t, 16), eraOptions{}),
	})
	fake.failListing.Store(true)

	srv := httptest.NewServer(fake)
	defer srv.Close()

	a := newTestArchive(t, testConfig(srv.URL, 0))

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

	fake := newFakeServer(map[string][]byte{
		eraName(2): encodeEra(t, 2, blocks, eraOptions{}),
		eraName(3): encodeEra(t, 3, testBlocksAt(t, 32), eraOptions{}),
	})

	srv := httptest.NewServer(fake)
	defer srv.Close()

	a := newTestArchive(t, testConfig(srv.URL, 0))

	_, found, err := a.Slots(context.Background(), 16)
	require.ErrorContains(t, err, "decodes to slot 17")
	require.ErrorIs(t, err, errCorrupt)
	assert.False(t, found)

	// Only the corrupt era is skipped; the archive stays available.
	assert.False(t, a.isUnavailable())

	_, found, err = a.Slots(context.Background(), 20)
	require.NoError(t, err)
	assert.False(t, found)

	_, found, err = a.Slots(context.Background(), 32)
	require.NoError(t, err)
	assert.True(t, found)
}

func TestArchiveEnvelopes(t *testing.T) {
	for _, rangeSlots := range []uint64{0, 4} {
		t.Run(fmt.Sprintf("rangeSlots=%d", rangeSlots), func(t *testing.T) {
			// Era 3 is slots 32..47 with gloas from slot 36; slot 38's payload is withheld and slot 39 is missed.
			pre := testBlocksAt(t, 32, 35)
			post := gloasChain(t, []uint64{36, 37, 38, 40, 41}, map[uint64]bool{38: true})
			blocks := append(pre, post...)

			fake := newFakeServer(map[string][]byte{
				eraName(3):     encodeEra(t, 3, blocks, eraOptions{}),
				sidecarName(3): encodeSidecar(t, 3, blocks),
			})

			srv := httptest.NewServer(fake)
			defer srv.Close()

			a := newTestArchive(t, testConfig(srv.URL, rangeSlots))

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

			assert.True(t, slots[38].EnvelopeKnown)
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

func TestArchiveWithoutSidecarServesBlocksOnly(t *testing.T) {
	blocks := gloasChain(t, []uint64{36, 37}, nil)

	fake := newFakeServer(map[string][]byte{eraName(3): encodeEra(t, 3, blocks, eraOptions{})})

	srv := httptest.NewServer(fake)
	defer srv.Close()

	a := newTestArchive(t, testConfig(srv.URL, 0))

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

	srv := httptest.NewServer(fake)
	defer srv.Close()

	a := newTestArchive(t, testConfig(srv.URL, 0))

	slots, found, err := a.Slots(context.Background(), 36)
	require.NoError(t, err)
	require.True(t, found, "blocks are still served")
	assert.NotNil(t, slots[37].Block)
	assert.False(t, slots[37].EnvelopeKnown)
	assert.True(t, fake.wasRequested(sidecarName(3)))

	has, err := a.HasEnvelopes(context.Background(), 36)
	require.NoError(t, err)
	assert.False(t, has, "the sidecar is skipped after failing to verify")
}

func TestArchiveSeparateListingURL(t *testing.T) {
	fake := newFakeServer(map[string][]byte{eraName(2): encodeEra(t, 2, testBlocksAt(t, 16), eraOptions{})})

	mux := http.NewServeMux()
	mux.Handle("/era/", fake)
	mux.HandleFunc("/api/v1/eras", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = fmt.Fprintf(w, `{"eras":[{"network":%q,"file":%q}]}`, testNetwork, eraName(2))
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	cfg := testConfig(srv.URL, 0)
	cfg.ListingURL = srv.URL + "/api/v1/eras"
	a := newTestArchive(t, cfg)

	_, found, err := a.Slots(context.Background(), 16)
	require.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, int64(0), fake.listings.Load())
}

func TestArchiveChainErrorFallsBack(t *testing.T) {
	cfg := testConfig("http://127.0.0.1:1", 0)
	require.NoError(t, cfg.Validate())

	chain := func(context.Context) (*Chain, error) { return nil, errors.New("beacon not ready") }
	a := New(cfg, logrus.New(), func() string { return testNetwork }, chain, testMetrics())

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
			Enabled: true, URL: "http://x", MaxConcurrentFetches: 1,
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
	}

	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			cfg := valid()
			mutate(cfg)
			require.Error(t, cfg.Validate())
		})
	}
}
