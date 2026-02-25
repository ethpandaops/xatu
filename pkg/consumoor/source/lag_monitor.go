package source

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/twmb/franz-go/pkg/kadm"
	"github.com/twmb/franz-go/pkg/kgo"

	"github.com/ethpandaops/xatu/pkg/consumoor/telemetry"
	"github.com/sirupsen/logrus"
)

// LagMonitor periodically polls Kafka to compute consumer group lag and
// updates a Prometheus gauge. It reuses the same broker/SASL/TLS config
// as the main consumer.
type LagMonitor struct {
	log     logrus.FieldLogger
	metrics *telemetry.Metrics

	interval       time.Duration
	consumerGroups []string

	admClient *kadm.Client
	kgoClient *kgo.Client
	done      chan struct{}
	exited    chan struct{}
}

// NewLagMonitor creates a new LagMonitor. Call Start to begin polling.
// The consumerGroups slice contains the consumer group names to monitor
// for lag (typically the single base consumer group shared by all streams).
func NewLagMonitor(
	log logrus.FieldLogger,
	cfg *KafkaConfig,
	consumerGroups []string,
	metrics *telemetry.Metrics,
) (*LagMonitor, error) {
	if cfg == nil {
		return nil, fmt.Errorf("nil kafka config")
	}

	if len(consumerGroups) == 0 {
		return nil, fmt.Errorf("no consumer groups to monitor")
	}

	opts := []kgo.Opt{
		kgo.SeedBrokers(cfg.Brokers...),
	}

	if cfg.TLS.Enabled {
		tlsCfg, err := cfg.TLS.Build()
		if err != nil {
			return nil, fmt.Errorf("building TLS config for lag monitor: %w", err)
		}

		opts = append(opts, kgo.DialTLSConfig(tlsCfg))
	}

	if cfg.SASLConfig != nil {
		mechanism, err := franzSASLMechanism(cfg.SASLConfig)
		if err != nil {
			return nil, fmt.Errorf("creating sasl mechanism for lag monitor: %w", err)
		}

		opts = append(opts, kgo.SASL(mechanism))
	}

	kgoClient, err := kgo.NewClient(opts...)
	if err != nil {
		return nil, fmt.Errorf("creating kafka client for lag monitor: %w", err)
	}

	return &LagMonitor{
		log:            log.WithField("component", "lag_monitor"),
		metrics:        metrics,
		interval:       cfg.LagPollInterval,
		consumerGroups: consumerGroups,
		admClient:      kadm.NewClient(kgoClient),
		kgoClient:      kgoClient,
		done:           make(chan struct{}),
		exited:         make(chan struct{}),
	}, nil
}

// Start begins the periodic lag polling loop. It blocks until Stop is
// called or the context is cancelled.
func (m *LagMonitor) Start(ctx context.Context) error {
	defer close(m.exited)

	ticker := time.NewTicker(m.interval)
	defer ticker.Stop()

	// Poll immediately on startup.
	m.poll(ctx)

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-m.done:
			return nil
		case <-ticker.C:
			m.poll(ctx)
		}
	}
}

// Stop signals the lag monitor to exit and waits for it to finish.
func (m *LagMonitor) Stop() error {
	close(m.done)
	<-m.exited

	m.kgoClient.Close()

	return nil
}

// poll uses kadm.Client.Lag to fetch and publish consumer group lag
// for all monitored consumer groups.
func (m *LagMonitor) poll(ctx context.Context) {
	lags, err := m.admClient.Lag(ctx, m.consumerGroups...)
	if err != nil {
		m.log.WithError(err).Warn("Failed to fetch consumer group lag")

		return
	}

	for _, group := range m.consumerGroups {
		groupLag, ok := lags[group]
		if !ok {
			m.log.WithField("consumer_group", group).
				Debug("Consumer group not found in lag response")

			continue
		}

		if groupLag.Error() != nil {
			m.log.WithError(groupLag.Error()).
				WithField("consumer_group", group).
				Warn("Error in consumer group lag response")

			continue
		}

		for topic, partitions := range groupLag.Lag {
			for partition := range partitions {
				ml := partitions[partition]

				if ml.Err != nil {
					m.log.WithError(ml.Err).
						WithField("topic", topic).
						WithField("partition", partition).
						WithField("consumer_group", group).
						Warn("Error computing lag for partition")

					continue
				}

				lag := ml.Lag
				if lag < 0 {
					lag = 0
				}

				m.metrics.KafkaConsumerLag().WithLabelValues(
					topic,
					strconv.FormatInt(int64(partition), 10),
					group,
				).Set(float64(lag))
			}
		}
	}
}
