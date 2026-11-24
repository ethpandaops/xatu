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
	listingTimeout     = 30 * time.Second
	maxListingSize     = 64 << 20
	maxListingPages    = 1000
	indexCacheSize     = 16
	windowMemoSize     = 64
	skippedCacheSize   = 1024
	modeEra            = "era"
	modeRange          = "range"
	resultHit          = "hit"
	resultUncovered    = "not_covered"
	resultNotCanonical = "not_canonical"
	resultNoBeacon     = "beacon_missing"
	resultBeaconError  = "beacon_error"
	skippedFile        = "file"
	// beaconRetryAfter holds off a window after a failed beacon lookup, so an outage costs one request per window.
	beaconRetryAfter = 10 * time.Second
)

var (
	// errCorrupt marks a file whose contents do not parse, decode or verify; only that file is skipped.
	errCorrupt = errors.New("corrupt archive file")
	// errMissing marks a listed file the archive no longer serves; only that file is skipped.
	errMissing = errors.New("archive file not found")
	// errUnreachable marks a failure talking to the archive; the whole archive is skipped for ListingTTL.
	errUnreachable = errors.New("archive unreachable")
	// errNotCanonical marks a window that disagrees with the beacon node's chain; only that window is skipped.
	errNotCanonical = errors.New("era window is not the beacon node's canonical chain")
	// errBeacon marks a failed beacon node lookup; the window is retried after beaconRetryAfter.
	errBeacon = errors.New("beacon node lookup failed")
)

// beaconMissingError is a window whose anchor block the beacon node does not have, such as one without history.
type beaconMissingError struct {
	slot uint64
}

func (e *beaconMissingError) Error() string {
	return fmt.Sprintf("beacon node has no block at slot %d", e.slot)
}

// Slot is one slot of a fetched era range.
type Slot struct {
	// Block is the canonical block, nil for a missed slot.
	Block *spec.VersionedSignedBeaconBlock
	// Envelope is nil for a withheld payload; only meaningful when EnvelopeKnown is set.
	Envelope *gloas.SignedExecutionPayloadEnvelope
	// EnvelopeKnown is set for Gloas+ blocks whose envelope, or withheld payload, a verified sidecar proves.
	EnvelopeKnown bool
}

// Slots maps the slots of a fetched era range the archive answers for to their contents.
type Slots map[phase0.Slot]Slot

// BlockID identifies a signed beacon block.
type BlockID struct {
	Root      phase0.Root
	Signature phase0.BLSSignature
}

// CanonicalBlock returns the beacon node's block at slot; found is false for a slot without a block.
type CanonicalBlock func(ctx context.Context, slot phase0.Slot) (id BlockID, found bool, err error)

// Archive serves finalized blocks from CL era files and Gloas envelopes from per-era sidecars.
type Archive struct {
	cfg        *Config
	log        observability.ContextualLogger
	client     *http.Client
	base       string
	listingURL string
	network    func() string
	chainFn    func(context.Context) (*Chain, error)
	canonical  CanonicalBlock
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
	// skipped holds skipped files and windows, with the lookup result a skipped window reports.
	skipped *ttlcache.Cache[string, string]
	indexes *ttlcache.Cache[string, *fileIndex]
	windows *ttlcache.Cache[string, *windowMemo]
}

// New creates an Archive; network, chain and canonical are resolved per request, after beacon metadata is ready.
func New(
	cfg *Config,
	log observability.ContextualLogger,
	network func() string,
	chain func(context.Context) (*Chain, error),
	canonical CanonicalBlock,
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
		canonical:  canonical,
		metrics:    metrics,
		sem:        make(chan struct{}, cfg.MaxConcurrentFetches),
		skipped: ttlcache.New(
			ttlcache.WithTTL[string, string](cfg.ListingTTL.Duration),
			ttlcache.WithCapacity[string, string](skippedCacheSize),
			ttlcache.WithDisableTouchOnHit[string, string](),
		),
		indexes: ttlcache.New(
			ttlcache.WithCapacity[string, *fileIndex](indexCacheSize),
		),
		windows: ttlcache.New(
			ttlcache.WithTTL[string, *windowMemo](cfg.ListingTTL.Duration),
			ttlcache.WithCapacity[string, *windowMemo](windowMemoSize),
			ttlcache.WithDisableTouchOnHit[string, *windowMemo](),
		),
	}
}

// Slots returns the proven slots of the era range holding slot; found is false when the beacon node must answer for it all.
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
	key := windowKey(file.name, first, last)

	if item := a.skipped.Get(key); item != nil {
		a.metrics.lookup(network, item.Value())

		return nil, false, nil
	}

	if memo := a.windows.Get(key); memo != nil && !memo.Value().blocks.has(uint64(slot)) {
		a.metrics.lookup(network, resultUncovered)

		return nil, false, nil
	}

	// The fetch is shared and outlives any one caller, so it runs detached with its own timeouts.
	ch := a.fetchGroup.DoChan(key, func() (any, error) {
		return a.fetch(context.WithoutCancel(ctx), network, chain, era, file, first, last)
	})

	var res singleflight.Result

	select {
	case res = <-ch:
	case <-ctx.Done():
		return nil, false, ctx.Err()
	}

	if res.Err != nil {
		return nil, false, a.failed(ctx, network, file, key, res.Err)
	}

	slots, ok := res.Val.(Slots)
	if !ok {
		return nil, false, fmt.Errorf("unexpected era result type %T", res.Val)
	}

	if _, ok := slots[slot]; ok {
		a.metrics.lookup(network, resultHit)
	} else {
		a.metrics.lookup(network, resultUncovered)
	}

	return slots, true, nil
}

// failed records a failed window fetch and returns the error the caller should see.
func (a *Archive) failed(ctx context.Context, network string, file eraFile, key string, err error) error {
	var missing *beaconMissingError

	switch {
	case errors.As(err, &missing):
		a.skipped.Set(key, resultNoBeacon, ttlcache.DefaultTTL)
		a.metrics.lookup(network, resultNoBeacon)
		a.log.WithField("window", key).WithField("slot", missing.slot).WithContext(ctx).
			Warn("Beacon node has no block at the era window's anchor slot, serving the window from the beacon node")

		return nil
	case errors.Is(err, errNotCanonical):
		a.skipped.Set(key, resultNotCanonical, ttlcache.DefaultTTL)
		a.metrics.lookup(network, resultNotCanonical)
		a.log.WithError(err).WithField("window", key).WithContext(ctx).
			Warn("Era window disagrees with the beacon node, serving it from the beacon node")

		return nil
	case errors.Is(err, errBeacon):
		a.skipped.Set(key, resultBeaconError, beaconRetryAfter)
		a.metrics.lookup(network, resultBeaconError)

		return err
	case errors.Is(err, errCorrupt), errors.Is(err, errMissing):
		a.skip(file.name)
	case errors.Is(err, errUnreachable):
		a.markUnavailable()
	}

	a.metrics.lookup(network, "error")

	return err
}

// HasEnvelopes reports whether Slots may return slot's envelope: a Gloas+ slot in an era with a sidecar.
func (a *Archive) HasEnvelopes(ctx context.Context, slot phase0.Slot) (bool, error) {
	chain, file, err := a.resolve(ctx, a.network(), uint64(slot))
	if err != nil || chain == nil {
		return false, err
	}

	if file.envelopes == "" || !hasEnvelopes(chain.forkAt(uint64(slot))) {
		return false, nil
	}

	first, last := a.window(chain, chain.eraOf(uint64(slot)), uint64(slot))
	key := windowKey(file.name, first, last)

	if a.skipped.Has(key) {
		return false, nil
	}

	if memo := a.windows.Get(key); memo != nil {
		return memo.Value().envelopes.has(uint64(slot)), nil
	}

	return true, nil
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
		if ctx.Err() == nil {
			a.markUnavailable()
		}

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

	if end-first < a.cfg.RangeSlots {
		return first, end
	}

	return first, first + a.cfg.RangeSlots - 1
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

// skip stops using a file for ListingTTL and forgets its index, so a replaced file is read afresh.
func (a *Archive) skip(name string) {
	a.skipped.Set(name, skippedFile, ttlcache.DefaultTTL)
	a.indexes.Delete(name)
}

// lookup returns era's usable files, refreshing a listing older than ListingTTL.
func (a *Archive) lookup(ctx context.Context, network string, era uint64) (file eraFile, err error) {
	a.mu.Lock()
	fresh := !a.listed.IsZero() && time.Since(a.listed) < a.cfg.ListingTTL.Duration
	a.mu.Unlock()

	if !fresh {
		ch := a.listingGroup.DoChan("listing", func() (any, error) {
			refreshCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), listingTimeout)
			defer cancel()

			return nil, a.refresh(refreshCtx, network)
		})

		select {
		case res := <-ch:
			if res.Err != nil {
				return file, res.Err
			}
		case <-ctx.Done():
			return file, ctx.Err()
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

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxListingSize+1))
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", target, err)
	}

	if len(body) > maxListingSize {
		return nil, fmt.Errorf("%s is larger than %d bytes", target, maxListingSize)
	}

	return body, nil
}

// fetch loads a window under the fetch concurrency limit; ctx carries no caller cancellation.
func (a *Archive) fetch(ctx context.Context, network string, chain *Chain, era uint64, file eraFile, first, last uint64) (Slots, error) {
	waitCtx, cancelWait := context.WithTimeout(ctx, a.cfg.Timeout.Duration)
	defer cancelWait()

	select {
	case a.sem <- struct{}{}:
	case <-waitCtx.Done():
		return nil, fmt.Errorf("waiting to fetch from the archive: %w", waitCtx.Err())
	}

	defer func() { <-a.sem }()

	fetchCtx, cancel := context.WithTimeout(ctx, a.cfg.Timeout.Duration)
	defer cancel()

	slots, err := a.load(fetchCtx, network, chain, era, file, first, last)
	if err != nil {
		return nil, err
	}

	a.windows.Set(windowKey(file.name, first, last), memoOf(slots, first, last), ttlcache.DefaultTTL)

	return slots, nil
}

// load downloads, decodes and verifies slots first..last of an era.
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
		WithField("proven", len(slots)).
		WithField("mode", mode).
		WithField("bytes", size).
		WithField("duration", time.Since(start).String()).
		WithContext(ctx).
		Debug("Loaded blocks from era archive")

	return slots, nil
}

// loadRange returns the slots of first..last that the era file proves, tied to the beacon node's chain by one root.
func (a *Archive) loadRange(ctx context.Context, chain *Chain, era uint64, file eraFile, first, last uint64) (Slots, int64, error) {
	idx, err := a.index(ctx, file.name, eraTailSize(chain.SlotsPerEra), func(tail []byte, size int64) (*fileIndex, error) {
		return parseEraTail(tail, size, era, chain.SlotsPerEra)
	})
	if err != nil {
		return nil, 0, err
	}

	p, ok := planWindow(idx.slots, first, last)
	if !ok {
		return Slots{}, 0, nil
	}

	want, found, err := a.canonical(ctx, phase0.Slot(p.anchor))
	if err != nil {
		return nil, 0, fmt.Errorf("%w: block at slot %d: %w", errBeacon, p.anchor, err)
	}

	if !found {
		return nil, 0, &beaconMissingError{slot: p.anchor}
	}

	raw, size, err := a.entries(ctx, file.name, idx, typeCompressedSignedBeaconBlock, p.readFirst, p.readLast)
	if err != nil {
		return nil, size, err
	}

	decoded, err := decodeBlocks(chain, raw)
	if err != nil {
		return nil, size, fmt.Errorf("%w: %s: %w", errCorrupt, file.name, err)
	}

	if err = checkLinks(decoded); err != nil {
		return nil, size, fmt.Errorf("%w: %s: %w", errCorrupt, file.name, err)
	}

	got, ok := decoded[p.anchor]
	if !ok {
		return nil, size, fmt.Errorf("%w: %s: no block at indexed slot %d", errCorrupt, file.name, p.anchor)
	}

	if got.root != want.Root || got.sig != want.Signature {
		return nil, size, fmt.Errorf("%w: %s has block %#x signed %#x at slot %d, the beacon node %#x signed %#x",
			errNotCanonical, file.name, got.root, got.sig, p.anchor, want.Root, want.Signature)
	}

	blocks := make(map[uint64]*decodedBlock, len(decoded))
	slots := make(Slots, p.hi-p.lo+1)

	for s := p.lo; s <= p.hi; s++ {
		var block *spec.VersionedSignedBeaconBlock

		if b, ok := decoded[s]; ok {
			blocks[s], block = b, b.block
		}

		slots[phase0.Slot(s)] = Slot{Block: block}
	}

	if file.envelopes == "" || !hasEnvelopes(chain.forkAt(p.hi)) {
		return slots, size, nil
	}

	var next *decodedBlock
	if p.succ {
		next = decoded[p.readLast]
	}

	envelopes, n, err := a.loadEnvelopes(ctx, chain, era, file.envelopes, blocks, next, first, last)
	size += n

	if err != nil {
		a.skip(file.envelopes)
		a.log.WithError(err).WithField("file", file.envelopes).WithContext(ctx).
			Warn("Skipping envelope sidecar, envelopes come from the beacon node")

		return slots, size, nil
	}

	for s, envelope := range envelopes {
		slots[phase0.Slot(s)] = Slot{Block: blocks[s].block, Envelope: envelope, EnvelopeKnown: true}
	}

	return slots, size, nil
}

func (a *Archive) loadEnvelopes(
	ctx context.Context,
	chain *Chain,
	era uint64,
	name string,
	blocks map[uint64]*decodedBlock,
	next *decodedBlock,
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

	envelopes, err = decodeEnvelopes(chain, blocks, raw, next)
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

// statusError classifies an unexpected response: a missing file is skipped alone, anything else skips the archive.
func statusError(status int, target string) error {
	if status == http.StatusNotFound || status == http.StatusGone {
		return fmt.Errorf("%w: %s", errMissing, target)
	}

	return fmt.Errorf("%w: unexpected status %d for %s", errUnreachable, status, target)
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
		if _, err := fmt.Sscanf(resp.Header.Get("Content-Range"), "bytes %d-%d/%d", &first, &last, &size); err != nil || last != size-1 {
			return nil, 0, fmt.Errorf("%w: unexpected Content-Range %q for the last %d bytes", errUnreachable, resp.Header.Get("Content-Range"), n)
		}

		if last-first+1 != n {
			return nil, 0, fmt.Errorf("%w: file of %d bytes is shorter than its %d byte index", errCorrupt, size, n)
		}
	case http.StatusOK:
		size = resp.ContentLength
		if size < n {
			return nil, 0, fmt.Errorf("%w: file of %d bytes is shorter than its %d byte index", errCorrupt, size, n)
		}

		if _, err := io.CopyN(io.Discard, resp.Body, size-n); err != nil {
			return nil, 0, fmt.Errorf("%w: failed to skip to the file tail: %w", errUnreachable, err)
		}
	default:
		return nil, 0, statusError(resp.StatusCode, target)
	}

	tail = make([]byte, n)
	if _, err := io.ReadFull(resp.Body, tail); err != nil {
		return nil, 0, fmt.Errorf("%w: failed to read file tail: %w", errUnreachable, err)
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
			return nil, fmt.Errorf("%w: unexpected Content-Range %q for bytes %d-%d", errUnreachable, resp.Header.Get("Content-Range"), first, last)
		}
	case http.StatusOK:
		if _, err := io.CopyN(io.Discard, resp.Body, first); err != nil {
			return nil, fmt.Errorf("%w: failed to skip to range start: %w", errUnreachable, err)
		}
	default:
		return nil, statusError(resp.StatusCode, target)
	}

	data := make([]byte, last-first+1)
	if _, err := io.ReadFull(resp.Body, data); err != nil {
		return nil, fmt.Errorf("%w: failed to read range: %w", errUnreachable, err)
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

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errUnreachable, err)
	}

	return resp, nil
}
