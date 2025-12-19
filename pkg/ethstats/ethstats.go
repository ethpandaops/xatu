package ethstats

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	//nolint:gosec // only exposed if pprofAddr config is set
	_ "net/http/pprof"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/beevik/ntp"
	"github.com/ethpandaops/ethcore/pkg/ethereum/clients"
	"github.com/ethpandaops/ethcore/pkg/ethereum/networks"
	"github.com/ethpandaops/xatu/pkg/auth"
	"github.com/ethpandaops/xatu/pkg/geoip"
	"github.com/ethpandaops/xatu/pkg/observability"
	"github.com/ethpandaops/xatu/pkg/output"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/google/uuid"
	perrors "github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/sdk/trace"
)

// Ethstats is the main ethstats service.
type Ethstats struct {
	config *Config

	sinks []output.Sink

	log logrus.FieldLogger

	id uuid.UUID

	metrics *Metrics

	clockDrift time.Duration

	authorization *auth.Authorization

	geoipProvider geoip.Provider

	server *Server

	shutdownFuncs []func(context.Context) error
}

// New creates a new Ethstats service.
func New(ctx context.Context, log logrus.FieldLogger, config *Config) (*Ethstats, error) {
	log = log.WithField("module", "ethstats")

	if config == nil {
		return nil, errors.New("config is required")
	}

	if err := config.Validate(); err != nil {
		return nil, err
	}

	sinks, err := config.CreateSinks(log)
	if err != nil {
		return nil, err
	}

	metrics := NewMetrics("xatu_ethstats")

	e := &Ethstats{
		config:        config,
		sinks:         sinks,
		log:           log,
		id:            uuid.New(),
		metrics:       metrics,
		clockDrift:    time.Duration(0),
		shutdownFuncs: []func(context.Context) error{},
	}

	// Setup authorization if configured
	if config.Authorization.Enabled {
		authorization, err := auth.NewAuthorization(log, config.Authorization)
		if err != nil {
			return nil, fmt.Errorf("failed to create authorization: %w", err)
		}

		e.authorization = authorization
	}

	// Setup GeoIP provider if configured
	if config.GeoIP.Enabled {
		geoipProvider, err := geoip.NewProvider(config.GeoIP.Type, config.GeoIP.Config, log)
		if err != nil {
			return nil, fmt.Errorf("failed to create geoip provider: %w", err)
		}

		e.geoipProvider = geoipProvider
	}

	// Create handler with clientMeta callback
	handler := NewHandler(
		log,
		sinks,
		e.createClientMeta,
		metrics,
		config.ClientNameSalt,
		e.geoipProvider,
	)

	// Create server
	e.server = NewServer(
		config,
		log,
		e.authorization,
		handler,
		metrics,
	)

	return e, nil
}

// Start starts the ethstats service.
func (e *Ethstats) Start(ctx context.Context) error {
	if err := e.ServeMetrics(ctx); err != nil {
		return err
	}

	if e.config.PProfAddr != nil {
		if err := e.ServePProf(ctx); err != nil {
			return err
		}
	}

	e.log.
		WithField("version", xatu.Full()).
		WithField("id", e.id.String()).
		Info("Starting Xatu in ethstats mode")

	// Start tracing if enabled
	if e.config.Tracing.Enabled {
		e.log.Info("Tracing enabled")

		res, err := observability.NewResource(xatu.WithModule(xatu.ModuleName_ETHSTATS), xatu.Short())
		if err != nil {
			return perrors.Wrap(err, "failed to create tracing resource")
		}

		opts := []trace.TracerProviderOption{
			trace.WithSampler(trace.ParentBased(trace.TraceIDRatioBased(e.config.Tracing.Sampling.Rate))),
		}

		tracer, err := observability.NewHTTPTraceProvider(ctx,
			res,
			e.config.Tracing.AsOTelOpts(),
			opts...,
		)
		if err != nil {
			return perrors.Wrap(err, "failed to create tracing provider")
		}

		shutdown, err := observability.SetupOTelSDK(ctx, tracer)
		if err != nil {
			return perrors.Wrap(err, "failed to setup tracing SDK")
		}

		e.shutdownFuncs = append(e.shutdownFuncs, shutdown)
	}

	// Start clock drift sync
	go e.runClockDriftSync(ctx)

	// Start authorization if configured
	if e.authorization != nil {
		if err := e.authorization.Start(ctx); err != nil {
			return perrors.Wrap(err, "failed to start authorization")
		}
	}

	// Start GeoIP provider if configured
	if e.geoipProvider != nil {
		if err := e.geoipProvider.Start(ctx); err != nil {
			return perrors.Wrap(err, "failed to start geoip provider")
		}
	}

	// Start sinks
	for _, sink := range e.sinks {
		e.log.WithField("type", sink.Type()).WithField("name", sink.Name()).Info("Starting sink")

		if err := sink.Start(ctx); err != nil {
			return err
		}
	}

	// Start WebSocket server
	if err := e.server.Start(ctx); err != nil {
		return err
	}

	// Wait for signal
	cancel := make(chan os.Signal, 1)
	signal.Notify(cancel, syscall.SIGTERM, syscall.SIGINT)

	sig := <-cancel
	e.log.Printf("Caught signal: %v", sig)

	e.log.Printf("Stopping server")

	if err := e.server.Stop(ctx); err != nil {
		e.log.WithError(err).Error("Error stopping server")
	}

	e.log.Printf("Flushing sinks")

	for _, sink := range e.sinks {
		if err := sink.Stop(ctx); err != nil {
			return err
		}
	}

	for _, f := range e.shutdownFuncs {
		if err := f(ctx); err != nil {
			return err
		}
	}

	return nil
}

// ServeMetrics starts the metrics server.
func (e *Ethstats) ServeMetrics(ctx context.Context) error {
	go func() {
		sm := http.NewServeMux()
		sm.Handle("/metrics", promhttp.Handler())

		server := &http.Server{
			Addr:              e.config.MetricsAddr,
			ReadHeaderTimeout: 15 * time.Second,
			Handler:           sm,
		}

		e.log.Infof("Serving metrics at %s", e.config.MetricsAddr)

		if err := server.ListenAndServe(); err != nil {
			e.log.Fatal(err)
		}
	}()

	return nil
}

// ServePProf starts the pprof server.
func (e *Ethstats) ServePProf(ctx context.Context) error {
	pprofServer := &http.Server{
		Addr:              *e.config.PProfAddr,
		ReadHeaderTimeout: 120 * time.Second,
	}

	go func() {
		e.log.Infof("Serving pprof at %s", *e.config.PProfAddr)

		if err := pprofServer.ListenAndServe(); err != nil {
			e.log.Fatal(err)
		}
	}()

	return nil
}

// runClockDriftSync syncs clock drift periodically.
func (e *Ethstats) runClockDriftSync(ctx context.Context) {
	// Sync immediately on startup
	if err := e.syncClockDrift(ctx); err != nil {
		e.log.WithError(err).Error("Failed to sync clock drift")
	}

	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := e.syncClockDrift(ctx); err != nil {
				e.log.WithError(err).Error("Failed to sync clock drift")
			}
		}
	}
}

// syncClockDrift queries NTP and updates clock drift.
func (e *Ethstats) syncClockDrift(_ context.Context) error {
	response, err := ntp.Query(e.config.NTPServer)
	if err != nil {
		return err
	}

	err = response.Validate()
	if err != nil {
		return err
	}

	e.clockDrift = response.ClockOffset

	e.log.WithField("drift", e.clockDrift).Debug("Updated clock drift")

	if e.clockDrift > 2*time.Second || e.clockDrift < -2*time.Second {
		e.log.WithField("drift", e.clockDrift).Warn("Large clock drift detected")
	}

	return nil
}

// createClientMeta creates a ClientMeta for a connected node.
func (e *Ethstats) createClientMeta(node *Node) *xatu.ClientMeta {
	info := node.Info()

	var networkMeta *xatu.ClientMeta_Ethereum_Network

	if info != nil && info.Network != "" {
		// Try to parse network as ID (e.g., "1" for mainnet)
		if networkID, err := strconv.ParseUint(info.Network, 10, 64); err == nil {
			network := networks.DeriveFromID(networkID)
			networkMeta = &xatu.ClientMeta_Ethereum_Network{
				Name: string(network.Name),
				Id:   networkID,
			}
		} else {
			// Fall back to using the raw network string as name
			networkMeta = &xatu.ClientMeta_Ethereum_Network{
				Name: info.Network,
			}
		}
	}

	// Apply network name override if configured (useful for testnets)
	if e.config.OverrideNetworkName != "" && networkMeta != nil {
		networkMeta.Name = e.config.OverrideNetworkName
	}

	// Client name is the ethstats server name from config
	clientName := e.config.Name

	var executionMeta *xatu.ClientMeta_Ethereum_Execution

	if info != nil && info.Node != "" {
		// Parse the full client version string (e.g., "Geth/v1.16.8-unstable-63c4383b/linux-amd64/go1.24.11")
		// In geth's ethstats, info.node contains the full client version string
		impl, version, major, minor, patch := clients.ParseExecutionClientVersion(info.Node)

		executionMeta = &xatu.ClientMeta_Ethereum_Execution{
			Implementation: impl,
			Version:        version,
			VersionMajor:   major,
			VersionMinor:   minor,
			VersionPatch:   patch,
		}
	}

	return &xatu.ClientMeta{
		Name:           clientName,
		Version:        xatu.Short(),
		Id:             e.id.String(),
		Implementation: xatu.Implementation,
		ModuleName:     xatu.ModuleName_ETHSTATS,
		Os:             runtime.GOOS,
		ClockDrift:     uint64(e.clockDrift.Milliseconds()), //nolint:gosec // clock drift is bounded
		Ethereum: &xatu.ClientMeta_Ethereum{
			Network:   networkMeta,
			Execution: executionMeta,
			Consensus: &xatu.ClientMeta_Ethereum_Consensus{},
		},
		Labels: e.config.Labels,
	}
}
