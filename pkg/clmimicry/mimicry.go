package clmimicry

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"time"

	//nolint:gosec // only exposed if pprofAddr config is set
	_ "net/http/pprof"

	"github.com/beevik/ntp"
	"github.com/ethpandaops/xatu/pkg/clmimicry/ethereum"
	"github.com/ethpandaops/xatu/pkg/output"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/go-co-op/gocron"
	"github.com/google/uuid"
	"github.com/probe-lab/hermes/eth"
	"github.com/probe-lab/hermes/host"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/signing"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/p2p/encoder"
	"github.com/prysmaticlabs/prysm/v5/config/params"
	"github.com/prysmaticlabs/prysm/v5/time/slots"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
)

type Mimicry struct {
	Config *Config

	sinks []output.Sink

	clockDrift time.Duration

	log logrus.FieldLogger

	id uuid.UUID

	metrics *Metrics

	startupTime time.Time

	node *eth.Node

	networkConfig *params.NetworkConfig
	beaconConfig  *params.BeaconChainConfig

	ethereum *ethereum.BeaconNode
}

func New(ctx context.Context, log logrus.FieldLogger, config *Config) (*Mimicry, error) {
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

	addr := fmt.Sprintf("%s:%d", config.Node.PrysmHost, config.Node.PrysmPortHTTP)

	if !strings.HasPrefix(addr, "http") {
		addr = "http://" + addr
	}

	client, err := ethereum.NewBeaconNode(ctx,
		config.Name,
		&config.Ethereum,
		log,
		addr,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create ethereum client: %w", err)
	}

	mimicry := &Mimicry{
		Config:      config,
		sinks:       sinks,
		clockDrift:  time.Duration(0),
		log:         log,
		id:          uuid.New(),
		metrics:     NewMetrics("xatu_cl_mimicry"),
		startupTime: time.Now(),
		ethereum:    client,
	}

	return mimicry, nil
}

func (m *Mimicry) startHermes(ctx context.Context) error {
	c, err := eth.DeriveKnownNetworkConfig(ctx, m.Config.Ethereum.Network)
	if err != nil {
		return fmt.Errorf("get config for %s: %w", m.Config.Ethereum.Network, err)
	}

	m.networkConfig = c.Network
	m.beaconConfig = c.Beacon

	genesisRoot := c.Genesis.GenesisValidatorRoot
	genesisTime := c.Genesis.GenesisTime

	// compute fork version and fork digest
	currentSlot := slots.Since(genesisTime)
	currentEpoch := slots.ToEpoch(currentSlot)

	currentForkVersion, err := eth.GetCurrentForkVersion(currentEpoch, m.beaconConfig)
	if err != nil {
		return fmt.Errorf("compute fork version for epoch %d: %w", currentEpoch, err)
	}

	forkDigest, err := signing.ComputeForkDigest(currentForkVersion[:], genesisRoot)
	if err != nil {
		return fmt.Errorf("create fork digest (%s, %x): %w", genesisTime, genesisRoot, err)
	}

	// Overriding configuration so that functions like ComputForkDigest take the
	// correct input data from the global configuration.
	params.OverrideBeaconConfig(m.beaconConfig)
	params.OverrideBeaconNetworkConfig(m.networkConfig)

	nodeConfig := m.Config.Node.AsHermesConfig()

	nodeConfig.GenesisConfig = c.Genesis
	nodeConfig.NetworkConfig = m.networkConfig
	nodeConfig.BeaconConfig = m.beaconConfig
	nodeConfig.ForkDigest = forkDigest
	nodeConfig.ForkVersion = currentForkVersion
	nodeConfig.PubSubSubscriptionRequestLimit = 200
	nodeConfig.PubSubQueueSize = 200
	nodeConfig.Libp2pPeerscoreSnapshotFreq = 60 * time.Second
	nodeConfig.GossipSubMessageEncoder = encoder.SszNetworkEncoder{}
	nodeConfig.RPCEncoder = encoder.SszNetworkEncoder{}
	nodeConfig.Tracer = otel.GetTracerProvider().Tracer("hermes")
	nodeConfig.Meter = otel.GetMeterProvider().Meter("hermes")

	err = nodeConfig.Validate()
	if err != nil {
		return fmt.Errorf("invalid Hermes node config: %w", err)
	}

	node, err := eth.NewNode(nodeConfig)
	if err != nil {
		if strings.Contains(err.Error(), "in correct fork_digest") {
			return fmt.Errorf("invalid fork digest (config.ethereum.network and prysm network probably don't match): %w", err)
		}

		return err
	}

	node.OnEvent(func(ctx context.Context, event *host.TraceEvent) {
		if err := m.handleHermesEvent(ctx, event); err != nil {
			m.log.WithError(err).Error("Failed to handle hermes event")
		}
	})

	m.node = node

	if err := m.node.Start(ctx); err != nil {
		return err
	}

	return nil
}

func (m *Mimicry) Start(ctx context.Context) error {
	if err := m.ServeMetrics(ctx); err != nil {
		return err
	}

	if m.Config.PProfAddr != nil {
		if err := m.ServePProf(ctx); err != nil {
			return err
		}
	}

	if m.Config.ProbeAddr != nil {
		if err := m.ServeProbe(ctx); err != nil {
			return err
		}
	}

	m.log.
		WithField("version", xatu.Full()).
		WithField("id", m.id.String()).
		Info("Starting Xatu in consensus layer mimicry mode")

	if err := m.startCrons(ctx); err != nil {
		m.log.WithError(err).Fatal("Failed to start crons")
	}

	for _, sink := range m.sinks {
		if err := sink.Start(ctx); err != nil {
			return err
		}
	}

	m.ethereum.OnReady(ctx, func(ctx context.Context) error {
		m.log.Info("Ethereum client is ready. Starting Hermes..")

		if err := m.startHermes(ctx); err != nil {
			m.log.Fatal("failed to start hermes: %w", err)
		}

		return nil
	})

	if err := m.ethereum.Start(ctx); err != nil {
		return err
	}

	cancel := make(chan os.Signal, 1)
	signal.Notify(cancel, syscall.SIGTERM, syscall.SIGINT)

	sig := <-cancel
	m.log.Printf("Caught signal: %v", sig)

	m.log.Printf("Flushing sinks")

	for _, sink := range m.sinks {
		if err := sink.Stop(ctx); err != nil {
			return err
		}
	}

	return nil
}

func (m *Mimicry) ServeMetrics(ctx context.Context) error {
	go func() {
		sm := http.NewServeMux()
		sm.Handle("/metrics", promhttp.Handler())

		server := &http.Server{
			Addr:              m.Config.MetricsAddr,
			ReadHeaderTimeout: 15 * time.Second,
			Handler:           sm,
		}

		m.log.Infof("Serving metrics at %s", m.Config.MetricsAddr)

		if err := server.ListenAndServe(); err != nil {
			m.log.Fatal(err)
		}
	}()

	return nil
}

func (m *Mimicry) ServePProf(ctx context.Context) error {
	pprofServer := &http.Server{
		Addr:              *m.Config.PProfAddr,
		ReadHeaderTimeout: 120 * time.Second,
	}

	go func() {
		m.log.Infof("Serving pprof at %s", *m.Config.PProfAddr)

		if err := pprofServer.ListenAndServe(); err != nil {
			m.log.Fatal(err)
		}
	}()

	return nil
}

func (m *Mimicry) ServeProbe(ctx context.Context) error {
	probeServer := &http.Server{
		Addr:              *m.Config.ProbeAddr,
		ReadHeaderTimeout: 120 * time.Second,
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, err := w.Write([]byte("OK"))
			if err != nil {
				m.log.Error("Failed to write response: ", err)
			}
		}),
	}

	go func() {
		m.log.Infof("Serving probe at %s", *m.Config.ProbeAddr)

		if err := probeServer.ListenAndServe(); err != nil {
			m.log.Fatal(err)
		}
	}()

	return nil
}

func (m *Mimicry) createNewClientMeta(ctx context.Context) (*xatu.ClientMeta, error) {
	return &xatu.ClientMeta{
		Name:           m.Config.Name,
		Version:        xatu.Short(),
		Id:             m.id.String(),
		Implementation: xatu.Implementation,
		Os:             runtime.GOOS,
		ModuleName:     xatu.ModuleName_CL_MIMICRY,
		ClockDrift:     uint64(m.clockDrift.Milliseconds()),
		Ethereum: &xatu.ClientMeta_Ethereum{
			Network: &xatu.ClientMeta_Ethereum_Network{
				Name: m.Config.Ethereum.Network,
				Id:   m.beaconConfig.DepositNetworkID,
			},
			Execution: &xatu.ClientMeta_Ethereum_Execution{},
			Consensus: &xatu.ClientMeta_Ethereum_Consensus{},
		},

		Labels: m.Config.Labels,
	}, nil
}

func (m *Mimicry) startCrons(ctx context.Context) error {
	c := gocron.NewScheduler(time.Local)

	if _, err := c.Every("5m").Do(func() {
		if err := m.syncClockDrift(ctx); err != nil {
			m.log.WithError(err).Error("Failed to sync clock drift")
		}
	}); err != nil {
		return err
	}

	c.StartAsync()

	return nil
}

func (m *Mimicry) syncClockDrift(ctx context.Context) error {
	response, err := ntp.Query(m.Config.NTPServer)
	if err != nil {
		return err
	}

	err = response.Validate()
	if err != nil {
		return err
	}

	m.clockDrift = response.ClockOffset
	m.log.WithField("drift", m.clockDrift).Info("Updated clock drift")

	return err
}

func (m *Mimicry) handleNewDecoratedEvent(ctx context.Context, event *xatu.DecoratedEvent) error {
	network := event.GetMeta().GetClient().GetEthereum().GetNetwork().GetId()
	networkStr := fmt.Sprintf("%d", network)

	if networkStr == "" || networkStr == "0" {
		networkStr = "unknown"
	}

	eventType := event.GetEvent().GetName().String()
	if eventType == "" {
		eventType = "unknown"
	}

	m.metrics.AddDecoratedEvent(1, eventType, networkStr)

	for _, sink := range m.sinks {
		if err := sink.HandleNewDecoratedEvent(ctx, event); err != nil {
			m.log.WithError(err).WithField("sink", sink.Type()).Error("Failed to send event to sink")
		}
	}

	return nil
}
