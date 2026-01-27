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
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// GenericDeriver is a universal deriver implementation that uses the registry
// pattern to handle all deriver types with minimal boilerplate.
type GenericDeriver struct {
	log               logrus.FieldLogger
	enabled           bool
	spec              *DeriverSpec
	iterator          iterator.Iterator
	beacon            cldata.BeaconClient
	ctx               cldata.ContextProvider
	onEventsCallbacks []func(ctx context.Context, events []*xatu.DecoratedEvent) error
}

// NewGenericDeriver creates a new generic deriver from a specification.
func NewGenericDeriver(
	log logrus.FieldLogger,
	deriverSpec *DeriverSpec,
	enabled bool,
	iter iterator.Iterator,
	beacon cldata.BeaconClient,
	ctxProvider cldata.ContextProvider,
) *GenericDeriver {
	return &GenericDeriver{
		log: log.WithFields(logrus.Fields{
			"module": "cldata/deriver/" + deriverSpec.Name,
			"type":   deriverSpec.CannonType.String(),
		}),
		enabled:  enabled,
		spec:     deriverSpec,
		iterator: iter,
		beacon:   beacon,
		ctx:      ctxProvider,
	}
}

// CannonType returns the cannon type of the deriver.
func (d *GenericDeriver) CannonType() xatu.CannonType {
	return d.spec.CannonType
}

// Name returns the name of the deriver.
func (d *GenericDeriver) Name() string {
	return d.spec.CannonType.String()
}

// ActivationFork returns the fork at which the deriver is activated.
func (d *GenericDeriver) ActivationFork() spec.DataVersion {
	return d.spec.ActivationFork
}

// OnEventsDerived registers a callback for when events are derived.
func (d *GenericDeriver) OnEventsDerived(
	_ context.Context,
	fn func(ctx context.Context, events []*xatu.DecoratedEvent) error,
) {
	d.onEventsCallbacks = append(d.onEventsCallbacks, fn)
}

// Start starts the deriver.
func (d *GenericDeriver) Start(ctx context.Context) error {
	if !d.enabled {
		d.log.Info("Deriver disabled")

		return nil
	}

	d.log.Info("Deriver enabled")

	if err := d.iterator.Start(ctx, d.ActivationFork()); err != nil {
		return errors.Wrap(err, "failed to start iterator")
	}

	d.run(ctx)

	return nil
}

// Stop stops the deriver.
func (d *GenericDeriver) Stop(_ context.Context) error {
	return nil
}

func (d *GenericDeriver) run(rctx context.Context) {
	bo := backoff.NewExponentialBackOff()
	bo.MaxInterval = 3 * time.Minute

	tracer := observability.Tracer()

	for {
		select {
		case <-rctx.Done():
			return
		default:
			operation := func() (string, error) {
				ctx, span := tracer.Start(rctx, fmt.Sprintf("Derive %s", d.Name()),
					trace.WithAttributes(
						attribute.String("network", d.ctx.NetworkName())),
				)
				defer span.End()

				time.Sleep(100 * time.Millisecond)

				if err := d.beacon.Synced(ctx); err != nil {
					span.SetStatus(codes.Error, err.Error())

					return "", err
				}

				position, err := d.iterator.Next(ctx)
				if err != nil {
					span.SetStatus(codes.Error, err.Error())

					return "", err
				}

				d.lookAhead(ctx, position.LookAheadEpochs)

				events, err := d.processEpoch(ctx, position.Epoch)
				if err != nil {
					d.log.WithError(err).WithField("epoch", position.Epoch).Error("Failed to process epoch")
					span.SetStatus(codes.Error, err.Error())

					return "", err
				}

				span.AddEvent("Epoch processing complete. Sending events...")

				for _, fn := range d.onEventsCallbacks {
					if err := fn(ctx, events); err != nil {
						span.SetStatus(codes.Error, err.Error())

						return "", errors.Wrap(err, "failed to send events")
					}
				}

				span.AddEvent("Events sent. Updating location...")

				if err := d.iterator.UpdateLocation(ctx, position); err != nil {
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
					d.log.WithError(err).WithField("next_attempt", timer).Warn("Failed to process")
				}),
			}
			if _, err := backoff.Retry(rctx, operation, retryOpts...); err != nil {
				d.log.WithError(err).Warn("Failed to process")
			}
		}
	}
}

func (d *GenericDeriver) lookAhead(ctx context.Context, epochs []phase0.Epoch) {
	_, span := observability.Tracer().Start(ctx, d.Name()+".lookAhead")
	defer span.End()

	sp, err := d.beacon.Node().Spec()
	if err != nil {
		d.log.WithError(err).Warn("Failed to look ahead at epoch")

		return
	}

	for _, epoch := range epochs {
		for i := uint64(0); i <= uint64(sp.SlotsPerEpoch-1); i++ {
			slot := phase0.Slot(i + uint64(epoch)*uint64(sp.SlotsPerEpoch))
			d.beacon.LazyLoadBeaconBlock(xatuethv1.SlotAsString(slot))
		}
	}
}

func (d *GenericDeriver) processEpoch(ctx context.Context, epoch phase0.Epoch) ([]*xatu.DecoratedEvent, error) {
	ctx, span := observability.Tracer().Start(ctx,
		d.Name()+".processEpoch",
		//nolint:gosec // epoch numbers won't exceed int64 max in practice
		trace.WithAttributes(attribute.Int64("epoch", int64(epoch))),
	)
	defer span.End()

	switch d.spec.Mode {
	case ProcessingModeSlot:
		return d.processEpochBySlot(ctx, epoch)
	case ProcessingModeEpoch:
		return d.spec.EpochProcessor(ctx, epoch, d.beacon, d.ctx)
	default:
		return nil, fmt.Errorf("unknown processing mode: %d", d.spec.Mode)
	}
}

func (d *GenericDeriver) processEpochBySlot(
	ctx context.Context,
	epoch phase0.Epoch,
) ([]*xatu.DecoratedEvent, error) {
	sp, err := d.beacon.Node().Spec()
	if err != nil {
		return nil, errors.Wrap(err, "failed to obtain spec")
	}

	allEvents := make([]*xatu.DecoratedEvent, 0)

	for i := uint64(0); i <= uint64(sp.SlotsPerEpoch-1); i++ {
		slot := phase0.Slot(i + uint64(epoch)*uint64(sp.SlotsPerEpoch))

		events, err := d.processSlot(ctx, slot)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to process slot %d", slot)
		}

		allEvents = append(allEvents, events...)
	}

	return allEvents, nil
}

func (d *GenericDeriver) processSlot(ctx context.Context, slot phase0.Slot) ([]*xatu.DecoratedEvent, error) {
	ctx, span := observability.Tracer().Start(ctx,
		d.Name()+".processSlot",
		//nolint:gosec // slot numbers won't exceed int64 max in practice
		trace.WithAttributes(attribute.Int64("slot", int64(slot))),
	)
	defer span.End()

	block, err := d.beacon.GetBeaconBlock(ctx, xatuethv1.SlotAsString(slot))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get beacon block for slot %d", slot)
	}

	if block == nil {
		return []*xatu.DecoratedEvent{}, nil
	}

	blockIdentifier, err := GetBlockIdentifier(block, d.ctx.Wallclock())
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get block identifier for slot %d", slot)
	}

	return d.spec.BlockExtractor(ctx, block, blockIdentifier, d.beacon, d.ctx)
}

// Verify GenericDeriver implements the EventDeriver interface.
var _ EventDeriver = (*GenericDeriver)(nil)
