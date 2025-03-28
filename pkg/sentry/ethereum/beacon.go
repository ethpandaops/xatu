package ethereum

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/ethpandaops/beacon/pkg/beacon"
	"github.com/ethpandaops/xatu/pkg/sentry/ethereum/services"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type BeaconNode struct {
	config *Config
	log    logrus.FieldLogger

	beacon beacon.Node

	services []services.Service

	onReadyCallbacks []func(ctx context.Context) error
}

func NewBeaconNode(ctx context.Context, name string, config *Config, log logrus.FieldLogger, opt *Options) (*BeaconNode, error) {
	opts := *beacon.
		DefaultOptions().
		DisablePrometheusMetrics()

	if config.BeaconSubscriptions != nil {
		opts.BeaconSubscription = beacon.BeaconSubscriptionOptions{
			Enabled: true,
			Topics:  *config.BeaconSubscriptions,
		}
	} else {
		opts.EnableDefaultBeaconSubscription()
	}

	opts.HealthCheck.Interval.Duration = time.Second * 3
	opts.HealthCheck.SuccessfulResponses = 1

	node := beacon.NewNode(log, &beacon.Config{
		Name:    name,
		Addr:    config.BeaconNodeAddress,
		Headers: config.BeaconNodeHeaders,
	}, "xatu_sentry", opts)

	metadata := services.NewMetadataService(log, node, config.OverrideNetworkName)
	duties := services.NewDutiesService(log, node, &metadata, opt.FetchProposerDuties, opt.FetchBeaconCommittees)

	svcs := []services.Service{
		&metadata,
		&duties,
	}

	return &BeaconNode{
		config:   config,
		log:      log.WithField("module", "sentry/ethereum/beacon"),
		beacon:   node,
		services: svcs,
	}, nil
}

func (b *BeaconNode) Start(ctx context.Context) error {
	errs := make(chan error, 1)
	healthyFirstTime := make(chan struct{})

	b.beacon.OnFirstTimeHealthy(ctx, func(ctx context.Context, event *beacon.FirstTimeHealthyEvent) error {
		b.log.Info("Upstream beacon node is healthy")

		close(healthyFirstTime)

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

			b.log.WithField("service", service.Name()).Info("Waiting for service to be ready")

			wg.Wait()
		}

		b.log.Info("All services are ready")

		for _, callback := range b.onReadyCallbacks {
			if err := callback(ctx); err != nil {
				errs <- fmt.Errorf("failed to run on ready callback: %w", err)
			}
		}

		return nil
	})

	b.beacon.StartAsync(ctx)

	select {
	case err := <-errs:
		return err
	case <-ctx.Done():
		return ctx.Err()
	case <-healthyFirstTime:
		// Beacon node is healthy, continue with normal operation
	case <-time.After(10 * time.Minute):
		return errors.New("upstream beacon node is not healthy. check your configuration.")
	}

	// Wait for any errors after the first healthy event
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
		// This should never happen. If it does, good luck.
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
		// This should never happen. If it does, good luck.
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

	wallclock := b.Metadata().Wallclock()
	if wallclock == nil {
		return errors.New("missing wallclock")
	}

	for _, service := range b.services {
		if err := service.Ready(ctx); err != nil {
			return errors.Wrapf(err, "service %s is not ready", service.Name())
		}
	}

	return nil
}
