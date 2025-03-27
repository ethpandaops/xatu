package ethereum

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/ethpandaops/beacon/pkg/beacon"
	"github.com/ethpandaops/xatu/pkg/clmimicry/ethereum/services"
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

func NewBeaconNode(ctx context.Context, name string, config *Config, log logrus.FieldLogger, addr string) (*BeaconNode, error) {
	opts := *beacon.
		DefaultOptions().
		DisablePrometheusMetrics()

	opts.HealthCheck.Interval.Duration = time.Second * 3
	opts.HealthCheck.SuccessfulResponses = 1

	node := beacon.NewNode(log, &beacon.Config{
		Name: name,
		Addr: addr,
	}, "xatu_cl_mimicry", opts)

	metadata := services.NewMetadataService(log, node, config.Network)
	duties := services.NewDutiesService(log, node, &metadata)

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

	if err := b.beacon.Start(ctx); err != nil {
		return err
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
