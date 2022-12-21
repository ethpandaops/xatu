package p2p

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/avast/retry-go/v4"
	"github.com/chuckpreslar/emission"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/sirupsen/logrus"
)

const (
	topicStatus = "status"
)

type Status struct {
	config *Config

	log logrus.FieldLogger

	broker *emission.Emitter

	active int

	mu sync.Mutex
}

func NewStatus(ctx context.Context, config *Config, log logrus.FieldLogger) *Status {
	return &Status{
		log:    log.WithField("module", "discovery/p2p/discV5"),
		config: config,
		broker: emission.NewEmitter(),
	}
}

func (s *Status) Start(ctx context.Context) error {
	return nil
}

func (s *Status) Stop(ctx context.Context) error {
	return nil
}

func (s *Status) Active() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.active
}

func (s *Status) AddNodeRecords(ctx context.Context, nodeRecords []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.active += len(nodeRecords)

	for _, nodeRecord := range nodeRecords {
		go func(record string) {
			_ = retry.Do(
				func() error {
					peer, err := NewPeer(ctx, s.log, record, s.publishStatus)
					if err != nil {
						return err
					}

					defer func() {
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

					timeout := time.After(15 * time.Second)

					var response error

					select {
					case response = <-disconnect:
					case <-timeout:
						response = errors.New("timeout")
					}

					return response
				},
				retry.Attempts(5),
				retry.DelayType(func(n uint, err error, config *retry.Config) time.Duration {
					s.log.WithError(err).Debug("peer failed")
					return 5 * time.Second
				}),
			)

			s.mu.Lock()
			defer s.mu.Unlock()

			s.active--
		}(nodeRecord)
	}
}

func (s *Status) publishStatus(ctx context.Context, status *xatu.ExecutionNodeStatus) {
	s.broker.Emit(topicStatus, status)
}

func (s *Status) handleSubscriberError(err error, topic string) {
	if err != nil {
		s.log.WithError(err).WithField("topic", topic).Error("Subscriber error")
	}
}

func (s *Status) OnStatus(ctx context.Context, handler func(ctx context.Context, status *xatu.ExecutionNodeStatus) error) {
	s.broker.On(topicStatus, func(status *xatu.ExecutionNodeStatus) {
		s.handleSubscriberError(handler(ctx, status), topicStatus)
	})
}
