package manual

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

const Type = "manual"

type Manual struct {
	config *Config

	handlers *handler.Peer

	log logrus.FieldLogger

	cache *cache.SharedCache
	peers *map[string]bool

	metrics *Metrics
}

func New(name string, config *Config, handlers *handler.Peer, log logrus.FieldLogger) (*Manual, error) {
	if config == nil {
		return nil, errors.New("config is required")
	}

	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &Manual{
		config:   config,
		handlers: handlers,
		log:      log,
		cache:    cache.NewSharedCache(),
		peers:    &map[string]bool{},
		metrics:  NewMetrics("xatu_mimicry_coordinator_manual"),
	}, nil
}

func (m *Manual) Type() string {
	return Type
}

func (m *Manual) Start(ctx context.Context) error {
	for _, nodeRecord := range m.config.NodeRecords {
		(*m.peers)[nodeRecord] = false
		go func(record string, peers *map[string]bool) {
			_ = retry.Do(
				func() error {
					peer, err := execution.New(ctx, m.log, record, m.handlers, m.cache)
					if err != nil {
						return err
					}

					defer func() {
						(*peers)[record] = false
						if peer != nil {
							if err = peer.Stop(ctx); err != nil {
								m.log.WithError(err).Warn("failed to stop peer")
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
					m.log.WithError(err).Debug("peer failed")
					return m.config.RetryInterval
				}),
			)
		}(nodeRecord, m.peers)
	}

	if err := m.startCrons(ctx); err != nil {
		return err
	}

	return nil
}

func (m *Manual) Stop(ctx context.Context) error {
	return nil
}

func (m *Manual) startCrons(ctx context.Context) error {
	c := gocron.NewScheduler(time.Local)

	if _, err := c.Every("5s").Do(func() {
		connectedPeers := 0
		for _, connected := range *m.peers {
			if connected {
				connectedPeers++
			}
		}
		m.metrics.SetPeers(connectedPeers, "connected")
		m.metrics.SetPeers(len(*m.peers)-connectedPeers, "disconnected")
	}); err != nil {
		return err
	}

	c.StartAsync()

	return nil
}
