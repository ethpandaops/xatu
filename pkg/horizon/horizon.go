package horizon

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

	"github.com/attestantio/go-eth2-client/spec"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	cldataderiver "github.com/ethpandaops/xatu/pkg/cldata/deriver"
	"github.com/ethpandaops/xatu/pkg/horizon/cache"
	"github.com/ethpandaops/xatu/pkg/horizon/coordinator"
	"github.com/ethpandaops/xatu/pkg/horizon/deriver"
	"github.com/ethpandaops/xatu/pkg/horizon/ethereum"
	"github.com/ethpandaops/xatu/pkg/horizon/iterator"
	"github.com/ethpandaops/xatu/pkg/horizon/subscription"
	"github.com/ethpandaops/xatu/pkg/observability"
	"github.com/ethpandaops/xatu/pkg/output"
	oxatu "github.com/ethpandaops/xatu/pkg/output/xatu"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/google/uuid"
	perrors "github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/sdk/trace"
)

type Horizon struct {
	Config *Config

	sinks []output.Sink

	log logrus.FieldLogger

	id uuid.UUID

	metrics *Metrics

	// Beacon node pool for connecting to multiple beacon nodes.
	beaconPool *ethereum.BeaconNodePool

	// Coordinator client for tracking locations.
	coordinatorClient *coordinator.Client

	// Deduplication cache for block events.
	dedupCache *cache.DedupCache

	// Broadcaster for deduplicated block events.
	blockBroadcaster *BlockEventBroadcaster

	// Block subscriptions from beacon nodes.
	blockSubscription *subscription.BlockSubscription

	// Reorg subscription for chain reorg events.
	reorgSubscription *subscription.ReorgSubscription

	// Reorg tracker for tagging derived events.
	reorgTracker *ReorgTracker

	// Event derivers for processing block data.
	eventDerivers []cldataderiver.EventDeriver

	// Dual iterators for coordinated HEAD/FILL processing.
	dualIterators []*iterator.DualIterator

	shutdownFuncs []func(ctx context.Context) error

	overrides *Override
}

func New(ctx context.Context, log logrus.FieldLogger, config *Config, overrides *Override) (*Horizon, error) {
	if config == nil {
		return nil, errors.New("config is required")
	}

	if overrides != nil {
		if err := config.ApplyOverrides(overrides, log); err != nil {
			return nil, fmt.Errorf("failed to apply overrides: %w", err)
		}
	}

	if err := config.Validate(); err != nil {
		return nil, err
	}

	sinks, err := config.CreateSinks(log)
	if err != nil {
		return nil, err
	}

	// Create beacon node pool.
	beaconPool, err := ethereum.NewBeaconNodePool(ctx, &config.Ethereum, log)
	if err != nil {
		return nil, fmt.Errorf("failed to create beacon node pool: %w", err)
	}

	// Create coordinator client.
	coordinatorClient, err := coordinator.New(&config.Coordinator, log)
	if err != nil {
		return nil, fmt.Errorf("failed to create coordinator client: %w", err)
	}

	// Create deduplication cache.
	dedupCache := cache.New(&config.DedupCache, "xatu_horizon")
	reorgTracker := NewReorgTracker(config.DedupCache.TTL)

	return &Horizon{
		Config:            config,
		sinks:             sinks,
		log:               log,
		id:                uuid.New(),
		metrics:           NewMetrics("xatu_horizon"),
		beaconPool:        beaconPool,
		coordinatorClient: coordinatorClient,
		dedupCache:        dedupCache,
		reorgTracker:      reorgTracker,
		eventDerivers:     nil, // Derivers are created once the beacon pool is ready.
		shutdownFuncs:     make([]func(ctx context.Context) error, 0),
		overrides:         overrides,
	}, nil
}

func (h *Horizon) Start(ctx context.Context) error {
	// Start tracing if enabled.
	if h.Config.Tracing.Enabled {
		h.log.Info("Tracing enabled")

		res, err := observability.NewResource(xatu.WithModule(xatu.ModuleName_HORIZON), xatu.Short())
		if err != nil {
			return perrors.Wrap(err, "failed to create tracing resource")
		}

		opts := []trace.TracerProviderOption{
			trace.WithSampler(trace.ParentBased(trace.TraceIDRatioBased(h.Config.Tracing.Sampling.Rate))),
		}

		tracer, err := observability.NewHTTPTraceProvider(ctx,
			res,
			h.Config.Tracing.AsOTelOpts(),
			opts...,
		)
		if err != nil {
			return perrors.Wrap(err, "failed to create tracing provider")
		}

		shutdown, err := observability.SetupOTelSDK(ctx, tracer)
		if err != nil {
			return perrors.Wrap(err, "failed to setup tracing SDK")
		}

		h.shutdownFuncs = append(h.shutdownFuncs, shutdown)
	}

	if err := h.ServeMetrics(ctx); err != nil {
		return err
	}

	if h.Config.PProfAddr != nil {
		if err := h.ServePProf(ctx); err != nil {
			return err
		}
	}

	h.log.
		WithField("version", xatu.Full()).
		WithField("id", h.id.String()).
		Info("Starting Xatu in horizon mode 🌅")

	// Start sinks.
	for _, sink := range h.sinks {
		if err := sink.Start(ctx); err != nil {
			return err
		}
	}

	if err := h.ApplyOverrideBeforeStartAfterCreation(ctx); err != nil {
		return fmt.Errorf("failed to apply overrides before start: %w", err)
	}

	// Start dedup cache.
	go h.dedupCache.Start()

	// Register on-ready callback for beacon pool.
	h.beaconPool.OnReady(func(ctx context.Context) error {
		return h.onBeaconPoolReady(ctx)
	})

	// Start beacon pool (will call onBeaconPoolReady when healthy).
	if err := h.beaconPool.Start(ctx); err != nil {
		return fmt.Errorf("failed to start beacon pool: %w", err)
	}

	cancel := make(chan os.Signal, 1)
	signal.Notify(cancel, syscall.SIGTERM, syscall.SIGINT)

	sig := <-cancel
	h.log.Printf("Caught signal: %v", sig)

	if err := h.Shutdown(ctx); err != nil {
		return err
	}

	return nil
}

// onBeaconPoolReady is called when the beacon pool has at least one healthy node.
// It initializes and starts all the event derivers.
func (h *Horizon) onBeaconPoolReady(ctx context.Context) error {
	h.log.Info("Beacon pool ready, initializing event derivers")

	metadata := h.beaconPool.Metadata()
	networkName := string(metadata.Network.Name)
	networkID := fmt.Sprintf("%d", metadata.Network.ID)
	wallclock := metadata.Wallclock()
	depositChainID := metadata.Spec.DepositChainID

	// Create block subscription for SSE events.
	h.blockSubscription = subscription.NewBlockSubscription(
		h.log,
		h.beaconPool,
		&h.Config.Subscription,
	)

	// Start block subscription.
	if err := h.blockSubscription.Start(ctx); err != nil {
		return fmt.Errorf("failed to start block subscription: %w", err)
	}

	// Start block broadcaster to deduplicate and fan-out events.
	h.blockBroadcaster = NewBlockEventBroadcaster(
		h.log,
		h.dedupCache,
		h.blockSubscription.Events(),
		h.Config.Subscription.BufferSize,
	)
	h.blockBroadcaster.Start(ctx)

	// Create and start reorg subscription for chain reorg handling.
	h.reorgSubscription = subscription.NewReorgSubscription(
		h.log,
		h.beaconPool,
		&h.Config.Reorg,
	)

	if err := h.reorgSubscription.Start(ctx); err != nil {
		return fmt.Errorf("failed to start reorg subscription: %w", err)
	}

	// Start goroutine to handle reorg events.
	go h.handleReorgEvents(ctx)

	// Create context provider adapter for all derivers.
	ctxProvider := deriver.NewContextProviderAdapter(
		h.id,
		h.Config.Name,
		networkName,
		metadata.Network.ID,
		wallclock,
		depositChainID,
		h.Config.Labels,
	)

	// Create beacon client adapter.
	beaconClient := deriver.NewBeaconClientAdapter(h.beaconPool)

	// Create HEAD iterators for each deriver type.
	// Each deriver gets its own HEAD iterator instance that tracks its progress.
	eventDerivers := []cldataderiver.EventDeriver{
		// BeaconBlockDeriver.
		cldataderiver.NewBeaconBlockDeriver(
			h.log,
			&cldataderiver.BeaconBlockDeriverConfig{Enabled: h.Config.Derivers.BeaconBlockConfig.Enabled},
			h.createDualIterator(xatu.HorizonType_HORIZON_TYPE_BEACON_API_ETH_V2_BEACON_BLOCK, networkID, networkName),
			beaconClient,
			ctxProvider,
		),
		// AttesterSlashingDeriver.
		cldataderiver.NewAttesterSlashingDeriver(
			h.log,
			&cldataderiver.AttesterSlashingDeriverConfig{Enabled: h.Config.Derivers.AttesterSlashingConfig.Enabled},
			h.createDualIterator(xatu.HorizonType_HORIZON_TYPE_BEACON_API_ETH_V2_BEACON_BLOCK_ATTESTER_SLASHING, networkID, networkName),
			beaconClient,
			ctxProvider,
		),
		// ProposerSlashingDeriver.
		cldataderiver.NewProposerSlashingDeriver(
			h.log,
			&cldataderiver.ProposerSlashingDeriverConfig{Enabled: h.Config.Derivers.ProposerSlashingConfig.Enabled},
			h.createDualIterator(xatu.HorizonType_HORIZON_TYPE_BEACON_API_ETH_V2_BEACON_BLOCK_PROPOSER_SLASHING, networkID, networkName),
			beaconClient,
			ctxProvider,
		),
		// DepositDeriver.
		cldataderiver.NewDepositDeriver(
			h.log,
			&cldataderiver.DepositDeriverConfig{Enabled: h.Config.Derivers.DepositConfig.Enabled},
			h.createDualIterator(xatu.HorizonType_HORIZON_TYPE_BEACON_API_ETH_V2_BEACON_BLOCK_DEPOSIT, networkID, networkName),
			beaconClient,
			ctxProvider,
		),
		// WithdrawalDeriver.
		cldataderiver.NewWithdrawalDeriver(
			h.log,
			&cldataderiver.WithdrawalDeriverConfig{Enabled: h.Config.Derivers.WithdrawalConfig.Enabled},
			h.createDualIterator(xatu.HorizonType_HORIZON_TYPE_BEACON_API_ETH_V2_BEACON_BLOCK_WITHDRAWAL, networkID, networkName),
			beaconClient,
			ctxProvider,
		),
		// VoluntaryExitDeriver.
		cldataderiver.NewVoluntaryExitDeriver(
			h.log,
			&cldataderiver.VoluntaryExitDeriverConfig{Enabled: h.Config.Derivers.VoluntaryExitConfig.Enabled},
			h.createDualIterator(xatu.HorizonType_HORIZON_TYPE_BEACON_API_ETH_V2_BEACON_BLOCK_VOLUNTARY_EXIT, networkID, networkName),
			beaconClient,
			ctxProvider,
		),
		// BLSToExecutionChangeDeriver.
		cldataderiver.NewBLSToExecutionChangeDeriver(
			h.log,
			&cldataderiver.BLSToExecutionChangeDeriverConfig{Enabled: h.Config.Derivers.BLSToExecutionChangeConfig.Enabled},
			h.createDualIterator(xatu.HorizonType_HORIZON_TYPE_BEACON_API_ETH_V2_BEACON_BLOCK_BLS_TO_EXECUTION_CHANGE, networkID, networkName),
			beaconClient,
			ctxProvider,
		),
		// ExecutionTransactionDeriver.
		cldataderiver.NewExecutionTransactionDeriver(
			h.log,
			&cldataderiver.ExecutionTransactionDeriverConfig{Enabled: h.Config.Derivers.ExecutionTransactionConfig.Enabled},
			h.createDualIterator(xatu.HorizonType_HORIZON_TYPE_BEACON_API_ETH_V2_BEACON_BLOCK_EXECUTION_TRANSACTION, networkID, networkName),
			beaconClient,
			ctxProvider,
		),
		// ElaboratedAttestationDeriver.
		cldataderiver.NewElaboratedAttestationDeriver(
			h.log,
			&cldataderiver.ElaboratedAttestationDeriverConfig{Enabled: h.Config.Derivers.ElaboratedAttestationConfig.Enabled},
			h.createDualIterator(xatu.HorizonType_HORIZON_TYPE_BEACON_API_ETH_V2_BEACON_BLOCK_ELABORATED_ATTESTATION, networkID, networkName),
			beaconClient,
			ctxProvider,
		),

		// --- Epoch-based derivers (triggered midway through epoch) ---

		// ProposerDutyDeriver.
		cldataderiver.NewProposerDutyDeriver(
			h.log,
			&cldataderiver.ProposerDutyDeriverConfig{Enabled: h.Config.Derivers.ProposerDutyConfig.Enabled},
			h.createEpochIterator(xatu.HorizonType_HORIZON_TYPE_BEACON_API_ETH_V1_PROPOSER_DUTY, networkID, networkName),
			beaconClient,
			ctxProvider,
		),
		// BeaconBlobDeriver.
		cldataderiver.NewBeaconBlobDeriver(
			h.log,
			&cldataderiver.BeaconBlobDeriverConfig{Enabled: h.Config.Derivers.BeaconBlobConfig.Enabled},
			h.createEpochIterator(xatu.HorizonType_HORIZON_TYPE_BEACON_API_ETH_V1_BEACON_BLOB_SIDECAR, networkID, networkName),
			beaconClient,
			ctxProvider,
		),
		// BeaconValidatorsDeriver.
		cldataderiver.NewBeaconValidatorsDeriver(
			h.log,
			&cldataderiver.BeaconValidatorsDeriverConfig{
				Enabled:   h.Config.Derivers.BeaconValidatorsConfig.Enabled,
				ChunkSize: h.Config.Derivers.BeaconValidatorsConfig.ChunkSize,
			},
			h.createEpochIterator(xatu.HorizonType_HORIZON_TYPE_BEACON_API_ETH_V1_BEACON_VALIDATORS, networkID, networkName),
			beaconClient,
			ctxProvider,
		),
		// BeaconCommitteeDeriver.
		cldataderiver.NewBeaconCommitteeDeriver(
			h.log,
			&cldataderiver.BeaconCommitteeDeriverConfig{Enabled: h.Config.Derivers.BeaconCommitteeConfig.Enabled},
			h.createEpochIterator(xatu.HorizonType_HORIZON_TYPE_BEACON_API_ETH_V1_BEACON_COMMITTEE, networkID, networkName),
			beaconClient,
			ctxProvider,
		),
	}

	h.eventDerivers = eventDerivers

	// Start each deriver.
	for _, d := range h.eventDerivers {
		// Register callback for derived events.
		d.OnEventsDerived(ctx, func(ctx context.Context, events []*xatu.DecoratedEvent) error {
			return h.handleNewDecoratedEvents(ctx, events)
		})

		// Start deriver in goroutine.
		go func() {
			if err := h.startDeriverWhenReady(ctx, d); err != nil {
				h.log.
					WithField("deriver", d.Name()).
					WithError(err).Fatal("Failed to start deriver")
			}
		}()
	}

	return nil
}

// createHeadIterator creates a HEAD iterator for a specific deriver type.
func (h *Horizon) createHeadIterator(
	horizonType xatu.HorizonType,
	networkID string,
	networkName string,
	blockEvents <-chan subscription.BlockEvent,
) *iterator.HeadIterator {
	return iterator.NewHeadIterator(
		h.log,
		h.beaconPool,
		h.coordinatorClient,
		horizonType,
		networkID,
		networkName,
		blockEvents,
	)
}

// createFillIterator creates a FILL iterator for a specific deriver type.
func (h *Horizon) createFillIterator(
	horizonType xatu.HorizonType,
	networkID string,
	networkName string,
) *iterator.FillIterator {
	return iterator.NewFillIterator(
		h.log,
		h.beaconPool,
		h.coordinatorClient,
		&h.Config.Iterators.Fill,
		horizonType,
		networkID,
		networkName,
	)
}

// createDualIterator creates a dual iterator that multiplexes HEAD and FILL.
func (h *Horizon) createDualIterator(
	horizonType xatu.HorizonType,
	networkID string,
	networkName string,
) *iterator.DualIterator {
	head := h.createHeadIterator(horizonType, networkID, networkName, h.blockBroadcaster.Subscribe())
	fill := h.createFillIterator(horizonType, networkID, networkName)
	dual := iterator.NewDualIterator(h.log, &h.Config.Iterators, head, fill)

	h.dualIterators = append(h.dualIterators, dual)

	return dual
}

// createEpochIterator creates an Epoch iterator for a specific deriver type.
func (h *Horizon) createEpochIterator(
	horizonType xatu.HorizonType,
	networkID string,
	networkName string,
) *iterator.EpochIterator {
	return iterator.NewEpochIterator(
		h.log,
		h.beaconPool,
		h.coordinatorClient,
		h.Config.EpochIterator,
		horizonType,
		networkID,
		networkName,
	)
}

// startDeriverWhenReady waits for the deriver's activation fork and then starts it.
func (h *Horizon) startDeriverWhenReady(ctx context.Context, d cldataderiver.EventDeriver) error {
	for {
		// Handle derivers that require phase0 - since it's not actually a fork, it'll never appear in the spec.
		if d.ActivationFork() != spec.DataVersionPhase0 {
			fork, err := h.beaconPool.Metadata().Spec.ForkEpochs.GetByName(d.ActivationFork().String())
			if err != nil {
				h.log.WithError(err).Errorf("unknown activation fork: %s", d.ActivationFork())

				epoch := h.beaconPool.Metadata().Wallclock().Epochs().Current()

				time.Sleep(time.Until(epoch.TimeWindow().End()))

				continue
			}

			currentEpoch := h.beaconPool.Metadata().Wallclock().Epochs().Current()

			if !fork.Active(phase0.Epoch(currentEpoch.Number())) {
				activationForkEpoch := h.beaconPool.Metadata().Wallclock().Epochs().FromNumber(uint64(fork.Epoch))

				sleepFor := time.Until(activationForkEpoch.TimeWindow().End())

				if activationForkEpoch.Number()-currentEpoch.Number() > 100000 {
					// If the fork epoch is over 100k epochs away, we are most likely dealing with a
					// placeholder fork epoch. Sleep until the end of the current fork epoch and then
					// wait for the spec to refresh.
					sleepFor = time.Until(currentEpoch.TimeWindow().End())
				}

				h.log.
					WithField("current_epoch", currentEpoch.Number()).
					WithField("activation_fork_name", d.ActivationFork()).
					WithField("activation_fork_epoch", fork.Epoch).
					WithField("estimated_time_until_fork", time.Until(activationForkEpoch.TimeWindow().Start())).
					WithField("check_again_in", sleepFor).
					Warn("Deriver required fork is not active yet")

				time.Sleep(sleepFor)

				continue
			}
		}

		h.log.
			WithField("deriver", d.Name()).
			Info("Starting horizon event deriver")

		return d.Start(ctx)
	}
}

// handleNewDecoratedEvents sends derived events to all configured sinks.
func (h *Horizon) handleNewDecoratedEvents(ctx context.Context, events []*xatu.DecoratedEvent) error {
	h.markReorgMetadata(events)

	for _, sink := range h.sinks {
		if err := sink.HandleNewDecoratedEvents(ctx, events); err != nil {
			return perrors.Wrapf(err, "failed to handle new decorated events in sink %s", sink.Name())
		}
	}

	networkName := string(h.beaconPool.Metadata().Network.Name)

	for _, event := range events {
		h.metrics.AddDecoratedEvent(1, event, networkName)
	}

	return nil
}

// handleReorgEvents handles chain reorg events by clearing affected block roots from the dedup cache.
// This allows the affected slots to be re-processed with the new canonical blocks.
func (h *Horizon) handleReorgEvents(ctx context.Context) {
	if h.reorgSubscription == nil || !h.reorgSubscription.Enabled() {
		return
	}

	log := h.log.WithField("component", "reorg_handler")
	log.Info("Starting reorg event handler")

	for {
		select {
		case <-ctx.Done():
			log.Info("Reorg event handler stopped (context cancelled)")

			return
		case event, ok := <-h.reorgSubscription.Events():
			if !ok {
				log.Info("Reorg event handler stopped (channel closed)")

				return
			}

			log.WithFields(logrus.Fields{
				"slot":           event.Slot,
				"depth":          event.Depth,
				"old_head_block": event.OldHeadBlock.String(),
				"new_head_block": event.NewHeadBlock.String(),
				"epoch":          event.Epoch,
				"node":           event.NodeName,
			}).Info("Processing chain reorg event")

			start, end := reorgSlotRange(event)
			if h.reorgTracker != nil {
				h.reorgTracker.AddRange(start, end)
			}

			h.rollbackReorgLocations(ctx, start)

			// Clear the old head block from dedup cache so the new canonical block can be processed.
			// The old head block root needs to be removed so that if we receive the new canonical
			// block for the same slot, it won't be deduplicated.
			h.dedupCache.Delete(event.OldHeadBlock.String())

			log.WithFields(logrus.Fields{
				"slot":          event.Slot,
				"depth":         event.Depth,
				"cleared_block": event.OldHeadBlock.String(),
				"new_canonical": event.NewHeadBlock.String(),
			}).Info("Cleared reorged block from dedup cache - slot can be re-processed")
		}
	}
}

func (h *Horizon) Shutdown(ctx context.Context) error {
	h.log.Printf("Shutting down")

	// Stop event derivers.
	for _, d := range h.eventDerivers {
		if err := d.Stop(ctx); err != nil {
			h.log.WithError(err).WithField("deriver", d.Name()).Warn("Error stopping deriver")
		}
	}

	// Stop dual iterators.
	for _, dual := range h.dualIterators {
		if err := dual.Stop(ctx); err != nil {
			h.log.WithError(err).Warn("Error stopping dual iterator")
		}
	}

	// Stop block broadcaster.
	if h.blockBroadcaster != nil {
		h.blockBroadcaster.Stop()
	}

	// Stop block subscription.
	if h.blockSubscription != nil {
		if err := h.blockSubscription.Stop(ctx); err != nil {
			h.log.WithError(err).Warn("Error stopping block subscription")
		}
	}

	// Stop reorg subscription.
	if h.reorgSubscription != nil {
		if err := h.reorgSubscription.Stop(ctx); err != nil {
			h.log.WithError(err).Warn("Error stopping reorg subscription")
		}
	}

	// Stop dedup cache.
	h.dedupCache.Stop()

	// Stop beacon pool.
	if err := h.beaconPool.Stop(ctx); err != nil {
		h.log.WithError(err).Warn("Error stopping beacon pool")
	}

	// Stop sinks.
	for _, sink := range h.sinks {
		if err := sink.Stop(ctx); err != nil {
			return err
		}
	}

	// Run shutdown functions.
	for _, fun := range h.shutdownFuncs {
		if err := fun(ctx); err != nil {
			return err
		}
	}

	return nil
}

func (h *Horizon) ApplyOverrideBeforeStartAfterCreation(ctx context.Context) error {
	if h.overrides == nil {
		return nil
	}

	if h.overrides.XatuOutputAuth.Enabled {
		h.log.Info("Overriding output authorization on xatu sinks")

		for _, sink := range h.sinks {
			if sink.Type() == string(output.SinkTypeXatu) {
				xatuSink, ok := sink.(*oxatu.Xatu)
				if !ok {
					return perrors.New("failed to assert xatu sink")
				}

				h.log.WithField("sink_name", sink.Name()).Info("Overriding xatu output authorization")

				xatuSink.SetAuthorization(h.overrides.XatuOutputAuth.Value)
			}
		}
	}

	return nil
}

func (h *Horizon) ServeMetrics(_ context.Context) error {
	go func() {
		sm := http.NewServeMux()
		sm.Handle("/metrics", promhttp.Handler())

		server := &http.Server{
			Addr:              h.Config.MetricsAddr,
			ReadHeaderTimeout: 15 * time.Second,
			Handler:           sm,
		}

		h.log.Infof("Serving metrics at %s", h.Config.MetricsAddr)

		if err := server.ListenAndServe(); err != nil {
			h.log.Fatal(err)
		}
	}()

	return nil
}

func (h *Horizon) ServePProf(_ context.Context) error {
	pprofServer := &http.Server{
		Addr:              *h.Config.PProfAddr,
		ReadHeaderTimeout: 120 * time.Second,
	}

	go func() {
		h.log.Infof("Serving pprof at %s", *h.Config.PProfAddr)

		if err := pprofServer.ListenAndServe(); err != nil {
			h.log.Fatal(err)
		}
	}()

	return nil
}
