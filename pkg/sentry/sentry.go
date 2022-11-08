package sentry

import (
	"context"
	"errors"
	"log"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/ethpandaops/xatu/pkg/sentry/ethereum"
	"github.com/ethpandaops/xatu/pkg/sentry/output"
	"github.com/ethpandaops/xatu/pkg/wallclock"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Sentry struct {
	Config *Config

	sinks []output.Sink

	beacon *ethereum.BeaconNode

	log logrus.FieldLogger

	wallclock *wallclock.EthereumBeaconChain
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

	return &Sentry{
		Config: config,
		sinks:  sinks,
		beacon: beacon,
		log:    log,
	}, nil
}

func (s *Sentry) Start(ctx context.Context) error {
	s.log.WithField("version", xatu.Full()).Info("Starting Xatu in sentry mode")

	s.beacon.OnReady(ctx, func(ctx context.Context) error {
		s.log.Info("Internal beacon node is ready, subscribing to events")

		s.beacon.Node().OnAttestation(ctx, s.handleAttestation)

		s.beacon.Node().OnBlock(ctx, s.handleBlock)

		s.beacon.Node().OnChainReOrg(ctx, s.handleChainReOrg)

		s.beacon.Node().OnHead(ctx, s.handleHead)

		s.beacon.Node().OnVoluntaryExit(ctx, s.handleVoluntaryExit)

		return nil
	})

	if err := s.beacon.Start(ctx); err != nil {
		return err
	}

	cancel := make(chan os.Signal, 1)
	signal.Notify(cancel, syscall.SIGTERM, syscall.SIGINT)

	sig := <-cancel
	log.Printf("Caught signal: %v", sig)

	return nil
}

func (s *Sentry) createNewClientMeta(ctx context.Context, topic xatu.ClientMeta_Event_Name) (*xatu.ClientMeta, error) {
	return &xatu.ClientMeta{
		Name:           s.Config.Name,
		Version:        xatu.Full(),
		Id:             "00000000000000000",
		Implementation: xatu.Implementation,
		Os:             runtime.GOOS,
		Event: &xatu.ClientMeta_Event{
			Name:     topic,
			DateTime: timestamppb.Now(),
		},
		Ethereum: &xatu.ClientMeta_Ethereum{
			Network: &xatu.ClientMeta_Ethereum_Network{
				Name: s.Config.Ethereum.Network,
				Id:   999, // TODO
			},
			Execution: &xatu.ClientMeta_Ethereum_Execution{
				Implementation: s.Config.Ethereum.ExecutionClient,
				Version:        "",
			},
			Consensus: &xatu.ClientMeta_Ethereum_Consensus{
				Implementation: s.Config.Ethereum.ConsensusClient,
				Version:        "",
			},
		},
		Labels: s.Config.Labels,
	}, nil
}

func (s *Sentry) handleNewDecoratedEvent(ctx context.Context, event xatu.DecoratedEvent) error {
	for _, sink := range s.sinks {
		if err := sink.HandleNewDecoratedEvent(ctx, event); err != nil {
			s.log.WithError(err).WithField("sink", sink.Type()).Error("Failed to send event to sink")
		}
	}

	return nil
}
