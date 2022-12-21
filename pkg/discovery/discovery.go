package discovery

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethpandaops/xatu/pkg/discovery/cache"
	"github.com/ethpandaops/xatu/pkg/discovery/coordinator"
	"github.com/ethpandaops/xatu/pkg/discovery/p2p"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/go-co-op/gocron"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
)

type Discovery struct {
	Config *Config

	coordinator *coordinator.Client

	discV5 *p2p.DiscV5
	status *p2p.Status

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

	d.discV5 = p2p.NewDiscV5(ctx, &d.Config.P2P, d.log)

	err := d.discV5.Start(ctx)
	if err != nil {
		return err
	}

	d.discV5.OnNodeRecord(ctx, func(ctx context.Context, node *enode.Node) error {
		return d.handleNewNodeRecord(ctx, node.String())
	})

	d.status = p2p.NewStatus(ctx, &d.Config.P2P, d.log)

	err = d.status.Start(ctx)
	if err != nil {
		return err
	}

	d.status.OnStatus(ctx, func(ctx context.Context, status *xatu.ExecutionNodeStatus) error {
		return d.coordinator.HandleExecutionNodeRecordStatus(ctx, status)
	})

	if err := d.startCrons(ctx); err != nil {
		return err
	}

	cancel := make(chan os.Signal, 1)
	signal.Notify(cancel, syscall.SIGTERM, syscall.SIGINT)

	sig := <-cancel
	d.log.Printf("Caught signal: %v", sig)

	if err := d.discV5.Stop(ctx); err != nil {
		return err
	}

	if err := d.status.Stop(ctx); err != nil {
		return err
	}

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

func (d *Discovery) startCrons(ctx context.Context) error {
	c := gocron.NewScheduler(time.Local)

	if _, err := c.Every("5s").Do(func() {
		d.log.WithFields(logrus.Fields{
			"records": d.status.Active(),
		}).Info("execution records currently trying to dial")
		if d.status.Active() == 0 {
			d.log.Info("no active execution records to dial, requesting stale records from coordinator")
			nodeRecords, err := d.coordinator.ListStaleNodeRecords(ctx)
			if err != nil {
				d.log.WithError(err).Error("Failed to list stale node records")
				return
			}
			d.log.WithField("records", len(nodeRecords)).Info("Adding stale node records to status")
			d.status.AddNodeRecords(ctx, nodeRecords)
		}
	}); err != nil {
		return err
	}

	c.StartAsync()

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
