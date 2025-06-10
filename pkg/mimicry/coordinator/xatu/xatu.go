package xatu

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/ethpandaops/xatu/pkg/mimicry/coordinator/cache"
	xatuCoordinator "github.com/ethpandaops/xatu/pkg/mimicry/coordinator/xatu/coordinator"
	xatuPeer "github.com/ethpandaops/xatu/pkg/mimicry/coordinator/xatu/peer"
	"github.com/ethpandaops/xatu/pkg/mimicry/ethereum"
	"github.com/ethpandaops/xatu/pkg/mimicry/p2p/handler"
	xatupb "github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/go-co-op/gocron/v2"
	"github.com/sirupsen/logrus"
)

const Type = "xatu"

type Xatu struct {
	handlers       *handler.Peer
	captureDelay   time.Duration
	ethereumConfig *ethereum.Config
	log            logrus.FieldLogger

	cache       *cache.SharedCache
	coordinator *xatuCoordinator.Coordinator

	mu    sync.Mutex
	peers map[string]*xatuPeer.Peer

	metrics *Metrics
}

func New(name string, config *xatuCoordinator.Config, handlers *handler.Peer, captureDelay time.Duration, ethereumConfig *ethereum.Config, log logrus.FieldLogger) (*Xatu, error) {
	if config == nil {
		return nil, errors.New("config is required")
	}

	if err := config.Validate(); err != nil {
		return nil, err
	}

	coordinator, err := xatuCoordinator.NewCoordinator(name, config, log)
	if err != nil {
		return nil, err
	}

	handlers.ExecutionStatus = coordinator.HandleExecutionNodeRecordStatus

	return &Xatu{
		handlers:       handlers,
		captureDelay:   captureDelay,
		ethereumConfig: ethereumConfig,
		log:            log,
		cache:          cache.NewSharedCache(),
		coordinator:    coordinator,
		mu:             sync.Mutex{},
		peers:          make(map[string]*xatuPeer.Peer),
		metrics:        NewMetrics("xatu_mimicry_coordinator_xatu"),
	}, nil
}

func (x *Xatu) Type() string {
	return Type
}

func (x *Xatu) Start(ctx context.Context) error {
	if err := x.coordinator.Start(ctx); err != nil {
		return err
	}

	if err := x.cache.Start(ctx); err != nil {
		return err
	}

	if err := x.startCrons(ctx); err != nil {
		return err
	}

	return nil
}

func (x *Xatu) Stop(ctx context.Context) error {
	return nil
}

func (x *Xatu) startCrons(ctx context.Context) error {
	c, err := gocron.NewScheduler(gocron.WithLocation(time.Local))
	if err != nil {
		return err
	}

	if _, err := c.NewJob(
		gocron.DurationJob(5*time.Second),
		gocron.NewTask(
			func(ctx context.Context) {
				x.mu.Lock()
				defer x.mu.Unlock()

				connectedPeers := 0
				connectionAttempts := 0
				for _, peer := range x.peers {
					if peer.Record.Connected {
						connectedPeers++
					}
					connectionAttempts += int(peer.Record.ConnectionAttempts)
				}
				x.metrics.SetPeers(connectedPeers, "connected")
				x.metrics.SetPeers(len(x.peers)-connectedPeers, "disconnected")
				x.metrics.SetPeerConnectionAttempts(connectionAttempts)
			},
			ctx,
		),
		gocron.WithStartAt(gocron.WithStartImmediately()),
	); err != nil {
		return err
	}

	if _, err := c.NewJob(
		gocron.DurationJob(5*time.Minute),
		gocron.NewTask(
			func(ctx context.Context) {
				x.mu.Lock()
				defer x.mu.Unlock()

				var records []*xatupb.CoordinatedNodeRecord
				for _, peer := range x.peers {
					records = append(records, peer.Record)
				}

				res, err := x.coordinator.CoordinateExecutionNodeRecords(ctx, records)
				if err != nil {
					x.log.WithError(err).Error("failed to coordinate execution node records")

					return
				}

				if res == nil {
					x.log.Error("failed to coordinate execution node records: nil response")

					return
				}

				retryDelay := time.Duration(res.RetryDelay) * time.Second

				for i, peer := range x.peers {
					found := false
					for _, record := range res.NodeRecords {
						if record == i {
							found = true
							peer.RetryDelay(retryDelay)

							break
						}
					}

					// remove peer
					if !found {
						if err := peer.Stop(); err != nil {
							x.log.WithError(err).Error("failed to stop peer")
						}
						delete(x.peers, i)
					}
				}

				for _, record := range res.NodeRecords {
					if _, ok := x.peers[record]; !ok {
						x.peers[record] = xatuPeer.NewPeer(x.log, x.handlers, x.cache, record, retryDelay, x.captureDelay, x.ethereumConfig)
						if err := x.peers[record].Start(ctx); err != nil {
							x.log.WithError(err).Error("failed to start peer")
							delete(x.peers, record)
						}
					}
				}
			},
			ctx,
		),
		gocron.WithStartAt(gocron.WithStartImmediately()),
	); err != nil {
		return err
	}

	c.Start()

	return nil
}
