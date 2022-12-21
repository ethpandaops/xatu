package manual

import (
	"context"
	"errors"
	"time"

	"github.com/avast/retry-go/v4"
	"github.com/ethpandaops/xatu/pkg/mimicry/coordinator/cache"
	"github.com/ethpandaops/xatu/pkg/mimicry/p2p/execution"
	"github.com/ethpandaops/xatu/pkg/mimicry/p2p/handler"
	"github.com/sirupsen/logrus"
)

const Type = "manual"

type Manual struct {
	config *Config

	handlers *handler.Peer

	log logrus.FieldLogger

	cache *cache.SharedCache
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
	}, nil
}

func (m *Manual) Type() string {
	return Type
}

func (m *Manual) Start(ctx context.Context) error {
	for _, nodeRecord := range m.config.NodeRecords {
		go func(record string) {
			_ = retry.Do(
				func() error {
					peer, err := execution.New(ctx, m.log, record, m.handlers, m.cache)
					if err != nil {
						return err
					}

					defer func() {
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

					response := <-disconnect

					return response
				},
				retry.Attempts(0),
				retry.DelayType(func(n uint, err error, config *retry.Config) time.Duration {
					m.log.WithError(err).Debug("peer failed")
					return m.config.RetryInterval
				}),
			)
		}(nodeRecord)
	}

	return nil
}

func (m *Manual) Stop(ctx context.Context) error {
	return nil
}
