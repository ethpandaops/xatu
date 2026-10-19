package blockarchive

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ethpandaops/go-eth2-client/spec"
	"github.com/ethpandaops/go-eth2-client/spec/phase0"
	"github.com/jellydator/ttlcache/v3"
	dynssz "github.com/pk910/dynamic-ssz"
	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/singleflight"

	"github.com/ethpandaops/xatu/pkg/observability"
)

const (
	// lookupTimeout bounds a discovery request so a slow archive falls back to the beacon node quickly.
	lookupTimeout   = 15 * time.Second
	indexCacheSize  = 16
	modeBundle      = "bundle"
	modeRange       = "range"
	resultHit       = "hit"
	resultUncovered = "not_covered"
)

// Blocks maps each slot of a fetched bundle range to its canonical block, nil for a missed slot.
type Blocks map[phase0.Slot]*spec.VersionedSignedBeaconBlock

// Descriptor is a published bundle as listed by the archive's /api/v1/bundles.
//
//nolint:tagliatelle // block-archiver's API uses snake_case keys.
type Descriptor struct {
	StartSlot  uint64 `json:"start_slot"`
	EndSlot    uint64 `json:"end_slot"`
	Span       uint64 `json:"span"`
	URL        string `json:"url"`
	SizeBytes  int64  `json:"size_bytes"`
	DataOffset int64  `json:"data_offset"`
}

func (d *Descriptor) covers(slot uint64) bool {
	return d.StartSlot <= slot && slot <= d.EndSlot
}

//nolint:tagliatelle // block-archiver's API uses snake_case keys.
type bundlesResponse struct {
	Bundles  []Descriptor `json:"bundles"`
	Coverage struct {
		MaxSlot *uint64 `json:"max_slot"`
		Span    *uint64 `json:"span"`
	} `json:"coverage"`
}

type coverage struct {
	maxSlot *uint64
	span    uint64
	fetched time.Time
}

// Archive serves finalized beacon blocks from block-archiver bundles. It only
// answers for slots inside a published bundle; callers fall back to the beacon
// node otherwise.
type Archive struct {
	cfg     *Config
	log     observability.ContextualLogger
	client  *http.Client
	base    string
	network func() string
	codec   func(context.Context) (*dynssz.DynSsz, error)
	metrics *Metrics

	fetchGroup  singleflight.Group
	lookupGroup singleflight.Group
	sem         chan struct{}

	mu          sync.Mutex
	coverage    coverage
	bundles     map[uint64]*Descriptor
	unavailable time.Time
	notCovered  *ttlcache.Cache[uint64, struct{}]
	indexes     *ttlcache.Cache[string, *indexEntry]
}

type indexEntry struct {
	index      *Index
	dataOffset int64
}

// New creates an Archive. network returns the archive network name and codec
// the SSZ codec for the network's preset; both are resolved per request so
// they can depend on beacon node metadata that is not ready at construction.
func New(
	cfg *Config,
	log observability.ContextualLogger,
	network func() string,
	codec func(context.Context) (*dynssz.DynSsz, error),
	metrics *Metrics,
) *Archive {
	return &Archive{
		cfg:     cfg,
		log:     log.WithField("module", "cannon/ethereum/blockarchive"),
		client:  &http.Client{},
		base:    strings.TrimRight(cfg.Address, "/"),
		network: network,
		codec:   codec,
		metrics: metrics,
		sem:     make(chan struct{}, cfg.MaxConcurrentFetches),
		bundles: make(map[uint64]*Descriptor),
		notCovered: ttlcache.New(
			ttlcache.WithTTL[uint64, struct{}](cfg.CoverageTTL.Duration),
			ttlcache.WithDisableTouchOnHit[uint64, struct{}](),
		),
		indexes: ttlcache.New(
			ttlcache.WithCapacity[string, *indexEntry](indexCacheSize),
		),
	}
}

// Blocks returns the decoded blocks of the bundle range holding slot: the
// whole bundle, or its RangeSlots-aligned window when RangeSlots is set. found
// is false when no published bundle covers slot, or the archive failed within
// the last CoverageTTL and is being skipped.
func (a *Archive) Blocks(ctx context.Context, slot phase0.Slot) (blocks Blocks, found bool, err error) {
	network := a.network()

	if a.isUnavailable() {
		a.metrics.lookup(network, "unavailable")

		return nil, false, nil
	}

	desc, err := a.lookup(ctx, uint64(slot))
	if err != nil {
		a.markUnavailable()
		a.metrics.lookup(network, "error")

		return nil, false, err
	}

	if desc == nil {
		a.metrics.lookup(network, resultUncovered)

		return nil, false, nil
	}

	first, last := a.window(desc, uint64(slot))

	v, err, _ := a.fetchGroup.Do(fmt.Sprintf("%d-%d", first, last), func() (any, error) {
		select {
		case a.sem <- struct{}{}:
		case <-ctx.Done():
			return nil, ctx.Err()
		}

		defer func() { <-a.sem }()

		fetchCtx, cancel := context.WithTimeout(ctx, a.cfg.Timeout.Duration)
		defer cancel()

		return a.load(fetchCtx, network, desc, first, last)
	})
	if err != nil {
		a.markUnavailable()
		a.metrics.lookup(network, "error")

		return nil, false, err
	}

	blocks, ok := v.(Blocks)
	if !ok {
		return nil, false, fmt.Errorf("unexpected bundle result type %T", v)
	}

	a.metrics.lookup(network, resultHit)

	return blocks, true, nil
}

func (a *Archive) window(desc *Descriptor, slot uint64) (first, last uint64) {
	if a.cfg.RangeSlots == 0 {
		return desc.StartSlot, desc.EndSlot
	}

	first = desc.StartSlot + (slot-desc.StartSlot)/a.cfg.RangeSlots*a.cfg.RangeSlots
	last = min(first+a.cfg.RangeSlots-1, desc.EndSlot)

	return first, last
}

func (a *Archive) markUnavailable() {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.unavailable = time.Now().Add(a.cfg.CoverageTTL.Duration)
}

func (a *Archive) isUnavailable() bool {
	a.mu.Lock()
	defer a.mu.Unlock()

	return time.Now().Before(a.unavailable)
}

// lookup returns the bundle covering slot, or nil when none does.
func (a *Archive) lookup(ctx context.Context, slot uint64) (*Descriptor, error) {
	if desc, decided := a.cachedLookup(slot); decided {
		return desc, nil
	}

	key := strconv.FormatUint(slot, 10)
	if span := a.span(); span > 0 {
		key = "bundle-" + strconv.FormatUint(slot/span, 10)
	}

	_, err, _ := a.lookupGroup.Do(key, func() (any, error) {
		return nil, a.refresh(ctx, slot)
	})
	if err != nil {
		return nil, err
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	return a.findLocked(slot), nil
}

func (a *Archive) span() uint64 {
	a.mu.Lock()
	defer a.mu.Unlock()

	return a.coverage.span
}

// cachedLookup answers from cached descriptors and coverage. decided is false
// when the archive has to be asked.
func (a *Archive) cachedLookup(slot uint64) (desc *Descriptor, decided bool) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if d := a.findLocked(slot); d != nil {
		return d, true
	}

	if time.Since(a.coverage.fetched) < a.cfg.CoverageTTL.Duration &&
		(a.coverage.maxSlot == nil || slot > *a.coverage.maxSlot) {
		return nil, true
	}

	if a.coverage.span > 0 && a.notCovered.Has(slot/a.coverage.span) {
		return nil, true
	}

	return nil, false
}

func (a *Archive) findLocked(slot uint64) *Descriptor {
	if a.coverage.span == 0 {
		return nil
	}

	if d, ok := a.bundles[slot/a.coverage.span*a.coverage.span]; ok && d.covers(slot) {
		return d
	}

	return nil
}

// refresh asks the archive for the bundle covering slot and its current coverage.
func (a *Archive) refresh(ctx context.Context, slot uint64) error {
	ctx, cancel := context.WithTimeout(ctx, lookupTimeout)
	defer cancel()

	q := url.Values{}
	q.Set("network", a.network())
	q.Set("slot", strconv.FormatUint(slot, 10))

	resp, err := a.get(ctx, a.base+"/api/v1/bundles?"+q.Encode(), "")
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bundle lookup for slot %d: unexpected status %d", slot, resp.StatusCode)
	}

	var out bundlesResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return fmt.Errorf("failed to decode bundle lookup: %w", err)
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	a.coverage = coverage{maxSlot: out.Coverage.MaxSlot, fetched: time.Now()}
	if out.Coverage.Span != nil {
		a.coverage.span = *out.Coverage.Span
	}

	for i := range out.Bundles {
		d := out.Bundles[i]
		if d.Span == a.coverage.span && d.URL != "" {
			a.bundles[d.StartSlot] = &d
		}
	}

	if a.findLocked(slot) == nil && a.coverage.span > 0 {
		a.notCovered.Set(slot/a.coverage.span, struct{}{}, ttlcache.DefaultTTL)
	}

	return nil
}

// load downloads and decodes slots first..last of a bundle.
func (a *Archive) load(ctx context.Context, network string, desc *Descriptor, first, last uint64) (Blocks, error) {
	start := time.Now()
	mode := modeRange

	if first == desc.StartSlot && last == desc.EndSlot {
		mode = modeBundle
	}

	blocks, size, err := a.loadRange(ctx, desc, first, last, mode == modeBundle)
	a.metrics.fetched(network, mode, time.Since(start), size, len(blocks), err)

	if err != nil {
		return nil, fmt.Errorf("bundle %d..%d: %w", desc.StartSlot, desc.EndSlot, err)
	}

	a.log.WithField("slots", fmt.Sprintf("%d..%d", first, last)).
		WithField("mode", mode).
		WithField("bytes", size).
		WithField("duration", time.Since(start).String()).
		WithContext(ctx).
		Debug("Loaded blocks from block archive")

	return blocks, nil
}

func (a *Archive) loadRange(ctx context.Context, desc *Descriptor, first, last uint64, whole bool) (Blocks, int64, error) {
	ds, err := a.codec(ctx)
	if err != nil {
		return nil, 0, err
	}

	var (
		idx  *Index
		data []byte
		base int64
	)

	if whole {
		idx, data, err = a.fetchBundle(ctx, desc)
	} else {
		idx, data, base, err = a.fetchWindow(ctx, desc, first, last)
	}

	if err != nil {
		return nil, int64(len(data)), err
	}

	blocks, err := decodeEntries(ds, idx.entries(first, last), data, base)

	return blocks, int64(len(data)), err
}

// fetchBundle downloads a whole bundle, returning its index and data section.
func (a *Archive) fetchBundle(ctx context.Context, desc *Descriptor) (*Index, []byte, error) {
	resp, err := a.get(ctx, a.base+desc.URL, "")
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	br := bufio.NewReaderSize(resp.Body, 1<<20)

	idx, dataOffset, err := readIndex(br)
	if err != nil {
		return nil, nil, err
	}

	if desc.SizeBytes <= dataOffset {
		data, rerr := io.ReadAll(br)

		return idx, data, rerr
	}

	data := make([]byte, desc.SizeBytes-dataOffset)
	if _, err := io.ReadFull(br, data); err != nil {
		return nil, nil, fmt.Errorf("failed to read bundle data: %w", err)
	}

	return idx, data, nil
}

// fetchWindow fetches the gzip members of slots first..last with one Range
// request. base is the data-section offset of the first returned byte.
func (a *Archive) fetchWindow(ctx context.Context, desc *Descriptor, first, last uint64) (idx *Index, data []byte, base int64, err error) {
	ie, err := a.index(ctx, desc)
	if err != nil {
		return nil, nil, 0, err
	}

	var lo, hi int64 = -1, -1

	for _, e := range ie.index.entries(first, last) {
		if e.Missed {
			continue
		}

		if lo < 0 {
			lo = e.Offset
		}

		hi = e.Offset + e.Length
	}

	if lo < 0 {
		return ie.index, nil, 0, nil
	}

	data, err = a.getRange(ctx, a.base+desc.URL, ie.dataOffset+lo, ie.dataOffset+hi-1)
	if err != nil {
		return nil, nil, 0, err
	}

	return ie.index, data, lo, nil
}

// index fetches and caches a bundle's index with a Range request over its header.
func (a *Archive) index(ctx context.Context, desc *Descriptor) (*indexEntry, error) {
	if item := a.indexes.Get(desc.URL); item != nil {
		return item.Value(), nil
	}

	if desc.DataOffset <= headerSize {
		return nil, fmt.Errorf("bundle descriptor has invalid data offset %d", desc.DataOffset)
	}

	raw, err := a.getRange(ctx, a.base+desc.URL, 0, desc.DataOffset-1)
	if err != nil {
		return nil, err
	}

	idx, dataOffset, err := readIndex(bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}

	ie := &indexEntry{index: idx, dataOffset: dataOffset}
	a.indexes.Set(desc.URL, ie, ttlcache.DefaultTTL)

	return ie, nil
}

// getRange returns bytes first..last of a URL. A server that ignores Range and
// answers 200 is handled by slicing the full body.
func (a *Archive) getRange(ctx context.Context, target string, first, last int64) ([]byte, error) {
	resp, err := a.get(ctx, target, fmt.Sprintf("bytes=%d-%d", first, last))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusPartialContent:
		if !strings.HasPrefix(resp.Header.Get("Content-Range"), fmt.Sprintf("bytes %d-%d/", first, last)) {
			return nil, fmt.Errorf("unexpected Content-Range %q for bytes %d-%d", resp.Header.Get("Content-Range"), first, last)
		}

		data := make([]byte, last-first+1)
		if _, err := io.ReadFull(resp.Body, data); err != nil {
			return nil, fmt.Errorf("failed to read range: %w", err)
		}

		return data, nil
	case http.StatusOK:
		if _, err := io.CopyN(io.Discard, resp.Body, first); err != nil {
			return nil, fmt.Errorf("failed to skip to range start: %w", err)
		}

		data := make([]byte, last-first+1)
		if _, err := io.ReadFull(resp.Body, data); err != nil {
			return nil, fmt.Errorf("failed to read range: %w", err)
		}

		return data, nil
	default:
		return nil, fmt.Errorf("unexpected status %d for range request", resp.StatusCode)
	}
}

func (a *Archive) get(ctx context.Context, target, byteRange string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, http.NoBody)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "xatu-cannon")

	for k, v := range a.cfg.Headers {
		req.Header.Set(k, v)
	}

	if byteRange != "" {
		req.Header.Set("Range", byteRange)
	}

	return a.client.Do(req)
}

// decodeEntries decodes entries in parallel. data holds gzip members starting
// at data-section offset base.
func decodeEntries(ds *dynssz.DynSsz, entries []Entry, data []byte, base int64) (Blocks, error) {
	decoded := make([]*spec.VersionedSignedBeaconBlock, len(entries))

	g := errgroup.Group{}
	g.SetLimit(runtime.GOMAXPROCS(0))

	for i := range entries {
		e := &entries[i]
		if e.Missed {
			continue
		}

		lo, hi := e.Offset-base, e.Offset-base+e.Length
		if lo < 0 || hi > int64(len(data)) {
			return nil, fmt.Errorf("slot %d: byte range outside fetched data", e.Slot)
		}

		g.Go(func() error {
			block, err := decodeEntry(ds, e, data[lo:hi])
			if err != nil {
				return err
			}

			decoded[i] = block

			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}

	blocks := make(Blocks, len(entries))
	for i := range entries {
		blocks[phase0.Slot(entries[i].Slot)] = decoded[i]
	}

	return blocks, nil
}
