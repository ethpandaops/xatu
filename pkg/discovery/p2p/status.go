package p2p

import (
	"context"
	"errors"
	"fmt"
	"net"
	"slices"
	"sync"
	"time"

	"github.com/avast/retry-go/v4"
	"github.com/chuckpreslar/emission"
	"github.com/ethpandaops/ethcore/pkg/consensus/mimicry/crawler"
	"github.com/ethpandaops/ethcore/pkg/consensus/mimicry/host"
	"github.com/ethpandaops/xatu/pkg/discovery/p2p/static"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/sirupsen/logrus"
)

const (
	topicExecutionStatus = "execution_status"
	topicConsensusStatus = "consensus_status"
)

type Status struct {
	config           *Config
	executionConfig  *static.ExecutionConfig
	consensusConfig  *static.ConsensusConfig
	log              logrus.FieldLogger
	broker           *emission.Emitter
	activeExecution  int
	activeConsensus  int
	mu               sync.Mutex
	metrics          *Metrics
	consensusCrawler *ConsensusCrawler
	activeWorkers    sync.WaitGroup
	ctx              context.Context //nolint:containedctx // requires much larger refactor into channels.
	cancel           context.CancelFunc
}

func NewStatus(ctx context.Context, config *Config, log logrus.FieldLogger) (*Status, error) {
	consensusConfig := config.GetConsensusConfig()

	s := &Status{
		log:             log.WithField("module", "discovery/p2p"),
		config:          config,
		executionConfig: config.GetExecutionConfig(),
		consensusConfig: consensusConfig,
		broker:          emission.NewEmitter(),
		metrics:         NewMetrics("xatu_discovery"),
	}

	if config.Ethereum != nil {
		cfg := &crawler.Config{
			Node: &host.Config{
				IPAddr: net.ParseIP("127.0.0.1"),
			},
			Beacon:           config.Ethereum,
			DialConcurrency:  consensusConfig.DialConcurrency,
			DialTimeout:      consensusConfig.DialTimeout,
			CooloffDuration:  consensusConfig.CooloffDuration,
			UserAgent:        xatu.Full(),
			MaxRetryAttempts: consensusConfig.RetryAttempts,
			RetryBackoff:     consensusConfig.RetryDelay,
		}

		c, err := NewConsensusCrawler(ctx, log, cfg)
		if err != nil {
			return nil, err
		}

		s.consensusCrawler = c
	}

	return s, nil
}

func (s *Status) Start(ctx context.Context) error {
	s.ctx, s.cancel = context.WithCancel(ctx)
	return nil
}

func (s *Status) Stop(ctx context.Context) error {
	// Cancel the context to signal all workers to stop
	if s.cancel != nil {
		s.cancel()
	}

	// Wait for active workers with timeout
	done := make(chan struct{})
	go func() {
		s.activeWorkers.Wait()
		close(done)
	}()

	select {
	case <-done:
		s.log.Info("All active workers finished")
	case <-time.After(5 * time.Second):
		s.log.Warn("Timeout waiting for active workers to finish")
	}

	if s.consensusCrawler != nil {
		return s.consensusCrawler.Stop(ctx)
	}

	return nil
}

func (s *Status) ActiveExecution() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.activeExecution
}

func (s *Status) ActiveConsensus() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.activeConsensus
}

func (s *Status) AddExecutionNodeRecords(ctx context.Context, nodeRecords []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.activeExecution += len(nodeRecords)
	s.metrics.SetActiveDialingNodeRecods(s.activeExecution, "execution")

	for _, nodeRecord := range nodeRecords {
		s.activeWorkers.Add(1)

		go func(record string) {
			defer s.activeWorkers.Done()

			// Check if context is already cancelled
			if s.ctx.Err() != nil {
				return
			}

			_ = retry.Do(
				func() error {
					// Check for cancellation before each attempt
					select {
					case <-s.ctx.Done():
						return retry.Unrecoverable(s.ctx.Err())
					default:
					}

					peer, err := NewExecutionPeer(s.ctx, s.log, record, s.publishExecutionStatus)
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

							s.metrics.AddDialedNodeRecod(1, status, "execution")

							if err = peer.Stop(s.ctx); err != nil {
								s.log.WithError(err).Warn("failed to stop peer")
							}
						}
					}()

					disconnect, err := peer.Start(s.ctx)
					if err != nil {
						return err
					}

					// Use context-aware timeout
					timer := time.NewTimer(s.executionConfig.DialTimeout)
					defer timer.Stop()

					select {
					case response = <-disconnect:
					case <-timer.C:
						response = errors.New("timeout")
					case <-s.ctx.Done():
						response = s.ctx.Err()
					}

					if response == nil {
						connected = true
					}

					return response
				},
				retry.Attempts(s.executionConfig.RetryAttempts),
				retry.DelayType(func(n uint, err error, config *retry.Config) time.Duration {
					// Check for cancellation during delay
					select {
					case <-s.ctx.Done():
						return 0
					default:
						s.log.WithError(err).Debug("peer failed")

						return s.executionConfig.RetryDelay
					}
				}),
				retry.RetryIf(func(err error) bool {
					// Don't retry if context is cancelled
					select {
					case <-s.ctx.Done():
						return false
					default:
						return true
					}
				}),
			)

			s.mu.Lock()
			defer s.mu.Unlock()

			s.activeExecution--
			s.metrics.SetActiveDialingNodeRecods(s.activeExecution, "execution")
		}(nodeRecord)
	}
}

func (s *Status) AddConsensusNodeRecords(_ context.Context, nodeRecords []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.activeConsensus += len(nodeRecords)
	s.metrics.SetActiveDialingNodeRecods(s.activeConsensus, "consensus")

	for _, nodeRecord := range nodeRecords {
		s.activeWorkers.Add(1)

		go func(record string) {
			defer s.activeWorkers.Done()

			// Check if context is already cancelled
			if s.ctx.Err() != nil {
				return
			}

			// Check for cancellation before proceeding
			select {
			case <-s.ctx.Done():
				return
			default:
			}

			if s.consensusCrawler == nil {
				s.log.Error("consensus crawler not initialized")

				s.mu.Lock()

				s.activeConsensus--
				s.metrics.SetActiveDialingNodeRecods(s.activeConsensus, "consensus")

				s.mu.Unlock()

				return
			}

			handler := func(status *xatu.ConsensusNodeStatus) {
				s.publishConsensusStatus(s.ctx, status)
			}

			peer, err := NewConsensusPeer(s.ctx, s.log, record, handler, s.consensusCrawler)
			if err != nil {
				s.log.WithError(err).Debug("failed to create consensus peer")

				s.mu.Lock()

				s.activeConsensus--
				s.metrics.SetActiveDialingNodeRecods(s.activeConsensus, "consensus")

				s.mu.Unlock()

				return
			}

			var (
				response  error
				connected = false
			)

			defer func() {
				if peer != nil {
					status := "failed"

					if connected {
						status = "success"
					}

					s.metrics.AddDialedNodeRecod(1, status, "consensus")

					if err = peer.Stop(s.ctx); err != nil {
						s.log.WithError(err).Warn("failed to stop peer")
					}
				}
			}()

			disconnect := peer.Start(s.ctx)

			// Use context-aware timeout
			timer := time.NewTimer(s.consensusConfig.ConnectionTimeout)
			defer timer.Stop()

			select {
			case response = <-disconnect:
			case <-timer.C:
				response = errors.New("timeout")
			case <-s.ctx.Done():
				response = s.ctx.Err()
			}

			if response == nil {
				connected = true
			}

			s.mu.Lock()
			defer s.mu.Unlock()

			s.activeConsensus--
			s.metrics.SetActiveDialingNodeRecods(s.activeConsensus, "consensus")
		}(nodeRecord)
	}
}

func (s *Status) publishExecutionStatus(_ context.Context, status *xatu.ExecutionNodeStatus) {
	s.broker.Emit(topicExecutionStatus, status)
}

func (s *Status) publishConsensusStatus(_ context.Context, status *xatu.ConsensusNodeStatus) {
	s.broker.Emit(topicConsensusStatus, status)
}

func (s *Status) handleSubscriberError(err error, topic string) {
	if err != nil {
		s.log.WithError(err).WithField("topic", topic).Error("Subscriber error")
	}
}

func (s *Status) OnExecutionStatus(ctx context.Context, handler func(ctx context.Context, status *xatu.ExecutionNodeStatus) error, networkIds []uint64, forkIdHashes []string) {
	s.broker.On(topicExecutionStatus, func(status *xatu.ExecutionNodeStatus) {
		// exclude execution status for network ids that are not in the list
		if len(networkIds) > 0 && !slices.Contains(networkIds, status.NetworkId) {
			s.log.WithField("network_id", status.NetworkId).Warn("skipping execution status for network id")

			return
		}

		// exclude execution status for fork id hashes that are not in the list
		if len(forkIdHashes) > 0 && !slices.Contains(forkIdHashes, fmt.Sprintf("0x%x", status.ForkId.Hash)) {
			s.log.WithField("fork_id_hash", fmt.Sprintf("0x%x", status.ForkId.Hash)).Warn("skipping execution status for fork id hash")

			return
		}

		s.handleSubscriberError(handler(ctx, status), topicExecutionStatus)
	})
}

func (s *Status) OnConsensusStatus(ctx context.Context, handler func(ctx context.Context, status *xatu.ConsensusNodeStatus) error) {
	s.broker.On(topicConsensusStatus, func(status *xatu.ConsensusNodeStatus) {
		s.handleSubscriberError(handler(ctx, status), topicConsensusStatus)
	})
}
