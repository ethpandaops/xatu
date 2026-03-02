package consumoor

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os/signal"
	"regexp"
	"strings"
	"sync"
	"syscall"
	"time"

	//nolint:gosec // only exposed if pprofAddr config is set
	_ "net/http/pprof"

	"github.com/ethpandaops/xatu/pkg/consumoor/clickhouse"
	"github.com/ethpandaops/xatu/pkg/consumoor/route/all"
	"github.com/ethpandaops/xatu/pkg/consumoor/router"
	"github.com/ethpandaops/xatu/pkg/consumoor/source"
	"github.com/ethpandaops/xatu/pkg/consumoor/telemetry"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redpanda-data/benthos/v4/public/service"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
)

// topicStream pairs a discovered Kafka topic with its dedicated Benthos stream.
type topicStream struct {
	topic  string
	stream *service.Stream
}

// Consumoor is the main service that consumes events from Kafka and
// writes them to ClickHouse. Each matched Kafka topic gets its own
// Benthos stream and consumer group while sharing a single ClickHouse
// writer for efficient connection reuse.
type Consumoor struct {
	log       logrus.FieldLogger
	parentLog logrus.FieldLogger
	config    *Config

	metrics    *telemetry.Metrics
	router     *router.Engine
	writer     source.Writer
	streams    []topicStream
	lagMonitor *source.LagMonitor

	// activeTopics tracks topics that have a running stream to avoid
	// creating duplicate streams when watchTopics re-discovers them.
	activeTopics map[string]struct{}

	mu            sync.Mutex
	metricsServer *http.Server
	pprofServer   *http.Server
}

// New creates a new Consumoor service. Call Start() to run it.
func New(
	ctx context.Context,
	log logrus.FieldLogger,
	config *Config,
	overrides *Override,
) (*Consumoor, error) {
	config.ApplyOverrides(overrides)

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	metrics := telemetry.NewMetrics("xatu")

	// Create the ClickHouse writer.
	writer, err := clickhouse.NewChGoWriter(
		log,
		&config.ClickHouse,
		metrics,
	)
	if err != nil {
		return nil, fmt.Errorf("creating clickhouse writer: %w", err)
	}

	// Create the router with all registered routes.
	registeredRoutes, err := all.All()
	if err != nil {
		return nil, fmt.Errorf("route registration: %w", err)
	}

	disabledEvents, err := config.DisabledEventEnums()
	if err != nil {
		return nil, fmt.Errorf("invalid disabledEvents config: %w", err)
	}

	rtr := router.New(log, registeredRoutes, disabledEvents, metrics)

	// Register columnar batch factories from routes on the writer so
	// each table gets zero-reflection inserts.
	writer.RegisterBatchFactories(registeredRoutes)

	// Discover matching Kafka topics and create one stream per topic.
	topics, err := source.DiscoverTopics(ctx, &config.Kafka)
	if err != nil {
		return nil, fmt.Errorf("discovering kafka topics: %w", err)
	}

	if len(topics) == 0 {
		return nil, fmt.Errorf(
			"no kafka topics matched patterns %v",
			config.Kafka.Topics,
		)
	}

	cLog := log.WithField("component", "consumoor")
	cLog.WithField("topics", topics).
		WithField("count", len(topics)).
		Info("Discovered Kafka topics for per-topic streams")

	streams := make([]topicStream, 0, len(topics))
	consumerGroups := make([]string, 0, len(topics))

	for _, topic := range topics {
		topicKafkaCfg := config.Kafka.ApplyTopicOverride(topic)
		topicKafkaCfg.Topics = []string{"^" + regexp.QuoteMeta(topic) + "$"}
		topicKafkaCfg.ConsumerGroup = config.Kafka.ConsumerGroup + "-" + topic

		consumerGroups = append(consumerGroups, topicKafkaCfg.ConsumerGroup)

		if _, hasOverride := config.Kafka.TopicOverrides[topic]; hasOverride {
			cLog.WithField("topic", topic).
				WithField("outputBatchCount", topicKafkaCfg.OutputBatchCount).
				WithField("outputBatchPeriod", topicKafkaCfg.OutputBatchPeriod).
				WithField("maxInFlight", topicKafkaCfg.MaxInFlight).
				Info("Applied per-topic batch overrides")
		}

		stream, sErr := source.NewBenthosStream(
			log.WithField("topic", topic),
			config.LoggingLevel,
			&topicKafkaCfg,
			metrics,
			rtr,
			writer,
			false, // writer lifecycle owned by Consumoor, not the output plugin
			source.GroupRetryConfig{
				MaxAttempts: config.ClickHouse.ChGo.GroupRetryMaxAttempts,
				BaseDelay:   config.ClickHouse.ChGo.GroupRetryBaseDelay,
				MaxDelay:    config.ClickHouse.ChGo.GroupRetryMaxDelay,
			},
		)
		if sErr != nil {
			return nil, fmt.Errorf(
				"creating benthos stream for topic %q: %w", topic, sErr,
			)
		}

		streams = append(streams, topicStream{
			topic:  topic,
			stream: stream,
		})

		metrics.OutputMaxInFlight().WithLabelValues(topic).Set(float64(topicKafkaCfg.MaxInFlight))
	}

	activeTopics := make(map[string]struct{}, len(topics))
	for _, t := range topics {
		activeTopics[t] = struct{}{}
	}

	if strings.TrimSpace(config.Kafka.RejectedTopic) == "" {
		cLog.Warn("No rejectedTopic configured — decode errors and permanent write failures will block Kafka offset advancement (messages will be redelivered indefinitely). Configure kafka.rejectedTopic to enable dead letter queue")
	}

	if len(config.Kafka.TopicOverrides) > 0 {
		for overrideTopic := range config.Kafka.TopicOverrides {
			if _, found := activeTopics[overrideTopic]; !found {
				cLog.WithField("topic", overrideTopic).
					Warn("Topic override configured but no matching topic was discovered — check for typos")
			}
		}
	}

	c := &Consumoor{
		log:          cLog,
		parentLog:    log,
		config:       config,
		metrics:      metrics,
		router:       rtr,
		writer:       writer,
		streams:      streams,
		activeTopics: activeTopics,
	}

	// Optionally create the Kafka consumer lag monitor.
	if config.Kafka.LagPollInterval > 0 {
		lagMon, lagErr := source.NewLagMonitor(
			log,
			&config.Kafka,
			consumerGroups,
			metrics,
		)
		if lagErr != nil {
			return nil, fmt.Errorf("creating lag monitor: %w", lagErr)
		}

		c.lagMonitor = lagMon
	}

	return c, nil
}

// Start runs the consumoor service until the context is cancelled or
// a SIGINT/SIGTERM is received.
func (c *Consumoor) Start(ctx context.Context) error {
	nctx, stop := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := c.writer.Start(nctx); err != nil {
		return fmt.Errorf("starting clickhouse writer: %w", err)
	}

	c.log.WithField("streams", len(c.streams)).
		Info("Consumoor started (per-topic benthos streams)")

	g, gCtx := errgroup.WithContext(nctx)

	g.Go(func() error {
		return c.startMetrics(ctx)
	})

	if c.config.PProfAddr != nil {
		g.Go(func() error {
			return c.startPProf(ctx)
		})
	}

	for _, ts := range c.streams {
		g.Go(func() error {
			c.log.WithField("topic", ts.topic).Info("Starting stream")

			if err := ts.stream.Run(gCtx); err != nil && !errors.Is(err, context.Canceled) {
				return fmt.Errorf("running stream for topic %q: %w", ts.topic, err)
			}

			return nil
		})
	}

	if c.lagMonitor != nil {
		g.Go(func() error {
			return c.lagMonitor.Start(gCtx)
		})
	}

	// Shut down HTTP servers when the group context is cancelled so that
	// g.Wait() can return. Writer cleanup happens after g.Wait() to
	// guarantee no stream is mid-WriteBatch when the writer stops.
	g.Go(func() error {
		<-gCtx.Done()
		c.stopHTTPServers(ctx)

		return nil
	})

	if c.config.Kafka.TopicRefreshInterval > 0 {
		g.Go(func() error {
			c.watchTopics(gCtx, g)

			return nil
		})
	}

	streamErr := g.Wait()

	// All streams and HTTP servers have exited. Now stop the writer.
	c.stopWriter(ctx)

	if streamErr != nil && !errors.Is(streamErr, context.Canceled) {
		return streamErr
	}

	return nil
}

// stopHTTPServers shuts down the metrics and pprof servers. Called from
// within the errgroup on context cancellation so that g.Wait() can return.
func (c *Consumoor) stopHTTPServers(_ context.Context) {
	c.mu.Lock()
	metricsServer := c.metricsServer
	pprofServer := c.pprofServer
	c.mu.Unlock()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if metricsServer != nil {
		if err := metricsServer.Shutdown(shutdownCtx); err != nil {
			c.log.WithError(err).Error("Error stopping metrics server")
		}
	}

	if pprofServer != nil {
		if err := pprofServer.Shutdown(shutdownCtx); err != nil {
			c.log.WithError(err).Error("Error stopping pprof server")
		}
	}
}

// stopWriter drains the shared ClickHouse writer. Must be called after
// all streams have fully exited to guarantee no in-flight writes.
func (c *Consumoor) stopWriter(ctx context.Context) {
	c.log.Info("Stopping consumoor")

	if c.lagMonitor != nil {
		if err := c.lagMonitor.Stop(); err != nil {
			c.log.WithError(err).Error("Error stopping lag monitor")
		}
	}

	if err := c.writer.Stop(ctx); err != nil {
		c.log.WithError(err).Error("Error stopping clickhouse writer")
	}

	c.log.Info("Consumoor stopped")
}

// healthResponse is the JSON body returned by health check endpoints.
type healthResponse struct {
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

func (c *Consumoor) handleHealthz(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	json.NewEncoder(w).Encode(healthResponse{Status: "ok"}) //nolint:errcheck // best-effort response
}

func (c *Consumoor) handleReadyz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if err := c.writer.Ping(r.Context()); err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)

		json.NewEncoder(w).Encode(healthResponse{ //nolint:errcheck // best-effort response
			Status: "not ready",
			Error:  err.Error(),
		})

		return
	}

	w.WriteHeader(http.StatusOK)

	json.NewEncoder(w).Encode(healthResponse{Status: "ok"}) //nolint:errcheck // best-effort response
}

func (c *Consumoor) startMetrics(ctx context.Context) error {
	sm := http.NewServeMux()
	sm.Handle("/metrics", promhttp.Handler())
	sm.HandleFunc("/healthz", c.handleHealthz)
	sm.HandleFunc("/readyz", c.handleReadyz)

	c.log.WithField("addr", c.config.MetricsAddr).Info("Starting metrics server")

	srv := &http.Server{
		Addr:              c.config.MetricsAddr,
		ReadHeaderTimeout: 15 * time.Second,
		Handler:           sm,
		BaseContext: func(l net.Listener) context.Context {
			return ctx
		},
	}

	c.mu.Lock()
	c.metricsServer = srv
	c.mu.Unlock()

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}

	return nil
}

// watchTopics periodically queries Kafka metadata to discover topics matching
// the configured regex patterns. Newly discovered topics automatically get
// their own Benthos stream without requiring a restart.
func (c *Consumoor) watchTopics(ctx context.Context, g *errgroup.Group) {
	interval := c.config.Kafka.TopicRefreshInterval

	c.log.WithField("interval", interval).
		Info("Starting topic discovery watcher")

	// Perform an initial discovery so the metric is populated immediately.
	knownTopics := c.discoverAndDiff(ctx, g, nil)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			c.log.Info("Topic discovery watcher stopped")

			return
		case <-ticker.C:
			knownTopics = c.discoverAndDiff(ctx, g, knownTopics)
		}
	}
}

// discoverAndDiff calls DiscoverTopics, compares the result to the previously
// known set, logs any changes, updates the active_topics gauge, and starts
// streams for newly discovered topics.
func (c *Consumoor) discoverAndDiff(
	ctx context.Context,
	g *errgroup.Group,
	previous map[string]struct{},
) map[string]struct{} {
	topics, err := source.DiscoverTopics(ctx, &c.config.Kafka)
	if err != nil {
		c.log.WithError(err).Warn("Topic discovery failed")

		return previous
	}

	current := make(map[string]struct{}, len(topics))
	for _, t := range topics {
		current[t] = struct{}{}
	}

	c.metrics.ActiveTopics().Set(float64(len(current)))

	if previous == nil {
		c.log.WithField("topics", topics).
			WithField("count", len(topics)).
			Info("Initial topic discovery complete")

		return current
	}

	// Find newly appeared topics and start streams for them.
	for _, t := range topics {
		if _, ok := previous[t]; !ok {
			c.log.WithField("topic", t).
				Info("Discovered new topic matching pattern")
			c.startTopicStream(ctx, g, t)
		}
	}

	// Find disappeared topics.
	for t := range previous {
		if _, ok := current[t]; !ok {
			c.log.WithField("topic", t).
				Warn("Previously matched topic no longer found")
		}
	}

	return current
}

// startTopicStream creates a Benthos stream for a dynamically discovered
// topic and launches it in the errgroup. If the topic already has an active
// stream, this is a no-op.
func (c *Consumoor) startTopicStream(
	ctx context.Context,
	g *errgroup.Group,
	topic string,
) {
	c.mu.Lock()
	if _, exists := c.activeTopics[topic]; exists {
		c.mu.Unlock()

		return
	}

	c.activeTopics[topic] = struct{}{}
	c.mu.Unlock()

	topicKafkaCfg := c.config.Kafka.ApplyTopicOverride(topic)
	topicKafkaCfg.Topics = []string{"^" + regexp.QuoteMeta(topic) + "$"}
	topicKafkaCfg.ConsumerGroup = c.config.Kafka.ConsumerGroup + "-" + topic

	if _, hasOverride := c.config.Kafka.TopicOverrides[topic]; hasOverride {
		c.log.WithField("topic", topic).
			WithField("outputBatchCount", topicKafkaCfg.OutputBatchCount).
			WithField("outputBatchPeriod", topicKafkaCfg.OutputBatchPeriod).
			WithField("maxInFlight", topicKafkaCfg.MaxInFlight).
			Info("Applied per-topic batch overrides for dynamically discovered topic")
	}

	stream, err := source.NewBenthosStream(
		c.parentLog.WithField("topic", topic),
		c.config.LoggingLevel,
		&topicKafkaCfg,
		c.metrics,
		c.router,
		c.writer,
		false, // writer lifecycle owned by Consumoor
		source.GroupRetryConfig{
			MaxAttempts: c.config.ClickHouse.ChGo.GroupRetryMaxAttempts,
			BaseDelay:   c.config.ClickHouse.ChGo.GroupRetryBaseDelay,
			MaxDelay:    c.config.ClickHouse.ChGo.GroupRetryMaxDelay,
		},
	)
	if err != nil {
		c.log.WithError(err).
			WithField("topic", topic).
			Error("Failed to create stream for dynamically discovered topic")

		c.mu.Lock()
		delete(c.activeTopics, topic)
		c.mu.Unlock()

		return
	}

	c.mu.Lock()
	c.streams = append(c.streams, topicStream{topic: topic, stream: stream})
	c.mu.Unlock()

	c.metrics.OutputMaxInFlight().WithLabelValues(topic).Set(float64(topicKafkaCfg.MaxInFlight))

	c.log.WithField("topic", topic).
		Info("Starting stream for dynamically discovered topic")

	if c.lagMonitor != nil {
		c.log.WithField("topic", topic).
			Warn("Dynamically discovered topic will not have consumer lag monitoring — restart to include it")
	}

	g.Go(func() error {
		if err := stream.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
			return fmt.Errorf("running stream for topic %q: %w", topic, err)
		}

		return nil
	})
}

func (c *Consumoor) startPProf(_ context.Context) error {
	c.log.WithField("addr", c.config.PProfAddr).Info("Starting pprof server")

	srv := &http.Server{
		Addr:              *c.config.PProfAddr,
		ReadHeaderTimeout: 120 * time.Second,
	}

	c.mu.Lock()
	c.pprofServer = srv
	c.mu.Unlock()

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}

	return nil
}
