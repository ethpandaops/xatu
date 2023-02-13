package static

import (
	"context"
	"errors"
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

	log logrus.FieldLogger

	cache *cache.SharedCache
	peers *map[string]bool

	metrics *Metrics
}

func New(name string, config *Config, handlers *handler.Peer, log logrus.FieldLogger) (*Static, error) {
	if config == nil {
		return nil, errors.New("config is required")
	}

	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &Static{
		config:   config,
		handlers: handlers,
		log:      log,
		cache:    cache.NewSharedCache(),
		peers:    &map[string]bool{},
		metrics:  NewMetrics("xatu_mimicry_coordinator_static"),
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
					peer, err := execution.New(ctx, s.log, record, s.handlers, s.cache)
					if err != nil {
						return err
					}

					defer func() {
						(*peers)[record] = false
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

					(*peers)[record] = true

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
		connectedPeers := 0
		for _, connected := range *s.peers {
			if connected {
				connectedPeers++
			}
		}
		s.metrics.SetPeers(connectedPeers, "connected")
		s.metrics.SetPeers(len(*s.peers)-connectedPeers, "disconnected")
	}); err != nil {
		return err
	}

	c.StartAsync()

	return nil
}
