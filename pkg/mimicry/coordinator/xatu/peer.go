package xatu

import (
	"context"
	"time"

	"github.com/avast/retry-go/v4"
	"github.com/ethpandaops/xatu/pkg/mimicry/coordinator/cache"
	"github.com/ethpandaops/xatu/pkg/mimicry/p2p/execution"
	"github.com/ethpandaops/xatu/pkg/mimicry/p2p/handler"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/sirupsen/logrus"
)

type Peer struct {
	log        logrus.FieldLogger
	handlers   *handler.Peer
	cache      *cache.SharedCache
	retryDelay time.Duration

	stopped bool

	Record *xatu.CoordinatedNodeRecord
}

func NewPeer(log logrus.FieldLogger, handlers *handler.Peer, sharedCache *cache.SharedCache, record string, retryDelay time.Duration) *Peer {
	return &Peer{
		log:        log,
		handlers:   handlers,
		cache:      sharedCache,
		retryDelay: retryDelay,
		stopped:    false,
		Record: &xatu.CoordinatedNodeRecord{
			NodeRecord:         record,
			Connected:          false,
			ConnectionAttempts: 0,
		},
	}
}

func (p *Peer) Start(ctx context.Context) error {
	// TODO: handle ExecutionStatus when connected
	go func() {
		p.Record.ConnectionAttempts++

		_ = retry.Do(
			func() error {
				peer, err := execution.New(ctx, p.log, p.Record.NodeRecord, p.handlers, p.cache)
				if err != nil {
					return err
				}

				defer func() {
					p.Record.Connected = false
					if peer != nil {
						if err = peer.Stop(ctx); err != nil {
							p.log.WithError(err).Warn("failed to stop peer")
						}
					}
				}()

				disconnect, err := peer.Start(ctx)
				if err != nil {
					return err
				}

				p.Record.Connected = true

				response := <-disconnect

				return response
			},
			retry.RetryIf(func(err error) bool {
				if !p.stopped {
					p.Record.ConnectionAttempts++
				}

				return !p.stopped
			}),
			retry.Attempts(0),
			retry.DelayType(func(n uint, err error, config *retry.Config) time.Duration {
				p.log.WithError(err).Debug("peer failed")
				return p.retryDelay
			}),
		)
	}()

	return nil
}

func (p *Peer) Stop() error {
	p.stopped = true
	return nil
}

func (p *Peer) RetryDelay(delay time.Duration) {
	p.retryDelay = delay
}
