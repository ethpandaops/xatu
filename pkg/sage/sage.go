package sage

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/ethpandaops/xatu/pkg/output"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/ethpandaops/xatu/pkg/sage/cache"
	"github.com/ethpandaops/xatu/pkg/sage/ethereum"
	"github.com/ethpandaops/xatu/pkg/sage/event/armiarma"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/r3labs/sse/v2"
	"github.com/sirupsen/logrus"
	"gopkg.in/cenkalti/backoff.v1"
)

type Sage struct {
	log            logrus.FieldLogger
	config         *Config
	armiarma       *sse.Client
	sinks          []output.Sink
	beacon         *ethereum.BeaconNode
	id             uuid.UUID
	duplicateCache *cache.DuplicateCache

	metrics *Metrics

	attestationCh chan *armiarma.TimedEthereumAttestation
}

func New(ctx context.Context, log logrus.FieldLogger, config *Config) (*Sage, error) {
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

	return &Sage{
		log:            log.WithField("module", "sage"),
		config:         config,
		armiarma:       client,
		beacon:         beacon,
		id:             uuid.New(),
		sinks:          sinks,
		metrics:        NewMetrics("xatu_sage"),
		duplicateCache: duplicateCache,
		attestationCh:  make(chan *armiarma.TimedEthereumAttestation, 20000),
	}, nil
}

func (a *Sage) Start(ctx context.Context) error {
	a.log.
		WithField("version", xatu.Full()).
		Info("Starting Xatu in sage mode")

	a.startWorkers(ctx)

	a.duplicateCache.Start()

	a.beacon.OnReady(ctx, func(ctx context.Context) error {
		a.log.Info("Beacon node is ready")

		//nolint:goconst // Not critical
		if a.beacon.Metadata().Network.Name == "unknown" {
			a.log.Fatal("Unable to determine network. Provide a network name override in the config")

			return errors.New("unable to determine network name")
		}

		if err := a.subscribeToEvents(ctx); err != nil {
			a.log.Fatal(err)
		}

		return nil
	})

	if a.config.PProfAddr != nil {
		if err := a.ServePProf(ctx); err != nil {
			return err
		}
	}

	if err := a.ServeMetrics(ctx); err != nil {
		return err
	}

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

func (a *Sage) startWorkers(ctx context.Context) {
	a.log.WithField("count", a.config.Workers).Info("Starting workers")

	for i := 0; i < a.config.Workers; i++ {
		go a.processChannels(ctx)
	}
}

func (a *Sage) ServeMetrics(ctx context.Context) error {
	go func() {
		sm := http.NewServeMux()
		sm.Handle("/metrics", promhttp.Handler())

		server := &http.Server{
			Addr:              a.config.MetricsAddr,
			ReadHeaderTimeout: 15 * time.Second,
			Handler:           sm,
		}

		a.log.Infof("Serving metrics at %s", a.config.MetricsAddr)

		if err := server.ListenAndServe(); err != nil {
			a.log.Fatal(err)
		}
	}()

	return nil
}

func (a *Sage) ServePProf(ctx context.Context) error {
	pprofServer := &http.Server{
		Addr:              *a.config.PProfAddr,
		ReadHeaderTimeout: 120 * time.Second,
	}

	go func() {
		a.log.Infof("Serving pprof at %s", *a.config.PProfAddr)

		if err := pprofServer.ListenAndServe(); err != nil {
			a.log.Fatal(err)
		}
	}()

	return nil
}

func (a *Sage) processChannels(ctx context.Context) {
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

func (a *Sage) subscribeToEvents(ctx context.Context) error {
	a.log.Info("Subscribing to events upstream")

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if err := a.armiarma.SubscribeWithContext(ctx, "timed_ethereum_attestation", func(msg *sse.Event) {
				event := &armiarma.TimedEthereumAttestation{}

				if err := json.Unmarshal(msg.Data, event); err != nil {
					a.log.WithError(err).Error("Failed to unmarshal attestation event")

					return
				}

				// Add the event to the channel
				a.attestationCh <- event
			}); err != nil {
				return err
			}
		}
	}
}

func (a *Sage) createNewClientMeta(ctx context.Context) (*xatu.ClientMeta, error) {
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

func (a *Sage) handleNewDecoratedEvent(ctx context.Context, event *xatu.DecoratedEvent) error {
	network := event.GetMeta().GetClient().GetEthereum().GetNetwork().GetId()
	networkStr := fmt.Sprintf("%d", network)

	if networkStr == "" || networkStr == "0" {
		networkStr = "unknown"
	}

	eventType := event.GetEvent().GetName().String()
	if eventType == "" {
		eventType = "unknown"
	}

	a.metrics.AddDecoratedEvent(1, eventType, networkStr)

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
