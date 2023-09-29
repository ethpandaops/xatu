package ethereum

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/attestantio/go-eth2-client/spec"
	"github.com/ethpandaops/beacon/pkg/beacon"
	"github.com/ethpandaops/xatu/pkg/cannon/ethereum/services"
	"github.com/go-co-op/gocron"
	"github.com/jellydator/ttlcache/v3"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
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
}

func NewBeaconNode(ctx context.Context, name string, config *Config, log logrus.FieldLogger) (*BeaconNode, error) {
	namespace := "xatu_cannon"

	opts := *beacon.
		DefaultOptions().
		DisableEmptySlotDetection().
		DisablePrometheusMetrics()

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

	svcs := []services.Service{
		&metadata,
	}

	return &BeaconNode{
		config:   config,
		log:      log.WithField("module", "cannon/ethereum/beacon"),
		beacon:   node,
		services: svcs,
		blockCache: ttlcache.New(
			ttlcache.WithTTL[string, *spec.VersionedSignedBeaconBlock](config.BlockCacheTTL.Duration),
			ttlcache.WithCapacity[string, *spec.VersionedSignedBeaconBlock](config.BlockCacheSize),
		),
		sfGroup:          &singleflight.Group{},
		blockPreloadChan: make(chan string, config.BlockPreloadQueueSize),
		metrics:          NewMetrics(namespace, name),
	}, nil
}

func (b *BeaconNode) Start(ctx context.Context) error {
	s := gocron.NewScheduler(time.Local)

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

		b.log.Info("All services are ready")

		for _, callback := range b.onReadyCallbacks {
			if err := callback(ctx); err != nil {
				errs <- fmt.Errorf("failed to run on ready callback: %w", err)
			}
		}
	}()

	s.StartAsync()

	if err := b.beacon.Start(ctx); err != nil {
		return err
	}

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

	return service.(*services.MetadataService)
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
	b.metrics.IncBlocksFetched(string(b.Metadata().Network.Name))

	// Create a buffered channel (semaphore) to limit the number of concurrent goroutines.
	sem := make(chan struct{}, b.config.BlockPreloadWorkers)

	// Check the cache first.
	if item := b.blockCache.Get(identifier); item != nil {
		if len(ignoreMetrics) != 0 && ignoreMetrics[0] {
			b.metrics.IncBlockCacheHit(string(b.Metadata().Network.Name))
		}

		return item.Value(), nil
	}

	if len(ignoreMetrics) != 0 && ignoreMetrics[0] {
		b.metrics.IncBlockCacheMiss(string(b.Metadata().Network.Name))
	}

	// Use singleflight to ensure we only make one request for a block at a time.
	x, err, _ := b.sfGroup.Do(identifier, func() (interface{}, error) {
		// Acquire a semaphore before proceeding.
		sem <- struct{}{}
		defer func() { <-sem }()

		// Not in the cache, so fetch it.
		block, err := b.beacon.FetchBlock(ctx, identifier)
		if err != nil {
			return nil, err
		}

		// Add it to the cache.
		b.blockCache.Set(identifier, block, time.Hour)

		return block, nil
	})
	if err != nil {
		if len(ignoreMetrics) != 0 && ignoreMetrics[0] {
			b.metrics.IncBlocksFetchErrors(string(b.Metadata().Network.Name))
		}

		return nil, err
	}

	return x.(*spec.VersionedSignedBeaconBlock), nil
}

func (b *BeaconNode) LazyLoadBeaconBlock(identifier string) {
	b.blockPreloadChan <- identifier
}
