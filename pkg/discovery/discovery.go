package discovery

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	//nolint:gosec // only exposed if pprofAddr config is set
	_ "net/http/pprof"

	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethpandaops/xatu/pkg/discovery/cache"
	"github.com/ethpandaops/xatu/pkg/discovery/coordinator"
	"github.com/ethpandaops/xatu/pkg/discovery/p2p"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/go-co-op/gocron"
	"github.com/google/uuid"
	"github.com/jellydator/ttlcache/v3"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
)

type Discovery struct {
	Config *Config

	coordinator *coordinator.Client

	p2p    p2p.P2P
	status *p2p.Status

	log logrus.FieldLogger

	duplicateCache *cache.DuplicateCache

	id uuid.UUID

	metrics *Metrics
}

func New(ctx context.Context, log logrus.FieldLogger, config *Config, overrides *Override) (*Discovery, error) {
	if config == nil {
		return nil, errors.New("config is required")
	}

	if err := config.Validate(); err != nil {
		return nil, err
	}

	if overrides != nil {
		if err := config.ApplyOverrides(overrides, log); err != nil {
			return nil, fmt.Errorf("failed to apply overrides: %w", err)
		}
	}

	client, err := coordinator.New(&config.Coordinator, log)
	if err != nil {
		return nil, err
	}

	duplicateCache := cache.NewDuplicateCache()

	err = duplicateCache.Start(ctx)
	if err != nil {
		return nil, err
	}

	return &Discovery{
		Config:         config,
		coordinator:    client,
		log:            log,
		duplicateCache: duplicateCache,
		id:             uuid.New(),
		metrics:        NewMetrics("xatu_discovery"),
	}, nil
}

func (d *Discovery) Start(ctx context.Context) error {
	if err := d.ServeMetrics(ctx); err != nil {
		return err
	}

	if d.Config.PProfAddr != nil {
		if err := d.ServePProf(ctx); err != nil {
			return err
		}
	}

	d.log.
		WithField("version", xatu.Full()).
		WithField("id", d.id.String()).
		Info("Starting Xatu in discovery mode")

	if err := d.coordinator.Start(ctx); err != nil {
		return err
	}

	p2pDisc, err := p2p.NewP2P(d.Config.P2P.Type, d.Config.P2P.Config, d.handleNewNodeRecord, d.log)
	if err != nil {
		return err
	}

	d.p2p = p2pDisc

	if errP := d.p2p.Start(ctx); errP != nil {
		return errP
	}

	d.status = p2p.NewStatus(ctx, &d.Config.P2P, d.log)

	if errS := d.status.Start(ctx); errS != nil {
		return errS
	}

	d.status.OnExecutionStatus(ctx, d.handleExecutionStatus)

	if err := d.startCrons(ctx); err != nil {
		return err
	}

	cancel := make(chan os.Signal, 1)
	signal.Notify(cancel, syscall.SIGTERM, syscall.SIGINT)

	sig := <-cancel
	d.log.Printf("Caught signal: %v", sig)

	if d.p2p != nil {
		if err := d.p2p.Stop(ctx); err != nil {
			return err
		}
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
		sm := http.NewServeMux()
		sm.Handle("/metrics", promhttp.Handler())

		server := &http.Server{
			Addr:              d.Config.MetricsAddr,
			ReadHeaderTimeout: 15 * time.Second,
			Handler:           sm,
		}

		d.log.Infof("Serving metrics at %s", d.Config.MetricsAddr)

		if err := server.ListenAndServe(); err != nil {
			d.log.Fatal(err)
		}
	}()

	return nil
}

func (d *Discovery) ServePProf(ctx context.Context) error {
	pprofServer := &http.Server{
		Addr:              *d.Config.PProfAddr,
		ReadHeaderTimeout: 120 * time.Second,
	}

	go func() {
		d.log.Infof("Serving pprof at %s", *d.Config.PProfAddr)

		if err := pprofServer.ListenAndServe(); err != nil {
			d.log.Fatal(err)
		}
	}()

	return nil
}

func (d *Discovery) startCrons(ctx context.Context) error {
	c := gocron.NewScheduler(time.Local)

	if _, err := c.Every("5s").Do(func() {
		d.log.WithFields(logrus.Fields{
			"records": d.status.ActiveExecution(),
		}).Info("execution records currently trying to dial")
		if d.status.ActiveExecution() == 0 {
			d.log.Info("no active execution records to dial, requesting stale records from coordinator")
			nodeRecords, err := d.coordinator.ListStaleNodeRecords(ctx)
			if err != nil {
				d.log.WithError(err).Error("Failed to list stale node records")

				return
			}
			d.log.WithField("records", len(nodeRecords)).Info("Adding stale node records to status")
			d.status.AddExecutionNodeRecords(ctx, nodeRecords)
		}
	}); err != nil {
		return err
	}

	c.StartAsync()

	return nil
}

func (d *Discovery) handleExecutionStatus(ctx context.Context, status *xatu.ExecutionNodeStatus) error {
	d.metrics.AddNodeRecordStatus(1, fmt.Sprintf("%d", status.GetNetworkId()), fmt.Sprintf("0x%x", status.GetForkId().GetHash()))

	return d.coordinator.HandleExecutionNodeRecordStatus(ctx, status)
}

func (d *Discovery) handleNewNodeRecord(ctx context.Context, node *enode.Node, source string) error {
	d.log.Debug("Node received")

	if node == nil {
		return errors.New("node is nil")
	}

	enr := node.String()

	item, retrieved := d.duplicateCache.Node.GetOrSet(enr, time.Now(), ttlcache.WithTTL[string, time.Time](ttlcache.DefaultTTL))
	if retrieved {
		d.log.WithFields(logrus.Fields{
			"enr":                   enr,
			"time_since_first_item": time.Since(item.Value()),
		}).Debug("Duplicate node received")

		return nil
	}

	err := d.coordinator.HandleNewNodeRecord(ctx, &enr)
	if err != nil {
		return err
	}

	d.metrics.AddDiscoveredNodeRecord(1, source)

	return nil
}
