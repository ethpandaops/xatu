package sentry

import (
	"context"
	"errors"
	"log"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	v1 "github.com/attestantio/go-eth2-client/api/v1"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/ethpandaops/xatu/pkg/sentry/ethereum"
	"github.com/ethpandaops/xatu/pkg/sentry/output"
	"github.com/ethpandaops/xatu/pkg/xatu"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Sentry struct {
	Config *Config

	sinks []output.Sink

	beacon *ethereum.BeaconNode

	log logrus.FieldLogger
}

func New(log logrus.FieldLogger, config *Config) (*Sentry, error) {
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

	beacon, err := ethereum.NewBeaconNode(config.Name, &config.Ethereum, log)
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
	if err := s.beacon.Start(ctx); err != nil {
		return err
	}

	s.beacon.Node().OnAttestation(ctx, func(ctx context.Context, event *phase0.Attestation) error {
		s.log.Info("Attestation received")

		return s.createNewDecoratedMessage(ctx, event, xatu.ClientMeta_Event_BEACON_API_ETH_V1_EVENTS_ATTESTATION)
	})

	s.beacon.Node().OnBlock(ctx, func(ctx context.Context, event *v1.BlockEvent) error {
		s.log.Info("BlockEvent received")

		return s.createNewDecoratedMessage(ctx, event, xatu.ClientMeta_Event_BEACON_API_ETH_V1_EVENTS_BLOCK)
	})

	s.beacon.Node().OnChainReOrg(ctx, func(ctx context.Context, event *v1.ChainReorgEvent) error {
		s.log.Info("ChainReorg received")

		return s.createNewDecoratedMessage(ctx, event, xatu.ClientMeta_Event_BEACON_API_ETH_V1_EVENTS_CHAIN_REORG)
	})

	s.beacon.Node().OnHead(ctx, func(ctx context.Context, event *v1.HeadEvent) error {
		s.log.Info("Head received")

		return s.createNewDecoratedMessage(ctx, event, xatu.ClientMeta_Event_BEACON_API_ETH_V1_EVENTS_HEAD)
	})

	s.beacon.Node().OnVoluntaryExit(ctx, func(ctx context.Context, event *phase0.VoluntaryExit) error {
		s.log.Info("VoluntaryExit received")

		return s.createNewDecoratedMessage(ctx, event, xatu.ClientMeta_Event_BEACON_API_ETH_V1_EVENTS_VOLUNTARY_EXIT)
	})

	s.beacon.Node().OnFinalizedCheckpoint(ctx, func(ctx context.Context, event *v1.FinalizedCheckpointEvent) error {
		s.log.Info("FinalizedCheckpoint received")

		return s.createNewDecoratedMessage(ctx, event, xatu.ClientMeta_Event_BEACON_API_ETH_V1_EVENTS_FINALIZED_CHECKPOINT)
	})

	cancel := make(chan os.Signal, 1)
	signal.Notify(cancel, syscall.SIGTERM, syscall.SIGINT)

	sig := <-cancel
	log.Printf("Caught signal: %v", sig)

	return nil
}

func (s *Sentry) createNewDecoratedMessage(ctx context.Context, rawEvent interface{}, topic xatu.ClientMeta_Event_Name) error {
	event := xatu.DecoratedEvent{
		Meta: &xatu.Meta{
			Client: &xatu.ClientMeta{
				Name:           s.Config.Name,
				Version:        "0.0.0/dev/dev",
				Id:             "00000000000000000",
				Implementation: "xatu",
				Os:             runtime.GOOS,
				Event: &xatu.ClientMeta_Event{
					Name:     topic,
					DateTime: timestamppb.Now(),
				},
				Ethereum: &xatu.ClientMeta_Ethereum{
					NetworkName: s.Config.Ethereum.Network,
					NetworkId:   999, // TODO
					Execution: &xatu.ClientMeta_Ethereum_Execution{
						Implementation: s.Config.Ethereum.ExecutionClient,
						Version:        "0.0.0/dev/dev",
					},
					Consensus: &xatu.ClientMeta_Ethereum_Consensus{
						Implementation: s.Config.Ethereum.ConsensusClient,
						Version:        "0.0.0/dev/dev",
					},
				},
				Labels: s.Config.Labels,
			},
		},
	}

	for _, sink := range s.sinks {
		if err := sink.HandleNewDecoratedEvent(ctx, event); err != nil {
			return err
		}
	}

	return nil
}
