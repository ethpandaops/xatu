package ethereum

import (
	"context"
	"time"

	"github.com/go-co-op/gocron"
	"github.com/pkg/errors"
	"github.com/samcm/beacon"
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
		DisableFetchingProposerDuties().
		EnableDefaultBeaconSubscription()

	opts.HealthCheck.Interval.Duration = time.Second * 3
	opts.HealthCheck.SuccessfulResponses = 1

	node := beacon.NewNode(log, &beacon.Config{
		Name: name,
		Addr: config.BeaconNodeAddress,
	}, "xatu", opts)

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
			b.log.WithError(err).Error("failed to check for ready publish")
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

func (b *BeaconNode) OnReady(ctx context.Context, callback func(ctx context.Context) error) {
	b.onReadyCallbacks = append(b.onReadyCallbacks, callback)
}

func (b *BeaconNode) checkForReadyPublish(ctx context.Context) error {
	if b.readyPublished {
		return nil
	}

	if err := b.metadata.Ready(); err != nil {
		return errors.Wrap(err, "metadata service is not ready")
	}

	status := b.beacon.GetStatus(ctx)
	if status == nil {
		return errors.New("failed to get beacon node status")
	}

	syncState := status.SyncState()
	if syncState == nil {
		return errors.New("missing beacon node status sync state")
	}

	if syncState.IsSyncing {
		return errors.New("beacon node is syncing")
	}

	for _, callback := range b.onReadyCallbacks {
		if err := callback(ctx); err != nil {
			b.log.WithError(err).Error("failed to run on ready callback")
		}
	}

	b.readyPublished = true

	return nil
}
