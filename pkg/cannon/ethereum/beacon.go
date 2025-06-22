package ethereum

import (
	"context"
	"fmt"
	"sync"
	"time"

	client "github.com/attestantio/go-eth2-client"
	"github.com/attestantio/go-eth2-client/api"
	apiv1 "github.com/attestantio/go-eth2-client/api/v1"
	ehttp "github.com/attestantio/go-eth2-client/http"
	"github.com/attestantio/go-eth2-client/spec"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/ethpandaops/beacon/pkg/beacon"
	"github.com/ethpandaops/xatu/pkg/cannon/ethereum/services"
	"github.com/ethpandaops/xatu/pkg/networks"
	"github.com/ethpandaops/xatu/pkg/observability"
	"github.com/jellydator/ttlcache/v3"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/sync/singleflight"
)

type BeaconNode struct {
	config *Config
	log    logrus.FieldLogger

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
}

func NewBeaconNode(ctx context.Context, name string, config *Config, log logrus.FieldLogger) (*BeaconNode, error) {
	namespace := "xatu_cannon"

	opts := *beacon.
		DefaultOptions().
		DisableEmptySlotDetection().
		DisablePrometheusMetrics()

	opts.GoEth2ClientParams = []ehttp.Parameter{
		// Default JSON until https://github.com/attestantio/go-eth2-client/pull/198 is merged.
		ehttp.WithEnforceJSON(true),
	}

	opts.HealthCheck.Interval.Duration = time.Second * 3
	opts.HealthCheck.SuccessfulResponses = 1

	opts.BeaconSubscription.Enabled = false

	node := beacon.NewNode(log, &beacon.Config{
		Name:    name,
		Addr:    config.BeaconNodeAddress,
		Headers: config.BeaconNodeHeaders,
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
	sem := make(chan struct{}, config.BlockPreloadWorkers)
	validatorsSem := make(chan struct{}, 1)

	return &BeaconNode{
		config:            config,
		log:               log.WithField("module", "cannon/ethereum/beacon"),
		beacon:            node,
		services:          svcs,
		sfGroup:           &singleflight.Group{},
		validatorsSfGroup: &singleflight.Group{},
		blockCache: ttlcache.New(
			ttlcache.WithTTL[string, *spec.VersionedSignedBeaconBlock](config.BlockCacheTTL.Duration),
			ttlcache.WithCapacity[string, *spec.VersionedSignedBeaconBlock](config.BlockCacheSize),
		),
		blockPreloadChan: make(chan string, config.BlockPreloadQueueSize),
		blockPreloadSem:  sem,
		validatorsCache: ttlcache.New(
			ttlcache.WithTTL[string, map[phase0.ValidatorIndex]*apiv1.Validator](5*time.Minute),
			ttlcache.WithCapacity[string, map[phase0.ValidatorIndex]*apiv1.Validator](4),
		),
		validatorsPreloadChan: make(chan string, 2),
		validatorsPreloadSem:  validatorsSem,
		metrics:               NewMetrics(namespace, name),
	}, nil
}

func (b *BeaconNode) Start(ctx context.Context) error {
	errs := make(chan error, 1)

	go func() {
		wg := sync.WaitGroup{}

		for _, service := range b.services {
			wg.Add(1)

			service.OnReady(ctx, func(ctx context.Context) error {
				b.log.WithField("service", service.Name()).Info("Service is ready")

				wg.Done()

				return nil
			})

			b.log.WithField("service", service.Name()).Info("Starting service")

			if err := service.Start(ctx); err != nil {
				errs <- fmt.Errorf("failed to start service: %w", err)
			}

			wg.Wait()
		}

		if b.Metadata().Network.Name == networks.NetworkNameUnknown {
			errs <- errors.New("unknown network detected. Please override the network name via config if you are using a custom network")
		}

		b.log.Info("All services are ready")

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
		b.log.WithField("identifier", item.Key()).WithField("reason", reason).Trace("Block evicted from cache")
	})

	go b.blockCache.Start()

	for i := 0; i < int(b.config.BlockPreloadWorkers); i++ {
		go func() {
			for identifier := range b.blockPreloadChan {
				b.metrics.SetPreloadBlockQueueSize(string(b.Metadata().Network.Name), len(b.blockPreloadChan))

				b.log.WithField("identifier", identifier).Trace("Preloading block")

				//nolint:errcheck // We don't care about errors here.
				b.GetBeaconBlock(ctx, identifier, true)
			}
		}()
	}

	go func() {
		for identifier := range b.validatorsPreloadChan {
			b.log.WithField("identifier", identifier).Trace("Preloading validators")

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

	metadataService, ok := service.(*services.MetadataService)
	if !ok {
		return nil
	}

	return metadataService
}

func (b *BeaconNode) Duties() *services.DutiesService {
	service, err := b.getServiceByName("duties")
	if err != nil {
		// This should never happen. If it does, good luck.
		return nil
	}

	dutiesService, ok := service.(*services.DutiesService)
	if !ok {
		return nil
	}

	return dutiesService
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
	x, err, shared := b.sfGroup.Do(identifier, func() (interface{}, error) {
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

	block, ok := x.(*spec.VersionedSignedBeaconBlock)
	if !ok {
		return nil, errors.New("failed to cast to VersionedSignedBeaconBlock")
	}

	return block, nil
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
		b.log.WithField("identifier", identifier).Debug("Validator cache hit")

		span.SetAttributes(attribute.Bool("cached", true))

		return item.Value(), nil
	}

	span.SetAttributes(attribute.Bool("cached", false))

	// Use singleflight to ensure we only make one request for validators at a time.
	x, err, shared := b.validatorsSfGroup.Do(identifier, func() (interface{}, error) {
		span.AddEvent("Acquiring semaphore...")

		// Acquire a semaphore before proceeding.
		b.validatorsPreloadSem <- struct{}{}
		defer func() { <-b.validatorsPreloadSem }()

		span.AddEvent("Semaphore acquired. Fetching validators from beacon api...")

		client, err := b.getValidatorsClient(ctx)
		if err != nil {
			return nil, err
		}

		// Not in the cache, so fetch it.
		validatorsResponse, err := client.Validators(ctx, &api.ValidatorsOpts{
			State: identifier,
			Common: api.CommonOpts{
				Timeout: 6 * time.Minute, // If we can't fetch 1 epoch of validators within 1 epoch we will never catch up.
			},
		})
		if err != nil {
			return nil, err
		}

		validators := validatorsResponse.Data

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

	validators, ok := x.(map[phase0.ValidatorIndex]*apiv1.Validator)
	if !ok {
		return nil, errors.New("failed to cast to validator map")
	}

	return validators, nil
}

func (b *BeaconNode) LazyLoadValidators(stateID string) {
	if item := b.validatorsCache.Get(stateID); item != nil {
		return
	}

	b.validatorsPreloadChan <- stateID
}

func (b *BeaconNode) getValidatorsClient(ctx context.Context) (client.ValidatorsProvider, error) {
	if provider, isProvider := b.beacon.Service().(client.ValidatorsProvider); isProvider {
		return provider, nil
	}

	return nil, errors.New("validator states client not found")
}

func (b *BeaconNode) DeleteValidatorsFromCache(stateID string) {
	b.validatorsCache.Delete(stateID)
}
