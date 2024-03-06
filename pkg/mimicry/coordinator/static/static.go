package static

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/avast/retry-go/v4"
	"github.com/ethpandaops/xatu/pkg/mimicry/coordinator/cache"
	"github.com/ethpandaops/xatu/pkg/mimicry/p2p/execution"
	"github.com/ethpandaops/xatu/pkg/mimicry/p2p/handler"
	"github.com/go-co-op/gocron"
	"github.com/sirupsen/logrus"
)

const Type = "static"

type Static struct {
	config *Config

	handlers *handler.Peer

	captureDelay time.Duration

	log logrus.FieldLogger

	cache    *cache.SharedCache
	peersMux sync.Mutex
	peers    *map[string]bool

	metrics *Metrics
}

func New(name string, config *Config, handlers *handler.Peer, captureDelay time.Duration, log logrus.FieldLogger) (*Static, error) {
	if config == nil {
		return nil, errors.New("config is required")
	}

	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &Static{
		config:       config,
		handlers:     handlers,
		captureDelay: captureDelay,
		log:          log,
		cache:        cache.NewSharedCache(),
		peers:        &map[string]bool{},
		metrics:      NewMetrics("xatu_mimicry_coordinator_static"),
	}, nil
}

func (s *Static) Type() string {
	return Type
}

func (s *Static) Start(ctx context.Context) error {
	for _, nodeRecord := range s.config.NodeRecords {
		(*s.peers)[nodeRecord] = false
		go func(record string, peers *map[string]bool) {
			_ = retry.Do(
				func() error {
					peer, err := execution.New(ctx, s.log, record, s.handlers, s.captureDelay, s.cache)
					if err != nil {
						return err
					}

					defer func() {
						s.peersMux.Lock()
						(*peers)[record] = false
						s.peersMux.Unlock()

						if peer != nil {
							if err = peer.Stop(ctx); err != nil {
								s.log.WithError(err).Warn("failed to stop peer")
							}
						}
					}()

					disconnect, err := peer.Start(ctx)
					if err != nil {
						return err
					}

					s.peersMux.Lock()
					(*peers)[record] = true
					s.peersMux.Unlock()

					response := <-disconnect

					return response
				},
				retry.Attempts(0),
				retry.DelayType(func(n uint, err error, config *retry.Config) time.Duration {
					s.log.WithError(err).Debug("peer failed")

					return s.config.RetryInterval
				}),
			)
		}(nodeRecord, s.peers)
	}

	if err := s.startCrons(ctx); err != nil {
		return err
	}

	return nil
}

func (s *Static) Stop(ctx context.Context) error {
	return nil
}

func (s *Static) startCrons(ctx context.Context) error {
	c := gocron.NewScheduler(time.Local)

	if _, err := c.Every("5s").Do(func() {
		s.peersMux.Lock()
		connectedPeers := 0
		for _, connected := range *s.peers {
			if connected {
				connectedPeers++
			}
		}
		s.metrics.SetPeers(connectedPeers, "connected")
		s.metrics.SetPeers(len(*s.peers)-connectedPeers, "disconnected")
		s.peersMux.Unlock()
	}); err != nil {
		return err
	}

	c.StartAsync()

	return nil
}
