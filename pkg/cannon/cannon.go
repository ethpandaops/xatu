package cannon

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"syscall"
	"time"

	//nolint:gosec // only exposed if pprofAddr config is set
	_ "net/http/pprof"

	"github.com/beevik/ntp"
	"github.com/ethpandaops/ethwallclock"
	"github.com/ethpandaops/go-eth2-client/spec"
	"github.com/ethpandaops/go-eth2-client/spec/phase0"
	"github.com/go-co-op/gocron/v2"
	"github.com/google/uuid"
	perrors "github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/ethpandaops/xatu/pkg/cannon/coordinator"
	"github.com/ethpandaops/xatu/pkg/cannon/deriver"
	v1 "github.com/ethpandaops/xatu/pkg/cannon/deriver/beacon/eth/v1"
	v2 "github.com/ethpandaops/xatu/pkg/cannon/deriver/beacon/eth/v2"
	"github.com/ethpandaops/xatu/pkg/cannon/ethereum"
	"github.com/ethpandaops/xatu/pkg/cannon/iterator"
	"github.com/ethpandaops/xatu/pkg/observability"
	"github.com/ethpandaops/xatu/pkg/output"
	oxatu "github.com/ethpandaops/xatu/pkg/output/xatu"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

type Cannon struct {
	Config *Config

	sinks []output.Sink

	beacon *ethereum.BeaconNode

	clockDrift time.Duration

	log observability.ContextualLogger

	id uuid.UUID

	metrics *Metrics

	scheduler gocron.Scheduler

	eventDerivers []deriver.EventDeriver

	coordinatorClient *coordinator.Client

	shutdownFuncs []func(ctx context.Context) error

	overrides *Override
}

func New(ctx context.Context, log observability.ContextualLogger, config *Config, overrides *Override) (*Cannon, error) {
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

	beacon, err := ethereum.NewBeaconNode(ctx, config.Name, &config.Ethereum, log)
	if err != nil {
		return nil, err
	}

	coordinatorClient, err := coordinator.New(&config.Coordinator, log)
	if err != nil {
		return nil, err
	}

	scheduler, err := gocron.NewScheduler(gocron.WithLocation(time.Local))
	if err != nil {
		return nil, err
	}

	return &Cannon{
		Config:            config,
		sinks:             sinks,
		beacon:            beacon,
		clockDrift:        time.Duration(0),
		log:               log,
		id:                uuid.New(),
		metrics:           NewMetrics("xatu_cannon"),
		scheduler:         scheduler,
		eventDerivers:     nil, // Derivers are created once the beacon node is ready
		coordinatorClient: coordinatorClient,
		shutdownFuncs:     []func(ctx context.Context) error{},
		overrides:         overrides,
	}, nil
}

func (c *Cannon) ApplyOverrideBeforeStartAfterCreation(ctx context.Context) error {
	if c.overrides == nil {
		return nil
	}

	if c.overrides.XatuOutputAuth.Enabled {
		c.log.WithContext(ctx).Info("Overriding output authorization on xatu sinks")

		for _, sink := range c.sinks {
			if sink.Type() == string(output.SinkTypeXatu) {
				xatuSink, ok := sink.(*oxatu.Xatu)
				if !ok {
					return perrors.New("failed to assert xatu sink")
				}

				c.log.WithField("sink_name", sink.Name()).WithContext(ctx).Info("Overriding xatu output authorization")

				xatuSink.SetAuthorization(c.overrides.XatuOutputAuth.Value)
			}
		}
	}

	return nil
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
		WithField("id", c.id.String()).WithContext(ctx).
		Info("Starting Xatu in cannon mode 💣")

	if err := c.startCrons(ctx); err != nil {
		c.log.WithError(err).WithContext(ctx).Fatal("Failed to start crons")
	}

	for _, sink := range c.sinks {
		if err := sink.Start(ctx); err != nil {
			return err
		}
	}

	if err := c.ApplyOverrideBeforeStartAfterCreation(ctx); err != nil {
		return fmt.Errorf("failed to apply overrides before start: %w", err)
	}

	if c.Config.Ethereum.OverrideNetworkName != "" {
		c.log.WithField("network", c.Config.Ethereum.OverrideNetworkName).WithContext(ctx).Info("Overriding network name")
	}

	if err := c.beacon.Start(ctx); err != nil {
		return err
	}

	cancel := make(chan os.Signal, 1)
	signal.Notify(cancel, syscall.SIGTERM, syscall.SIGINT)

	sig := <-cancel
	c.log.WithContext(ctx).Printf("Caught signal: %v", sig)

	if err := c.Shutdown(ctx); err != nil {
		return err
	}

	return nil
}

func (c *Cannon) Shutdown(ctx context.Context) error {
	c.log.WithContext(ctx).Printf("Shutting down")

	for _, sink := range c.sinks {
		if err := sink.Stop(ctx); err != nil {
			return err
		}
	}

	for _, fun := range c.shutdownFuncs {
		if err := fun(ctx); err != nil {
			return err
		}
	}

	if err := c.scheduler.Shutdown(); err != nil {
		return err
	}

	for _, deriver := range c.eventDerivers {
		if err := deriver.Stop(ctx); err != nil {
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

		c.log.WithContext(ctx).Infof("Serving metrics at %s", c.Config.MetricsAddr)

		if err := server.ListenAndServe(); err != nil {
			c.log.WithContext(ctx).Fatal(err)
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
		c.log.WithContext(ctx).Infof("Serving pprof at %s", *c.Config.PProfAddr)

		if err := pprofServer.ListenAndServe(); err != nil {
			c.log.WithContext(ctx).Fatal(err)
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
		ModuleName:     xatu.ModuleName_CANNON,
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
	if _, err := c.scheduler.NewJob(
		gocron.DurationJob(5*time.Minute),
		gocron.NewTask(
			func(ctx context.Context) {
				if err := c.syncClockDrift(ctx); err != nil {
					c.log.WithError(err).WithContext(ctx).Error("Failed to sync clock drift")
				}
			},
			ctx,
		),
		gocron.WithStartAt(gocron.WithStartImmediately()),
	); err != nil {
		return err
	}

	c.scheduler.Start()

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
	c.log.WithField("drift", c.clockDrift).WithContext(ctx).Info("Updated clock drift")

	return err
}

func (c *Cannon) handleNewDecoratedEvents(ctx context.Context, events []*xatu.DecoratedEvent) error {
	if err := fanOutToSinks(ctx, c.sinks, events); err != nil {
		return err
	}

	for _, event := range events {
		c.metrics.AddDecoratedEvent(1, event, string(c.beacon.Metadata().Network.Name))
	}

	return nil
}

// fanOutToSinks dispatches a batch to every configured sink concurrently
// and waits for all of them to return before reporting the joined error
// (or nil). All sinks are attempted on every call regardless of whether
// any other sink failed — so a flaky sink does not prevent siblings from
// making progress on an attempt. The deriver still gates checkpoint
// advance on a nil return, so a single sink failure forces an epoch
// retry against every sink, with ReplacingMergeTree absorbing the
// resulting duplicate writes on the previously-succeeded paths.
//
// Goroutine cost is per-call rather than persistent: cannon's fan-out
// fires at most a few times per second per deriver in steady state, so
// per-call goroutines (~200 ns spawn) are cheaper overall than parking
// persistent workers on channels (~80 ns per send/recv plus 2 KB of
// resident stack each, even when idle).
func fanOutToSinks(ctx context.Context, sinks []output.Sink, events []*xatu.DecoratedEvent) error {
	if len(sinks) == 0 {
		return nil
	}

	if len(sinks) == 1 {
		// Skip the fan-out machinery entirely for the single-sink case so
		// the common deployment shape pays nothing for concurrency.
		if err := sinks[0].HandleNewDecoratedEvents(ctx, events); err != nil {
			return perrors.Wrapf(err, "failed to handle new decorated events in sink %s", sinks[0].Name())
		}

		return nil
	}

	var wg sync.WaitGroup

	errs := make([]error, len(sinks))

	for i, sink := range sinks {
		wg.Add(1)

		go func(i int, s output.Sink) {
			defer wg.Done()

			if err := s.HandleNewDecoratedEvents(ctx, events); err != nil {
				errs[i] = perrors.Wrapf(err, "failed to handle new decorated events in sink %s", s.Name())
			}
		}(i, sink)
	}

	wg.Wait()

	return errors.Join(errs...)
}

func (c *Cannon) startBeaconBlockProcessor(ctx context.Context) error {
	c.beacon.OnReady(ctx, func(ctx context.Context) error {
		c.log.WithContext(ctx).Info("Internal beacon node is ready, firing up event derivers")

		networkName := string(c.beacon.Metadata().Network.Name)
		networkID := fmt.Sprintf("%d", c.beacon.Metadata().Network.ID)

		wallclock := c.beacon.Metadata().Wallclock()

		clientMeta, err := c.createNewClientMeta(ctx)
		if err != nil {
			return err
		}

		backfillingCheckpointIteratorMetrics := iterator.NewBackfillingCheckpointMetrics("xatu_cannon")

		finalizedCheckpoint := "finalized"

		eventDerivers := []deriver.EventDeriver{
			v2.NewAttesterSlashingDeriver(
				c.log,
				&c.Config.Derivers.AttesterSlashingConfig,
				iterator.NewBackfillingCheckpoint(
					c.log,
					networkName,
					networkID,
					xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_ATTESTER_SLASHING,
					c.coordinatorClient,
					wallclock,
					&backfillingCheckpointIteratorMetrics,
					c.beacon,
					finalizedCheckpoint,
					3,
					&c.Config.Derivers.AttesterSlashingConfig.Iterator,
				),
				c.beacon,
				clientMeta,
			),
			v2.NewProposerSlashingDeriver(
				c.log,
				&c.Config.Derivers.ProposerSlashingConfig,
				iterator.NewBackfillingCheckpoint(
					c.log,
					networkName,
					networkID,
					xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_PROPOSER_SLASHING,
					c.coordinatorClient,
					wallclock,
					&backfillingCheckpointIteratorMetrics,
					c.beacon,
					finalizedCheckpoint,
					3,
					&c.Config.Derivers.ProposerSlashingConfig.Iterator,
				),
				c.beacon,
				clientMeta,
			),
			v2.NewVoluntaryExitDeriver(
				c.log,
				&c.Config.Derivers.VoluntaryExitConfig,
				iterator.NewBackfillingCheckpoint(
					c.log,
					networkName,
					networkID,
					xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_VOLUNTARY_EXIT,
					c.coordinatorClient,
					wallclock,
					&backfillingCheckpointIteratorMetrics,
					c.beacon,
					finalizedCheckpoint,
					3,
					&c.Config.Derivers.VoluntaryExitConfig.Iterator,
				),
				c.beacon,
				clientMeta,
			),
			v2.NewDepositDeriver(
				c.log,
				&c.Config.Derivers.DepositConfig,
				iterator.NewBackfillingCheckpoint(
					c.log,
					networkName,
					networkID,
					xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_DEPOSIT,
					c.coordinatorClient,
					wallclock,
					&backfillingCheckpointIteratorMetrics,
					c.beacon,
					finalizedCheckpoint,
					3,
					&c.Config.Derivers.DepositConfig.Iterator,
				),
				c.beacon,
				clientMeta,
			),
			v2.NewBLSToExecutionChangeDeriver(
				c.log,
				&c.Config.Derivers.BLSToExecutionConfig,
				iterator.NewBackfillingCheckpoint(
					c.log,
					networkName,
					networkID,
					xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_BLS_TO_EXECUTION_CHANGE,
					c.coordinatorClient,
					wallclock,
					&backfillingCheckpointIteratorMetrics,
					c.beacon,
					finalizedCheckpoint,
					3,
					&c.Config.Derivers.BLSToExecutionConfig.Iterator,
				),
				c.beacon,
				clientMeta,
			),
			v2.NewExecutionTransactionDeriver(
				c.log,
				&c.Config.Derivers.ExecutionTransactionConfig,
				iterator.NewBackfillingCheckpoint(
					c.log,
					networkName,
					networkID,
					xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_EXECUTION_TRANSACTION,
					c.coordinatorClient,
					wallclock,
					&backfillingCheckpointIteratorMetrics,
					c.beacon,
					finalizedCheckpoint,
					3,
					&c.Config.Derivers.ExecutionTransactionConfig.Iterator,
				),
				c.beacon,
				clientMeta,
			),
			v2.NewWithdrawalDeriver(
				c.log,
				&c.Config.Derivers.WithdrawalConfig,
				iterator.NewBackfillingCheckpoint(
					c.log,
					networkName,
					networkID,
					xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_WITHDRAWAL,
					c.coordinatorClient,
					wallclock,
					&backfillingCheckpointIteratorMetrics,
					c.beacon,
					finalizedCheckpoint,
					3,
					&c.Config.Derivers.WithdrawalConfig.Iterator,
				),
				c.beacon,
				clientMeta,
			),
			v2.NewBeaconBlockDeriver(
				c.log,
				&c.Config.Derivers.BeaconBlockConfig,
				iterator.NewBackfillingCheckpoint(
					c.log,
					networkName,
					networkID,
					xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK,
					c.coordinatorClient,
					wallclock,
					&backfillingCheckpointIteratorMetrics,
					c.beacon,
					finalizedCheckpoint,
					3,
					&c.Config.Derivers.BeaconBlockConfig.Iterator,
				),
				c.beacon,
				clientMeta,
			),
			v1.NewBeaconBlobDeriver(
				c.log,
				&c.Config.Derivers.BeaconBlobSidecarConfig,
				iterator.NewBackfillingCheckpoint(
					c.log,
					networkName,
					networkID,
					xatu.CannonType_BEACON_API_ETH_V1_BEACON_BLOB_SIDECAR,
					c.coordinatorClient,
					wallclock,
					&backfillingCheckpointIteratorMetrics,
					c.beacon,
					finalizedCheckpoint,
					3,
					&c.Config.Derivers.BeaconBlobSidecarConfig.Iterator,
				),
				c.beacon,
				clientMeta,
			),
			v1.NewProposerDutyDeriver(
				c.log,
				&c.Config.Derivers.ProposerDutyConfig,
				iterator.NewBackfillingCheckpoint(
					c.log,
					networkName,
					networkID,
					xatu.CannonType_BEACON_API_ETH_V1_PROPOSER_DUTY,
					c.coordinatorClient,
					wallclock,
					&backfillingCheckpointIteratorMetrics,
					c.beacon,
					finalizedCheckpoint,
					3,
					&c.Config.Derivers.ProposerDutyConfig.Iterator,
				),
				c.beacon,
				clientMeta,
			),
			v2.NewElaboratedAttestationDeriver(
				c.log,
				&c.Config.Derivers.ElaboratedAttestationConfig,
				iterator.NewBackfillingCheckpoint(
					c.log,
					networkName,
					networkID,
					xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_ELABORATED_ATTESTATION,
					c.coordinatorClient,
					wallclock,
					&backfillingCheckpointIteratorMetrics,
					c.beacon,
					finalizedCheckpoint,
					3,
					&c.Config.Derivers.ElaboratedAttestationConfig.Iterator,
				),
				c.beacon,
				clientMeta,
			),
			v1.NewBeaconValidatorsDeriver(
				c.log,
				&c.Config.Derivers.BeaconValidatorsConfig,
				iterator.NewBackfillingCheckpoint(
					c.log,
					networkName,
					networkID,
					xatu.CannonType_BEACON_API_ETH_V1_BEACON_VALIDATORS,
					c.coordinatorClient,
					wallclock,
					&backfillingCheckpointIteratorMetrics,
					c.beacon,
					finalizedCheckpoint,
					2,
					&c.Config.Derivers.BeaconValidatorsConfig.Iterator,
				),
				c.beacon,
				clientMeta,
			),
			v1.NewBeaconCommitteeDeriver(
				c.log,
				&c.Config.Derivers.BeaconCommitteeConfig,
				iterator.NewBackfillingCheckpoint(
					c.log,
					networkName,
					networkID,
					xatu.CannonType_BEACON_API_ETH_V1_BEACON_COMMITTEE,
					c.coordinatorClient,
					wallclock,
					&backfillingCheckpointIteratorMetrics,
					c.beacon,
					finalizedCheckpoint,
					2,
					&c.Config.Derivers.BeaconCommitteeConfig.Iterator,
				),
				c.beacon,
				clientMeta,
			),
			v1.NewBeaconSyncCommitteeDeriver(
				c.log,
				&c.Config.Derivers.BeaconSyncCommitteeConfig,
				iterator.NewBackfillingCheckpoint(
					c.log,
					networkName,
					networkID,
					xatu.CannonType_BEACON_API_ETH_V1_BEACON_SYNC_COMMITTEE,
					c.coordinatorClient,
					wallclock,
					&backfillingCheckpointIteratorMetrics,
					c.beacon,
					finalizedCheckpoint,
					3,
					&c.Config.Derivers.BeaconSyncCommitteeConfig.Iterator,
				),
				c.beacon,
				clientMeta,
			),
			v2.NewBeaconBlockSyncAggregateDeriver(
				c.log,
				&c.Config.Derivers.BeaconBlockSyncAggregateConfig,
				iterator.NewBackfillingCheckpoint(
					c.log,
					networkName,
					networkID,
					xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_SYNC_AGGREGATE,
					c.coordinatorClient,
					wallclock,
					&backfillingCheckpointIteratorMetrics,
					c.beacon,
					finalizedCheckpoint,
					3,
					&c.Config.Derivers.BeaconBlockSyncAggregateConfig.Iterator,
				),
				c.beacon,
				clientMeta,
			),
			v2.NewBlockAccessListDeriver(
				c.log,
				&c.Config.Derivers.BlockAccessListConfig,
				iterator.NewBackfillingCheckpoint(
					c.log,
					networkName,
					networkID,
					xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_ACCESS_LIST,
					c.coordinatorClient,
					wallclock,
					&backfillingCheckpointIteratorMetrics,
					c.beacon,
					finalizedCheckpoint,
					3,
					&c.Config.Derivers.BlockAccessListConfig.Iterator,
				),
				c.beacon,
				clientMeta,
			),
			v2.NewPayloadAttestationDeriver(
				c.log,
				&c.Config.Derivers.PayloadAttestationConfig,
				iterator.NewBackfillingCheckpoint(
					c.log,
					networkName,
					networkID,
					xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_PAYLOAD_ATTESTATION,
					c.coordinatorClient,
					wallclock,
					&backfillingCheckpointIteratorMetrics,
					c.beacon,
					finalizedCheckpoint,
					3,
					&c.Config.Derivers.PayloadAttestationConfig.Iterator,
				),
				c.beacon,
				clientMeta,
			),
			v2.NewExecutionPayloadBidDeriver(
				c.log,
				&c.Config.Derivers.ExecutionPayloadBidConfig,
				iterator.NewBackfillingCheckpoint(
					c.log,
					networkName,
					networkID,
					xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_EXECUTION_PAYLOAD_BID,
					c.coordinatorClient,
					wallclock,
					&backfillingCheckpointIteratorMetrics,
					c.beacon,
					finalizedCheckpoint,
					3,
					&c.Config.Derivers.ExecutionPayloadBidConfig.Iterator,
				),
				c.beacon,
				clientMeta,
			),
		}

		c.eventDerivers = eventDerivers

		// Refresh the spec every epoch
		c.beacon.Metadata().Wallclock().OnEpochChanged(func(current ethwallclock.Epoch) {
			_, err := c.beacon.Node().FetchSpec(ctx)
			if err != nil {
				c.log.WithError(err).WithContext(ctx).Error("Failed to refresh spec")
			}
		})

		for _, deriver := range c.eventDerivers {
			d := deriver

			d.OnEventsDerived(ctx, func(ctx context.Context, events []*xatu.DecoratedEvent) error {
				return c.handleNewDecoratedEvents(ctx, events)
			})

			go func() {
				if err := c.startDeriverWhenReady(ctx, d); err != nil {
					c.log.
						WithField("deriver", d.Name()).
						WithError(err).WithContext(ctx).Fatal("Failed to start deriver")
				}
			}()
		}

		return nil
	})

	return nil
}

func (c *Cannon) startDeriverWhenReady(ctx context.Context, d deriver.EventDeriver) error {
	for {
		// Handle derivers that require phase0, since its not actually a fork it'll never appear
		// in the spec.
		if d.ActivationFork() != spec.DataVersionPhase0 {
			spec, err := c.beacon.Node().Spec()
			if err != nil {
				c.log.WithError(err).WithContext(ctx).Error("Failed to get spec")

				time.Sleep(5 * time.Second)

				continue
			}

			fork, err := spec.ForkEpochs.GetByName(d.ActivationFork().String())
			if err != nil {
				c.log.WithError(err).WithContext(ctx).Errorf("unknown activation fork: %s", d.ActivationFork())

				epoch := c.beacon.Metadata().Wallclock().Epochs().Current()

				time.Sleep(time.Until(epoch.TimeWindow().End()))

				continue
			}

			currentEpoch := c.beacon.Metadata().Wallclock().Epochs().Current()

			if !fork.Active(phase0.Epoch(currentEpoch.Number())) {
				// Sleep until the next epochl and then retrty
				activationForkEpoch := c.beacon.Node().Wallclock().Epochs().FromNumber(uint64(fork.Epoch))

				sleepFor := time.Until(activationForkEpoch.TimeWindow().End())

				if activationForkEpoch.Number()-currentEpoch.Number() > 100000 {
					// If the fork epoch is over 100k epochs away we are most likely dealing with a
					// placeholder fork epoch. We should sleep until the end of the current fork epoch and then
					// wait for the spec to refresh. This gives the beacon node a chance to give us the real
					// fork epoch once its scheduled.
					sleepFor = time.Until(currentEpoch.TimeWindow().End())
				}

				c.log.
					WithField("current_epoch", currentEpoch.Number()).
					WithField("activation_fork_name", d.ActivationFork()).
					WithField("activation_fork_epoch", fork.Epoch).
					WithField("estimated_time_until_fork", time.Until(
						activationForkEpoch.TimeWindow().Start(),
					)).
					WithField("check_again_in", sleepFor).WithContext(ctx).
					Warn("Deriver required fork is not active yet")

				time.Sleep(sleepFor)

				continue
			}
		}

		c.log.
			WithField("deriver", d.Name()).WithContext(ctx).
			Info("Starting cannon event deriver")

		return d.Start(ctx)
	}
}
