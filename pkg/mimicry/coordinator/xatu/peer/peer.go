package xatu

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/avast/retry-go/v4"
	"github.com/ethpandaops/xatu/pkg/mimicry/coordinator/cache"
	"github.com/ethpandaops/xatu/pkg/mimicry/p2p/execution"
	"github.com/ethpandaops/xatu/pkg/mimicry/p2p/handler"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/sirupsen/logrus"
)

type Peer struct {
	log          logrus.FieldLogger
	handlers     *handler.Peer
	cache        *cache.SharedCache
	retryDelay   time.Duration
	captureDelay time.Duration

	stopped bool
	mu      sync.Mutex

	Record *xatu.CoordinatedNodeRecord
}

func NewPeer(log logrus.FieldLogger, handlers *handler.Peer, sharedCache *cache.SharedCache, record string, retryDelay, captureDelay time.Duration) *Peer {
	return &Peer{
		log:          log,
		handlers:     handlers,
		cache:        sharedCache,
		retryDelay:   retryDelay,
		captureDelay: captureDelay,
		stopped:      false,
		Record: &xatu.CoordinatedNodeRecord{
			NodeRecord:         record,
			Connected:          false,
			ConnectionAttempts: 0,
		},
	}
}

func (p *Peer) Start(ctx context.Context) error {
	go func() {
		p.Record.ConnectionAttempts++

		_ = retry.Do(
			func() error {
				p.mu.Lock()

				// return unrecoverable error if peer has been stopped
				if p.stopped {
					p.log.Debug("peer stopped")
					p.mu.Unlock()

					return retry.Unrecoverable(errors.New("peer stopped"))
				}

				p.mu.Unlock()

				peer, err := execution.New(ctx, p.log, p.Record.NodeRecord, p.handlers, p.captureDelay, p.cache)
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
			retry.Attempts(0),
			retry.DelayType(func(n uint, err error, config *retry.Config) time.Duration {
				p.mu.Lock()
				defer p.mu.Unlock()

				p.log.WithError(err).Debug("peer failed")

				if !p.stopped {
					p.Record.ConnectionAttempts++
				} else {
					return 0
				}

				return p.retryDelay
			}),
		)
	}()

	return nil
}

func (p *Peer) Stop() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.stopped = true

	return nil
}

func (p *Peer) RetryDelay(delay time.Duration) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.retryDelay = delay
}
