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
	WithdrawalDeriverName = xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_WITHDRAWAL
)

// WithdrawalDeriverConfig holds the configuration for the WithdrawalDeriver.
type WithdrawalDeriverConfig struct {
	Enabled bool `yaml:"enabled" default:"true"`
}

// WithdrawalDeriver derives withdrawal events from the consensus layer.
// It processes epochs of blocks and emits decorated events for each withdrawal.
type WithdrawalDeriver struct {
	log               logrus.FieldLogger
	cfg               *WithdrawalDeriverConfig
	iterator          iterator.Iterator
	onEventsCallbacks []func(ctx context.Context, events []*xatu.DecoratedEvent) error
	beacon            cldata.BeaconClient
	ctx               cldata.ContextProvider
}

// NewWithdrawalDeriver creates a new WithdrawalDeriver instance.
func NewWithdrawalDeriver(
	log logrus.FieldLogger,
	config *WithdrawalDeriverConfig,
	iter iterator.Iterator,
	beacon cldata.BeaconClient,
	ctxProvider cldata.ContextProvider,
) *WithdrawalDeriver {
	return &WithdrawalDeriver{
		log: log.WithFields(logrus.Fields{
			"module": "cldata/deriver/withdrawal",
			"type":   WithdrawalDeriverName.String(),
		}),
		cfg:      config,
		iterator: iter,
		beacon:   beacon,
		ctx:      ctxProvider,
	}
}

func (w *WithdrawalDeriver) CannonType() xatu.CannonType {
	return WithdrawalDeriverName
}

func (w *WithdrawalDeriver) ActivationFork() spec.DataVersion {
	return spec.DataVersionCapella
}

func (w *WithdrawalDeriver) Name() string {
	return WithdrawalDeriverName.String()
}

func (w *WithdrawalDeriver) OnEventsDerived(
	ctx context.Context,
	fn func(ctx context.Context, events []*xatu.DecoratedEvent) error,
) {
	w.onEventsCallbacks = append(w.onEventsCallbacks, fn)
}

func (w *WithdrawalDeriver) Start(ctx context.Context) error {
	if !w.cfg.Enabled {
		w.log.Info("Withdrawal deriver disabled")

		return nil
	}

	w.log.Info("Withdrawal deriver enabled")

	if err := w.iterator.Start(ctx, w.ActivationFork()); err != nil {
		return errors.Wrap(err, "failed to start iterator")
	}

	// Start our main loop
	w.run(ctx)

	return nil
}

func (w *WithdrawalDeriver) Stop(ctx context.Context) error {
	return nil
}

func (w *WithdrawalDeriver) run(rctx context.Context) {
	bo := backoff.NewExponentialBackOff()
	bo.MaxInterval = 3 * time.Minute

	tracer := observability.Tracer()

	for {
		select {
		case <-rctx.Done():
			return
		default:
			operation := func() (string, error) {
				ctx, span := tracer.Start(rctx, fmt.Sprintf("Derive %s", w.Name()),
					trace.WithAttributes(
						attribute.String("network", w.ctx.NetworkName())),
				)
				defer span.End()

				time.Sleep(100 * time.Millisecond)

				if err := w.beacon.Synced(ctx); err != nil {
					span.SetStatus(codes.Error, err.Error())

					return "", err
				}

				// Get the next position
				position, err := w.iterator.Next(ctx)
				if err != nil {
					span.SetStatus(codes.Error, err.Error())

					return "", err
				}

				// Look ahead
				w.lookAhead(ctx, position.LookAheadEpochs)

				// Process the epoch
				events, err := w.processEpoch(ctx, position.Epoch)
				if err != nil {
					w.log.WithError(err).Error("Failed to process epoch")

					span.SetStatus(codes.Error, err.Error())

					return "", err
				}

				span.AddEvent("Epoch processing complete. Sending events...")

				// Send the events
				for _, fn := range w.onEventsCallbacks {
					if err := fn(ctx, events); err != nil {
						span.SetStatus(codes.Error, err.Error())

						return "", errors.Wrap(err, "failed to send events")
					}
				}

				span.AddEvent("Events sent. Updating location...")

				// Update our location
				if err := w.iterator.UpdateLocation(ctx, position); err != nil {
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
					w.log.WithError(err).WithField("next_attempt", timer).Warn("Failed to process")
				}),
			}
			if _, err := backoff.Retry(rctx, operation, retryOpts...); err != nil {
				w.log.WithError(err).Warn("Failed to process")
			}
		}
	}
}

// lookAhead takes the upcoming epochs and looks ahead to do any pre-processing that might be required.
func (w *WithdrawalDeriver) lookAhead(ctx context.Context, epochs []phase0.Epoch) {
	_, span := observability.Tracer().Start(ctx,
		"WithdrawalDeriver.lookAhead",
	)
	defer span.End()

	sp, err := w.beacon.Node().Spec()
	if err != nil {
		w.log.WithError(err).Warn("Failed to look ahead at epoch")

		return
	}

	for _, epoch := range epochs {
		for i := uint64(0); i <= uint64(sp.SlotsPerEpoch-1); i++ {
			slot := phase0.Slot(i + uint64(epoch)*uint64(sp.SlotsPerEpoch))

			// Add the block to the preload queue so it's available when we need it
			w.beacon.LazyLoadBeaconBlock(xatuethv1.SlotAsString(slot))
		}
	}
}

func (w *WithdrawalDeriver) processEpoch(ctx context.Context, epoch phase0.Epoch) ([]*xatu.DecoratedEvent, error) {
	ctx, span := observability.Tracer().Start(ctx,
		"WithdrawalDeriver.processEpoch",
		//nolint:gosec // epoch numbers won't exceed int64 max in practice
		trace.WithAttributes(attribute.Int64("epoch", int64(epoch))),
	)
	defer span.End()

	sp, err := w.beacon.Node().Spec()
	if err != nil {
		return nil, errors.Wrap(err, "failed to obtain spec")
	}

	allEvents := make([]*xatu.DecoratedEvent, 0)

	for i := uint64(0); i <= uint64(sp.SlotsPerEpoch-1); i++ {
		slot := phase0.Slot(i + uint64(epoch)*uint64(sp.SlotsPerEpoch))

		events, err := w.processSlot(ctx, slot)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to process slot %d", slot)
		}

		allEvents = append(allEvents, events...)
	}

	return allEvents, nil
}

func (w *WithdrawalDeriver) processSlot(ctx context.Context, slot phase0.Slot) ([]*xatu.DecoratedEvent, error) {
	ctx, span := observability.Tracer().Start(ctx,
		"WithdrawalDeriver.processSlot",
		//nolint:gosec // slot numbers won't exceed int64 max in practice
		trace.WithAttributes(attribute.Int64("slot", int64(slot))),
	)
	defer span.End()

	// Get the block
	block, err := w.beacon.GetBeaconBlock(ctx, xatuethv1.SlotAsString(slot))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get beacon block for slot %d", slot)
	}

	if block == nil {
		return []*xatu.DecoratedEvent{}, nil
	}

	blockIdentifier, err := GetBlockIdentifier(block, w.ctx.Wallclock())
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get block identifier for slot %d", slot)
	}

	events := make([]*xatu.DecoratedEvent, 0)

	withdrawals, err := w.getWithdrawals(ctx, block)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get withdrawals for slot %d", slot)
	}

	for _, withdrawal := range withdrawals {
		event, err := w.createEvent(ctx, withdrawal, blockIdentifier)
		if err != nil {
			w.log.WithError(err).Error("Failed to create event")

			return nil, errors.Wrapf(err, "failed to create event for withdrawal %s", withdrawal.String())
		}

		events = append(events, event)
	}

	return events, nil
}

func (w *WithdrawalDeriver) getWithdrawals(
	_ context.Context,
	block *spec.VersionedSignedBeaconBlock,
) ([]*xatuethv1.WithdrawalV2, error) {
	withdrawals := make([]*xatuethv1.WithdrawalV2, 0)

	withd, err := block.Withdrawals()
	if err != nil {
		return nil, errors.Wrap(err, "failed to obtain withdrawals")
	}

	for _, withdrawal := range withd {
		withdrawals = append(withdrawals, &xatuethv1.WithdrawalV2{
			Index:          &wrapperspb.UInt64Value{Value: uint64(withdrawal.Index)},
			ValidatorIndex: &wrapperspb.UInt64Value{Value: uint64(withdrawal.ValidatorIndex)},
			Address:        withdrawal.Address.String(),
			Amount:         &wrapperspb.UInt64Value{Value: uint64(withdrawal.Amount)},
		})
	}

	return withdrawals, nil
}

func (w *WithdrawalDeriver) createEvent(
	ctx context.Context,
	withdrawal *xatuethv1.WithdrawalV2,
	identifier *xatu.BlockIdentifier,
) (*xatu.DecoratedEvent, error) {
	// Get client metadata
	clientMeta, err := w.ctx.CreateClientMeta(ctx)
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
			Name:     xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_WITHDRAWAL,
			DateTime: timestamppb.New(time.Now()),
			Id:       uuid.New().String(),
		},
		Meta: &xatu.Meta{
			Client: metadata,
		},
		Data: &xatu.DecoratedEvent_EthV2BeaconBlockWithdrawal{
			EthV2BeaconBlockWithdrawal: withdrawal,
		},
	}

	decoratedEvent.Meta.Client.AdditionalData = &xatu.ClientMeta_EthV2BeaconBlockWithdrawal{
		EthV2BeaconBlockWithdrawal: &xatu.ClientMeta_AdditionalEthV2BeaconBlockWithdrawalData{
			Block: identifier,
		},
	}

	return decoratedEvent, nil
}
