package blockarchive

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/ethpandaops/go-eth2-client/spec"
	"github.com/ethpandaops/go-eth2-client/spec/gloas"
	"github.com/ethpandaops/go-eth2-client/spec/phase0"
	"github.com/jellydator/ttlcache/v3"
	"golang.org/x/sync/singleflight"

	"github.com/ethpandaops/xatu/pkg/observability"
)

const (
	// listingTimeout bounds a listing refresh, every page included, so a slow archive falls back to the beacon node quickly.
	listingTimeout  = 30 * time.Second
	maxListingSize  = 64 << 20
	maxListingPages = 1000
	indexCacheSize  = 16
	modeEra         = "era"
	modeRange       = "range"
	resultHit       = "hit"
	resultUncovered = "not_covered"
)

// errCorrupt marks a file whose contents do not parse, decode or verify; only that file is skipped.
var errCorrupt = errors.New("corrupt archive file")

// Slot is one slot of a fetched era range.
type Slot struct {
	// Block is the canonical block, nil for a missed slot.
	Block *spec.VersionedSignedBeaconBlock
	// Envelope is nil for a withheld payload; only meaningful when EnvelopeKnown is set.
	Envelope *gloas.SignedExecutionPayloadEnvelope
	// EnvelopeKnown is set for Gloas+ blocks whose era has a published envelope sidecar.
	EnvelopeKnown bool
}

// Slots maps each slot of a fetched era range to its contents.
type Slots map[phase0.Slot]Slot

// Archive serves finalized blocks from CL era files and Gloas envelopes from per-era sidecars.
type Archive struct {
	cfg        *Config
	log        observability.ContextualLogger
	client     *http.Client
	base       string
	listingURL string
	network    func() string
	chainFn    func(context.Context) (*Chain, error)
	metrics    *Metrics

	fetchGroup   singleflight.Group
	listingGroup singleflight.Group
	indexGroup   singleflight.Group
	sem          chan struct{}

	mu          sync.Mutex
	chain       *Chain
	eras        map[uint64]eraFile
	listed      time.Time
	unavailable time.Time
	skipped     *ttlcache.Cache[string, struct{}]
	indexes     *ttlcache.Cache[string, *fileIndex]
}

// New creates an Archive; network and chain are resolved per request, after beacon metadata is ready.
func New(
	cfg *Config,
	log observability.ContextualLogger,
	network func() string,
	chain func(context.Context) (*Chain, error),
	metrics *Metrics,
) *Archive {
	base := strings.TrimRight(cfg.URL, "/")

	listingURL := cfg.ListingURL
	if listingURL == "" {
		listingURL = base + "/"
	}

	return &Archive{
		cfg:        cfg,
		log:        log.WithField("module", "cannon/ethereum/blockarchive"),
		client:     &http.Client{},
		base:       base,
		listingURL: listingURL,
		network:    network,
		chainFn:    chain,
		metrics:    metrics,
		sem:        make(chan struct{}, cfg.MaxConcurrentFetches),
		skipped: ttlcache.New(
			ttlcache.WithTTL[string, struct{}](cfg.ListingTTL.Duration),
			ttlcache.WithDisableTouchOnHit[string, struct{}](),
		),
		indexes: ttlcache.New(
			ttlcache.WithCapacity[string, *fileIndex](indexCacheSize),
		),
	}
}

// Slots returns the era range (whole era or RangeSlots window) holding slot; found is false when the beacon node must answer.
func (a *Archive) Slots(ctx context.Context, slot phase0.Slot) (slots Slots, found bool, err error) {
	network := a.network()

	chain, file, err := a.resolve(ctx, network, uint64(slot))
	if err != nil {
		a.metrics.lookup(network, "error")

		return nil, false, err
	}

	if chain == nil {
		a.metrics.lookup(network, "unavailable")

		return nil, false, nil
	}

	if file.name == "" {
		a.metrics.lookup(network, resultUncovered)

		return nil, false, nil
	}

	era := chain.eraOf(uint64(slot))
	first, last := a.window(chain, era, uint64(slot))

	v, err, _ := a.fetchGroup.Do(fmt.Sprintf("%s/%d-%d", file.name, first, last), func() (any, error) {
		select {
		case a.sem <- struct{}{}:
		case <-ctx.Done():
			return nil, ctx.Err()
		}

		defer func() { <-a.sem }()

		fetchCtx, cancel := context.WithTimeout(ctx, a.cfg.Timeout.Duration)
		defer cancel()

		return a.load(fetchCtx, network, chain, era, file, first, last)
	})
	if err != nil {
		if errors.Is(err, errCorrupt) {
			a.skipped.Set(file.name, struct{}{}, ttlcache.DefaultTTL)
		} else {
			a.markUnavailable()
		}

		a.metrics.lookup(network, "error")

		return nil, false, err
	}

	slots, ok := v.(Slots)
	if !ok {
		return nil, false, fmt.Errorf("unexpected era result type %T", v)
	}

	a.metrics.lookup(network, resultHit)

	return slots, true, nil
}

// HasEnvelopes reports whether Slots will return slot's envelope: a Gloas+ slot in an era with a sidecar.
func (a *Archive) HasEnvelopes(ctx context.Context, slot phase0.Slot) (bool, error) {
	chain, file, err := a.resolve(ctx, a.network(), uint64(slot))
	if err != nil || chain == nil {
		return false, err
	}

	return file.envelopes != "" && hasEnvelopes(chain.forkAt(uint64(slot))), nil
}

// resolve returns the chain and slot's era file; chain is nil while unavailable, file.name empty when uncovered.
func (a *Archive) resolve(ctx context.Context, network string, slot uint64) (chain *Chain, file eraFile, err error) {
	if a.isUnavailable() {
		return nil, file, nil
	}

	chain, err = a.chainConfig(ctx)
	if err != nil {
		return nil, file, err
	}

	file, err = a.lookup(ctx, network, chain.eraOf(slot))
	if err != nil {
		a.markUnavailable()

		return nil, file, err
	}

	return chain, file, nil
}

func (a *Archive) chainConfig(ctx context.Context) (*Chain, error) {
	a.mu.Lock()
	chain := a.chain
	a.mu.Unlock()

	if chain != nil {
		return chain, nil
	}

	chain, err := a.chainFn(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve chain configuration: %w", err)
	}

	if err := chain.validate(); err != nil {
		return nil, err
	}

	a.mu.Lock()
	a.chain = chain
	a.mu.Unlock()

	return chain, nil
}

func (a *Archive) window(chain *Chain, era, slot uint64) (first, last uint64) {
	start := chain.eraStart(era)
	end := start + chain.SlotsPerEra - 1

	if a.cfg.RangeSlots == 0 {
		return start, end
	}

	first = start + (slot-start)/a.cfg.RangeSlots*a.cfg.RangeSlots

	return first, min(first+a.cfg.RangeSlots-1, end)
}

func (a *Archive) markUnavailable() {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.unavailable = time.Now().Add(a.cfg.ListingTTL.Duration)
}

func (a *Archive) isUnavailable() bool {
	a.mu.Lock()
	defer a.mu.Unlock()

	return time.Now().Before(a.unavailable)
}

// lookup returns era's usable files, refreshing a listing older than ListingTTL.
func (a *Archive) lookup(ctx context.Context, network string, era uint64) (file eraFile, err error) {
	a.mu.Lock()
	fresh := !a.listed.IsZero() && time.Since(a.listed) < a.cfg.ListingTTL.Duration
	a.mu.Unlock()

	if !fresh {
		if _, err, _ := a.listingGroup.Do("listing", func() (any, error) {
			return nil, a.refresh(ctx, network)
		}); err != nil {
			return file, err
		}
	}

	a.mu.Lock()
	file = a.eras[era]
	a.mu.Unlock()

	if file.name == "" || a.skipped.Has(file.name) {
		return eraFile{}, nil
	}

	if file.envelopes != "" && a.skipped.Has(file.envelopes) {
		file.envelopes = ""
	}

	return file, nil
}

// refresh fetches every page of the listing and replaces the known era files.
func (a *Archive) refresh(ctx context.Context, network string) error {
	ctx, cancel := context.WithTimeout(ctx, listingTimeout)
	defer cancel()

	found := newListing(network)
	target := a.listingURL
	pages := 0

	for target != "" {
		if pages == maxListingPages {
			return fmt.Errorf("era listing: more than %d pages", maxListingPages)
		}

		body, err := a.listingPage(ctx, target)
		if err != nil {
			return fmt.Errorf("era listing: %w", err)
		}

		pages++

		found.add(body)

		if target, err = nextPage(target, body); err != nil {
			return fmt.Errorf("era listing: %w", err)
		}
	}

	eras := found.files()

	a.mu.Lock()
	a.eras, a.listed = eras, time.Now()
	a.mu.Unlock()

	log := a.log.WithField("network", network).WithField("eras", len(eras)).WithField("pages", pages).WithContext(ctx)
	if len(eras) == 0 {
		log.Warn("Era listing names no era files for the network")
	} else {
		log.Debug("Refreshed era listing")
	}

	return nil
}

func (a *Archive) listingPage(ctx context.Context, target string) ([]byte, error) {
	resp, err := a.get(ctx, target, "")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d from %s", resp.StatusCode, target)
	}

	return io.ReadAll(io.LimitReader(resp.Body, maxListingSize))
}

// load downloads and decodes slots first..last of an era.
func (a *Archive) load(ctx context.Context, network string, chain *Chain, era uint64, file eraFile, first, last uint64) (Slots, error) {
	start := time.Now()
	mode := modeRange

	if first == chain.eraStart(era) && last == chain.eraStart(era)+chain.SlotsPerEra-1 {
		mode = modeEra
	}

	slots, size, err := a.loadRange(ctx, chain, era, file, first, last)
	a.metrics.fetched(network, mode, time.Since(start), size, slots, err)

	if err != nil {
		return nil, fmt.Errorf("era %d slots %d..%d: %w", era, first, last, err)
	}

	a.log.WithField("slots", fmt.Sprintf("%d..%d", first, last)).
		WithField("mode", mode).
		WithField("bytes", size).
		WithField("duration", time.Since(start).String()).
		WithContext(ctx).
		Debug("Loaded blocks from era archive")

	return slots, nil
}

func (a *Archive) loadRange(ctx context.Context, chain *Chain, era uint64, file eraFile, first, last uint64) (Slots, int64, error) {
	idx, err := a.index(ctx, file.name, eraTailSize(chain.SlotsPerEra), func(tail []byte, size int64) (*fileIndex, error) {
		return parseEraTail(tail, size, era, chain.SlotsPerEra)
	})
	if err != nil {
		return nil, 0, err
	}

	raw, size, err := a.entries(ctx, file.name, idx, typeCompressedSignedBeaconBlock, first, last)
	if err != nil {
		return nil, size, err
	}

	blocks, err := decodeBlocks(chain, raw)
	if err != nil {
		return nil, size, fmt.Errorf("%w: %s: %w", errCorrupt, file.name, err)
	}

	slots := make(Slots, last-first+1)
	for s := first; s <= last; s++ {
		slots[phase0.Slot(s)] = Slot{Block: blocks[s]}
	}

	if file.envelopes == "" || !hasEnvelopes(chain.forkAt(last)) {
		return slots, size, nil
	}

	envelopes, n, err := a.loadEnvelopes(ctx, chain, era, file.envelopes, blocks, first, last)
	size += n

	switch {
	case errors.Is(err, errCorrupt):
		a.skipped.Set(file.envelopes, struct{}{}, ttlcache.DefaultTTL)
		a.log.WithError(err).WithContext(ctx).Warn("Skipping envelope sidecar that failed to verify")

		return slots, size, nil
	case err != nil:
		return nil, size, err
	}

	for s, envelope := range envelopes {
		slots[phase0.Slot(s)] = Slot{Block: blocks[s], Envelope: envelope, EnvelopeKnown: true}
	}

	return slots, size, nil
}

func (a *Archive) loadEnvelopes(
	ctx context.Context,
	chain *Chain,
	era uint64,
	name string,
	blocks map[uint64]*spec.VersionedSignedBeaconBlock,
	first, last uint64,
) (envelopes map[uint64]*gloas.SignedExecutionPayloadEnvelope, size int64, err error) {
	idx, err := a.index(ctx, name, envelopeTailSize(chain.SlotsPerEra), func(tail []byte, fileSize int64) (*fileIndex, error) {
		return parseEnvelopeTail(tail, fileSize, era, chain.SlotsPerEra)
	})
	if err != nil {
		return nil, 0, err
	}

	raw, size, err := a.entries(ctx, name, idx, typeCompressedEnvelope, first, last)
	if err != nil {
		return nil, size, err
	}

	envelopes, err = decodeEnvelopes(chain, blocks, raw)
	if err != nil {
		return nil, size, fmt.Errorf("%w: %s: %w", errCorrupt, name, err)
	}

	return envelopes, size, nil
}

// entries fetches the typ entries of slots first..last of a file with one Range request.
func (a *Archive) entries(ctx context.Context, name string, idx *fileIndex, typ uint16, first, last uint64) (raw map[uint64][]byte, size int64, err error) {
	lo, hi, ok := idx.slots.span(first, last, idx.end)
	if !ok {
		return map[uint64][]byte{}, 0, nil
	}

	buf, err := a.getRange(ctx, a.fileURL(name), lo, hi-1)
	if err != nil {
		return nil, int64(len(buf)), err
	}

	raw, err = indexedEntries(idx.slots, typ, first, last, buf, lo)
	if err != nil {
		return nil, int64(len(buf)), fmt.Errorf("%w: %s: %w", errCorrupt, name, err)
	}

	return raw, int64(len(buf)), nil
}

// index fetches, parses and caches a file's trailing slot index of tailSize bytes.
func (a *Archive) index(ctx context.Context, name string, tailSize int64, parse func(tail []byte, size int64) (*fileIndex, error)) (*fileIndex, error) {
	if item := a.indexes.Get(name); item != nil {
		return item.Value(), nil
	}

	v, err, _ := a.indexGroup.Do(name, func() (any, error) {
		tail, size, err := a.getTail(ctx, a.fileURL(name), tailSize)
		if err != nil {
			return nil, err
		}

		idx, err := parse(tail, size)
		if err != nil {
			return nil, fmt.Errorf("%w: %s: %w", errCorrupt, name, err)
		}

		a.indexes.Set(name, idx, ttlcache.DefaultTTL)

		return idx, nil
	})
	if err != nil {
		return nil, err
	}

	idx, ok := v.(*fileIndex)
	if !ok {
		return nil, fmt.Errorf("unexpected index result type %T", v)
	}

	return idx, nil
}

func (a *Archive) fileURL(name string) string {
	return a.base + "/" + name
}

// getTail returns the last n bytes of target and its size, also against servers that ignore Range.
func (a *Archive) getTail(ctx context.Context, target string, n int64) (tail []byte, size int64, err error) {
	resp, err := a.get(ctx, target, fmt.Sprintf("bytes=-%d", n))
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusPartialContent:
		var first, last int64
		if _, err := fmt.Sscanf(resp.Header.Get("Content-Range"), "bytes %d-%d/%d", &first, &last, &size); err != nil ||
			last != size-1 || last-first+1 != n {
			return nil, 0, fmt.Errorf("unexpected Content-Range %q for the last %d bytes", resp.Header.Get("Content-Range"), n)
		}
	case http.StatusOK:
		size = resp.ContentLength
		if size < n {
			return nil, 0, fmt.Errorf("file of %d bytes is shorter than its %d byte index", size, n)
		}

		if _, err := io.CopyN(io.Discard, resp.Body, size-n); err != nil {
			return nil, 0, fmt.Errorf("failed to skip to the file tail: %w", err)
		}
	default:
		return nil, 0, fmt.Errorf("unexpected status %d for tail request", resp.StatusCode)
	}

	tail = make([]byte, n)
	if _, err := io.ReadFull(resp.Body, tail); err != nil {
		return nil, 0, fmt.Errorf("failed to read file tail: %w", err)
	}

	return tail, size, nil
}

// getRange returns bytes first..last of target, also against servers that ignore Range.
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
	case http.StatusOK:
		if _, err := io.CopyN(io.Discard, resp.Body, first); err != nil {
			return nil, fmt.Errorf("failed to skip to range start: %w", err)
		}
	default:
		return nil, fmt.Errorf("unexpected status %d for range request", resp.StatusCode)
	}

	data := make([]byte, last-first+1)
	if _, err := io.ReadFull(resp.Body, data); err != nil {
		return nil, fmt.Errorf("failed to read range: %w", err)
	}

	return data, nil
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
