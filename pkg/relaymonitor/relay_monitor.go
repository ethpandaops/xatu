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
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"

	//nolint:gosec // only exposed if pprofAddr config is set
	_ "net/http/pprof"

	"github.com/beevik/ntp"
	apiv1 "github.com/ethpandaops/go-eth2-client/api/v1"
	"github.com/go-co-op/gocron/v2"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/ethpandaops/xatu/pkg/observability"
	"github.com/ethpandaops/xatu/pkg/output"
	"github.com/ethpandaops/xatu/pkg/proto/mevrelay"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/ethpandaops/xatu/pkg/relaymonitor/coordinator"
	"github.com/ethpandaops/xatu/pkg/relaymonitor/ethereum"
	"github.com/ethpandaops/xatu/pkg/relaymonitor/registrations"
	"github.com/ethpandaops/xatu/pkg/relaymonitor/relay"
)

type RelayMonitor struct {
	Config *Config

	sinks []output.Sink

	clockDrift time.Duration

	log observability.ContextualLogger

	id uuid.UUID

	metrics *Metrics

	ethereum *ethereum.BeaconNode

	relays []*relay.Client

	bidCache *DuplicateBidCache

	validatorRegistrationMonitor *registrations.ValidatorMonitor

	regestationEventsCh chan *registrations.ValidatorRegistrationEvent

	coordinatorClient *coordinator.Client
}

const (
	namespace = "xatu_relay_monitor"
)

func New(ctx context.Context, log observability.ContextualLogger, config *Config, overrides *Override) (*RelayMonitor, error) {
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

	client, err := ethereum.NewBeaconNode(
		config.Name,
		log,
		&config.Ethereum,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create ethereum client: %w", err)
	}

	relays := make([]*relay.Client, 0, len(config.Relays))

	for _, r := range config.Relays {
		relayClient, err := relay.NewClient(namespace, r, config.Ethereum.Network)
		if err != nil {
			return nil, perrors.Wrap(err, "failed to create relay client")
		}

		relays = append(relays, relayClient)
	}

	regestationEventsCh := make(chan *registrations.ValidatorRegistrationEvent, 10000)

	relayMonitor := &RelayMonitor{
		Config:                       config,
		sinks:                        sinks,
		clockDrift:                   time.Duration(0),
		log:                          log,
		id:                           uuid.New(),
		metrics:                      NewMetrics(namespace, config.Ethereum.Network),
		ethereum:                     client,
		relays:                       relays,
		bidCache:                     NewDuplicateBidCache(time.Minute * 13),
		regestationEventsCh:          regestationEventsCh,
		validatorRegistrationMonitor: registrations.NewValidatorMonitor(log, &config.ValidatorRegistrations, relays, client, regestationEventsCh),
	}

	// Initialize coordinator client if configured
	if config.Coordinator != nil {
		coordinatorClient, err := coordinator.New(config.Coordinator, log)
		if err != nil {
			return nil, fmt.Errorf("failed to create coordinator client: %w", err)
		}

		relayMonitor.coordinatorClient = coordinatorClient
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

	go r.handleValidatorRegistrationEvents(ctx)

	r.log.
		WithField("version", xatu.Full()).
		WithField("id", r.id.String()).WithContext(ctx).
		Info("Starting Xatu in relay monitor mode")

	// Sync clock drift
	if err := r.syncClockDrift(ctx); err != nil {
		// This is not fatal, as we will sync it on the first poll
		r.log.WithError(err).WithContext(ctx).Warn("Failed to sync clock drift")
	}

	if r.clockDrift != 0 {
		r.log.WithField("drift", r.clockDrift).WithContext(ctx).Info("Clock drift detected")

		maxDrift := time.Second * 60 * 5

		if r.clockDrift > maxDrift || r.clockDrift < -maxDrift {
			r.log.WithField("drift", r.clockDrift).WithContext(ctx).Fatal("Clock drift is too high, exiting")
		}
	}

	for _, sink := range r.sinks {
		if err := sink.Start(ctx); err != nil {
			return err
		}
	}

	if err := r.ethereum.Start(ctx); err != nil {
		return errors.Join(err, fmt.Errorf("failed to start ethereum beacon node client"))
	}

	if err := r.startCrons(ctx); err != nil {
		r.log.WithError(err).WithContext(ctx).Fatal("Failed to start crons")
	}

	if r.Config.ValidatorRegistrations.Enabled != nil && *r.Config.ValidatorRegistrations.Enabled {
		if err := r.validatorRegistrationMonitor.Start(ctx); err != nil {
			return errors.Join(err, fmt.Errorf("failed to start validator registration"))
		}
	}

	// Start coordinator client if configured
	if r.coordinatorClient != nil {
		if err := r.coordinatorClient.Start(ctx); err != nil {
			r.log.WithError(err).WithContext(ctx).Error("Failed to start coordinator client")
		}

		// Start consistency processes if configured
		if err := r.startConsistencyProcesses(ctx); err != nil {
			r.log.WithError(err).WithContext(ctx).Error("Failed to start consistency processes")
		}
	}

	cancel := make(chan os.Signal, 1)
	signal.Notify(cancel, syscall.SIGTERM, syscall.SIGINT)

	sig := <-cancel
	r.log.WithContext(ctx).Printf("Caught signal: %v", sig)

	// Stop coordinator client if running
	if r.coordinatorClient != nil {
		if err := r.coordinatorClient.Stop(ctx); err != nil {
			r.log.WithError(err).WithContext(ctx).Error("Failed to stop coordinator client")
		}
	}

	r.log.WithContext(ctx).Printf("Flushing sinks")

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

		r.log.WithContext(ctx).Infof("Serving metrics at %s", r.Config.MetricsAddr)

		if err := server.ListenAndServe(); err != nil {
			r.log.WithContext(ctx).Fatal(err)
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
		r.log.WithContext(ctx).Infof("Serving pprof at %s", *r.Config.PProfAddr)

		if err := pprofServer.ListenAndServe(); err != nil {
			r.log.WithContext(ctx).Fatal(err)
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
				r.log.WithContext(ctx).Error("Failed to write response: ", err)
			}
		}),
	}

	go func() {
		r.log.WithContext(ctx).Infof("Serving probe at %s", *r.Config.ProbeAddr)

		if err := probeServer.ListenAndServe(); err != nil {
			r.log.WithContext(ctx).Fatal(err)
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
		ModuleName:     xatu.ModuleName_RELAY_MONITOR,
		Os:             runtime.GOOS,
		ClockDrift:     uint64(r.clockDrift.Milliseconds()), //nolint:gosec // drift won't overflow
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
	c, err := gocron.NewScheduler(gocron.WithLocation(time.Local))
	if err != nil {
		return err
	}

	if _, err := c.NewJob(
		gocron.DurationJob(5*time.Minute),
		gocron.NewTask(
			func(ctx context.Context) {
				if err := r.syncClockDrift(ctx); err != nil {
					r.log.WithError(err).WithContext(ctx).Error("Failed to sync clock drift")
				}
			},
			ctx,
		),
		gocron.WithStartAt(gocron.WithStartImmediately()),
	); err != nil {
		return err
	}

	if r.Config.Schedule.AtSlotTimes != nil {
		for _, timer := range r.Config.Schedule.AtSlotTimes {
			for _, relayClient := range r.relays {
				r.scheduleBidTraceFetchingAtSlotTime(ctx, timer.Duration, relayClient)
			}
		}
	}

	if !r.Config.FetchProposerPayloadDelivered {
		r.log.WithContext(ctx).Warn("Proposer payload delivered fetching is disabled")
	} else {
		for _, relayClient := range r.relays {
			r.scheduleProposerPayloadDeliveredFetching(ctx, relayClient)
		}
	}

	c.Start()

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
	r.log.WithField("drift", r.clockDrift).WithContext(ctx).Info("Updated clock drift")

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
			r.log.WithError(err).WithField("sink", sink.Type()).WithContext(ctx).Error("Failed to send event to sink")
		}
	}

	return nil
}

func (r *RelayMonitor) handleValidatorRegistrationEvents(ctx context.Context) {
	for {
		select {
		case event := <-r.regestationEventsCh:
			if event == nil {
				continue
			}

			ev, err := r.createValidatorRegistrationEvent(ctx, event.Validator, event.Relay, event.Registration)
			if err != nil {
				r.log.WithError(err).WithContext(ctx).Error("Failed to create validator registration event")

				continue
			}

			if err := r.handleNewDecoratedEvent(ctx, ev); err != nil {
				r.log.WithError(err).WithContext(ctx).Error("Failed to handle new decorated event")
			}
		case <-ctx.Done():
			return
		}
	}
}

func (r *RelayMonitor) createValidatorRegistrationAdditionalData(
	_ context.Context,
	now time.Time,
	validator *apiv1.Validator,
	re *relay.Client,
	registration *mevrelay.ValidatorRegistration,
) (*xatu.ClientMeta_AdditionalMevRelayValidatorRegistrationData, error) {
	// Calculate slot and epoch from registration timestamp
	// Convert timestamp to time.Time
	data := &xatu.ClientMeta_AdditionalMevRelayValidatorRegistrationData{
		Relay: &mevrelay.Relay{
			Url:  wrapperspb.String(re.URL()),
			Name: wrapperspb.String(re.Name()),
		},
		ValidatorIndex: &wrapperspb.UInt64Value{
			Value: uint64(validator.Index),
		},
	}

	//nolint:gosec // Not a concern - heat death of universe stuff
	timestamp := time.Unix(int64(registration.GetMessage().GetTimestamp().GetValue()), 0).UTC()

	slot, epoch, err := r.ethereum.Wallclock().FromTime(timestamp)
	if err != nil {
		r.log.WithError(err).Error("Failed to get slot and epoch from timestamp when creating validator registration event")
	} else {
		data.Slot = &xatu.SlotV2{
			Number: &wrapperspb.UInt64Value{
				Value: slot.Number(),
			},
			StartDateTime: timestamppb.New(slot.TimeWindow().Start()),
		}
		data.Epoch = &xatu.EpochV2{
			Number: &wrapperspb.UInt64Value{
				Value: epoch.Number(),
			},
			StartDateTime: timestamppb.New(epoch.TimeWindow().Start()),
		}
	}

	slot, epoch, err = r.ethereum.Wallclock().FromTime(now)
	if err != nil {
		r.log.WithError(err).Error("Failed to get slot and epoch from wallclock when creating validator registration event")
	} else {
		data.WallclockSlot = &xatu.SlotV2{
			Number: &wrapperspb.UInt64Value{
				Value: slot.Number(),
			},
			StartDateTime: timestamppb.New(slot.TimeWindow().Start()),
		}

		data.WallclockEpoch = &xatu.EpochV2{
			Number: &wrapperspb.UInt64Value{
				Value: epoch.Number(),
			},
			StartDateTime: timestamppb.New(epoch.TimeWindow().Start()),
		}
	}

	return data, nil
}

func (r *RelayMonitor) createValidatorRegistrationEvent(
	ctx context.Context,
	validator *apiv1.Validator,
	re *relay.Client,
	registration *mevrelay.ValidatorRegistration,
) (*xatu.DecoratedEvent, error) {
	now := time.Now()

	clientMeta, err := r.createNewClientMeta(ctx)
	if err != nil {
		return nil, err
	}

	metadata, ok := proto.Clone(clientMeta).(*xatu.ClientMeta)
	if !ok {
		return nil, fmt.Errorf("failed to clone client metadata")
	}

	additionalData, err := r.createValidatorRegistrationAdditionalData(ctx, now, validator, re, registration)
	if err == nil {
		metadata.AdditionalData = &xatu.ClientMeta_MevRelayValidatorRegistration{
			MevRelayValidatorRegistration: additionalData,
		}
	} else {
		r.log.WithError(err).WithContext(ctx).Error("Failed to create validator registration additional data")
	}

	decoratedEvent := &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_MEV_RELAY_VALIDATOR_REGISTRATION,
			DateTime: timestamppb.New(now.Add(r.clockDrift)),
			Id:       uuid.New().String(),
		},
		Meta: &xatu.Meta{
			Client: metadata,
		},
		Data: &xatu.DecoratedEvent_MevRelayValidatorRegistration{
			MevRelayValidatorRegistration: &mevrelay.ValidatorRegistration{
				Message:   registration.Message,
				Signature: nil, // Signature is not needed for the event - lets save the bytes
			},
		},
	}

	return decoratedEvent, nil
}
