package ethereum

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/ethpandaops/beacon/pkg/beacon"
	client "github.com/ethpandaops/go-eth2-client"
	"github.com/ethpandaops/go-eth2-client/api"
	apiv1 "github.com/ethpandaops/go-eth2-client/api/v1"
	ehttp "github.com/ethpandaops/go-eth2-client/http"
	"github.com/ethpandaops/go-eth2-client/spec"
	"github.com/ethpandaops/go-eth2-client/spec/gloas"
	"github.com/ethpandaops/go-eth2-client/spec/phase0"
	"github.com/ethpandaops/go-eth2-client/spec/pq"
	"github.com/jellydator/ttlcache/v3"
	dynssz "github.com/pk910/dynamic-ssz"
	"github.com/pkg/errors"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/sync/singleflight"

	"github.com/ethpandaops/xatu/pkg/cannon/ethereum/blockarchive"
	"github.com/ethpandaops/xatu/pkg/cannon/ethereum/services"
	"github.com/ethpandaops/xatu/pkg/networks"
	"github.com/ethpandaops/xatu/pkg/observability"
)

// blockArchive serves finalized blocks and envelopes in bulk; found is false for slots it does not cover.
type blockArchive interface {
	Slots(ctx context.Context, slot phase0.Slot) (slots blockarchive.Slots, found bool, err error)
	HasEnvelopes(ctx context.Context, slot phase0.Slot) (bool, error)
}

type BeaconNode struct {
	config *Config
	log    observability.ContextualLogger

	// archive is nil unless ethereum.blockArchive is enabled.
	archive blockArchive

	beacon  beacon.Node
	metrics *Metrics

	services []services.Service

	onReadyCallbacks []func(ctx context.Context) error

	sfGroup          *singleflight.Group
	blockCache       *ttlcache.Cache[string, *spec.VersionedSignedBeaconBlock]
	blockPreloadChan chan string
	blockPreloadSem  chan struct{}

	validatorsSfGroup     *singleflight.Group
	validatorsCache       *ttlcache.Cache[string, map[phase0.ValidatorIndex]*apiv1.Validator]
	validatorsPreloadChan chan string
	validatorsPreloadSem  chan struct{}

	// envelopeCache holds Gloas (EIP-7732) ExecutionPayloadEnvelopes keyed by
	// block root. The envelope is fetched separately from the block — many
	// derivers consume it per slot, so we cache to avoid duplicate fetches.
	// A nil value means "envelope known to be absent" (builder withheld payload —
	// payload_status = EMPTY); we cache the negative answer too.
	envelopeSfGroup *singleflight.Group
	envelopeCache   *ttlcache.Cache[string, *gloas.SignedExecutionPayloadEnvelope]

	// The Gloas state derivers share one state download per epoch. Only the
	// reduced GloasEpochState is cached, and stateSem keeps a single full state
	// in memory at a time.
	stateSfGroup *singleflight.Group
	stateCache   *ttlcache.Cache[string, *GloasEpochState]
	stateSem     chan struct{}
	stateCodecMu sync.Mutex
	stateCodec   *dynssz.DynSsz

	// slotGaps are runs of empty slots longer than nextCanonicalBlock's lookahead, newest last.
	slotGapsMu sync.Mutex
	slotGaps   []slotGap
}

func NewBeaconNode(ctx context.Context, name string, config *Config, log observability.ContextualLogger) (*BeaconNode, error) {
	namespace := "xatu_cannon"

	opts := *beacon.
		DefaultOptions().
		DisableEmptySlotDetection().
		DisablePrometheusMetrics()

	opts.GoEth2ClientParams = []ehttp.Parameter{
		ehttp.WithEnforceJSON(true),
	}

	opts.HealthCheck.Interval.Duration = time.Second * 3
	opts.HealthCheck.SuccessfulResponses = 1

	opts.BeaconSubscription.Enabled = false

	node := beacon.NewNode(log, &beacon.Config{
		Name:    name,
		Addr:    config.Beacon.Address,
		Headers: config.Beacon.Headers,
	}, namespace, opts)

	metadata := services.NewMetadataService(log, node)

	if config.OverrideNetworkName != "" {
		metadata.OverrideNetworkName(config.OverrideNetworkName)
	}

	duties := services.NewDutiesService(log, node, &metadata)

	svcs := []services.Service{
		&metadata,
		&duties,
	}

	// Create a buffered channel (semaphore) to limit the number of concurrent goroutines.
	sem := make(chan struct{}, config.Beacon.BlockPreloadWorkers)
	validatorsSem := make(chan struct{}, 1)

	blockCacheSize := config.Beacon.BlockCacheSize
	envelopeCacheSize := uint64(256)

	if config.BlockArchive.Enabled {
		// A fetched era range is cached whole, blocks and envelopes alike, so it must fit.
		blockCacheSize = max(blockCacheSize, config.BlockArchive.MaxFetchSlots())
		envelopeCacheSize = max(envelopeCacheSize, blockCacheSize)
	}

	b := &BeaconNode{
		config:            config,
		log:               log.WithField("module", "cannon/ethereum/beacon"),
		beacon:            node,
		services:          svcs,
		sfGroup:           &singleflight.Group{},
		validatorsSfGroup: &singleflight.Group{},
		blockCache: ttlcache.New(
			ttlcache.WithTTL[string, *spec.VersionedSignedBeaconBlock](config.Beacon.BlockCacheTTL.Duration),
			ttlcache.WithCapacity[string, *spec.VersionedSignedBeaconBlock](blockCacheSize),
		),
		blockPreloadChan: make(chan string, config.Beacon.BlockPreloadQueueSize),
		blockPreloadSem:  sem,
		validatorsCache: ttlcache.New(
			ttlcache.WithTTL[string, map[phase0.ValidatorIndex]*apiv1.Validator](5*time.Minute),
			ttlcache.WithCapacity[string, map[phase0.ValidatorIndex]*apiv1.Validator](4),
		),
		validatorsPreloadChan: make(chan string, 2),
		validatorsPreloadSem:  validatorsSem,
		envelopeSfGroup:       &singleflight.Group{},
		envelopeCache: ttlcache.New(
			ttlcache.WithTTL[string, *gloas.SignedExecutionPayloadEnvelope](config.Beacon.BlockCacheTTL.Duration),
			ttlcache.WithCapacity[string, *gloas.SignedExecutionPayloadEnvelope](envelopeCacheSize),
		),
		stateSfGroup: &singleflight.Group{},
		stateCache: ttlcache.New(
			ttlcache.WithTTL[string, *GloasEpochState](gloasEpochStateCacheTTL),
			ttlcache.WithCapacity[string, *GloasEpochState](gloasEpochStateCacheSize),
		),
		stateSem: make(chan struct{}, 1),
		metrics:  NewMetrics(namespace, name),
	}

	if config.BlockArchive.Enabled {
		b.archive = blockarchive.New(&config.BlockArchive, log, b.archiveNetwork, b.archiveChain, b.archiveCanonicalBlock,
			blockarchive.NewMetrics(namespace+"_ethereum"))
	}

	return b, nil
}

// archiveNetwork is the network name the block archive is queried with.
func (b *BeaconNode) archiveNetwork() string {
	if b.config.BlockArchive.Network != "" {
		return b.config.BlockArchive.Network
	}

	return string(b.Metadata().Network.Name)
}

// archiveChain is the network configuration era files are decoded with.
func (b *BeaconNode) archiveChain(ctx context.Context) (*blockarchive.Chain, error) {
	codec, err := b.sszCodec(ctx)
	if err != nil {
		return nil, err
	}

	sp, err := b.beacon.Spec()
	if err != nil {
		return nil, errors.Wrap(err, "failed to obtain spec")
	}

	slotsPerEra, ok := sp.FullSpec["SLOTS_PER_HISTORICAL_ROOT"].(uint64)
	if !ok {
		return nil, errors.New("spec has no SLOTS_PER_HISTORICAL_ROOT")
	}

	return &blockarchive.Chain{
		Codec:         codec,
		SlotsPerEpoch: uint64(sp.SlotsPerEpoch),
		SlotsPerEra:   slotsPerEra,
		Forks:         sp.ForkEpochs,
	}, nil
}

// archiveCanonicalBlock is the beacon node's block root and signature at slot, which each era window is tied to.
func (b *BeaconNode) archiveCanonicalBlock(ctx context.Context, slot phase0.Slot) (blockarchive.BlockID, bool, error) {
	provider, ok := b.beacon.Service().(client.BeaconBlockHeadersProvider)
	if !ok {
		return blockarchive.BlockID{}, false, errors.New("beacon block headers provider not available")
	}

	resp, err := provider.BeaconBlockHeader(ctx, &api.BeaconBlockHeaderOpts{Block: strconv.FormatUint(uint64(slot), 10)})
	if err != nil {
		var apiErr *api.Error
		if errors.As(err, &apiErr) && apiErr.StatusCode == 404 {
			return blockarchive.BlockID{}, false, nil
		}

		return blockarchive.BlockID{}, false, errors.Wrapf(err, "failed to fetch block header at slot %d", slot)
	}

	if resp == nil || resp.Data == nil || resp.Data.Header == nil {
		return blockarchive.BlockID{}, false, errors.Errorf("empty block header response at slot %d", slot)
	}

	return blockarchive.BlockID{Root: resp.Data.Root, Signature: resp.Data.Header.Signature}, true, nil
}

func (b *BeaconNode) Start(ctx context.Context) error {
	errs := make(chan error, 1)

	go func() {
		wg := sync.WaitGroup{}

		for _, service := range b.services {
			wg.Add(1)

			service.OnReady(ctx, func(ctx context.Context) error {
				b.log.WithField("service", service.Name()).WithContext(ctx).Info("Service is ready")

				wg.Done()

				return nil
			})

			b.log.WithField("service", service.Name()).WithContext(ctx).Info("Starting service")

			if err := service.Start(ctx); err != nil {
				errs <- fmt.Errorf("failed to start service: %w", err)
			}

			wg.Wait()
		}

		if b.Metadata().Network.Name == networks.NetworkNameUnknown {
			errs <- errors.New("unknown network detected. Please override the network name via config if you are using a custom network")
		}

		b.log.WithContext(ctx).Info("All services are ready")

		for _, callback := range b.onReadyCallbacks {
			if err := callback(ctx); err != nil {
				errs <- fmt.Errorf("failed to run on ready callback: %w", err)
			}
		}
	}()

	if err := b.beacon.Start(ctx); err != nil {
		return err
	}

	b.blockCache.OnEviction(func(ctx context.Context, reason ttlcache.EvictionReason, item *ttlcache.Item[string, *spec.VersionedSignedBeaconBlock]) {
		b.log.WithField("identifier", item.Key()).WithField("reason", reason).WithContext(ctx).Trace("Block evicted from cache")
	})

	go b.blockCache.Start()

	for i := uint64(0); i < b.config.Beacon.BlockPreloadWorkers; i++ {
		go func() {
			for identifier := range b.blockPreloadChan {
				b.metrics.SetPreloadBlockQueueSize(string(b.Metadata().Network.Name), len(b.blockPreloadChan))

				b.log.WithField("identifier", identifier).WithContext(ctx).Trace("Preloading block")

				//nolint:errcheck // We don't care about errors here.
				b.GetBeaconBlock(ctx, identifier, true)
			}
		}()
	}

	go func() {
		for identifier := range b.validatorsPreloadChan {
			b.log.WithField("identifier", identifier).WithContext(ctx).Trace("Preloading validators")

			//nolint:errcheck // We don't care about errors here.
			b.GetValidators(ctx, identifier)
		}
	}()

	select {
	case err := <-errs:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Finality returns the current finality checkpoints as reported by the
// upstream beacon node.
func (b *BeaconNode) Finality() (*apiv1.Finality, error) {
	return b.beacon.Finality()
}

func (b *BeaconNode) Node() beacon.Node {
	return b.beacon
}

func (b *BeaconNode) getServiceByName(name services.Name) (services.Service, error) {
	for _, service := range b.services {
		if service.Name() == name {
			return service, nil
		}
	}

	return nil, errors.New("service not found")
}

func (b *BeaconNode) Metadata() *services.MetadataService {
	service, err := b.getServiceByName("metadata")
	if err != nil {
		// This should never happen. If it does, good luck.
		return nil
	}

	return service.(*services.MetadataService)
}

func (b *BeaconNode) Duties() *services.DutiesService {
	service, err := b.getServiceByName("duties")
	if err != nil {
		// This should never happen. If it does, good luck.
		return nil
	}

	return service.(*services.DutiesService)
}

func (b *BeaconNode) OnReady(_ context.Context, callback func(ctx context.Context) error) {
	b.onReadyCallbacks = append(b.onReadyCallbacks, callback)
}

func (b *BeaconNode) Synced(ctx context.Context) error {
	status := b.beacon.Status()
	if status == nil {
		return errors.New("missing beacon status")
	}

	syncState := status.SyncState()
	if syncState == nil {
		return errors.New("missing beacon node status sync state")
	}

	if syncState.SyncDistance > 3 {
		return errors.New("beacon node is not synced")
	}

	wallclock := b.Metadata().Wallclock()
	if wallclock == nil {
		return errors.New("missing wallclock")
	}

	currentSlot := wallclock.Slots().Current()

	if currentSlot.Number()-uint64(syncState.HeadSlot) > 32 {
		return fmt.Errorf("beacon node is too far behind head, head slot is %d, current slot is %d", syncState.HeadSlot, currentSlot.Number())
	}

	for _, service := range b.services {
		if err := service.Ready(ctx); err != nil {
			return errors.Wrapf(err, "service %s is not ready", service.Name())
		}
	}

	return nil
}

// GetBeaconBlock returns a beacon block by its identifier. Blocks can be cached internally.
func (b *BeaconNode) GetBeaconBlock(ctx context.Context, identifier string, ignoreMetrics ...bool) (*spec.VersionedSignedBeaconBlock, error) {
	ctx, span := observability.Tracer().Start(ctx, "ethereum.beacon.GetBeaconBlock", trace.WithAttributes(attribute.String("identifier", identifier)))

	defer span.End()

	b.metrics.IncBlocksFetched(string(b.Metadata().Network.Name))

	// Check the cache first.
	if item := b.blockCache.Get(identifier); item != nil {
		if len(ignoreMetrics) != 0 && ignoreMetrics[0] {
			b.metrics.IncBlockCacheHit(string(b.Metadata().Network.Name))
		}

		span.SetAttributes(attribute.Bool("cached", true))

		return item.Value(), nil
	}

	span.SetAttributes(attribute.Bool("cached", false))

	if len(ignoreMetrics) != 0 && ignoreMetrics[0] {
		b.metrics.IncBlockCacheMiss(string(b.Metadata().Network.Name))
	}

	// Use singleflight to ensure we only make one request for a block at a time.
	x, err, shared := b.sfGroup.Do(identifier, func() (any, error) {
		if block, ok := b.blockFromArchive(ctx, identifier); ok {
			span.AddEvent("Block served from block archive.")

			return block, nil
		}

		span.AddEvent("Acquiring semaphore...")

		// Acquire a semaphore before proceeding.
		b.blockPreloadSem <- struct{}{}
		defer func() { <-b.blockPreloadSem }()

		span.AddEvent("Semaphore acquired. Fetching block from beacon api...")

		// Not in the cache, so fetch it.
		block, err := b.beacon.FetchBlock(ctx, identifier)
		if err != nil {
			return nil, err
		}

		// Only slot numbers are checked: a root or named identifier has no slot to confirm.
		if slot, parseErr := strconv.ParseUint(identifier, 10, 64); block == nil && parseErr == nil {
			if err := b.confirmMissedSlot(ctx, slot); err != nil {
				return nil, err
			}
		}

		span.AddEvent("Block fetched from beacon node.")

		// Add it to the cache.
		b.blockCache.Set(identifier, block, time.Hour)

		return block, nil
	})
	if err != nil {
		span.SetStatus(codes.Error, err.Error())

		if len(ignoreMetrics) != 0 && ignoreMetrics[0] {
			b.metrics.IncBlocksFetchErrors(string(b.Metadata().Network.Name))
		}

		return nil, err
	}

	span.AddEvent("Block fetching complete.", trace.WithAttributes(attribute.Bool("shared", shared)))

	return x.(*spec.VersionedSignedBeaconBlock), nil
}

// confirmMissedSlot checks that a slot the beacon node has no block for is empty on the canonical chain: the next
// block's parent must be older than the slot. A node whose block history does not reach the slot (checkpoint synced,
// still backfilling, or pruned) answers 404 for it and has no such parent, so its 404 is an error the caller retries
// instead of a missed slot.
func (b *BeaconNode) confirmMissedSlot(ctx context.Context, slot uint64) error {
	_, next, parentSlot, err := b.nextCanonicalBlock(ctx, slot)
	if err != nil {
		return err
	}

	if uint64(parentSlot) >= slot {
		return errors.Errorf("beacon node has no block at slot %d, but the block at slot %d builds on a block at slot %d", slot, next, parentSlot)
	}

	return nil
}

// slotGap is a run of empty slots: the canonical block after slot `after` is at slot `next`.
type slotGap struct {
	after, next uint64
}

const maxSlotGaps = 32

// nextCanonicalBlock returns the first canonical block after slot, its slot and its parent's slot. It looks at the
// following slots one by one, and past two epochs of empty slots, such as a chain stall, it finds any later block
// up to the node's head and walks its ancestry back to the slot.
func (b *BeaconNode) nextCanonicalBlock(ctx context.Context, slot uint64) (*spec.VersionedSignedBeaconBlock, uint64, phase0.Slot, error) {
	if gap, ok := b.knownSlotGap(slot); ok {
		block, err := b.fetchSlotBlock(ctx, gap.next)
		if err != nil {
			return nil, 0, 0, err
		}

		if block != nil {
			return block, gap.next, phase0.Slot(gap.after), nil
		}
	}

	sp, err := b.beacon.Spec()
	if err != nil {
		return nil, 0, 0, errors.Wrap(err, "failed to obtain spec")
	}

	lookahead := 2 * uint64(sp.SlotsPerEpoch)

	for next := slot + 1; next <= slot+lookahead; next++ {
		block, fetchErr := b.fetchSlotBlock(ctx, next)
		if fetchErr != nil {
			return nil, 0, 0, errors.Wrapf(fetchErr, "failed to get block at slot %d after slot %d", next, slot)
		}

		if block == nil {
			continue
		}

		parentRoot, err := block.ParentRoot()
		if err != nil {
			return nil, 0, 0, errors.Wrapf(err, "failed to get the parent root of the block at slot %d", next)
		}

		parent, ok, err := b.blockHeader(ctx, parentRoot.String())
		if err != nil {
			return nil, 0, 0, err
		}

		if !ok {
			return nil, 0, 0, errors.Errorf("beacon node has no block at slot %d nor the parent of the block at slot %d: its block history does not reach slot %d", slot, next, slot)
		}

		return block, next, parent.Slot, nil
	}

	head, ok, err := b.blockHeader(ctx, "head")
	if err != nil {
		return nil, 0, 0, err
	}

	if !ok {
		return nil, 0, 0, errors.New("beacon node has no head block")
	}

	for probe := slot + 2*lookahead; probe <= uint64(head.Slot)+lookahead; probe += lookahead {
		block, err := b.fetchSlotBlock(ctx, min(probe, uint64(head.Slot)))
		if err != nil {
			return nil, 0, 0, errors.Wrapf(err, "failed to get block at slot %d after slot %d", probe, slot)
		}

		if block == nil {
			continue
		}

		childSlot, err := block.Slot()
		if err != nil {
			return nil, 0, 0, errors.Wrap(err, "failed to get block slot")
		}

		parentRoot, err := block.ParentRoot()
		if err != nil {
			return nil, 0, 0, errors.Wrapf(err, "failed to get the parent root of the block at slot %d", childSlot)
		}

		for {
			parent, ok, err := b.blockHeader(ctx, parentRoot.String())
			if err != nil {
				return nil, 0, 0, err
			}

			if !ok {
				return nil, 0, 0, errors.Errorf("beacon node has no block at slot %d nor the parent of the block at slot %d: its block history does not reach slot %d", slot, childSlot, slot)
			}

			if uint64(parent.Slot) <= slot {
				b.recordSlotGap(slotGap{after: uint64(parent.Slot), next: uint64(childSlot)})

				first, err := b.fetchSlotBlock(ctx, uint64(childSlot))
				if err != nil {
					return nil, 0, 0, err
				}

				if first == nil {
					return nil, 0, 0, errors.Errorf("beacon node has no block at slot %d, the child of the block at slot %d", childSlot, parent.Slot)
				}

				return first, uint64(childSlot), parent.Slot, nil
			}

			childSlot, parentRoot = parent.Slot, parent.ParentRoot
		}
	}

	return nil, 0, 0, errors.Errorf("beacon node has no block after slot %d up to its head at slot %d", slot, head.Slot)
}

// knownSlotGap returns the recorded gap that slot is the start of or inside.
func (b *BeaconNode) knownSlotGap(slot uint64) (slotGap, bool) {
	b.slotGapsMu.Lock()
	defer b.slotGapsMu.Unlock()

	for _, gap := range b.slotGaps {
		if gap.after <= slot && slot < gap.next {
			return gap, true
		}
	}

	return slotGap{}, false
}

func (b *BeaconNode) recordSlotGap(gap slotGap) {
	b.slotGapsMu.Lock()
	defer b.slotGapsMu.Unlock()

	b.slotGaps = append(b.slotGaps, gap)
	if len(b.slotGaps) > maxSlotGaps {
		b.slotGaps = b.slotGaps[1:]
	}
}

// fetchSlotBlock returns the block at slot from the block cache, the block archive or the beacon node, nil for none.
func (b *BeaconNode) fetchSlotBlock(ctx context.Context, slot uint64) (*spec.VersionedSignedBeaconBlock, error) {
	id := strconv.FormatUint(slot, 10)

	if item := b.blockCache.Get(id); item != nil && item.Value() != nil {
		return item.Value(), nil
	}

	if block, ok := b.blockFromArchive(ctx, id); ok {
		return block, nil
	}

	block, err := b.beacon.FetchBlock(ctx, id)
	if err != nil {
		return nil, err
	}

	if block != nil {
		b.blockCache.Set(id, block, time.Hour)
	}

	return block, nil
}

// blockHeader is the header of the block with the given id; ok is false when the node does not have it.
func (b *BeaconNode) blockHeader(ctx context.Context, id string) (*phase0.BeaconBlockHeader, bool, error) {
	provider, ok := b.beacon.Service().(client.BeaconBlockHeadersProvider)
	if !ok {
		return nil, false, errors.New("beacon block headers provider not available")
	}

	resp, err := provider.BeaconBlockHeader(ctx, &api.BeaconBlockHeaderOpts{Block: id})
	if err != nil {
		var apiErr *api.Error
		if errors.As(err, &apiErr) && apiErr.StatusCode == 404 {
			return nil, false, nil
		}

		return nil, false, errors.Wrapf(err, "failed to fetch block header %s", id)
	}

	if resp == nil || resp.Data == nil || resp.Data.Header == nil || resp.Data.Header.Message == nil {
		return nil, false, errors.Errorf("empty block header response for %s", id)
	}

	return resp.Data.Header.Message, true, nil
}

// blockFromArchive serves a slot identifier from the block archive; ok is false when the beacon node must answer.
func (b *BeaconNode) blockFromArchive(ctx context.Context, identifier string) (block *spec.VersionedSignedBeaconBlock, ok bool) {
	content, ok := b.archiveSlot(ctx, identifier)
	if !ok {
		return nil, false
	}

	return content.Block, true
}

// envelopeFromArchive is blockFromArchive for envelopes; a nil envelope with ok means the payload was withheld.
func (b *BeaconNode) envelopeFromArchive(ctx context.Context, identifier string) (envelope *gloas.SignedExecutionPayloadEnvelope, ok bool) {
	if b.archive == nil {
		return nil, false
	}

	slot, err := strconv.ParseUint(identifier, 10, 64)
	if err != nil {
		return nil, false
	}

	// Only eras with an envelope sidecar answer for envelopes; skip the era fetch otherwise.
	has, err := b.archive.HasEnvelopes(ctx, phase0.Slot(slot))
	if err != nil {
		b.log.WithError(err).WithField("slot", slot).WithContext(ctx).Warn("Block archive lookup failed, falling back to beacon node")

		return nil, false
	}

	if !has {
		return nil, false
	}

	content, ok := b.archiveSlot(ctx, identifier)
	if !ok || !content.EnvelopeKnown {
		return nil, false
	}

	return content.Envelope, true
}

// archiveSlot fetches the era range holding a slot identifier and caches its blocks and known Gloas+ envelopes.
func (b *BeaconNode) archiveSlot(ctx context.Context, identifier string) (content blockarchive.Slot, ok bool) {
	if b.archive == nil {
		return content, false
	}

	slot, err := strconv.ParseUint(identifier, 10, 64)
	if err != nil {
		return content, false
	}

	slots, found, err := b.archive.Slots(ctx, phase0.Slot(slot))
	if err != nil {
		b.log.WithError(err).WithField("slot", slot).WithContext(ctx).Warn("Block archive fetch failed, falling back to beacon node")

		return content, false
	}

	if !found {
		return content, false
	}

	// Cache the whole range even when it cannot answer for this slot, so its other slots skip the archive.
	for s, c := range slots {
		key := strconv.FormatUint(uint64(s), 10)
		b.blockCache.Set(key, c.Block, time.Hour)

		if c.EnvelopeKnown {
			b.envelopeCache.Set(key, c.Envelope, time.Hour)
		}
	}

	content, ok = slots[phase0.Slot(slot)]

	return content, ok
}

func (b *BeaconNode) LazyLoadBeaconBlock(identifier string) {
	// Don't add the block to the preload queue if it's already in the cache.
	if item := b.blockCache.Get(identifier); item != nil {
		return
	}

	b.blockPreloadChan <- identifier
}

// GetValidators returns a list of validators by its identifier. Validators can be cached internally.
func (b *BeaconNode) GetValidators(ctx context.Context, identifier string) (map[phase0.ValidatorIndex]*apiv1.Validator, error) {
	ctx, span := observability.Tracer().Start(ctx, "ethereum.beacon.GetValidators", trace.WithAttributes(attribute.String("identifier", identifier)))

	defer span.End()

	// Check the cache first.
	if item := b.validatorsCache.Get(identifier); item != nil {
		b.log.WithField("identifier", identifier).WithContext(ctx).Debug("Validator cache hit")

		span.SetAttributes(attribute.Bool("cached", true))

		return item.Value(), nil
	}

	span.SetAttributes(attribute.Bool("cached", false))

	// Use singleflight to ensure we only make one request for validators at a time.
	x, err, shared := b.validatorsSfGroup.Do(identifier, func() (any, error) {
		span.AddEvent("Acquiring semaphore...")

		// Acquire a semaphore before proceeding.
		b.validatorsPreloadSem <- struct{}{}
		defer func() { <-b.validatorsPreloadSem }()

		span.AddEvent("Semaphore acquired. Fetching validators from beacon api...")

		validators, err := b.fetchValidators(ctx, identifier)
		if err != nil {
			return nil, err
		}

		span.AddEvent("Validators fetched from beacon node.")

		// Add it to the cache.
		b.validatorsCache.Set(identifier, validators, 5*time.Minute)

		return validators, nil
	})
	if err != nil {
		span.SetStatus(codes.Error, err.Error())

		return nil, err
	}

	span.AddEvent("Validators fetching complete.", trace.WithAttributes(attribute.Bool("shared", shared)))

	return x.(map[phase0.ValidatorIndex]*apiv1.Validator), nil
}

func (b *BeaconNode) fetchValidators(ctx context.Context, identifier string) (map[phase0.ValidatorIndex]*apiv1.Validator, error) {
	if slot, err := strconv.ParseUint(identifier, 10, 64); err == nil {
		sp, err := b.beacon.Spec()
		if err != nil {
			return nil, errors.Wrap(err, "failed to obtain spec")
		}

		isPQ, err := b.isPQ(ctx, phase0.Epoch(slot/uint64(sp.SlotsPerEpoch)))
		if err != nil {
			return nil, err
		}

		if isPQ {
			return b.fetchPQValidators(ctx, identifier)
		}
	}

	provider, err := b.getValidatorsClient()
	if err != nil {
		return nil, err
	}

	validatorsResponse, err := provider.Validators(ctx, &api.ValidatorsOpts{
		State: identifier,
		Common: api.CommonOpts{
			Timeout: 6 * time.Minute, // If we can't fetch 1 epoch of validators within 1 epoch we will never catch up.
		},
	})
	if err != nil {
		return nil, err
	}

	return validatorsResponse.Data, nil
}

func (b *BeaconNode) LazyLoadValidators(stateID string) {
	if item := b.validatorsCache.Get(stateID); item != nil {
		return
	}

	b.validatorsPreloadChan <- stateID
}

func (b *BeaconNode) getValidatorsClient() (client.ValidatorsProvider, error) {
	if provider, isProvider := b.beacon.Service().(client.ValidatorsProvider); isProvider {
		return provider, nil
	}

	return nil, errors.New("validator states client not found")
}

// GetExecutionPayloadEnvelope returns the Gloas (EIP-7732) ExecutionPayloadEnvelope
// for a given block ID (root, slot, "head", etc.). Multiple derivers consume the
// envelope per slot; the cache + singleflight pattern prevents duplicate fetches.
//
// A nil-but-no-error result means the envelope is known to be absent — typically
// the builder withheld the payload (payload_status = EMPTY). Callers should treat
// this as a no-op for envelope-sourced data and emit nothing for the slot.
func (b *BeaconNode) GetExecutionPayloadEnvelope(
	ctx context.Context,
	blockID string,
) (*gloas.SignedExecutionPayloadEnvelope, error) {
	ctx, span := observability.Tracer().Start(ctx, "ethereum.beacon.GetExecutionPayloadEnvelope",
		trace.WithAttributes(attribute.String("block_id", blockID)))
	defer span.End()

	if item := b.envelopeCache.Get(blockID); item != nil {
		span.SetAttributes(attribute.Bool("cached", true))

		return item.Value(), nil
	}

	span.SetAttributes(attribute.Bool("cached", false))

	x, err, shared := b.envelopeSfGroup.Do(blockID, func() (any, error) {
		if envelope, ok := b.envelopeFromArchive(ctx, blockID); ok {
			span.AddEvent("Envelope served from block archive.")

			return envelope, nil
		}

		provider, err := b.getExecutionPayloadProvider()
		if err != nil {
			return nil, err
		}

		resp, err := provider.SignedExecutionPayloadEnvelope(ctx, &api.SignedExecutionPayloadEnvelopeOpts{
			Block: blockID,
		})
		if err != nil {
			var apiErr *api.Error
			if errors.As(err, &apiErr) && apiErr.StatusCode == 404 {
				withheld, werr := b.payloadWithheld(ctx, blockID)
				if werr != nil {
					return nil, errors.Wrapf(err,
						"envelope absent and reveal status undetermined (%s)", werr.Error())
				}

				if withheld {
					b.log.WithField("block_id", blockID).
						Info("Execution payload envelope permanently absent: builder withheld the payload")

					var absent *gloas.SignedExecutionPayloadEnvelope

					b.envelopeCache.Set(blockID, absent, time.Hour)

					return absent, nil
				}
			}

			return nil, err
		}

		var envelope *gloas.SignedExecutionPayloadEnvelope
		if resp != nil && resp.Data != nil {
			envelope = resp.Data.Gloas
			if resp.Data.PQ != nil {
				envelope = gloasEnvelopeFromPQ(resp.Data.PQ)
			}
		}

		b.envelopeCache.Set(blockID, envelope, time.Hour)

		return envelope, nil
	})
	if err != nil {
		span.SetStatus(codes.Error, err.Error())

		return nil, err
	}

	span.AddEvent("Envelope fetch complete.", trace.WithAttributes(attribute.Bool("shared", shared)))

	envelope, ok := x.(*gloas.SignedExecutionPayloadEnvelope)
	if !ok {
		return nil, fmt.Errorf("envelope singleflight returned unexpected type %T", x)
	}

	return envelope, nil
}

func (b *BeaconNode) getExecutionPayloadProvider() (client.ExecutionPayloadProvider, error) {
	if provider, isProvider := b.beacon.Service().(client.ExecutionPayloadProvider); isProvider {
		return provider, nil
	}

	return nil, errors.New("execution payload provider not available")
}

// payloadWithheld reports whether the canonical execution payload for the given
// block was withheld by its builder. Under EIP-7732 the EL head only advances
// when a payload is revealed, so the next canonical block's bid builds on this
// block's promised block hash iff this payload was revealed. This separates a
// permanently-absent envelope (withheld — the beacon API 404s it forever, on
// every client) from one the node should be able to serve but can't (a data
// gap worth retrying). Only block data is used, so the answer is available on
// any node that retains blocks, including during backfill.
func (b *BeaconNode) payloadWithheld(ctx context.Context, blockID string) (bool, error) {
	block, err := b.GetBeaconBlock(ctx, blockID)
	if err != nil {
		return false, errors.Wrapf(err, "failed to get block %s", blockID)
	}

	if block == nil {
		return false, errors.Errorf("no canonical block for id %s", blockID)
	}

	bid, slot, err := gloasBid(block)
	if err != nil {
		return false, err
	}

	next, _, _, err := b.nextCanonicalBlock(ctx, uint64(slot))
	if err != nil {
		return false, errors.Wrapf(err, "failed to get the canonical block after slot %d", slot)
	}

	nextBid, _, err := gloasBid(next)
	if err != nil {
		return false, err
	}

	return bidPayloadWithheld(bid, nextBid)
}

// gloasEnvelopeFromPQ carries a PQ envelope's payload and the execution requests Gloas also has. The
// XMSS signature and the PQ-only requests (preregistrations, voluntary exits) have no place in it.
func gloasEnvelopeFromPQ(envelope *pq.SignedExecutionPayloadEnvelope) *gloas.SignedExecutionPayloadEnvelope {
	if envelope.Message == nil {
		return &gloas.SignedExecutionPayloadEnvelope{}
	}

	message := &gloas.ExecutionPayloadEnvelope{
		Payload:               envelope.Message.Payload,
		BuilderIndex:          envelope.Message.BuilderIndex,
		BeaconBlockRoot:       envelope.Message.BeaconBlockRoot,
		ParentBeaconBlockRoot: envelope.Message.ParentBeaconBlockRoot,
	}

	if requests := envelope.Message.ExecutionRequests; requests != nil {
		message.ExecutionRequests = &gloas.ExecutionRequests{
			Deposits:        requests.Deposits,
			Withdrawals:     requests.Withdrawals,
			Consolidations:  requests.Consolidations,
			BuilderDeposits: requests.BuilderDeposits,
			BuilderExits:    requests.BuilderExits,
		}
	}

	return &gloas.SignedExecutionPayloadEnvelope{Message: message}
}

// gloasBid extracts the execution payload bid and slot from a Gloas or PQ block.
func gloasBid(block *spec.VersionedSignedBeaconBlock) (*gloas.ExecutionPayloadBid, phase0.Slot, error) {
	if block.Version < spec.DataVersionGloas {
		return nil, 0, errors.Errorf("block version %s predates gloas", block.Version)
	}

	if block.Version == spec.DataVersionPQ {
		if block.PQ == nil || block.PQ.Message == nil || block.PQ.Message.Body == nil ||
			block.PQ.Message.Body.SignedExecutionPayloadBid == nil ||
			block.PQ.Message.Body.SignedExecutionPayloadBid.Message == nil {
			return nil, 0, errors.New("pq block is missing its execution payload bid")
		}

		return block.PQ.Message.Body.SignedExecutionPayloadBid.Message, block.PQ.Message.Slot, nil
	}

	if block.Gloas == nil || block.Gloas.Message == nil || block.Gloas.Message.Body == nil ||
		block.Gloas.Message.Body.SignedExecutionPayloadBid == nil ||
		block.Gloas.Message.Body.SignedExecutionPayloadBid.Message == nil {
		return nil, 0, errors.New("gloas block is missing its execution payload bid")
	}

	return block.Gloas.Message.Body.SignedExecutionPayloadBid.Message, block.Gloas.Message.Slot, nil
}

// bidPayloadWithheld decides from two consecutive canonical bids whether the
// earlier block's payload was withheld. The next bid builds on the earlier
// bid's block hash iff that payload was revealed, and on the earlier bid's
// parent hash iff it was withheld. Anything else means our view of the chain
// is inconsistent (e.g. a reorg between fetches), so fail rather than guess.
func bidPayloadWithheld(bid, nextBid *gloas.ExecutionPayloadBid) (bool, error) {
	switch nextBid.ParentBlockHash {
	case bid.BlockHash:
		return false, nil
	case bid.ParentBlockHash:
		return true, nil
	default:
		return false, errors.Errorf(
			"next bid parent block hash %#x matches neither promised block hash %#x nor parent %#x",
			nextBid.ParentBlockHash, bid.BlockHash, bid.ParentBlockHash)
	}
}

func (b *BeaconNode) DeleteValidatorsFromCache(stateID string) {
	b.validatorsCache.Delete(stateID)
}
