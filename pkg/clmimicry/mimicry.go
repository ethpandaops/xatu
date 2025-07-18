package clmimicry

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	//nolint:gosec // only exposed if pprofAddr config is set
	_ "net/http/pprof"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/OffchainLabs/prysm/v6/beacon-chain/core/signing"
	"github.com/OffchainLabs/prysm/v6/beacon-chain/p2p/encoder"
	"github.com/OffchainLabs/prysm/v6/config/params"
	"github.com/OffchainLabs/prysm/v6/time/slots"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/beevik/ntp"
	"github.com/ethpandaops/ethwallclock"
	"github.com/ethpandaops/xatu/pkg/clmimicry/ethereum"
	"github.com/ethpandaops/xatu/pkg/output"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/go-co-op/gocron/v2"
	"github.com/google/uuid"
	"github.com/probe-lab/hermes/eth"
	"github.com/probe-lab/hermes/host"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
)

const unknown = "unknown"

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

	sharder *UnifiedSharder

	processor *Processor
}

func New(ctx context.Context, log logrus.FieldLogger, config *Config, overrides *Override) (*Mimicry, error) {
	if config == nil {
		return nil, errors.New("config is required")
	}

	if err := config.Validate(); err != nil {
		return nil, err
	}

	if overrides != nil {
		if err := config.ApplyOverrides(overrides, log); err != nil {
			return nil, fmt.Errorf("failed to apply overrides: %w", err)
		}
	}

	sinks, err := config.CreateSinks(log)
	if err != nil {
		return nil, err
	}

	addr := fmt.Sprintf("%s:%d", config.Node.PrysmHost, config.Node.PrysmPortHTTP)

	if !strings.HasPrefix(addr, "http") && !strings.HasPrefix(addr, "https") {
		if config.Node.PrysmUseTLS {
			addr = "https://" + addr
		} else {
			addr = "http://" + addr
		}
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

	// Create the unified sharder
	sharder, err := NewUnifiedSharder(&config.Sharding, true)
	if err != nil {
		return nil, fmt.Errorf("failed to create sharder: %w", err)
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
		sharder:     sharder,
	}

	return mimicry, nil
}

func (m *Mimicry) startHermes(ctx context.Context) error {
	var (
		err error
		c   *eth.NetworkConfig
	)

	m.log.Info("Deriving network config:", "chain", m.Config.Ethereum.Network)

	switch m.Config.Ethereum.Network {
	case params.DevnetName:
		c, err = eth.DeriveDevnetConfig(ctx, eth.DevnetOptions{
			ConfigURL:               m.Config.Ethereum.Devnet.ConfigURL,
			BootnodesURL:            m.Config.Ethereum.Devnet.BootnodesURL,
			DepositContractBlockURL: m.Config.Ethereum.Devnet.DepositContractBlockURL,
			GenesisSSZURL:           m.Config.Ethereum.Devnet.GenesisSSZURL,
		})
	default:
		c, err = eth.DeriveKnownNetworkConfig(ctx, m.Config.Ethereum.Network)
	}

	if err != nil {
		return fmt.Errorf("get config for %s: %w", m.Config.Ethereum.Network, err)
	}

	m.networkConfig = c.Network
	m.beaconConfig = c.Beacon

	// Initialize the processor now that beaconConfig is available
	m.log.Info("Initializing processor..")

	m.processor = NewProcessor(
		m,                                 // DutiesProvider
		m,                                 // OutputHandler
		m.metrics,                         // MetricsCollector
		m,                                 // MetaProvider
		m.sharder,                         // UnifiedSharder
		NewEventCategorizer(),             // EventCategorizer
		m.ethereum.Metadata().Wallclock(), // EthereumBeaconChain
		m.clockDrift,                      // clockDrift
		m.Config.Events,                   // EventConfig
		m.log.WithField("component", "processor"),
	)

	m.log.Info("Processor initialized successfully")

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
		if err := m.processor.HandleHermesEvent(ctx, event); err != nil {
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

	m.log.Info(m.Config.Sharding.LogSummary())

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
			m.log.Fatalf("failed to start hermes: %v", err)
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

func (m *Mimicry) GetClientMeta(ctx context.Context) (*xatu.ClientMeta, error) {
	return m.createNewClientMeta(ctx)
}

func (m *Mimicry) createNewClientMeta(ctx context.Context) (*xatu.ClientMeta, error) {
	return &xatu.ClientMeta{
		Name:           m.Config.Name,
		Version:        xatu.Short(),
		Id:             m.id.String(),
		Implementation: xatu.Implementation,
		Os:             runtime.GOOS,
		ModuleName:     xatu.ModuleName_CL_MIMICRY,
		ClockDrift: func() uint64 {
			ms := m.clockDrift.Milliseconds()
			if ms < 0 {
				return 0
			}

			return uint64(ms)
		}(),
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
	c, err := gocron.NewScheduler(gocron.WithLocation(time.Local))
	if err != nil {
		return err
	}

	if _, err := c.NewJob(
		gocron.DurationJob(5*time.Minute),
		gocron.NewTask(
			func(ctx context.Context) {
				if err := m.syncClockDrift(ctx); err != nil {
					m.log.WithError(err).Error("Failed to sync clock drift")
				}
			},
			ctx,
		),
		gocron.WithStartAt(gocron.WithStartImmediately()),
	); err != nil {
		return err
	}

	c.Start()

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
		networkStr = unknown
	}

	eventType := event.GetEvent().GetName().String()
	if eventType == "" {
		eventType = unknown
	}

	m.metrics.AddDecoratedEvent(1, eventType, networkStr)

	for _, sink := range m.sinks {
		if err := sink.HandleNewDecoratedEvent(ctx, event); err != nil {
			m.log.WithError(err).WithField("sink", sink.Type()).Error("Failed to send event to sink")
		}
	}

	return nil
}

func (m *Mimicry) handleNewDecoratedEvents(ctx context.Context, events []*xatu.DecoratedEvent) error {
	if len(events) == 0 {
		return nil
	}

	// Grab the first event and use it to get the network and event type.
	// Saves us parsing the same data multiple times.
	var (
		event      = events[0]
		network    = event.GetMeta().GetClient().GetEthereum().GetNetwork().GetId()
		networkStr = fmt.Sprintf("%d", network)
	)

	if networkStr == "" || networkStr == "0" {
		networkStr = unknown
	}

	for _, event := range events {
		eventType := event.GetEvent().GetName().String()
		if eventType == "" {
			eventType = unknown
		}

		m.metrics.AddDecoratedEvent(1, eventType, networkStr)
	}

	for _, sink := range m.sinks {
		if err := sink.HandleNewDecoratedEvents(ctx, events); err != nil {
			m.log.WithError(err).WithField("sink", sink.Type()).Error("Failed to send events to sink")
		}
	}

	return nil
}

// Implement MetadataProvider interface
func (m *Mimicry) Wallclock() *ethwallclock.EthereumBeaconChain {
	return m.ethereum.Metadata().Wallclock()
}

func (m *Mimicry) ClockDrift() *time.Duration {
	return &m.clockDrift
}

func (m *Mimicry) Network() *xatu.ClientMeta_Ethereum_Network {
	return &xatu.ClientMeta_Ethereum_Network{
		Name: m.Config.Ethereum.Network,
		Id:   m.beaconConfig.DepositNetworkID,
	}
}

// Implement DutiesProvider interface
func (m *Mimicry) GetValidatorIndex(epoch phase0.Epoch, slot phase0.Slot, committeeIndex phase0.CommitteeIndex, position uint64) (phase0.ValidatorIndex, error) {
	return m.ethereum.Duties().GetValidatorIndex(epoch, slot, committeeIndex, position)
}

// Implement OutputHandler interface
func (m *Mimicry) HandleDecoratedEvent(ctx context.Context, event *xatu.DecoratedEvent) error {
	return m.handleNewDecoratedEvent(ctx, event)
}

func (m *Mimicry) HandleDecoratedEvents(ctx context.Context, events []*xatu.DecoratedEvent) error {
	return m.handleNewDecoratedEvents(ctx, events)
}

// GetProcessor returns the processor for testing purposes
func (m *Mimicry) GetProcessor() *Processor {
	return m.processor
}
