package sentry

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	//nolint:gosec // only exposed if pprofAddr config is set
	_ "net/http/pprof"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"syscall"
	"time"

	"github.com/attestantio/go-eth2-client/api"
	eth2v1 "github.com/attestantio/go-eth2-client/api/v1"
	"github.com/attestantio/go-eth2-client/spec"
	"github.com/attestantio/go-eth2-client/spec/altair"
	"github.com/attestantio/go-eth2-client/spec/deneb"
	"github.com/attestantio/go-eth2-client/spec/electra"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/beevik/ntp"
	"github.com/ethpandaops/beacon/pkg/beacon"
	"github.com/ethpandaops/xatu/pkg/networks"
	"github.com/ethpandaops/xatu/pkg/observability"
	"github.com/ethpandaops/xatu/pkg/output"
	oxatu "github.com/ethpandaops/xatu/pkg/output/xatu"
	xatuethv1 "github.com/ethpandaops/xatu/pkg/proto/eth/v1"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/ethpandaops/xatu/pkg/sentry/cache"
	"github.com/ethpandaops/xatu/pkg/sentry/ethereum"
	v1 "github.com/ethpandaops/xatu/pkg/sentry/event/beacon/eth/v1"
	v2 "github.com/ethpandaops/xatu/pkg/sentry/event/beacon/eth/v2"
	"github.com/ethpandaops/xatu/pkg/sentry/execution"
	"github.com/go-co-op/gocron/v2"
	"github.com/google/uuid"
	perrors "github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/sdk/trace"
	"gopkg.in/yaml.v3"
)

const unknown = "unknown"

type Sentry struct {
	Config *Config

	sinks []output.Sink

	beacon *ethereum.BeaconNode

	execution *execution.Client

	clockDrift time.Duration

	log logrus.FieldLogger

	duplicateCache *cache.DuplicateCache

	id uuid.UUID

	metrics *Metrics

	scheduler gocron.Scheduler

	latestForkChoice   *v1.ForkChoice
	latestForkChoiceMu sync.RWMutex

	preset *Preset

	shutdownFuncs []func(context.Context) error

	summary *Summary
}

func New(ctx context.Context, log logrus.FieldLogger, config *Config, overrides *Override) (*Sentry, error) {
	log = log.WithField("module", "sentry")

	if config == nil {
		return nil, errors.New("config is required")
	}

	var preset *Preset

	// Merge preset if its set
	if overrides.Preset.Enabled || config.Preset != "" {
		var presetName string

		if overrides.Preset.Enabled {
			presetName = overrides.Preset.Value
		} else {
			presetName = config.Preset
		}

		p, err := GetPreset(presetName)
		if err != nil {
			return nil, fmt.Errorf("failed to get preset: %w", err)
		}

		preset = p

		log.WithField("preset", presetName).Info("Applying preset")

		// Merge the 2 yaml files
		if err := yaml.Unmarshal(preset.Value, config); err != nil {
			return nil, fmt.Errorf("failed to merge config and preset: %w", err)
		}
	}

	// If the beacon node override is set, use it
	if overrides.BeaconNodeURL.Enabled {
		log.Info("Overriding beacon node URL")

		config.Ethereum.BeaconNodeAddress = overrides.BeaconNodeURL.Value
	}

	// If the metrics address override is set, use it
	if overrides.MetricsAddr.Enabled {
		log.WithField("address", overrides.MetricsAddr.Value).Info("Overriding metrics address")

		config.MetricsAddr = overrides.MetricsAddr.Value
	}

	if err := config.Validate(); err != nil {
		return nil, err
	}

	sinks, err := config.CreateSinks(log)
	if err != nil {
		return nil, err
	}

	beaconOpts := ethereum.Options{}

	hasAttestationSubscription := false

	attestationTopic := "attestation"

	if config.Ethereum.BeaconSubscriptions != nil {
		for _, topic := range *config.Ethereum.BeaconSubscriptions {
			if topic == attestationTopic {
				hasAttestationSubscription = true

				break
			}
		}
	} else if beacon.DefaultEnabledBeaconSubscriptionOptions().Enabled {
		// If no subscriptions have been provided in config, we need to check if the default options have it enabled
		for _, topic := range beacon.DefaultEnabledBeaconSubscriptionOptions().Topics {
			if topic == attestationTopic {
				hasAttestationSubscription = true

				break
			}
		}
	}

	if hasAttestationSubscription {
		log.Info("Enabling beacon committees as we are subscribed to attestation events")

		beaconOpts.WithFetchBeaconCommittees(true)
	}

	if config.BeaconCommittees != nil && config.BeaconCommittees.Enabled {
		log.Info("Enabling beacon committees as we need to fetch them on interval")
		beaconOpts.WithFetchBeaconCommittees(true)
	}

	if config.ProposerDuty != nil && config.ProposerDuty.Enabled {
		log.Info("Enabling proposer duties as we need to fetch them on interval")
		beaconOpts.WithFetchProposerDuties(true)
	}

	b, err := ethereum.NewBeaconNode(ctx, config.Name, &config.Ethereum, log, &beaconOpts)
	if err != nil {
		return nil, err
	}

	duplicateCache := cache.NewDuplicateCache()
	duplicateCache.Start()

	scheduler, err := gocron.NewScheduler(gocron.WithLocation(time.Local))
	if err != nil {
		return nil, err
	}

	s := &Sentry{
		Config:             config,
		sinks:              sinks,
		beacon:             b,
		execution:          nil,
		clockDrift:         time.Duration(0),
		log:                log,
		duplicateCache:     duplicateCache,
		id:                 uuid.New(),
		metrics:            NewMetrics("xatu_sentry"),
		scheduler:          scheduler,
		latestForkChoice:   nil,
		latestForkChoiceMu: sync.RWMutex{},
		shutdownFuncs:      []func(context.Context) error{},
		preset:             preset,
		summary:            NewSummary(log, time.Duration(60)*time.Second, b),
	}

	// If the output authorization override is set, use it
	if overrides.XatuOutputAuth.Enabled {
		log.Info("Overriding output authorization")

		for _, sink := range s.sinks {
			if sink.Type() == string(output.SinkTypeXatu) {
				xatuSink, ok := sink.(*oxatu.Xatu)
				if !ok {
					return nil, errors.New("failed to assert xatu sink")
				}

				xatuSink.SetAuthorization(overrides.XatuOutputAuth.Value)
			}
		}
	}

	return s, nil
}

//nolint:gocyclo // Needs refactoring
func (s *Sentry) Start(ctx context.Context) error {
	if err := s.ServeMetrics(ctx); err != nil {
		return err
	}

	if s.Config.PProfAddr != nil {
		if err := s.ServePProf(ctx); err != nil {
			return err
		}
	}

	s.log.
		WithField("version", xatu.Full()).
		WithField("id", s.id.String()).
		Info("Starting Xatu in sentry mode")

	// Start tracing if enabled
	if s.Config.Tracing.Enabled {
		s.log.Info("Tracing enabled")

		res, err := observability.NewResource(xatu.WithModule(xatu.ModuleName_SENTRY), xatu.Short())
		if err != nil {
			return perrors.Wrap(err, "failed to create tracing resource")
		}

		opts := []trace.TracerProviderOption{
			trace.WithSampler(trace.ParentBased(trace.TraceIDRatioBased(s.Config.Tracing.Sampling.Rate))),
		}

		tracer, err := observability.NewHTTPTraceProvider(ctx,
			res,
			s.Config.Tracing.AsOTelOpts(),
			opts...,
		)
		if err != nil {
			return perrors.Wrap(err, "failed to create tracing provider")
		}

		shutdown, err := observability.SetupOTelSDK(ctx, tracer)
		if err != nil {
			return perrors.Wrap(err, "failed to setup tracing SDK")
		}

		s.shutdownFuncs = append(s.shutdownFuncs, shutdown)
	}

	if err := s.startBeaconCommitteesWatcher(ctx); err != nil {
		return err
	}

	if err := s.startProposerDutyWatcher(ctx); err != nil {
		return err
	}

	s.beacon.Node().OnEvent(ctx, func(ctx context.Context, event *eth2v1.Event) error {
		s.summary.AddEventStreamEvents(event.Topic, 1)

		return nil
	})

	s.beacon.OnReady(ctx, func(ctx context.Context) error {
		s.log.Info("Internal beacon node is ready, subscribing to events")

		if s.beacon.Metadata().Network.Name == networks.NetworkNameUnknown {
			s.log.Fatal("Unable to determine Ethereum network. Provide an override network name via ethereum.overrideNetworkName")
		}

		s.beacon.Node().OnSingleAttestation(ctx, func(ctx context.Context, ev *electra.SingleAttestation) error {
			now := time.Now().Add(s.clockDrift)

			meta, err := s.createNewClientMeta(ctx)
			if err != nil {
				return err
			}

			event, err := v1.NewEventsSingleAttestation(s.log, ev, now, s.beacon, s.duplicateCache.BeaconETHV1EventsAttestation, meta)
			if err != nil {
				return err
			}

			ignore, err := event.ShouldIgnore(ctx)
			if err != nil {
				return err
			}

			if ignore {
				return nil
			}

			decoratedEvent, err := event.Decorate(ctx)
			if err != nil {
				return err
			}

			return s.handleNewDecoratedEvent(ctx, decoratedEvent)
		})

		s.beacon.Node().OnAttestation(ctx, func(ctx context.Context, ev *spec.VersionedAttestation) error {
			now := time.Now().Add(s.clockDrift)

			meta, err := s.createNewClientMeta(ctx)
			if err != nil {
				return err
			}

			event, err := v1.NewEventsAttestation(s.log, ev, now, s.beacon, s.duplicateCache.BeaconETHV1EventsAttestation, meta)
			if err != nil {
				return err
			}

			ignore, err := event.ShouldIgnore(ctx)
			if err != nil {
				return err
			}

			if ignore {
				return nil
			}

			decoratedEvent, err := event.Decorate(ctx)
			if err != nil {
				return err
			}

			return s.handleNewDecoratedEvent(ctx, decoratedEvent)
		})

		s.beacon.Node().OnBlock(ctx, func(ctx context.Context, block *eth2v1.BlockEvent) error {
			now := time.Now().Add(s.clockDrift)

			meta, err := s.createNewClientMeta(ctx)
			if err != nil {
				return err
			}

			event := v1.NewEventsBlock(s.log, block, now, s.beacon, s.duplicateCache.BeaconETHV1EventsBlock, meta)

			ignore, err := event.ShouldIgnore(ctx)
			if err != nil {
				return err
			}

			if ignore {
				return nil
			}

			decoratedEvent, err := event.Decorate(ctx)
			if err != nil {
				return err
			}

			err = s.handleNewDecoratedEvent(ctx, decoratedEvent)
			if err != nil {
				return err
			}

			go func() {
				// some clients require a small delay before being able to fetch the block
				time.Sleep(1 * time.Second)

				blockRoot := xatuethv1.RootAsString(block.Block)

				beaconBlock, err := s.beacon.Node().FetchBlock(ctx, blockRoot)
				if err != nil {
					s.log.WithError(err).Error("Failed to fetch block")
				} else {
					beaconBlockMeta, err := s.createNewClientMeta(ctx)
					if err != nil {
						s.log.WithError(err).Error("Failed to create client meta")

						return
					}

					event := v2.NewBeaconBlock(s.log, blockRoot, beaconBlock, now, s.beacon, s.duplicateCache.BeaconETHV2BeaconBlock, beaconBlockMeta)

					ignore, err := event.ShouldIgnore(ctx)
					if err != nil {
						s.log.
							WithError(err).
							WithField("event", xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK).
							Error("Failed to check if event should be ignored")

						return
					}

					if ignore {
						return
					}

					decoratedEvent, err := event.Decorate(ctx)
					if err != nil {
						s.log.WithError(err).Error("Failed to decorate event")

						return
					}

					if err := s.handleNewDecoratedEvent(ctx, decoratedEvent); err != nil {
						s.log.WithError(err).Error("Failed to handle new decorated event")

						return
					}
				}
			}()

			return nil
		})

		s.beacon.Node().OnBlockGossip(ctx, func(ctx context.Context, block *eth2v1.BlockGossipEvent) error {
			now := time.Now().Add(s.clockDrift)

			meta, err := s.createNewClientMeta(ctx)
			if err != nil {
				return err
			}

			event := v1.NewEventsBlockGossip(s.log, block, now, s.beacon, s.duplicateCache.BeaconETHV1EventsBlockGossip, meta)

			ignore, err := event.ShouldIgnore(ctx)
			if err != nil {
				return err
			}

			if ignore {
				return nil
			}

			decoratedEvent, err := event.Decorate(ctx)
			if err != nil {
				return err
			}

			return s.handleNewDecoratedEvent(ctx, decoratedEvent)
		})

		s.beacon.Node().OnChainReOrg(ctx, func(ctx context.Context, chainReorg *eth2v1.ChainReorgEvent) error {
			now := time.Now().Add(s.clockDrift)

			meta, err := s.createNewClientMeta(ctx)
			if err != nil {
				return err
			}

			event := v1.NewEventsChainReorg(s.log, chainReorg, now, s.beacon, s.duplicateCache.BeaconETHV1EventsChainReorg, meta)

			ignore, err := event.ShouldIgnore(ctx)
			if err != nil {
				return err
			}

			if ignore {
				return nil
			}

			decoratedEvent, err := event.Decorate(ctx)
			if err != nil {
				return err
			}

			return s.handleNewDecoratedEvent(ctx, decoratedEvent)
		})

		s.beacon.Node().OnHead(ctx, func(ctx context.Context, head *eth2v1.HeadEvent) error {
			now := time.Now().Add(s.clockDrift)

			meta, err := s.createNewClientMeta(ctx)
			if err != nil {
				return err
			}

			event := v1.NewEventsHead(s.log, head, now, s.beacon, s.duplicateCache.BeaconETHV1EventsHead, meta)

			ignore, err := event.ShouldIgnore(ctx)
			if err != nil {
				return err
			}

			if ignore {
				return nil
			}

			decoratedEvent, err := event.Decorate(ctx)
			if err != nil {
				return err
			}

			if err := s.handleNewDecoratedEvent(ctx, decoratedEvent); err != nil {
				return err
			}

			// Trigger state size polling if enabled in "head" mode.
			if err := s.onHeadEventForStateSize(ctx); err != nil {
				s.log.WithError(err).Debug("Failed to trigger state size polling on head event")
			}

			return nil
		})

		s.beacon.Node().OnVoluntaryExit(ctx, func(ctx context.Context, voluntaryExit *phase0.SignedVoluntaryExit) error {
			now := time.Now().Add(s.clockDrift)

			meta, err := s.createNewClientMeta(ctx)
			if err != nil {
				return err
			}

			event := v1.NewEventsVoluntaryExit(s.log, voluntaryExit, now, s.beacon, s.duplicateCache.BeaconETHV1EventsVoluntaryExit, meta)

			ignore, err := event.ShouldIgnore(ctx)
			if err != nil {
				return err
			}

			if ignore {
				return nil
			}

			decoratedEvent, err := event.Decorate(ctx)
			if err != nil {
				return err
			}

			return s.handleNewDecoratedEvent(ctx, decoratedEvent)
		})

		s.beacon.Node().OnContributionAndProof(ctx, func(ctx context.Context, contributionAndProof *altair.SignedContributionAndProof) error {
			now := time.Now().Add(s.clockDrift)

			meta, err := s.createNewClientMeta(ctx)
			if err != nil {
				return err
			}

			event := v1.NewEventsContributionAndProof(s.log, contributionAndProof, now, s.beacon, s.duplicateCache.BeaconETHV1EventsContributionAndProof, meta)

			ignore, err := event.ShouldIgnore(ctx)
			if err != nil {
				return err
			}

			if ignore {
				return nil
			}

			decoratedEvent, err := event.Decorate(ctx)
			if err != nil {
				return err
			}

			return s.handleNewDecoratedEvent(ctx, decoratedEvent)
		})

		s.beacon.Node().OnFinalizedCheckpoint(ctx, func(ctx context.Context, finalizedCheckpoint *eth2v1.FinalizedCheckpointEvent) error {
			now := time.Now().Add(s.clockDrift)

			meta, err := s.createNewClientMeta(ctx)
			if err != nil {
				return err
			}

			event := v1.NewEventsFinalizedCheckpoint(s.log, finalizedCheckpoint, now, s.beacon, s.duplicateCache.BeaconETHV1EventsFinalizedCheckpoint, meta)

			ignore, err := event.ShouldIgnore(ctx)
			if err != nil {
				return err
			}

			if ignore {
				return nil
			}

			decoratedEvent, err := event.Decorate(ctx)
			if err != nil {
				return err
			}

			return s.handleNewDecoratedEvent(ctx, decoratedEvent)
		})

		s.beacon.Node().OnBlobSidecar(ctx, func(ctx context.Context, blobSidecar *eth2v1.BlobSidecarEvent) error {
			now := time.Now().Add(s.clockDrift)

			meta, err := s.createNewClientMeta(ctx)
			if err != nil {
				return err
			}

			event := v1.NewEventsBlobSidecar(s.log, blobSidecar, now, s.beacon, s.duplicateCache.BeaconEthV1EventsBlobSidecar, meta)

			ignore, err := event.ShouldIgnore(ctx)
			if err != nil {
				return err
			}

			if ignore {
				return nil
			}

			decoratedEvent, err := event.Decorate(ctx)
			if err != nil {
				return err
			}

			return s.handleNewDecoratedEvent(ctx, decoratedEvent)
		})

		s.beacon.Node().OnDataColumnSidecar(ctx, func(ctx context.Context, dataColumnSidecar *eth2v1.DataColumnSidecarEvent) error {
			now := time.Now().Add(s.clockDrift)

			meta, err := s.createNewClientMeta(ctx)
			if err != nil {
				return err
			}

			event := v1.NewEventsDataColumnSidecar(s.log, dataColumnSidecar, now, s.beacon, s.duplicateCache.BeaconEthV1EventsDataColumnSidecar, meta)

			ignore, err := event.ShouldIgnore(ctx)
			if err != nil {
				return err
			}

			if ignore {
				return nil
			}

			decoratedEvent, err := event.Decorate(ctx)
			if err != nil {
				return err
			}

			return s.handleNewDecoratedEvent(ctx, decoratedEvent)
		})

		// Blob sidecar fetching on block receipt
		if s.Config.BlobSidecar != nil && s.Config.BlobSidecar.Enabled {
			s.log.Info("Blob sidecar fetching enabled")

			s.beacon.Node().OnBlock(ctx, func(ctx context.Context, block *eth2v1.BlockEvent) error {
				// Small delay to ensure block is available for blob fetching
				time.Sleep(1 * time.Second)

				blockRoot := xatuethv1.RootAsString(block.Block)
				s.fetchAndEmitBlobSidecars(ctx, blockRoot, block.Slot)

				return nil
			})
		}

		// Beacon blob metadata emission on block receipt
		if s.Config.BeaconBlob != nil && s.Config.BeaconBlob.Enabled {
			s.log.Info("Beacon blob metadata emission enabled")

			s.beacon.Node().OnBlock(ctx, func(ctx context.Context, block *eth2v1.BlockEvent) error {
				// Small delay to ensure block is available for fetching
				time.Sleep(1 * time.Second)

				blockRoot := xatuethv1.RootAsString(block.Block)
				s.fetchAndEmitBeaconBlobs(ctx, blockRoot, block.Slot)

				return nil
			})
		}

		if err := s.startForkChoiceSchedule(ctx); err != nil {
			return err
		}

		if err := s.startAttestationDataSchedule(ctx); err != nil {
			return err
		}

		if err := s.startValidatorBlockSchedule(ctx); err != nil {
			return err
		}

		if err := s.startMempoolTransactionWatcher(ctx); err != nil {
			// If we can't reach the execution node, we can't start the mempool transaction watcher.
			// Instead of preventing the sentry from starting, we'll log an error and continue.
			s.log.WithError(err).Error("failed to start mempool transaction watcher")

			return nil
		}

		if err := s.startExecutionStateSizeWatcher(ctx); err != nil {
			s.log.WithError(err).Error("failed to start execution debug state size watcher")

			return err
		}

		return nil
	})

	if err := s.startCrons(ctx); err != nil {
		s.log.WithError(err).Fatal("Failed to start crons")
	}

	for _, sink := range s.sinks {
		s.log.WithField("type", sink.Type()).WithField("name", sink.Name()).Info("Starting sink")

		if err := sink.Start(ctx); err != nil {
			return err
		}
	}

	go s.summary.Start(ctx)

	if s.Config.Ethereum.OverrideNetworkName != "" {
		s.log.WithField("network", s.Config.Ethereum.OverrideNetworkName).Info("Overriding network name")
	}

	if err := s.beacon.Start(ctx); err != nil {
		return err
	}

	cancel := make(chan os.Signal, 1)
	signal.Notify(cancel, syscall.SIGTERM, syscall.SIGINT)

	sig := <-cancel
	s.log.Printf("Caught signal: %v", sig)

	s.log.Printf("Flushing sinks")

	for _, sink := range s.sinks {
		if err := sink.Stop(ctx); err != nil {
			return err
		}
	}

	for _, f := range s.shutdownFuncs {
		if err := f(ctx); err != nil {
			return err
		}
	}

	return nil
}

func (s *Sentry) ServeMetrics(ctx context.Context) error {
	go func() {
		sm := http.NewServeMux()
		sm.Handle("/metrics", promhttp.Handler())

		server := &http.Server{
			Addr:              s.Config.MetricsAddr,
			ReadHeaderTimeout: 15 * time.Second,
			Handler:           sm,
		}

		s.log.Infof("Serving metrics at %s", s.Config.MetricsAddr)

		if err := server.ListenAndServe(); err != nil {
			s.log.Fatal(err)
		}
	}()

	return nil
}

func (s *Sentry) ServePProf(ctx context.Context) error {
	pprofServer := &http.Server{
		Addr:              *s.Config.PProfAddr,
		ReadHeaderTimeout: 120 * time.Second,
	}

	go func() {
		s.log.Infof("Serving pprof at %s", *s.Config.PProfAddr)

		if err := pprofServer.ListenAndServe(); err != nil {
			s.log.Fatal(err)
		}
	}()

	return nil
}

func (s *Sentry) createNewClientMeta(ctx context.Context) (*xatu.ClientMeta, error) {
	var networkMeta *xatu.ClientMeta_Ethereum_Network

	network := s.beacon.Metadata().Network
	if network != nil {
		networkMeta = &xatu.ClientMeta_Ethereum_Network{
			Name: string(network.Name),
			Id:   network.ID,
		}

		if s.Config.Ethereum.OverrideNetworkName != "" {
			networkMeta.Name = s.Config.Ethereum.OverrideNetworkName
		}
	}

	clientName := s.Config.Name
	if clientName == "" {
		hashed, err := s.beacon.Metadata().NodeIDHash()
		if err != nil {
			return nil, err
		}

		clientName = hashed
	}

	presetName := ""
	if s.preset != nil {
		presetName = s.preset.Name
	}

	return &xatu.ClientMeta{
		Name:           clientName,
		Version:        xatu.Short(),
		Id:             s.id.String(),
		Implementation: xatu.Implementation,
		ModuleName:     xatu.ModuleName_SENTRY,
		Os:             runtime.GOOS,
		ClockDrift:     uint64(s.clockDrift.Milliseconds()),
		Ethereum: &xatu.ClientMeta_Ethereum{
			Network:   networkMeta,
			Execution: &xatu.ClientMeta_Ethereum_Execution{},
			Consensus: &xatu.ClientMeta_Ethereum_Consensus{
				Implementation: s.beacon.Metadata().Client(ctx),
				Version:        s.beacon.Metadata().NodeVersion(ctx),
			},
		},
		Labels:     s.Config.Labels,
		PresetName: presetName,
	}, nil
}

func (s *Sentry) startCrons(ctx context.Context) error {
	if _, err := s.scheduler.NewJob(
		gocron.DurationJob(5*time.Minute),
		gocron.NewTask(
			func(ctx context.Context) {
				if err := s.syncClockDrift(ctx); err != nil {
					s.log.WithError(err).Error("Failed to sync clock drift")
				}
			},
			ctx,
		),
		gocron.WithStartAt(gocron.WithStartImmediately()),
	); err != nil {
		return err
	}

	s.scheduler.Start()

	return nil
}

func (s *Sentry) syncClockDrift(_ context.Context) error {
	response, err := ntp.Query(s.Config.NTPServer)
	if err != nil {
		return err
	}

	err = response.Validate()
	if err != nil {
		return err
	}

	s.clockDrift = response.ClockOffset

	s.log.WithField("drift", s.clockDrift).Debug("Updated clock drift")

	if s.clockDrift > 2*time.Second || s.clockDrift < -2*time.Second {
		s.log.WithField("drift", s.clockDrift).Warn("Large clock drift detected, consider configuring an NTP server on your instance")
	}

	return err
}

func (s *Sentry) handleNewDecoratedEvent(ctx context.Context, event *xatu.DecoratedEvent) error {
	if err := s.beacon.Synced(ctx); err != nil {
		return err
	}

	network := event.GetMeta().GetClient().GetEthereum().GetNetwork().GetId()
	networkStr := fmt.Sprintf("%d", network)

	if networkStr == "" || networkStr == "0" {
		networkStr = unknown
	}

	eventType := event.GetEvent().GetName().String()
	if eventType == "" {
		eventType = unknown
	}

	s.metrics.AddDecoratedEvent(1, eventType, networkStr)

	s.summary.AddEventsExported(1)

	for _, sink := range s.sinks {
		if err := sink.HandleNewDecoratedEvent(ctx, event); err != nil {
			s.log.
				WithError(err).
				WithField("sink", sink.Type()).
				WithField("event_type", event.GetEvent().GetName()).
				Error("Failed to send event to sink")
		}
	}

	return nil
}

// fetchAndEmitBlobSidecars fetches blob sidecars for a given block and emits them as events.
// This ensures blob data is captured even if gossip events are missed.
func (s *Sentry) fetchAndEmitBlobSidecars(ctx context.Context, blockRoot string, slot phase0.Slot) {
	blobs, err := s.beacon.Node().FetchBeaconBlockBlobs(ctx, blockRoot)
	if err != nil {
		var apiErr *api.Error
		if errors.As(err, &apiErr) {
			switch apiErr.StatusCode {
			case 404:
				// No blobs for this block - this is normal for blocks without blob transactions
				s.log.WithFields(logrus.Fields{
					"block_root": blockRoot,
					"slot":       slot,
				}).Debug("No blob sidecars found for block")

				return
			case 503:
				s.log.WithFields(logrus.Fields{
					"block_root": blockRoot,
					"slot":       slot,
				}).Warn("Beacon node is syncing, cannot fetch blob sidecars")

				return
			}
		}

		s.log.WithError(err).WithFields(logrus.Fields{
			"block_root": blockRoot,
			"slot":       slot,
		}).Error("Failed to fetch blob sidecars")

		return
	}

	if len(blobs) == 0 {
		s.log.WithFields(logrus.Fields{
			"block_root": blockRoot,
			"slot":       slot,
		}).Debug("No blob sidecars found for block")

		return
	}

	now := time.Now().Add(s.clockDrift)

	for _, blob := range blobs {
		blobEvent, err := ethereum.BlobSidecarToBlobSidecarEvent(blob)
		if err != nil {
			s.log.WithError(err).WithFields(logrus.Fields{
				"block_root": blockRoot,
				"slot":       slot,
				"index":      blob.Index,
			}).Error("Failed to convert blob sidecar to event")

			continue
		}

		meta, err := s.createNewClientMeta(ctx)
		if err != nil {
			s.log.WithError(err).Error("Failed to create client meta for blob sidecar")

			continue
		}

		event := v1.NewEventsBlobSidecar(
			s.log,
			blobEvent,
			now,
			s.beacon,
			s.duplicateCache.BeaconEthV1EventsBlobSidecar,
			meta,
		)

		ignore, err := event.ShouldIgnore(ctx)
		if err != nil {
			s.log.WithError(err).WithFields(logrus.Fields{
				"block_root": blockRoot,
				"slot":       slot,
				"index":      blob.Index,
			}).Error("Failed to check if blob sidecar event should be ignored")

			continue
		}

		if ignore {
			// Already processed via gossip - duplicate cache prevents re-emission
			continue
		}

		decoratedEvent, err := event.Decorate(ctx)
		if err != nil {
			s.log.WithError(err).WithFields(logrus.Fields{
				"block_root": blockRoot,
				"slot":       slot,
				"index":      blob.Index,
			}).Error("Failed to decorate blob sidecar event")

			continue
		}

		if err := s.handleNewDecoratedEvent(ctx, decoratedEvent); err != nil {
			s.log.WithError(err).WithFields(logrus.Fields{
				"block_root": blockRoot,
				"slot":       slot,
				"index":      blob.Index,
			}).Error("Failed to handle blob sidecar event")

			continue
		}
	}

	s.log.WithFields(logrus.Fields{
		"block_root": blockRoot,
		"slot":       slot,
		"count":      len(blobs),
	}).Debug("Fetched and emitted blob sidecars for block")
}

// fetchAndEmitBeaconBlobs fetches the block and emits beacon blob events for each KZG commitment.
// This captures blob metadata with versioned_hash for joining with execution_engine_get_blobs events.
func (s *Sentry) fetchAndEmitBeaconBlobs(ctx context.Context, blockRoot string, slot phase0.Slot) {
	block, err := s.beacon.Node().FetchBlock(ctx, blockRoot)
	if err != nil {
		var apiErr *api.Error
		if errors.As(err, &apiErr) {
			switch apiErr.StatusCode {
			case 404:
				s.log.WithFields(logrus.Fields{
					"block_root": blockRoot,
					"slot":       slot,
				}).Debug("Block not found for beacon blob extraction")

				return
			case 503:
				s.log.WithFields(logrus.Fields{
					"block_root": blockRoot,
					"slot":       slot,
				}).Warn("Beacon node is syncing, cannot fetch block for beacon blob extraction")

				return
			}
		}

		s.log.WithError(err).WithFields(logrus.Fields{
			"block_root": blockRoot,
			"slot":       slot,
		}).Error("Failed to fetch block for beacon blob extraction")

		return
	}

	if block == nil {
		s.log.WithFields(logrus.Fields{
			"block_root": blockRoot,
			"slot":       slot,
		}).Debug("Block is nil for beacon blob extraction")

		return
	}

	// Extract KZG commitments from the block body based on the block version
	kzgCommitments, err := extractKZGCommitments(block)
	if err != nil {
		s.log.WithFields(logrus.Fields{
			"block_root": blockRoot,
			"slot":       slot,
			"version":    block.Version.String(),
		}).Debug("Block version does not support KZG commitments")

		return
	}

	if len(kzgCommitments) == 0 {
		s.log.WithFields(logrus.Fields{
			"block_root": blockRoot,
			"slot":       slot,
		}).Debug("No KZG commitments found in block")

		return
	}

	// Extract block metadata
	proposerIndex, err := block.ProposerIndex()
	if err != nil {
		s.log.WithError(err).WithFields(logrus.Fields{
			"block_root": blockRoot,
			"slot":       slot,
		}).Error("Failed to get proposer index from block")

		return
	}

	parentRoot, err := block.ParentRoot()
	if err != nil {
		s.log.WithError(err).WithFields(logrus.Fields{
			"block_root": blockRoot,
			"slot":       slot,
		}).Error("Failed to get parent root from block")

		return
	}

	now := time.Now().Add(s.clockDrift)

	blockRootParsed, err := xatuethv1.StringToRoot(blockRoot)
	if err != nil {
		s.log.WithError(err).WithFields(logrus.Fields{
			"block_root": blockRoot,
			"slot":       slot,
		}).Error("Failed to parse block root for beacon blob extraction")

		return
	}

	for i, commitment := range kzgCommitments {
		versionedHash := ethereum.ConvertKzgCommitmentToVersionedHash(commitment[:])

		blobData := &v1.BlobData{
			Slot:            slot,
			Index:           uint64(i), //nolint:gosec // G115: blob index is always small
			BlockRoot:       blockRootParsed,
			BlockParentRoot: parentRoot,
			ProposerIndex:   proposerIndex,
			KZGCommitment:   commitment,
			VersionedHash:   versionedHash,
		}

		meta, err := s.createNewClientMeta(ctx)
		if err != nil {
			s.log.WithError(err).Error("Failed to create client meta for beacon blob")

			continue
		}

		event := v1.NewBeaconBlob(
			s.log,
			blobData,
			now,
			s.beacon,
			s.duplicateCache.BeaconEthV1BeaconBlob,
			meta,
		)

		ignore, err := event.ShouldIgnore(ctx)
		if err != nil {
			s.log.WithError(err).WithFields(logrus.Fields{
				"block_root": blockRoot,
				"slot":       slot,
				"index":      i,
			}).Error("Failed to check if beacon blob event should be ignored")

			continue
		}

		if ignore {
			continue
		}

		decoratedEvent, err := event.Decorate(ctx)
		if err != nil {
			s.log.WithError(err).WithFields(logrus.Fields{
				"block_root": blockRoot,
				"slot":       slot,
				"index":      i,
			}).Error("Failed to decorate beacon blob event")

			continue
		}

		if err := s.handleNewDecoratedEvent(ctx, decoratedEvent); err != nil {
			s.log.WithError(err).WithFields(logrus.Fields{
				"block_root": blockRoot,
				"slot":       slot,
				"index":      i,
			}).Error("Failed to handle beacon blob event")

			continue
		}
	}

	s.log.WithFields(logrus.Fields{
		"block_root": blockRoot,
		"slot":       slot,
		"count":      len(kzgCommitments),
	}).Debug("Fetched and emitted beacon blobs for block")
}

// extractKZGCommitments extracts KZG commitments from a block based on its version.
func extractKZGCommitments(block *spec.VersionedSignedBeaconBlock) ([]deneb.KZGCommitment, error) {
	switch block.Version {
	case spec.DataVersionDeneb:
		if block.Deneb != nil && block.Deneb.Message != nil && block.Deneb.Message.Body != nil {
			return block.Deneb.Message.Body.BlobKZGCommitments, nil
		}
	case spec.DataVersionElectra:
		if block.Electra != nil && block.Electra.Message != nil && block.Electra.Message.Body != nil {
			return block.Electra.Message.Body.BlobKZGCommitments, nil
		}
	case spec.DataVersionFulu:
		if block.Fulu != nil && block.Fulu.Message != nil && block.Fulu.Message.Body != nil {
			return block.Fulu.Message.Body.BlobKZGCommitments, nil
		}
	}

	return nil, fmt.Errorf("block version %s does not support KZG commitments", block.Version)
}
