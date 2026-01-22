package xatu

import (
	"context"
	"errors"
	"fmt"
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

func New(ctx context.Context, name string, config *xatuCoordinator.Config, handlers *handler.Peer, captureDelay time.Duration, ethereumConfig *ethereum.Config, log logrus.FieldLogger) (*Xatu, error) {
	if config == nil {
		return nil, errors.New("config is required")
	}

	if err := config.Validate(); err != nil {
		return nil, err
	}

	coordinator, err := xatuCoordinator.NewCoordinator(ctx, name, config, log)
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

	// Add seed peers from network config enodes
	if err := x.addSeedPeers(ctx); err != nil {
		return err
	}

	if err := x.startCrons(ctx); err != nil {
		return err
	}

	return nil
}

// addSeedPeers adds the enodes from network config as initial seed peers.
// Returns an error if enodes are configured but none could be started.
func (x *Xatu) addSeedPeers(ctx context.Context) error {
	enodes := x.coordinator.GetEnodes()
	if len(enodes) == 0 {
		return nil
	}

	x.log.WithField("count", len(enodes)).Info("Adding seed peers from network config enodes")

	x.mu.Lock()
	defer x.mu.Unlock()

	started := 0

	for _, enode := range enodes {
		if _, ok := x.peers[enode]; !ok {
			x.peers[enode] = xatuPeer.NewPeer(x.log, x.handlers, x.cache, enode, 60*time.Second, x.captureDelay, x.ethereumConfig)

			if err := x.peers[enode].Start(ctx); err != nil {
				x.log.WithError(err).WithField("enode", enode).Error("Failed to start seed peer")
				delete(x.peers, enode)
			} else {
				started++
			}
		}
	}

	if started == 0 {
		return fmt.Errorf("failed to start any seed peers from %d configured enodes", len(enodes))
	}

	x.log.WithFields(logrus.Fields{
		"started": started,
		"total":   len(enodes),
	}).Info("Started seed peers from network config")

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
