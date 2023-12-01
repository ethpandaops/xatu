package sage

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	"github.com/ethpandaops/xatu/pkg/output"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/ethpandaops/xatu/pkg/sage/cache"
	"github.com/ethpandaops/xatu/pkg/sage/ethereum"
	"github.com/ethpandaops/xatu/pkg/sage/event/armiarma"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/r3labs/sse/v2"
	"github.com/sirupsen/logrus"
	"gopkg.in/cenkalti/backoff.v1"
)

type sage struct {
	log            logrus.FieldLogger
	config         *Config
	armiarma       *sse.Client
	sinks          []output.Sink
	beacon         *ethereum.BeaconNode
	id             uuid.UUID
	duplicateCache *cache.DuplicateCache

	attestationCh chan *armiarma.TimedEthereumAttestation
}

func New(ctx context.Context, log logrus.FieldLogger, config *Config) (*sage, error) {
	if config == nil {
		return nil, errors.New("config is required")
	}

	if err := config.Validate(); err != nil {
		return nil, errors.Wrap(err, "invalid config")
	}

	sinks, err := config.createSinks(log)
	if err != nil {
		return nil, err
	}

	beacon, err := ethereum.NewBeaconNode(ctx, "beacon_node", &config.Ethereum, log)
	if err != nil {
		return nil, err
	}

	client := sse.NewClient(config.ArmiarmaURL)

	client.OnConnect(func(client *sse.Client) {
		log.Info("Connected to Armiarma")
	})

	client.OnDisconnect(func(client *sse.Client) {
		log.Info("Disconnected from Armiarma")
	})

	bkOff := backoff.NewExponentialBackOff()
	bkOff.MaxElapsedTime = 0

	client.ReconnectStrategy = bkOff

	duplicateCache := cache.NewDuplicateCache()

	return &sage{
		log:            log.WithField("module", "sage"),
		config:         config,
		armiarma:       client,
		beacon:         beacon,
		id:             uuid.New(),
		sinks:          sinks,
		duplicateCache: duplicateCache,
		attestationCh:  make(chan *armiarma.TimedEthereumAttestation, 20000),
	}, nil
}

func (a *sage) Start(ctx context.Context) error {
	a.log.
		WithField("version", xatu.Full()).
		Info("Starting Xatu in sage mode")

	a.startWorkers(ctx)

	a.duplicateCache.Start()

	a.beacon.OnReady(ctx, func(ctx context.Context) error {
		a.log.Info("Beacon node is ready")

		if a.beacon.Metadata().Network.Name == "unknown" {
			a.log.Fatal("Unable to determine network. Provide a network name override in the config")

			return errors.New("unable to determine network name")
		}

		a.subscribeToEvents(ctx)

		return nil
	})

	if err := a.beacon.Start(ctx); err != nil {
		return errors.Wrap(err, "failed to start beacon node")
	}

	cancel := make(chan os.Signal, 1)
	signal.Notify(cancel, syscall.SIGTERM, syscall.SIGINT)

	sig := <-cancel
	a.log.Printf("Caught signal: %v", sig)

	a.log.Printf("Flushing sinks")

	for _, sink := range a.sinks {
		if err := sink.Stop(ctx); err != nil {
			return err
		}
	}

	return nil
}

func (a *sage) startWorkers(ctx context.Context) {
	a.log.WithField("count", a.config.Workers).Info("Starting workers")

	for i := 0; i < a.config.Workers; i++ {
		go a.processChannels(ctx)
	}
}

func (a *sage) processChannels(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case event := <-a.attestationCh:
			if err := a.handleAttestationEvent(ctx, event); err != nil {
				a.log.WithError(err).Error("Failed to handle attestation event")
			}
		}
	}
}

func (a *sage) subscribeToEvents(ctx context.Context) error {
	a.log.Info("Subscribing to events upstream")

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			a.armiarma.SubscribeWithContext(ctx, "timed_ethereum_attestation", func(msg *sse.Event) {
				event := &armiarma.TimedEthereumAttestation{}

				if err := json.Unmarshal(msg.Data, event); err != nil {
					a.log.WithError(err).Error("Failed to unmarshal attestation event")

					return
				}

				// Add the event to the channel
				a.attestationCh <- event
			})
		}
	}
}

func (a *sage) createNewClientMeta(ctx context.Context) (*xatu.ClientMeta, error) {
	var networkMeta *xatu.ClientMeta_Ethereum_Network

	network := a.beacon.Metadata().Network
	if network != nil {
		networkMeta = &xatu.ClientMeta_Ethereum_Network{
			Name: string(network.Name),
			Id:   network.ID,
		}

		if a.config.Ethereum.OverrideNetworkName != "" {
			networkMeta.Name = a.config.Ethereum.OverrideNetworkName
		}
	}

	return &xatu.ClientMeta{
		Name:           "sage",
		Version:        xatu.Short(),
		Id:             a.id.String(),
		Implementation: xatu.Implementation,
		Os:             runtime.GOOS,
		ClockDrift:     0, // If we move Armiarma within-process we can calculate this
		Ethereum: &xatu.ClientMeta_Ethereum{
			Network:   networkMeta,
			Execution: &xatu.ClientMeta_Ethereum_Execution{},
			Consensus: &xatu.ClientMeta_Ethereum_Consensus{
				Implementation: "armiarma",
				Version:        "0.0.1",
			},
		},
		Labels: map[string]string{},
	}, nil
}

func (a *sage) handleNewDecoratedEvent(ctx context.Context, event *xatu.DecoratedEvent) error {
	network := event.GetMeta().GetClient().GetEthereum().GetNetwork().GetId()
	networkStr := fmt.Sprintf("%d", network)

	if networkStr == "" || networkStr == "0" {
		networkStr = "unknown"
	}

	eventType := event.GetEvent().GetName().String()
	if eventType == "" {
		eventType = "unknown"
	}

	// s.metrics.AddDecoratedEvent(1, eventType, networkStr)

	for _, sink := range a.sinks {
		if err := sink.HandleNewDecoratedEvent(ctx, event); err != nil {
			a.log.
				WithError(err).
				WithField("sink", sink.Type()).
				WithField("event_type", event.GetEvent().GetName()).
				Error("Failed to send event to sink")
		}
	}

	return nil
}
