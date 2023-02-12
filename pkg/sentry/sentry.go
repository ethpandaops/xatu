package sentry

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/beevik/ntp"
	"github.com/ethpandaops/xatu/pkg/output"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/ethpandaops/xatu/pkg/sentry/cache"
	"github.com/ethpandaops/xatu/pkg/sentry/ethereum"
	"github.com/go-co-op/gocron"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
)

type Sentry struct {
	Config *Config

	sinks []output.Sink

	beacon *ethereum.BeaconNode

	clockDrift time.Duration

	log logrus.FieldLogger

	duplicateCache *cache.DuplicateCache

	id uuid.UUID

	metrics *Metrics
}

func New(ctx context.Context, log logrus.FieldLogger, config *Config) (*Sentry, error) {
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

	beacon, err := ethereum.NewBeaconNode(ctx, config.Name, &config.Ethereum, log)
	if err != nil {
		return nil, err
	}

	duplicateCache := cache.NewDuplicateCache()
	duplicateCache.Start()

	return &Sentry{
		Config:         config,
		sinks:          sinks,
		beacon:         beacon,
		clockDrift:     time.Duration(0),
		log:            log,
		duplicateCache: duplicateCache,
		id:             uuid.New(),
		metrics:        NewMetrics("xatu_sentry"),
	}, nil
}

func (s *Sentry) Start(ctx context.Context) error {
	if err := s.ServeMetrics(ctx); err != nil {
		return err
	}

	s.log.
		WithField("version", xatu.Full()).
		WithField("id", s.id.String()).
		Info("Starting Xatu in sentry mode")

	s.beacon.OnReady(ctx, func(ctx context.Context) error {
		s.log.Info("Internal beacon node is ready, subscribing to events")

		s.beacon.Node().OnAttestation(ctx, s.handleAttestation)

		s.beacon.Node().OnBlock(ctx, s.handleBlock)

		s.beacon.Node().OnChainReOrg(ctx, s.handleChainReOrg)

		s.beacon.Node().OnHead(ctx, s.handleHead)

		s.beacon.Node().OnVoluntaryExit(ctx, s.handleVoluntaryExit)

		s.beacon.Node().OnContributionAndProof(ctx, s.handleContributionAndProof)

		return nil
	})

	if err := s.startCrons(ctx); err != nil {
		s.log.WithError(err).Fatal("Failed to start crons")
	}

	for _, sink := range s.sinks {
		if err := sink.Start(ctx); err != nil {
			return err
		}
	}

	if s.Config.Ethereum.OverrideNetworkName != "" {
		s.log.WithField("network", s.Config.Ethereum.OverrideNetworkName).Info("Overriding network name")
	}

	if err := s.beacon.Start(ctx); err != nil {
		return err
	}

	cancel := make(chan os.Signal, 1)
	signal.Notify(cancel, syscall.SIGTERM, syscall.SIGINT)

	sig := <-cancel
	s.log.Printf("Caught signal: %v", sig)

	s.log.Printf("Flushing sinks")

	for _, sink := range s.sinks {
		if err := sink.Stop(ctx); err != nil {
			return err
		}
	}

	return nil
}

func (s *Sentry) ServeMetrics(ctx context.Context) error {
	go func() {
		server := &http.Server{
			Addr:              s.Config.MetricsAddr,
			ReadHeaderTimeout: 15 * time.Second,
		}

		server.Handler = promhttp.Handler()

		s.log.Infof("Serving metrics at %s", s.Config.MetricsAddr)

		if err := server.ListenAndServe(); err != nil {
			s.log.Fatal(err)
		}
	}()

	return nil
}

func (s *Sentry) createNewClientMeta(ctx context.Context) (*xatu.ClientMeta, error) {
	var networkMeta *xatu.ClientMeta_Ethereum_Network

	network := s.beacon.Metadata().Network
	if network != nil {
		networkMeta = &xatu.ClientMeta_Ethereum_Network{
			Name: string(network.Name),
			Id:   network.ID,
		}

		if s.Config.Ethereum.OverrideNetworkName != "" {
			networkMeta.Name = s.Config.Ethereum.OverrideNetworkName
		}
	}

	return &xatu.ClientMeta{
		Name:           s.Config.Name,
		Version:        xatu.Short(),
		Id:             s.id.String(),
		Implementation: xatu.Implementation,
		Os:             runtime.GOOS,
		ClockDrift:     uint64(s.clockDrift.Milliseconds()),
		Ethereum: &xatu.ClientMeta_Ethereum{
			Network:   networkMeta,
			Execution: &xatu.ClientMeta_Ethereum_Execution{},
			Consensus: &xatu.ClientMeta_Ethereum_Consensus{
				Implementation: s.beacon.Metadata().Client(ctx),
				Version:        s.beacon.Metadata().NodeVersion(ctx),
			},
		},
		Labels: s.Config.Labels,
	}, nil
}

func (s *Sentry) startCrons(ctx context.Context) error {
	c := gocron.NewScheduler(time.Local)

	if _, err := c.Every("5m").Do(func() {
		if err := s.syncClockDrift(ctx); err != nil {
			s.log.WithError(err).Error("Failed to sync clock drift")
		}
	}); err != nil {
		return err
	}

	c.StartAsync()

	return nil
}

func (s *Sentry) syncClockDrift(ctx context.Context) error {
	response, err := ntp.Query(s.Config.NTPServer)
	if err != nil {
		return err
	}

	err = response.Validate()
	if err != nil {
		return err
	}

	s.clockDrift = response.ClockOffset
	s.log.WithField("drift", s.clockDrift).Info("Updated clock drift")

	return err
}

func (s *Sentry) handleNewDecoratedEvent(ctx context.Context, event *xatu.DecoratedEvent) error {
	if err := s.beacon.Synced(ctx); err != nil {
		return err
	}

	network := event.GetMeta().GetClient().GetEthereum().GetNetwork().GetId()
	networkStr := fmt.Sprintf("%d", network)

	if networkStr == "" || networkStr == "0" {
		networkStr = "unknown"
	}

	eventType := event.GetEvent().GetName().String()
	if eventType == "" {
		eventType = "unknown"
	}

	s.metrics.AddDecoratedEvent(1, eventType, networkStr)

	for _, sink := range s.sinks {
		if err := sink.HandleNewDecoratedEvent(ctx, event); err != nil {
			s.log.WithError(err).WithField("sink", sink.Type()).Error("Failed to send event to sink")
		}
	}

	return nil
}
