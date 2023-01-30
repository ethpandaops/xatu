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
	topicExecutionStatus = "execution_status"
)

type Status struct {
	config *Config

	log logrus.FieldLogger

	broker *emission.Emitter

	activeExecution int

	mu sync.Mutex

	metrics *Metrics
}

func NewStatus(ctx context.Context, config *Config, log logrus.FieldLogger) *Status {
	return &Status{
		log:     log.WithField("module", "discovery/p2p"),
		config:  config,
		broker:  emission.NewEmitter(),
		metrics: NewMetrics("xatu_discovery"),
	}
}

func (s *Status) Start(ctx context.Context) error {
	return nil
}

func (s *Status) Stop(ctx context.Context) error {
	return nil
}

func (s *Status) ActiveExecution() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.activeExecution
}

func (s *Status) AddExecutionNodeRecords(ctx context.Context, nodeRecords []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.activeExecution += len(nodeRecords)
	s.metrics.SetActiveDialingNodeRecods(s.activeExecution, "execution")

	for _, nodeRecord := range nodeRecords {
		go func(record string) {
			_ = retry.Do(
				func() error {
					peer, err := NewPeer(ctx, s.log, record, s.publishExecutionStatus)
					if err != nil {
						return err
					}

					var response error

					connected := false

					defer func() {
						if peer != nil {
							status := "failed"

							if connected {
								status = "success"
							}

							s.metrics.AddDialedNodeRecod(1, status)

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

					select {
					case response = <-disconnect:
					case <-timeout:
						response = errors.New("timeout")
					}

					if response == nil {
						connected = true
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

			s.activeExecution--
			s.metrics.SetActiveDialingNodeRecods(s.activeExecution, "execution")
		}(nodeRecord)
	}
}

func (s *Status) publishExecutionStatus(ctx context.Context, status *xatu.ExecutionNodeStatus) {
	s.broker.Emit(topicExecutionStatus, status)
}

func (s *Status) handleSubscriberError(err error, topic string) {
	if err != nil {
		s.log.WithError(err).WithField("topic", topic).Error("Subscriber error")
	}
}

func (s *Status) OnExecutionStatus(ctx context.Context, handler func(ctx context.Context, status *xatu.ExecutionNodeStatus) error) {
	s.broker.On(topicExecutionStatus, func(status *xatu.ExecutionNodeStatus) {
		s.handleSubscriberError(handler(ctx, status), topicExecutionStatus)
	})
}
