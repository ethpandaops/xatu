package ethereum

import (
	"context"
	"fmt"
	"time"

	"github.com/ethpandaops/beacon/pkg/beacon"
	"github.com/go-co-op/gocron"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type BeaconNode struct {
	config *Config
	log    logrus.FieldLogger

	beacon   beacon.Node
	metadata *MetadataService

	onReadyCallbacks []func(ctx context.Context) error
	readyPublished   bool
}

func NewBeaconNode(ctx context.Context, name string, config *Config, log logrus.FieldLogger) (*BeaconNode, error) {
	opts := *beacon.
		DefaultOptions().
		EnableDefaultBeaconSubscription()

	opts.HealthCheck.Interval.Duration = time.Second * 3
	opts.HealthCheck.SuccessfulResponses = 1

	node := beacon.NewNode(log, &beacon.Config{
		Name: name,
		Addr: config.BeaconNodeAddress,
	}, "xatu_sentry", opts)

	metadata := NewMetadataService(log, node)

	if err := metadata.Start(ctx); err != nil {
		return nil, errors.Wrap(err, "failed to start metadata service")
	}

	return &BeaconNode{
		config:   config,
		log:      log.WithField("module", "sentry/ethereum/beacon"),
		beacon:   node,
		metadata: &metadata,
	}, nil
}

func (b *BeaconNode) Start(ctx context.Context) error {
	s := gocron.NewScheduler(time.Local)

	// TODO(sam.calder-mason): Make this entirely event driven.
	if _, err := s.Every("2s").Do(func() {
		if err := b.checkForReadyPublish(ctx); err != nil {
			b.log.WithError(err).Warn("failed to check for ready publish")
		}
	}); err != nil {
		return err
	}

	s.StartAsync()

	return b.beacon.Start(ctx)
}

func (b *BeaconNode) Node() beacon.Node {
	return b.beacon
}

func (b *BeaconNode) Metadata() *MetadataService {
	return b.metadata
}

func (b *BeaconNode) OnReady(_ context.Context, callback func(ctx context.Context) error) {
	b.onReadyCallbacks = append(b.onReadyCallbacks, callback)
}

func (b *BeaconNode) Synced(_ context.Context) error {
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

	wallclock := b.metadata.Wallclock()
	if wallclock == nil {
		return errors.New("missing wallclock")
	}

	currentSlot := wallclock.Slots().Current()

	if currentSlot.Number()-uint64(syncState.HeadSlot) > 3 {
		return fmt.Errorf("beacon node is too far behind head, head slot is %d, current slot is %d", syncState.HeadSlot, currentSlot.Number())
	}

	if !b.readyPublished {
		return errors.New("internal beacon node is not ready")
	}

	return nil
}

func (b *BeaconNode) checkForReadyPublish(ctx context.Context) error {
	if b.readyPublished {
		return nil
	}

	if err := b.metadata.Ready(); err != nil {
		return errors.Wrap(err, "metadata service is not ready")
	}

	status := b.beacon.Status()
	if status == nil {
		return errors.New("failed to get beacon node status")
	}

	syncState := status.SyncState()
	if syncState == nil {
		return errors.New("missing beacon node status sync state")
	}

	if syncState.SyncDistance > 3 {
		return errors.New("beacon node is not synced")
	}

	for _, callback := range b.onReadyCallbacks {
		if err := callback(ctx); err != nil {
			b.log.WithError(err).Warn("failed to run on ready callback")
		}
	}

	b.readyPublished = true

	return nil
}
