package consumoor

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/IBM/sarama"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// KafkaConsumer wraps a Sarama consumer group and dispatches decoded
// DecoratedEvents to a handler function.
type KafkaConsumer struct {
	log     logrus.FieldLogger
	config  *KafkaConfig
	metrics *Metrics

	group   sarama.ConsumerGroup
	handler *consumerGroupHandler

	done chan struct{}
	wg   sync.WaitGroup
}

// MessageHandler is called for each successfully decoded DecoratedEvent.
type MessageHandler func(event *xatu.DecoratedEvent)

// NewKafkaConsumer creates a Kafka consumer group. Call Start() to begin
// consuming messages.
func NewKafkaConsumer(
	log logrus.FieldLogger,
	config *KafkaConfig,
	metrics *Metrics,
	handler MessageHandler,
	writer Writer,
) (*KafkaConsumer, error) {
	saramaConfig, err := buildSaramaConfig(config)
	if err != nil {
		return nil, fmt.Errorf("building sarama config: %w", err)
	}

	group, err := sarama.NewConsumerGroup(config.Brokers, config.ConsumerGroup, saramaConfig)
	if err != nil {
		return nil, fmt.Errorf("creating consumer group: %w", err)
	}

	return &KafkaConsumer{
		log:     log.WithField("component", "kafka_consumer"),
		config:  config,
		metrics: metrics,
		group:   group,
		handler: &consumerGroupHandler{
			log:            log.WithField("component", "kafka_handler"),
			encoding:       config.Encoding,
			metrics:        metrics,
			handler:        handler,
			writer:         writer,
			commitInterval: config.CommitInterval,
		},
		done: make(chan struct{}),
	}, nil
}

// Start begins consuming messages in a background goroutine.
// The consumer will rejoin the group and rebalance automatically.
func (c *KafkaConsumer) Start(ctx context.Context) error {
	topics, err := c.resolveTopics()
	if err != nil {
		return fmt.Errorf("resolving topics: %w", err)
	}

	if len(topics) == 0 {
		return fmt.Errorf("no topics matched patterns %v", c.config.Topics)
	}

	c.log.WithField("topics", topics).Info("Starting Kafka consumer")

	c.wg.Go(func() {
		for {
			select {
			case <-c.done:
				return
			default:
			}

			if err := c.group.Consume(ctx, topics, c.handler); err != nil {
				c.log.WithError(err).Error("Consumer group error")

				select {
				case <-c.done:
					return
				case <-time.After(5 * time.Second):
					// Retry after backoff
				}
			}
		}
	})

	// Log consumer group errors
	c.wg.Go(func() {
		for {
			select {
			case <-c.done:
				return
			case err, ok := <-c.group.Errors():
				if !ok {
					return
				}

				c.log.WithError(err).Error("Consumer group error")
			}
		}
	})

	return nil
}

// Stop gracefully shuts down the Kafka consumer.
func (c *KafkaConsumer) Stop() error {
	close(c.done)

	if err := c.group.Close(); err != nil {
		return fmt.Errorf("closing consumer group: %w", err)
	}

	c.wg.Wait()

	return nil
}

// resolveTopics expands regex topic patterns against the broker's
// actual topic list.
func (c *KafkaConsumer) resolveTopics() ([]string, error) {
	saramaConfig, err := buildSaramaConfig(c.config)
	if err != nil {
		return nil, err
	}

	client, err := sarama.NewClient(c.config.Brokers, saramaConfig)
	if err != nil {
		return nil, fmt.Errorf("creating client for topic resolution: %w", err)
	}

	defer client.Close()

	allTopics, err := client.Topics()
	if err != nil {
		return nil, fmt.Errorf("listing topics: %w", err)
	}

	matched := make(map[string]struct{}, len(allTopics))

	for _, pattern := range c.config.Topics {
		re, reErr := regexp.Compile(pattern)
		if reErr != nil {
			return nil, fmt.Errorf("compiling topic regex %q: %w", pattern, reErr)
		}

		for _, topic := range allTopics {
			if re.MatchString(topic) {
				matched[topic] = struct{}{}
			}
		}
	}

	topics := make([]string, 0, len(matched))
	for t := range matched {
		topics = append(topics, t)
	}

	return topics, nil
}

// consumerGroupHandler implements sarama.ConsumerGroupHandler.
type consumerGroupHandler struct {
	log      logrus.FieldLogger
	encoding string
	metrics  *Metrics
	handler  MessageHandler
	writer   Writer

	commitInterval time.Duration

	coordinator *commitCoordinator
}

// Setup is called at the start of a new consumer group session. It creates
// the commit coordinator that owns offset tracking for this session.
func (h *consumerGroupHandler) Setup(session sarama.ConsumerGroupSession) error {
	h.coordinator = newCommitCoordinator(
		h.log,
		h.writer,
		h.metrics,
		session,
		h.commitInterval,
	)
	h.coordinator.Start()

	return nil
}

// Cleanup is called at the end of a consumer group session. It triggers
// a final flush+commit before the partition is revoked.
func (h *consumerGroupHandler) Cleanup(_ sarama.ConsumerGroupSession) error {
	if h.coordinator != nil {
		h.coordinator.Stop()
		h.coordinator = nil
	}

	return nil
}

// ConsumeClaim processes messages from a single partition claim. It decodes
// each message, routes it through the handler (which writes to the writer's
// buffer), and tracks the offset for deferred commit by the coordinator.
func (h *consumerGroupHandler) ConsumeClaim(
	session sarama.ConsumerGroupSession,
	claim sarama.ConsumerGroupClaim,
) error {
	for msg := range claim.Messages() {
		h.metrics.messagesConsumed.WithLabelValues(msg.Topic).Inc()

		event, err := h.decode(msg.Value)
		if err != nil {
			h.log.WithError(err).
				WithField("topic", msg.Topic).
				WithField("partition", msg.Partition).
				WithField("offset", msg.Offset).
				Warn("Failed to decode message")

			h.metrics.decodeErrors.WithLabelValues(msg.Topic).Inc()

			// Track even failed decodes so offset advances past them
			h.coordinator.Track(msg)

			continue
		}

		h.handler(event)
		h.coordinator.Track(msg)
	}

	return nil
}

// decode deserializes a Kafka message value into a DecoratedEvent.
func (h *consumerGroupHandler) decode(data []byte) (*xatu.DecoratedEvent, error) {
	event := &xatu.DecoratedEvent{}

	switch h.encoding {
	case "protobuf":
		if err := proto.Unmarshal(data, event); err != nil {
			return nil, fmt.Errorf("protobuf unmarshal: %w", err)
		}
	default:
		if err := protojson.Unmarshal(data, event); err != nil {
			return nil, fmt.Errorf("json unmarshal: %w", err)
		}
	}

	return event, nil
}

// trackedMsg holds the Kafka coordinates for a message whose rows have been
// written to the writer's buffer but not yet flushed to ClickHouse.
type trackedMsg struct {
	topic     string
	partition int32
	offset    int64
}

// commitCoordinator manages the flush+commit cycle. A single goroutine
// owns all offset tracking so that FlushAll + MarkOffset + Commit happen
// atomically — no partition can commit unflushed data.
type commitCoordinator struct {
	log     logrus.FieldLogger
	writer  Writer
	metrics *Metrics
	session sarama.ConsumerGroupSession

	interval time.Duration

	mu      sync.Mutex
	pending []trackedMsg

	done chan struct{}
	wg   sync.WaitGroup
}

// newCommitCoordinator creates a commit coordinator for the given session.
func newCommitCoordinator(
	log logrus.FieldLogger,
	writer Writer,
	metrics *Metrics,
	session sarama.ConsumerGroupSession,
	interval time.Duration,
) *commitCoordinator {
	return &commitCoordinator{
		log:      log.WithField("component", "commit_coordinator"),
		writer:   writer,
		metrics:  metrics,
		session:  session,
		interval: interval,
		pending:  make([]trackedMsg, 0, 1024),
		done:     make(chan struct{}),
	}
}

// Start launches the commit loop goroutine.
func (cc *commitCoordinator) Start() {
	cc.wg.Go(func() {
		cc.run()
	})
}

// Track records a message whose rows are buffered but not yet flushed.
// Called by each ConsumeClaim goroutine after Write().
func (cc *commitCoordinator) Track(msg *sarama.ConsumerMessage) {
	cc.mu.Lock()
	cc.pending = append(cc.pending, trackedMsg{
		topic:     msg.Topic,
		partition: msg.Partition,
		offset:    msg.Offset,
	})
	cc.mu.Unlock()
}

// run is the commit loop. It exits when done is closed.
func (cc *commitCoordinator) run() {
	var ticker *time.Ticker

	var tickC <-chan time.Time

	if cc.interval > 0 {
		ticker = time.NewTicker(cc.interval)
		tickC = ticker.C

		defer ticker.Stop()
	}

	for {
		select {
		case <-tickC:
			cc.commit()
		case <-cc.done:
			cc.commit() // final flush

			return
		}
	}
}

// commit performs one flush+commit cycle. It snapshots pending messages,
// calls FlushAll on the writer, and if successful, marks offsets and
// commits them. On failure, the snapshot is prepended back to pending.
func (cc *commitCoordinator) commit() {
	cc.mu.Lock()

	if len(cc.pending) == 0 {
		cc.mu.Unlock()

		return
	}

	snapshot := cc.pending
	cc.pending = make([]trackedMsg, 0, cap(snapshot))

	cc.mu.Unlock()

	start := time.Now()

	if err := cc.writer.FlushAll(context.Background()); err != nil {
		cc.log.WithError(err).Error("FlushAll failed, not committing offsets")
		cc.metrics.commitErrors.WithLabelValues("flush_failed").Inc()

		// Put messages back — their data is preserved in table writer batches
		cc.mu.Lock()
		cc.pending = append(snapshot, cc.pending...)
		cc.mu.Unlock()

		return
	}

	cc.metrics.flushAllDuration.Observe(time.Since(start).Seconds())

	// Mark the highest offset per topic/partition
	highwater := make(map[string]map[int32]int64, 4)

	for _, msg := range snapshot {
		parts, ok := highwater[msg.topic]
		if !ok {
			parts = make(map[int32]int64, 4)
			highwater[msg.topic] = parts
		}

		if msg.offset > parts[msg.partition] {
			parts[msg.partition] = msg.offset
		}
	}

	for topic, parts := range highwater {
		for partition, offset := range parts {
			// MarkOffset expects the NEXT offset to read
			cc.session.MarkOffset(topic, partition, offset+1, "")
		}
	}

	cc.session.Commit()

	cc.metrics.commitsTotal.Inc()
	cc.log.WithField("messages", len(snapshot)).Debug("Committed offsets")
}

// Stop signals the commit loop to do a final flush+commit and exit.
func (cc *commitCoordinator) Stop() {
	close(cc.done)
	cc.wg.Wait()
}

// buildSaramaConfig creates a Sarama configuration from our KafkaConfig.
func buildSaramaConfig(config *KafkaConfig) (*sarama.Config, error) {
	c := sarama.NewConfig()

	c.Consumer.Fetch.Min = config.FetchMinBytes
	c.Consumer.MaxWaitTime = time.Duration(config.FetchWaitMaxMs) * time.Millisecond
	c.Consumer.Fetch.Default = config.MaxPartitionFetchBytes
	c.Consumer.Group.Session.Timeout = time.Duration(config.SessionTimeoutMs) * time.Millisecond
	c.Consumer.Group.Heartbeat.Interval = time.Duration(config.HeartbeatIntervalMs) * time.Millisecond
	c.Consumer.Return.Errors = true
	c.Net.TLS.Enable = config.TLS
	c.Metadata.Full = false

	// Disable auto-commit — the commit coordinator owns offset commits.
	c.Consumer.Offsets.AutoCommit.Enable = false

	switch config.OffsetDefault {
	case "newest":
		c.Consumer.Offsets.Initial = sarama.OffsetNewest
	default:
		c.Consumer.Offsets.Initial = sarama.OffsetOldest
	}

	if config.Version != "" {
		version, err := sarama.ParseKafkaVersion(config.Version)
		if err != nil {
			return nil, fmt.Errorf("parsing kafka version: %w", err)
		}

		c.Version = version
	}

	if config.SASLConfig != nil {
		var password string

		if config.SASLConfig.Password != "" {
			password = config.SASLConfig.Password
		} else if config.SASLConfig.PasswordFile != "" {
			data, err := os.ReadFile(config.SASLConfig.PasswordFile)
			if err != nil {
				return nil, fmt.Errorf("reading SASL password file: %w", err)
			}

			password = strings.TrimSpace(string(data))
		}

		c.Net.SASL.Enable = true
		c.Net.SASL.User = config.SASLConfig.User
		c.Net.SASL.Password = password

		switch config.SASLConfig.Mechanism {
		case "SCRAM-SHA-256":
			c.Net.SASL.Mechanism = sarama.SASLTypeSCRAMSHA256
		case "SCRAM-SHA-512":
			c.Net.SASL.Mechanism = sarama.SASLTypeSCRAMSHA512
		case "OAUTHBEARER":
			c.Net.SASL.Mechanism = sarama.SASLTypeOAuth
		default:
			c.Net.SASL.Mechanism = sarama.SASLTypePlaintext
		}
	}

	return c, nil
}
