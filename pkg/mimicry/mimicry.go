package mimicry

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
	"github.com/ethpandaops/xatu/pkg/mimicry/coordinator"
	"github.com/ethpandaops/xatu/pkg/mimicry/p2p/handler"
	"github.com/ethpandaops/xatu/pkg/output"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/go-co-op/gocron"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
)

type Mimicry struct {
	Config *Config

	sinks []output.Sink

	coordinator coordinator.Coordinator

	clockDrift time.Duration

	log logrus.FieldLogger

	id uuid.UUID

	metrics *Metrics

	startupTime time.Time
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

	mimicry := &Mimicry{
		Config:      config,
		sinks:       sinks,
		clockDrift:  time.Duration(0),
		log:         log,
		id:          uuid.New(),
		metrics:     NewMetrics("xatu_mimicry"),
		startupTime: time.Now(),
	}

	mimicry.coordinator, err = coordinator.NewCoordinator(config.Name, config.Coordinator.Type, config.Coordinator.Config, &handler.Peer{
		CreateNewClientMeta: mimicry.createNewClientMeta,
		DecoratedEvent:      mimicry.handleNewDecoratedEvent,
	}, config.CaptureDelay, log)
	if err != nil {
		return nil, err
	}

	return mimicry, nil
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
		Info("Starting Xatu in mimicry mode")

	if err := m.startCrons(ctx); err != nil {
		m.log.WithError(err).Fatal("Failed to start crons")
	}

	for _, sink := range m.sinks {
		if err := sink.Start(ctx); err != nil {
			return err
		}
	}

	if err := m.coordinator.Start(ctx); err != nil {
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
			if time.Since(m.startupTime) > m.Config.CaptureDelay {
				w.WriteHeader(http.StatusOK)
				_, err := w.Write([]byte("OK"))
				if err != nil {
					m.log.Error("Failed to write response: ", err)
				}
			} else {
				w.WriteHeader(http.StatusServiceUnavailable)
				_, err := w.Write([]byte("Service is not ready yet"))
				if err != nil {
					m.log.Error("Failed to write response: ", err)
				}
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
		ClockDrift:     uint64(m.clockDrift.Milliseconds()),
		Labels:         m.Config.Labels,
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
