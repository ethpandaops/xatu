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

	// Block subscriptions from beacon nodes.
	blockSubscription *subscription.BlockSubscription

	// Event derivers for processing block data.
	eventDerivers []cldataderiver.EventDeriver

	shutdownFuncs []func(ctx context.Context) error

	overrides *Override
}

func New(ctx context.Context, log logrus.FieldLogger, config *Config, overrides *Override) (*Horizon, error) {
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

	return &Horizon{
		Config:            config,
		sinks:             sinks,
		log:               log,
		id:                uuid.New(),
		metrics:           NewMetrics("xatu_horizon"),
		beaconPool:        beaconPool,
		coordinatorClient: coordinatorClient,
		dedupCache:        dedupCache,
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

	// Get the block events channel from the subscription.
	blockEventsChan := h.blockSubscription.Events()

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
			h.createHeadIterator(xatu.HorizonType_HORIZON_TYPE_BEACON_API_ETH_V2_BEACON_BLOCK, networkID, networkName, blockEventsChan),
			beaconClient,
			ctxProvider,
		),
		// AttesterSlashingDeriver.
		cldataderiver.NewAttesterSlashingDeriver(
			h.log,
			&cldataderiver.AttesterSlashingDeriverConfig{Enabled: h.Config.Derivers.AttesterSlashingConfig.Enabled},
			h.createHeadIterator(xatu.HorizonType_HORIZON_TYPE_BEACON_API_ETH_V2_BEACON_BLOCK_ATTESTER_SLASHING, networkID, networkName, blockEventsChan),
			beaconClient,
			ctxProvider,
		),
		// ProposerSlashingDeriver.
		cldataderiver.NewProposerSlashingDeriver(
			h.log,
			&cldataderiver.ProposerSlashingDeriverConfig{Enabled: h.Config.Derivers.ProposerSlashingConfig.Enabled},
			h.createHeadIterator(xatu.HorizonType_HORIZON_TYPE_BEACON_API_ETH_V2_BEACON_BLOCK_PROPOSER_SLASHING, networkID, networkName, blockEventsChan),
			beaconClient,
			ctxProvider,
		),
		// DepositDeriver.
		cldataderiver.NewDepositDeriver(
			h.log,
			&cldataderiver.DepositDeriverConfig{Enabled: h.Config.Derivers.DepositConfig.Enabled},
			h.createHeadIterator(xatu.HorizonType_HORIZON_TYPE_BEACON_API_ETH_V2_BEACON_BLOCK_DEPOSIT, networkID, networkName, blockEventsChan),
			beaconClient,
			ctxProvider,
		),
		// WithdrawalDeriver.
		cldataderiver.NewWithdrawalDeriver(
			h.log,
			&cldataderiver.WithdrawalDeriverConfig{Enabled: h.Config.Derivers.WithdrawalConfig.Enabled},
			h.createHeadIterator(xatu.HorizonType_HORIZON_TYPE_BEACON_API_ETH_V2_BEACON_BLOCK_WITHDRAWAL, networkID, networkName, blockEventsChan),
			beaconClient,
			ctxProvider,
		),
		// VoluntaryExitDeriver.
		cldataderiver.NewVoluntaryExitDeriver(
			h.log,
			&cldataderiver.VoluntaryExitDeriverConfig{Enabled: h.Config.Derivers.VoluntaryExitConfig.Enabled},
			h.createHeadIterator(xatu.HorizonType_HORIZON_TYPE_BEACON_API_ETH_V2_BEACON_BLOCK_VOLUNTARY_EXIT, networkID, networkName, blockEventsChan),
			beaconClient,
			ctxProvider,
		),
		// BLSToExecutionChangeDeriver.
		cldataderiver.NewBLSToExecutionChangeDeriver(
			h.log,
			&cldataderiver.BLSToExecutionChangeDeriverConfig{Enabled: h.Config.Derivers.BLSToExecutionChangeConfig.Enabled},
			h.createHeadIterator(xatu.HorizonType_HORIZON_TYPE_BEACON_API_ETH_V2_BEACON_BLOCK_BLS_TO_EXECUTION_CHANGE, networkID, networkName, blockEventsChan),
			beaconClient,
			ctxProvider,
		),
		// ExecutionTransactionDeriver.
		cldataderiver.NewExecutionTransactionDeriver(
			h.log,
			&cldataderiver.ExecutionTransactionDeriverConfig{Enabled: h.Config.Derivers.ExecutionTransactionConfig.Enabled},
			h.createHeadIterator(xatu.HorizonType_HORIZON_TYPE_BEACON_API_ETH_V2_BEACON_BLOCK_EXECUTION_TRANSACTION, networkID, networkName, blockEventsChan),
			beaconClient,
			ctxProvider,
		),
		// ElaboratedAttestationDeriver.
		cldataderiver.NewElaboratedAttestationDeriver(
			h.log,
			&cldataderiver.ElaboratedAttestationDeriverConfig{Enabled: h.Config.Derivers.ElaboratedAttestationConfig.Enabled},
			h.createHeadIterator(xatu.HorizonType_HORIZON_TYPE_BEACON_API_ETH_V2_BEACON_BLOCK_ELABORATED_ATTESTATION, networkID, networkName, blockEventsChan),
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
		h.dedupCache,
		horizonType,
		networkID,
		networkName,
		blockEvents,
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

func (h *Horizon) Shutdown(ctx context.Context) error {
	h.log.Printf("Shutting down")

	// Stop event derivers.
	for _, d := range h.eventDerivers {
		if err := d.Stop(ctx); err != nil {
			h.log.WithError(err).WithField("deriver", d.Name()).Warn("Error stopping deriver")
		}
	}

	// Stop block subscription.
	if h.blockSubscription != nil {
		if err := h.blockSubscription.Stop(ctx); err != nil {
			h.log.WithError(err).Warn("Error stopping block subscription")
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
