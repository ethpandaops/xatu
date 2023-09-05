package cannon

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

	//nolint:gosec // only exposed if pprofAddr config is set
	_ "net/http/pprof"

	"github.com/beevik/ntp"
	"github.com/ethpandaops/xatu/pkg/cannon/ethereum"
	v2 "github.com/ethpandaops/xatu/pkg/cannon/event/beacon/eth/v2"
	"github.com/ethpandaops/xatu/pkg/output"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/go-co-op/gocron"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
)

type Cannon struct {
	Config *Config

	sinks []output.Sink

	beacon *ethereum.BeaconNode

	clockDrift time.Duration

	log logrus.FieldLogger

	id uuid.UUID

	metrics *Metrics

	scheduler *gocron.Scheduler

	beaconBlockDerivers []v2.BeaconBlockEventDeriver
}

func New(ctx context.Context, log logrus.FieldLogger, config *Config) (*Cannon, error) {
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

	beaconBlockDerivers := []v2.BeaconBlockEventDeriver{
		v2.NewAttesterSlashingDeriver(log),
		v2.NewProposerSlashingDeriver(log),
		v2.NewVoluntaryExitDeriver(log),
		v2.NewDepositDeriver(log),
		v2.NewBLSToExecutionChangeDeriver(log),
		v2.NewExecutionTransactionDeriver(log),
	}

	return &Cannon{
		Config:              config,
		sinks:               sinks,
		beacon:              beacon,
		clockDrift:          time.Duration(0),
		log:                 log,
		id:                  uuid.New(),
		metrics:             NewMetrics("xatu_cannon"),
		scheduler:           gocron.NewScheduler(time.Local),
		beaconBlockDerivers: beaconBlockDerivers,
	}, nil
}

func (c *Cannon) Start(ctx context.Context) error {
	if err := c.ServeMetrics(ctx); err != nil {
		return err
	}

	if c.Config.PProfAddr != nil {
		if err := c.ServePProf(ctx); err != nil {
			return err
		}
	}

	if err := c.startBeaconBlockProcessor(ctx); err != nil {
		return err
	}

	c.log.
		WithField("version", xatu.Full()).
		WithField("id", c.id.String()).
		Info("Starting Xatu in cannon mode 💣")

	if err := c.startCrons(ctx); err != nil {
		c.log.WithError(err).Fatal("Failed to start crons")
	}

	for _, sink := range c.sinks {
		if err := sink.Start(ctx); err != nil {
			return err
		}
	}

	if c.Config.Ethereum.OverrideNetworkName != "" {
		c.log.WithField("network", c.Config.Ethereum.OverrideNetworkName).Info("Overriding network name")
	}

	if err := c.beacon.Start(ctx); err != nil {
		return err
	}

	cancel := make(chan os.Signal, 1)
	signal.Notify(cancel, syscall.SIGTERM, syscall.SIGINT)

	sig := <-cancel
	c.log.Printf("Caught signal: %v", sig)

	c.log.Printf("Flushing sinks")

	for _, sink := range c.sinks {
		if err := sink.Stop(ctx); err != nil {
			return err
		}
	}

	return nil
}

func (c *Cannon) ServeMetrics(ctx context.Context) error {
	go func() {
		sm := http.NewServeMux()
		sm.Handle("/metrics", promhttp.Handler())

		server := &http.Server{
			Addr:              c.Config.MetricsAddr,
			ReadHeaderTimeout: 15 * time.Second,
			Handler:           sm,
		}

		c.log.Infof("Serving metrics at %s", c.Config.MetricsAddr)

		if err := server.ListenAndServe(); err != nil {
			c.log.Fatal(err)
		}
	}()

	return nil
}

func (c *Cannon) ServePProf(ctx context.Context) error {
	pprofServer := &http.Server{
		Addr:              *c.Config.PProfAddr,
		ReadHeaderTimeout: 120 * time.Second,
	}

	go func() {
		c.log.Infof("Serving pprof at %s", *c.Config.PProfAddr)

		if err := pprofServer.ListenAndServe(); err != nil {
			c.log.Fatal(err)
		}
	}()

	return nil
}

func (c *Cannon) createNewClientMeta(ctx context.Context) (*xatu.ClientMeta, error) {
	var networkMeta *xatu.ClientMeta_Ethereum_Network

	network := c.beacon.Metadata().Network
	if network != nil {
		networkMeta = &xatu.ClientMeta_Ethereum_Network{
			Name: string(network.Name),
			Id:   network.ID,
		}

		if c.Config.Ethereum.OverrideNetworkName != "" {
			networkMeta.Name = c.Config.Ethereum.OverrideNetworkName
		}
	}

	return &xatu.ClientMeta{
		Name:           c.Config.Name,
		Version:        xatu.Short(),
		Id:             c.id.String(),
		Implementation: xatu.Implementation,
		Os:             runtime.GOOS,
		ClockDrift:     uint64(c.clockDrift.Milliseconds()),
		Ethereum: &xatu.ClientMeta_Ethereum{
			Network:   networkMeta,
			Execution: &xatu.ClientMeta_Ethereum_Execution{},
			Consensus: &xatu.ClientMeta_Ethereum_Consensus{
				Implementation: c.beacon.Metadata().Client(ctx),
				Version:        c.beacon.Metadata().NodeVersion(ctx),
			},
		},
		Labels: c.Config.Labels,
	}, nil
}

func (c *Cannon) startCrons(ctx context.Context) error {
	if _, err := c.scheduler.Every("5m").Do(func() {
		if err := c.syncClockDrift(ctx); err != nil {
			c.log.WithError(err).Error("Failed to sync clock drift")
		}
	}); err != nil {
		return err
	}

	c.scheduler.StartAsync()

	return nil
}

func (c *Cannon) syncClockDrift(ctx context.Context) error {
	response, err := ntp.Query(c.Config.NTPServer)
	if err != nil {
		return err
	}

	err = response.Validate()
	if err != nil {
		return err
	}

	c.clockDrift = response.ClockOffset
	c.log.WithField("drift", c.clockDrift).Info("Updated clock drift")

	return err
}

func (c *Cannon) handleNewDecoratedEvent(ctx context.Context, event *xatu.DecoratedEvent) error {
	if err := c.beacon.Synced(ctx); err != nil {
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

	c.metrics.AddDecoratedEvent(1, eventType, networkStr)

	for _, sink := range c.sinks {
		if err := sink.HandleNewDecoratedEvent(ctx, event); err != nil {
			c.log.
				WithError(err).
				WithField("sink", sink.Type()).
				WithField("event_type", event.GetEvent().GetName()).
				Error("Failed to send event to sink")
		}
	}

	return nil
}

func (c *Cannon) startBeaconBlockProcessor(ctx context.Context) error {
	c.beacon.OnReady(ctx, func(ctx context.Context) error {
		c.log.Info("Internal beacon node is ready, firing up beacon block processor")

		// TODO: Fetch our starting point from xatu-server
		start := uint64(5193791)

		for {
			select {
			case <-ctx.Done():
				return nil
			default:
				time.Sleep(1 * time.Second)

				c.log.WithField("slot", start).Info("Processing beacon block")

				if err := c.processBeaconBlock(ctx, start); err != nil {
					c.log.WithError(err).Error("Failed to process beacon block")

					// TODO: Make sure we don't tell xatu-server we've processed this block
					// If we can't process it, we should stop here and wait for
					// human intervention.
				}

				start++
			}
		}
	})

	return nil
}

func (c *Cannon) processBeaconBlock(ctx context.Context, slot uint64) error {
	block, err := c.beacon.Node().FetchBlock(ctx, fmt.Sprintf("%d", slot))
	if err != nil {
		return err
	}

	meta, err := c.createNewClientMeta(ctx)
	if err != nil {
		return err
	}

	event := v2.NewBeaconBlockMetadata(c.log, block, time.Now(), c.beacon, meta, c.beaconBlockDerivers)

	events, err := event.Process(ctx)
	if err != nil {
		return err
	}

	for _, event := range events {
		if err := c.handleNewDecoratedEvent(ctx, event); err != nil {
			return err
		}
	}

	return nil
}
