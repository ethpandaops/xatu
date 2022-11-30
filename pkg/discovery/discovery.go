package discovery

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ethpandaops/xatu/pkg/discovery/cache"
	"github.com/ethpandaops/xatu/pkg/discovery/coordinator"
	"github.com/ethpandaops/xatu/pkg/discovery/p2p"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
)

type Discovery struct {
	Config *Config

	coordinator *coordinator.Client

	discV5 *p2p.DiscV5

	log logrus.FieldLogger

	duplicateCache *cache.DuplicateCache

	id uuid.UUID
}

func New(ctx context.Context, log logrus.FieldLogger, config *Config) (*Discovery, error) {
	if config == nil {
		return nil, errors.New("config is required")
	}

	if err := config.Validate(); err != nil {
		return nil, err
	}

	client, err := coordinator.New(&config.Coordinator, log)
	if err != nil {
		return nil, err
	}

	duplicateCache := cache.NewDuplicateCache()
	duplicateCache.Start()

	return &Discovery{
		Config:         config,
		coordinator:    client,
		log:            log,
		duplicateCache: duplicateCache,
		id:             uuid.New(),
	}, nil
}

func (d *Discovery) Start(ctx context.Context) error {
	if err := d.ServeMetrics(ctx); err != nil {
		return err
	}

	d.log.
		WithField("version", xatu.Full()).
		WithField("id", d.id.String()).
		Info("Starting Xatu in discovery mode")

	if err := d.coordinator.Start(ctx); err != nil {
		return err
	}

	d.discV5 = p2p.NewDiscV5(ctx, &d.Config.P2P, d.log, d.handleNode)
	err := d.discV5.Start(ctx)

	if err != nil {
		return err
	}

	cancel := make(chan os.Signal, 1)
	signal.Notify(cancel, syscall.SIGTERM, syscall.SIGINT)

	sig := <-cancel
	d.log.Printf("Caught signal: %v", sig)

	d.log.Printf("Flushing sinks")

	if err := d.coordinator.Stop(ctx); err != nil {
		return err
	}

	return nil
}

func (d *Discovery) ServeMetrics(ctx context.Context) error {
	go func() {
		server := &http.Server{
			Addr:              d.Config.MetricsAddr,
			ReadHeaderTimeout: 15 * time.Second,
		}

		server.Handler = promhttp.Handler()

		d.log.Infof("Serving metrics at %s", d.Config.MetricsAddr)

		if err := server.ListenAndServe(); err != nil {
			d.log.Fatal(err)
		}
	}()

	return nil
}

func (d *Discovery) handleNewNodeRecord(ctx context.Context, record string) error {
	d.log.WithField("node record", record).Debug("Received new node record")

	err := d.coordinator.HandleNewNodeRecord(ctx, &record)
	if err != nil {
		return err
	}

	return nil
}
