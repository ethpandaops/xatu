package discovery

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"net/http"
	//nolint:gosec // only exposed if pprofAddr config is set
	_ "net/http/pprof"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethpandaops/ethcore/pkg/cache"
	"github.com/ethpandaops/ethcore/pkg/ethereum/networks"
	"github.com/ethpandaops/xatu/pkg/discovery/coordinator"
	"github.com/ethpandaops/xatu/pkg/discovery/p2p"
	"github.com/ethpandaops/xatu/pkg/output"
	"github.com/ethpandaops/xatu/pkg/proto/noderecord"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/go-co-op/gocron/v2"
	"github.com/google/uuid"
	"github.com/jellydator/ttlcache/v3"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

type Discovery struct {
	Config         *Config
	sinks          []output.Sink
	coordinator    *coordinator.Client
	p2p            p2p.P2P
	status         *p2p.Status
	log            logrus.FieldLogger
	duplicateCache cache.DuplicateCache[string, time.Time]
	id             uuid.UUID
	metrics        *Metrics
	scheduler      gocron.Scheduler
	metricsServer  *http.Server
	pprofServer    *http.Server
	ctx            context.Context //nolint:containedctx // requires much larger refactor into channels.
	cancel         context.CancelFunc
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

	sinks, err := config.CreateSinks(log)
	if err != nil {
		return nil, err
	}

	client, err := coordinator.New(&config.Coordinator, log)
	if err != nil {
		return nil, err
	}

	// Configure cache with metrics
	cacheConfig := cache.Config{
		TTL: 120 * time.Minute,
		Metrics: &cache.MetricsConfig{
			Namespace:      "xatu_discovery",
			Subsystem:      "cache_duplicate",
			InstanceLabels: map[string]string{"store": "node"},
			UpdateInterval: 5 * time.Second,
			Registerer:     prometheus.DefaultRegisterer,
		},
	}

	duplicateCache := cache.NewDuplicateCacheWithConfig[string, time.Time](log, cacheConfig)

	err = duplicateCache.Start(ctx)
	if err != nil {
		return nil, err
	}

	return &Discovery{
		Config:         config,
		sinks:          sinks,
		coordinator:    client,
		log:            log,
		duplicateCache: duplicateCache,
		id:             uuid.New(),
		metrics:        NewMetrics("xatu_discovery"),
	}, nil
}

func (d *Discovery) Start(ctx context.Context) error {
	// Create a cancellable context for this discovery instance
	d.ctx, d.cancel = context.WithCancel(ctx)

	if err := d.ServeMetrics(d.ctx); err != nil {
		return err
	}

	if d.Config.PProfAddr != nil {
		if err := d.ServePProf(d.ctx); err != nil {
			return err
		}
	}

	d.log.
		WithField("version", xatu.Full()).
		WithField("id", d.id.String()).
		Info("Starting Xatu in discovery mode")

	if err := d.coordinator.Start(d.ctx); err != nil {
		return err
	}

	p2pDisc, err := p2p.NewP2P(d.Config.P2P.Type, d.Config.P2P.Config, d.handleNewNodeRecord, d.log)
	if err != nil {
		return err
	}

	d.p2p = p2pDisc

	if errP := d.p2p.Start(d.ctx); errP != nil {
		return errP
	}

	status, err := p2p.NewStatus(d.ctx, &d.Config.P2P, d.log)
	if err != nil {
		return err
	}

	d.status = status

	if errS := d.status.Start(d.ctx); errS != nil {
		return errS
	}

	d.status.OnExecutionStatus(d.ctx, d.handleExecutionStatus)
	d.status.OnConsensusStatus(d.ctx, d.handleConsensusStatus)

	if err := d.startCrons(d.ctx); err != nil {
		return err
	}

	for _, sink := range d.sinks {
		d.log.WithField("type", sink.Type()).WithField("name", sink.Name()).Info("Starting sink")

		if err := sink.Start(d.ctx); err != nil {
			return err
		}
	}

	cancel := make(chan os.Signal, 1)
	signal.Notify(cancel, syscall.SIGTERM, syscall.SIGINT)

	sig := <-cancel
	d.log.Printf("Caught signal: %v", sig)

	// Cancel the context to signal all components to stop
	if d.cancel != nil {
		d.cancel()
	}

	d.log.Printf("Flushing sinks")

	for _, sink := range d.sinks {
		if err := sink.Stop(ctx); err != nil {
			return err
		}
	}

	// Stop the scheduler
	if d.scheduler != nil {
		if err := d.scheduler.Shutdown(); err != nil {
			d.log.WithError(err).Error("Failed to shutdown scheduler")
		}
	}

	// Stop HTTP servers
	if d.metricsServer != nil {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := d.metricsServer.Shutdown(shutdownCtx); err != nil {
			d.log.WithError(err).Error("Failed to shutdown metrics server")
		}
	}

	if d.pprofServer != nil {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := d.pprofServer.Shutdown(shutdownCtx); err != nil {
			d.log.WithError(err).Error("Failed to shutdown pprof server")
		}
	}

	// Stop duplicate cache
	if d.duplicateCache != nil {
		if err := d.duplicateCache.Stop(); err != nil {
			d.log.WithError(err).Error("Failed to stop duplicate cache")
		}
	}

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
	sm := http.NewServeMux()
	sm.Handle("/metrics", promhttp.Handler())

	d.metricsServer = &http.Server{
		Addr:              d.Config.MetricsAddr,
		ReadHeaderTimeout: 15 * time.Second,
		Handler:           sm,
	}

	go func() {
		d.log.Infof("Serving metrics at %s", d.Config.MetricsAddr)

		if err := d.metricsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			d.log.WithError(err).Error("Metrics server error")
		}
	}()

	return nil
}

func (d *Discovery) ServePProf(ctx context.Context) error {
	d.pprofServer = &http.Server{
		Addr:              *d.Config.PProfAddr,
		ReadHeaderTimeout: 120 * time.Second,
	}

	go func() {
		d.log.Infof("Serving pprof at %s", *d.Config.PProfAddr)

		if err := d.pprofServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			d.log.WithError(err).Error("PProf server error")
		}
	}()

	return nil
}

func (d *Discovery) startCrons(ctx context.Context) error {
	scheduler, err := gocron.NewScheduler(gocron.WithLocation(time.Local))
	if err != nil {
		return err
	}

	d.scheduler = scheduler

	if _, err := d.scheduler.NewJob(
		gocron.DurationJob(5*time.Second),
		gocron.NewTask(
			func(ctx context.Context) {
				// Check if context is cancelled
				select {
				case <-ctx.Done():
					return
				default:
				}

				d.log.WithFields(logrus.Fields{
					"records": d.status.ActiveExecution(),
				}).Info("Execution records currently trying to dial")

				if d.status.ActiveExecution() == 0 {
					d.log.Info("No active execution records to dial, requesting stale records from coordinator")

					nodeRecords, err := d.coordinator.ListStaleExecutionNodeRecords(ctx)
					if err != nil {
						d.log.WithError(err).Error("Failed to list stale execution node records")

						return
					}

					d.log.WithField("records", len(nodeRecords)).Info("Adding stale execution node records to status")

					d.status.AddExecutionNodeRecords(ctx, nodeRecords)
				}
			},
			ctx,
		),
		gocron.WithStartAt(gocron.WithStartImmediately()),
	); err != nil {
		return err
	}

	if _, err := d.scheduler.NewJob(
		gocron.DurationJob(5*time.Second),
		gocron.NewTask(
			func(ctx context.Context) {
				// Check if context is cancelled
				select {
				case <-ctx.Done():
					return
				default:
				}

				d.log.WithFields(logrus.Fields{
					"records": d.status.ActiveConsensus(),
				}).Info("Consensus records currently trying to dial")

				if d.status.ActiveConsensus() == 0 {
					d.log.Info("No active consensus records to dial, requesting stale records from coordinator")

					nodeRecords, err := d.coordinator.ListStaleConsensusNodeRecords(ctx)
					if err != nil {
						d.log.WithError(err).Error("Failed to list stale consensus node records")

						return
					}

					d.log.WithField("records", len(nodeRecords)).Info("Adding stale consensus node records to status")

					d.status.AddConsensusNodeRecords(ctx, nodeRecords)
				}
			},
			ctx,
		),
		gocron.WithStartAt(gocron.WithStartImmediately()),
	); err != nil {
		return err
	}

	d.scheduler.Start()

	return nil
}

func (d *Discovery) handleExecutionStatus(ctx context.Context, status *xatu.ExecutionNodeStatus) error {
	d.metrics.AddNodeRecordStatus(1, fmt.Sprintf("%d", status.GetNetworkId()), "execution", fmt.Sprintf("0x%x", status.GetForkId().GetHash()))

	// Create and send status event for ClickHouse
	decoratedEvent, err := d.createExecutionStatusEvent(ctx, status)
	if err != nil {
		d.log.WithError(err).Error("Failed to create execution status event")
	} else {
		for _, sink := range d.sinks {
			if sinkErr := sink.HandleNewDecoratedEvent(ctx, decoratedEvent); sinkErr != nil {
				d.log.
					WithError(sinkErr).
					WithField("sink", sink.Type()).
					WithField("event_type", decoratedEvent.GetEvent().GetName()).
					Error("Failed to send event to sink")
			}
		}
	}

	return d.coordinator.HandleExecutionNodeRecordStatus(ctx, status)
}

func (d *Discovery) handleConsensusStatus(ctx context.Context, status *xatu.ConsensusNodeStatus) error {
	d.metrics.AddNodeRecordStatus(1, fmt.Sprintf("%d", status.GetNetworkId()), "consensus", fmt.Sprintf("0x%x", status.GetForkDigest()))

	// Create and send status event for ClickHouse
	decoratedEvent, err := d.createConsensusStatusEvent(ctx, status)
	if err != nil {
		d.log.WithError(err).Error("Failed to create consensus status event")
	} else {
		for _, sink := range d.sinks {
			if sinkErr := sink.HandleNewDecoratedEvent(ctx, decoratedEvent); sinkErr != nil {
				d.log.
					WithError(sinkErr).
					WithField("sink", sink.Type()).
					WithField("event_type", decoratedEvent.GetEvent().GetName()).
					Error("Failed to send event to sink")
			}
		}
	}

	return d.coordinator.HandleConsensusNodeRecordStatus(ctx, status)
}

func (d *Discovery) handleNewNodeRecord(ctx context.Context, node *enode.Node, source string) error {
	d.log.Debug("Node received")

	if node == nil {
		return errors.New("node is nil")
	}

	enr := node.String()

	item, retrieved := d.duplicateCache.GetCache().GetOrSet(enr, time.Now(), ttlcache.WithTTL[string, time.Time](ttlcache.DefaultTTL))
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

func (d *Discovery) createNewConsensusClientMeta(_ context.Context, networkID uint64) (*xatu.ClientMeta, error) {
	var (
		network     = networks.DeriveFromID(networkID)
		networkMeta = &xatu.ClientMeta_Ethereum_Network{
			Name: string(network.Name),
			Id:   network.ID,
		}
	)

	if d.Config.P2P.Ethereum.NetworkOverride != "" {
		networkMeta.Name = d.Config.P2P.Ethereum.NetworkOverride
	}

	return &xatu.ClientMeta{
		Name:           "xatu-discovery", // Fixed client name
		Version:        xatu.Short(),
		Id:             d.id.String(),
		Implementation: xatu.Implementation,
		ModuleName:     xatu.ModuleName_DISCOVERY,
		Os:             runtime.GOOS,
		Ethereum: &xatu.ClientMeta_Ethereum{
			Network: networkMeta,
		},
	}, nil
}

func (d *Discovery) createNewExecutionClientMeta(_ context.Context, networkID uint64) (*xatu.ClientMeta, error) {
	var (
		network     = networks.DeriveFromID(networkID)
		networkMeta = &xatu.ClientMeta_Ethereum_Network{
			Name: string(network.Name),
			Id:   network.ID,
		}
	)

	return &xatu.ClientMeta{
		Name:           "xatu-discovery", // Fixed client name
		Version:        xatu.Short(),
		Id:             d.id.String(),
		Implementation: xatu.Implementation,
		ModuleName:     xatu.ModuleName_DISCOVERY,
		Os:             runtime.GOOS,
		Ethereum: &xatu.ClientMeta_Ethereum{
			Network: networkMeta,
		},
	}, nil
}

func (d *Discovery) createExecutionStatusEvent(ctx context.Context, status *xatu.ExecutionNodeStatus) (*xatu.DecoratedEvent, error) {
	now := time.Now()
	eventID := uuid.New()

	meta, err := d.createNewExecutionClientMeta(ctx, status.GetNetworkId())
	if err != nil {
		return nil, err
	}

	// Convert ExecutionNodeStatus to noderecord.Execution
	capabilities := make([]string, 0, len(status.GetCapabilities()))
	for _, cap := range status.GetCapabilities() {
		capabilities = append(capabilities, fmt.Sprintf("%s/%d", cap.GetName(), cap.GetVersion()))
	}

	forkIDHash := fmt.Sprintf("0x%x", status.GetForkId().GetHash())
	forkIDNext := status.GetForkId().GetNext()

	executionData := &noderecord.Execution{
		Enr:             &wrapperspb.StringValue{Value: status.GetNodeRecord()},
		Timestamp:       timestamppb.New(now),
		Name:            &wrapperspb.StringValue{Value: status.GetName()},
		Capabilities:    &wrapperspb.StringValue{Value: strings.Join(capabilities, ",")},
		ProtocolVersion: &wrapperspb.StringValue{Value: fmt.Sprintf("%d", status.GetProtocolVersion())},
		TotalDifficulty: &wrapperspb.StringValue{Value: status.GetTotalDifficulty()},
		Head:            &wrapperspb.StringValue{Value: fmt.Sprintf("0x%x", status.GetHead())},
		Genesis:         &wrapperspb.StringValue{Value: fmt.Sprintf("0x%x", status.GetGenesis())},
		ForkIdHash:      &wrapperspb.StringValue{Value: forkIDHash},
		ForkIdNext:      &wrapperspb.StringValue{Value: fmt.Sprintf("%d", forkIDNext)},
	}

	decoratedEvent := &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_NODE_RECORD_EXECUTION,
			DateTime: timestamppb.New(now),
			Id:       eventID.String(),
		},
		Meta: &xatu.Meta{
			Client: meta,
		},
		Data: &xatu.DecoratedEvent_NodeRecordExecution{
			NodeRecordExecution: executionData,
		},
	}

	return decoratedEvent, nil
}

func (d *Discovery) createConsensusStatusEvent(ctx context.Context, status *xatu.ConsensusNodeStatus) (*xatu.DecoratedEvent, error) {
	now := time.Now()
	eventID := uuid.New()

	meta, err := d.createNewConsensusClientMeta(ctx, status.GetNetworkId())
	if err != nil {
		return nil, err
	}

	// Convert ConsensusNodeStatus to noderecord.Consensus
	consensusData := &noderecord.Consensus{
		Enr:            &wrapperspb.StringValue{Value: status.GetNodeRecord()},
		NodeId:         &wrapperspb.StringValue{Value: status.GetNodeId()},
		PeerId:         &wrapperspb.StringValue{Value: status.GetPeerId()},
		Timestamp:      &wrapperspb.Int64Value{Value: now.Unix()},
		Name:           &wrapperspb.StringValue{Value: status.GetName()},
		ForkDigest:     &wrapperspb.StringValue{Value: fmt.Sprintf("0x%x", status.GetForkDigest())},
		FinalizedRoot:  &wrapperspb.StringValue{Value: fmt.Sprintf("0x%x", status.GetFinalizedRoot())},
		FinalizedEpoch: &wrapperspb.UInt64Value{Value: uint64FromBytes(status.GetFinalizedEpoch())},
		HeadRoot:       &wrapperspb.StringValue{Value: fmt.Sprintf("0x%x", status.GetHeadRoot())},
		HeadSlot:       &wrapperspb.UInt64Value{Value: uint64FromBytes(status.GetHeadSlot())},
	}

	if status.GetCgc() != nil {
		consensusData.Cgc = &wrapperspb.StringValue{Value: fmt.Sprintf("0x%x", status.GetCgc())}
	}

	if status.GetNextForkDigest() != nil {
		consensusData.NextForkDigest = &wrapperspb.StringValue{Value: fmt.Sprintf("0x%x", status.GetNextForkDigest())}
	}

	decoratedEvent := &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_NODE_RECORD_CONSENSUS,
			DateTime: timestamppb.New(now),
			Id:       eventID.String(),
		},
		Meta: &xatu.Meta{
			Client: meta,
		},
		Data: &xatu.DecoratedEvent_NodeRecordConsensus{
			NodeRecordConsensus: consensusData,
		},
	}

	return decoratedEvent, nil
}

// Helper function to convert bytes to uint64.
func uint64FromBytes(b []byte) uint64 {
	if len(b) != 8 {
		return 0
	}

	return binary.BigEndian.Uint64(b)
}
