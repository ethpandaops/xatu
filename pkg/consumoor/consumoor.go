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
	log    logrus.FieldLogger
	config *Config

	metrics    *telemetry.Metrics
	router     *router.Engine
	writer     source.Writer
	streams    []topicStream
	lagMonitor *source.LagMonitor

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
	}

	metrics.OutputMaxInFlight().Set(float64(config.Kafka.MaxInFlight))

	c := &Consumoor{
		log:     cLog,
		config:  config,
		metrics: metrics,
		router:  rtr,
		writer:  writer,
		streams: streams,
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

	if err := c.writer.Start(ctx); err != nil {
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
			c.watchTopics(gCtx)

			return nil
		})
	}

	streamErr := g.Wait()

	// All streams and HTTP servers have exited. Now stop the writer.
	c.stopWriter(ctx)

	if streamErr != nil && streamErr != context.Canceled {
		return streamErr
	}

	return nil
}

// stopHTTPServers shuts down the metrics and pprof servers. Called from
// within the errgroup on context cancellation so that g.Wait() can return.
func (c *Consumoor) stopHTTPServers(ctx context.Context) {
	c.mu.Lock()
	metricsServer := c.metricsServer
	pprofServer := c.pprofServer
	c.mu.Unlock()

	if metricsServer != nil {
		if err := metricsServer.Shutdown(ctx); err != nil {
			c.log.WithError(err).Error("Error stopping metrics server")
		}
	}

	if pprofServer != nil {
		if err := pprofServer.Shutdown(ctx); err != nil {
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
// the configured regex patterns. It logs newly appeared and disappeared topics
// and updates the active_topics gauge. The actual consumption of new topics is
// handled by Benthos via metadata_max_age; this goroutine provides visibility.
func (c *Consumoor) watchTopics(ctx context.Context) {
	interval := c.config.Kafka.TopicRefreshInterval

	c.log.WithField("interval", interval).
		Info("Starting topic discovery watcher")

	// Perform an initial discovery so the metric is populated immediately.
	knownTopics := c.discoverAndDiff(ctx, nil)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			c.log.Info("Topic discovery watcher stopped")

			return
		case <-ticker.C:
			knownTopics = c.discoverAndDiff(ctx, knownTopics)
		}
	}
}

// discoverAndDiff calls DiscoverTopics, compares the result to the previously
// known set, logs any changes, and updates the active_topics gauge. It returns
// the current set of discovered topics for the next comparison cycle.
func (c *Consumoor) discoverAndDiff(
	ctx context.Context,
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

	// Find newly appeared topics.
	for _, t := range topics {
		if _, ok := previous[t]; !ok {
			c.log.WithField("topic", t).
				Info("Discovered new topic matching pattern")
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
