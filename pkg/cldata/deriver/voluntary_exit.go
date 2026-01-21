package deriver

import (
	"context"
	"fmt"
	"time"

	"github.com/attestantio/go-eth2-client/spec"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	backoff "github.com/cenkalti/backoff/v5"
	"github.com/ethpandaops/xatu/pkg/cldata"
	"github.com/ethpandaops/xatu/pkg/cldata/iterator"
	"github.com/ethpandaops/xatu/pkg/observability"
	xatuethv1 "github.com/ethpandaops/xatu/pkg/proto/eth/v1"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

const (
	VoluntaryExitDeriverName = xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_VOLUNTARY_EXIT
)

// VoluntaryExitDeriverConfig holds the configuration for the VoluntaryExitDeriver.
type VoluntaryExitDeriverConfig struct {
	Enabled bool `yaml:"enabled" default:"true"`
}

// VoluntaryExitDeriver derives voluntary exit events from the consensus layer.
// It processes epochs of blocks and emits decorated events for each voluntary exit.
type VoluntaryExitDeriver struct {
	log               logrus.FieldLogger
	cfg               *VoluntaryExitDeriverConfig
	iterator          iterator.Iterator
	onEventsCallbacks []func(ctx context.Context, events []*xatu.DecoratedEvent) error
	beacon            cldata.BeaconClient
	ctx               cldata.ContextProvider
}

// NewVoluntaryExitDeriver creates a new VoluntaryExitDeriver instance.
func NewVoluntaryExitDeriver(
	log logrus.FieldLogger,
	config *VoluntaryExitDeriverConfig,
	iter iterator.Iterator,
	beacon cldata.BeaconClient,
	ctxProvider cldata.ContextProvider,
) *VoluntaryExitDeriver {
	return &VoluntaryExitDeriver{
		log: log.WithFields(logrus.Fields{
			"module": "cldata/deriver/voluntary_exit",
			"type":   VoluntaryExitDeriverName.String(),
		}),
		cfg:      config,
		iterator: iter,
		beacon:   beacon,
		ctx:      ctxProvider,
	}
}

func (v *VoluntaryExitDeriver) CannonType() xatu.CannonType {
	return VoluntaryExitDeriverName
}

func (v *VoluntaryExitDeriver) ActivationFork() spec.DataVersion {
	return spec.DataVersionPhase0
}

func (v *VoluntaryExitDeriver) Name() string {
	return VoluntaryExitDeriverName.String()
}

func (v *VoluntaryExitDeriver) OnEventsDerived(
	ctx context.Context,
	fn func(ctx context.Context, events []*xatu.DecoratedEvent) error,
) {
	v.onEventsCallbacks = append(v.onEventsCallbacks, fn)
}

func (v *VoluntaryExitDeriver) Start(ctx context.Context) error {
	if !v.cfg.Enabled {
		v.log.Info("Voluntary exit deriver disabled")

		return nil
	}

	v.log.Info("Voluntary exit deriver enabled")

	if err := v.iterator.Start(ctx, v.ActivationFork()); err != nil {
		return errors.Wrap(err, "failed to start iterator")
	}

	// Start our main loop
	v.run(ctx)

	return nil
}

func (v *VoluntaryExitDeriver) Stop(ctx context.Context) error {
	return nil
}

func (v *VoluntaryExitDeriver) run(rctx context.Context) {
	bo := backoff.NewExponentialBackOff()
	bo.MaxInterval = 3 * time.Minute

	tracer := observability.Tracer()

	for {
		select {
		case <-rctx.Done():
			return
		default:
			operation := func() (string, error) {
				ctx, span := tracer.Start(rctx, fmt.Sprintf("Derive %s", v.Name()),
					trace.WithAttributes(
						attribute.String("network", v.ctx.NetworkName())),
				)
				defer span.End()

				time.Sleep(100 * time.Millisecond)

				if err := v.beacon.Synced(ctx); err != nil {
					span.SetStatus(codes.Error, err.Error())

					return "", err
				}

				// Get the next position
				position, err := v.iterator.Next(ctx)
				if err != nil {
					span.SetStatus(codes.Error, err.Error())

					return "", err
				}

				// Look ahead
				v.lookAhead(ctx, position.LookAheadEpochs)

				// Process the epoch
				events, err := v.processEpoch(ctx, position.Epoch)
				if err != nil {
					v.log.WithError(err).Error("Failed to process epoch")

					span.SetStatus(codes.Error, err.Error())

					return "", err
				}

				span.AddEvent("Epoch processing complete. Sending events...")

				// Send the events
				for _, fn := range v.onEventsCallbacks {
					if err := fn(ctx, events); err != nil {
						span.SetStatus(codes.Error, err.Error())

						return "", errors.Wrap(err, "failed to send events")
					}
				}

				span.AddEvent("Events sent. Updating location...")

				// Update our location
				if err := v.iterator.UpdateLocation(ctx, position); err != nil {
					span.SetStatus(codes.Error, err.Error())

					return "", err
				}

				span.AddEvent("Location updated. Done.")

				bo.Reset()

				return "", nil
			}

			retryOpts := []backoff.RetryOption{
				backoff.WithBackOff(bo),
				backoff.WithNotify(func(err error, timer time.Duration) {
					v.log.WithError(err).WithField("next_attempt", timer).Warn("Failed to process")
				}),
			}
			if _, err := backoff.Retry(rctx, operation, retryOpts...); err != nil {
				v.log.WithError(err).Warn("Failed to process")
			}
		}
	}
}

// lookAhead takes the upcoming epochs and looks ahead to do any pre-processing that might be required.
func (v *VoluntaryExitDeriver) lookAhead(ctx context.Context, epochs []phase0.Epoch) {
	_, span := observability.Tracer().Start(ctx,
		"VoluntaryExitDeriver.lookAhead",
	)
	defer span.End()

	sp, err := v.beacon.Node().Spec()
	if err != nil {
		v.log.WithError(err).Warn("Failed to look ahead at epoch")

		return
	}

	for _, epoch := range epochs {
		for i := uint64(0); i <= uint64(sp.SlotsPerEpoch-1); i++ {
			slot := phase0.Slot(i + uint64(epoch)*uint64(sp.SlotsPerEpoch))

			// Add the block to the preload queue so it's available when we need it
			v.beacon.LazyLoadBeaconBlock(xatuethv1.SlotAsString(slot))
		}
	}
}

func (v *VoluntaryExitDeriver) processEpoch(ctx context.Context, epoch phase0.Epoch) ([]*xatu.DecoratedEvent, error) {
	ctx, span := observability.Tracer().Start(ctx,
		"VoluntaryExitDeriver.processEpoch",
		//nolint:gosec // epoch numbers won't exceed int64 max in practice
		trace.WithAttributes(attribute.Int64("epoch", int64(epoch))),
	)
	defer span.End()

	sp, err := v.beacon.Node().Spec()
	if err != nil {
		return nil, errors.Wrap(err, "failed to obtain spec")
	}

	allEvents := make([]*xatu.DecoratedEvent, 0)

	for i := uint64(0); i <= uint64(sp.SlotsPerEpoch-1); i++ {
		slot := phase0.Slot(i + uint64(epoch)*uint64(sp.SlotsPerEpoch))

		events, err := v.processSlot(ctx, slot)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to process slot %d", slot)
		}

		allEvents = append(allEvents, events...)
	}

	return allEvents, nil
}

func (v *VoluntaryExitDeriver) processSlot(ctx context.Context, slot phase0.Slot) ([]*xatu.DecoratedEvent, error) {
	ctx, span := observability.Tracer().Start(ctx,
		"VoluntaryExitDeriver.processSlot",
		//nolint:gosec // slot numbers won't exceed int64 max in practice
		trace.WithAttributes(attribute.Int64("slot", int64(slot))),
	)
	defer span.End()

	// Get the block
	block, err := v.beacon.GetBeaconBlock(ctx, xatuethv1.SlotAsString(slot))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get beacon block for slot %d", slot)
	}

	if block == nil {
		return []*xatu.DecoratedEvent{}, nil
	}

	blockIdentifier, err := GetBlockIdentifier(block, v.ctx.Wallclock())
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get block identifier for slot %d", slot)
	}

	events := make([]*xatu.DecoratedEvent, 0)

	exits, err := v.getVoluntaryExits(ctx, block)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get voluntary exits for slot %d", slot)
	}

	for _, exit := range exits {
		event, err := v.createEvent(ctx, exit, blockIdentifier)
		if err != nil {
			v.log.WithError(err).Error("Failed to create event")

			return nil, errors.Wrapf(err, "failed to create event for voluntary exit %s", exit.String())
		}

		events = append(events, event)
	}

	return events, nil
}

func (v *VoluntaryExitDeriver) getVoluntaryExits(
	_ context.Context,
	block *spec.VersionedSignedBeaconBlock,
) ([]*xatuethv1.SignedVoluntaryExitV2, error) {
	exits := make([]*xatuethv1.SignedVoluntaryExitV2, 0)

	voluntaryExits, err := block.VoluntaryExits()
	if err != nil {
		return nil, errors.Wrap(err, "failed to obtain voluntary exits")
	}

	for _, exit := range voluntaryExits {
		exits = append(exits, &xatuethv1.SignedVoluntaryExitV2{
			Message: &xatuethv1.VoluntaryExitV2{
				Epoch:          wrapperspb.UInt64(uint64(exit.Message.Epoch)),
				ValidatorIndex: wrapperspb.UInt64(uint64(exit.Message.ValidatorIndex)),
			},
			Signature: exit.Signature.String(),
		})
	}

	return exits, nil
}

func (v *VoluntaryExitDeriver) createEvent(
	ctx context.Context,
	exit *xatuethv1.SignedVoluntaryExitV2,
	identifier *xatu.BlockIdentifier,
) (*xatu.DecoratedEvent, error) {
	// Get client metadata
	clientMeta, err := v.ctx.CreateClientMeta(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create client metadata")
	}

	// Make a clone of the metadata
	metadata, ok := proto.Clone(clientMeta).(*xatu.ClientMeta)
	if !ok {
		return nil, errors.New("failed to clone client metadata")
	}

	decoratedEvent := &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_VOLUNTARY_EXIT,
			DateTime: timestamppb.New(time.Now()),
			Id:       uuid.New().String(),
		},
		Meta: &xatu.Meta{
			Client: metadata,
		},
		Data: &xatu.DecoratedEvent_EthV2BeaconBlockVoluntaryExit{
			EthV2BeaconBlockVoluntaryExit: exit,
		},
	}

	decoratedEvent.Meta.Client.AdditionalData = &xatu.ClientMeta_EthV2BeaconBlockVoluntaryExit{
		EthV2BeaconBlockVoluntaryExit: &xatu.ClientMeta_AdditionalEthV2BeaconBlockVoluntaryExitData{
			Block: identifier,
		},
	}

	return decoratedEvent, nil
}
