package relaymonitor

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

	perrors "github.com/pkg/errors"

	//nolint:gosec // only exposed if pprofAddr config is set
	_ "net/http/pprof"

	"github.com/beevik/ntp"
	"github.com/ethpandaops/xatu/pkg/output"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/ethpandaops/xatu/pkg/relaymonitor/ethereum"
	"github.com/ethpandaops/xatu/pkg/relaymonitor/relay"
	"github.com/go-co-op/gocron"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
)

type RelayMonitor struct {
	Config *Config

	sinks []output.Sink

	clockDrift time.Duration

	log logrus.FieldLogger

	id uuid.UUID

	metrics *Metrics

	ethereum *ethereum.BeaconNetwork

	relays []*relay.Client

	bidCache *DuplicateBidCache
}

const (
	namespace = "xatu_relay_monitor"
)

func New(ctx context.Context, log logrus.FieldLogger, config *Config) (*RelayMonitor, error) {
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

	client, err := ethereum.NewBeaconNetwork(
		log,
		&config.Ethereum,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create ethereum client: %w", err)
	}

	relays := make([]*relay.Client, 0, len(config.Relays))

	for _, r := range config.Relays {
		//
		relayClient, err := relay.NewClient(namespace, r, config.Ethereum.Network)
		if err != nil {
			return nil, perrors.Wrap(err, "failed to create relay client")
		}

		relays = append(relays, relayClient)
	}

	relayMonitor := &RelayMonitor{
		Config:     config,
		sinks:      sinks,
		clockDrift: time.Duration(0),
		log:        log,
		id:         uuid.New(),
		metrics:    NewMetrics(namespace, config.Ethereum.Network),
		ethereum:   client,
		relays:     relays,
		bidCache:   NewDuplicateBidCache(time.Minute * 13),
	}

	return relayMonitor, nil
}

func (r *RelayMonitor) Start(ctx context.Context) error {
	if err := r.ServeMetrics(ctx); err != nil {
		return err
	}

	if r.Config.PProfAddr != nil {
		if err := r.ServePProf(ctx); err != nil {
			return err
		}
	}

	if r.Config.ProbeAddr != nil {
		if err := r.ServeProbe(ctx); err != nil {
			return err
		}
	}

	r.log.
		WithField("version", xatu.Full()).
		WithField("id", r.id.String()).
		Info("Starting Xatu in relay monitor mode")

	// Sync clock drift
	if err := r.syncClockDrift(ctx); err != nil {
		// This is not fatal, as we will sync it on the first poll
		r.log.WithError(err).Warn("Failed to sync clock drift")
	}

	for _, sink := range r.sinks {
		if err := sink.Start(ctx); err != nil {
			return err
		}
	}

	if err := r.ethereum.Start(ctx); err != nil {
		return err
	}

	if err := r.startCrons(ctx); err != nil {
		r.log.WithError(err).Fatal("Failed to start crons")
	}

	cancel := make(chan os.Signal, 1)
	signal.Notify(cancel, syscall.SIGTERM, syscall.SIGINT)

	sig := <-cancel
	r.log.Printf("Caught signal: %v", sig)

	r.log.Printf("Flushing sinks")

	for _, sink := range r.sinks {
		if err := sink.Stop(ctx); err != nil {
			return err
		}
	}

	return nil
}

func (r *RelayMonitor) ServeMetrics(ctx context.Context) error {
	go func() {
		sm := http.NewServeMux()
		sm.Handle("/metrics", promhttp.Handler())

		server := &http.Server{
			Addr:              r.Config.MetricsAddr,
			ReadHeaderTimeout: 15 * time.Second,
			Handler:           sm,
		}

		r.log.Infof("Serving metrics at %s", r.Config.MetricsAddr)

		if err := server.ListenAndServe(); err != nil {
			r.log.Fatal(err)
		}
	}()

	return nil
}

func (r *RelayMonitor) ServePProf(ctx context.Context) error {
	pprofServer := &http.Server{
		Addr:              *r.Config.PProfAddr,
		ReadHeaderTimeout: 120 * time.Second,
	}

	go func() {
		r.log.Infof("Serving pprof at %s", *r.Config.PProfAddr)

		if err := pprofServer.ListenAndServe(); err != nil {
			r.log.Fatal(err)
		}
	}()

	return nil
}

func (r *RelayMonitor) ServeProbe(ctx context.Context) error {
	probeServer := &http.Server{
		Addr:              *r.Config.ProbeAddr,
		ReadHeaderTimeout: 120 * time.Second,
		Handler: http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, err := w.Write([]byte("OK"))
			if err != nil {
				r.log.Error("Failed to write response: ", err)
			}
		}),
	}

	go func() {
		r.log.Infof("Serving probe at %s", *r.Config.ProbeAddr)

		if err := probeServer.ListenAndServe(); err != nil {
			r.log.Fatal(err)
		}
	}()

	return nil
}

func (r *RelayMonitor) createNewClientMeta(ctx context.Context) (*xatu.ClientMeta, error) {
	return &xatu.ClientMeta{
		Name:           r.Config.Name,
		Version:        xatu.Short(),
		Id:             r.id.String(),
		Implementation: xatu.Implementation,
		Os:             runtime.GOOS,
		ClockDrift:     uint64(r.clockDrift.Milliseconds()),
		Ethereum: &xatu.ClientMeta_Ethereum{
			Network: &xatu.ClientMeta_Ethereum_Network{
				Name: r.Config.Ethereum.Network,
			},
			Execution: &xatu.ClientMeta_Ethereum_Execution{},
			Consensus: &xatu.ClientMeta_Ethereum_Consensus{},
		},

		Labels: r.Config.Labels,
	}, nil
}

func (r *RelayMonitor) startCrons(ctx context.Context) error {
	c := gocron.NewScheduler(time.Local)

	if _, err := c.Every("5m").Do(func() {
		if err := r.syncClockDrift(ctx); err != nil {
			r.log.WithError(err).Error("Failed to sync clock drift")
		}
	}); err != nil {
		return err
	}

	if r.Config.Schedule.AtSlotTimes != nil {
		for _, timer := range r.Config.Schedule.AtSlotTimes {
			for _, relay := range r.relays {
				r.scheduleBidTraceFetchingAtSlotTime(ctx, timer.Duration, relay)
			}
		}
	}

	c.StartAsync()

	return nil
}

func (r *RelayMonitor) syncClockDrift(ctx context.Context) error {
	response, err := ntp.Query(r.Config.NTPServer)
	if err != nil {
		return err
	}

	err = response.Validate()
	if err != nil {
		return err
	}

	r.clockDrift = response.ClockOffset
	r.log.WithField("drift", r.clockDrift).Info("Updated clock drift")

	return err
}

func (r *RelayMonitor) handleNewDecoratedEvent(ctx context.Context, event *xatu.DecoratedEvent) error {
	eventType := event.GetEvent().GetName().String()
	if eventType == "" {
		eventType = "unknown"
	}

	r.metrics.AddDecoratedEvent(1, eventType)

	for _, sink := range r.sinks {
		if err := sink.HandleNewDecoratedEvent(ctx, event); err != nil {
			r.log.WithError(err).WithField("sink", sink.Type()).Error("Failed to send event to sink")
		}
	}

	return nil
}
