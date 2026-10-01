package v1

import (
	"context"
	"fmt"
	"time"

	backoff "github.com/cenkalti/backoff/v5"
	"github.com/ethpandaops/go-eth2-client/spec"
	"github.com/ethpandaops/go-eth2-client/spec/phase0"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/ethpandaops/xatu/pkg/cannon/ethereum"
	"github.com/ethpandaops/xatu/pkg/cannon/iterator"
	"github.com/ethpandaops/xatu/pkg/observability"
	xatuethv1 "github.com/ethpandaops/xatu/pkg/proto/eth/v1"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

// gloasStateInput is the state of one epoch plus the chain parameters derivers need.
type gloasStateInput struct {
	epoch   phase0.Epoch
	stateID string
	state   *ethereum.GloasEpochState

	slotsPerEpoch uint64
	forkEpoch     phase0.Epoch
}

type gloasStateDeriveFunc func(in *gloasStateInput) ([]*xatu.DecoratedEvent, error)

// gloasStateDeriver is the loop shared by the derivers that read the Gloas state at each finalized epoch's first slot.
type gloasStateDeriver struct {
	log               observability.ContextualLogger
	cannonType        xatu.CannonType
	enabled           bool
	iterator          *iterator.BackfillingCheckpoint
	onEventsCallbacks []func(ctx context.Context, events []*xatu.DecoratedEvent) error
	beacon            *ethereum.BeaconNode
	clientMeta        *xatu.ClientMeta
	derive            gloasStateDeriveFunc

	epochData func(epoch phase0.Epoch) *xatu.EpochV2
	slotData  func(slot phase0.Slot) *xatu.SlotV2
}

func newGloasStateDeriver(
	log observability.ContextualLogger,
	cannonType xatu.CannonType,
	module string,
	enabled bool,
	iter *iterator.BackfillingCheckpoint,
	beacon *ethereum.BeaconNode,
	clientMeta *xatu.ClientMeta,
	derive gloasStateDeriveFunc,
) *gloasStateDeriver {
	return &gloasStateDeriver{
		log: log.WithFields(logrus.Fields{
			moduleLogField: "cannon/event/beacon/eth/v1/" + module,
			typeLogField:   cannonType.String(),
		}),
		cannonType: cannonType,
		enabled:    enabled,
		iterator:   iter,
		beacon:     beacon,
		clientMeta: clientMeta,
		derive:     derive,
		epochData: func(epoch phase0.Epoch) *xatu.EpochV2 {
			epochInfo := beacon.Metadata().Wallclock().Epochs().FromNumber(uint64(epoch))

			return &xatu.EpochV2{
				Number:        wrapperspb.UInt64(uint64(epoch)),
				StartDateTime: timestamppb.New(epochInfo.TimeWindow().Start()),
			}
		},
		slotData: func(slot phase0.Slot) *xatu.SlotV2 {
			slotInfo := beacon.Metadata().Wallclock().Slots().FromNumber(uint64(slot))

			return &xatu.SlotV2{
				Number:        wrapperspb.UInt64(uint64(slot)),
				StartDateTime: timestamppb.New(slotInfo.TimeWindow().Start()),
			}
		},
	}
}

func (g *gloasStateDeriver) CannonType() xatu.CannonType {
	return g.cannonType
}

func (g *gloasStateDeriver) Name() string {
	return g.cannonType.String()
}

// ActivationFork is Gloas, the first fork with these state fields.
func (g *gloasStateDeriver) ActivationFork() spec.DataVersion {
	return spec.DataVersionGloas
}

func (g *gloasStateDeriver) OnEventsDerived(_ context.Context, fn func(ctx context.Context, events []*xatu.DecoratedEvent) error) {
	g.onEventsCallbacks = append(g.onEventsCallbacks, fn)
}

func (g *gloasStateDeriver) Start(ctx context.Context) error {
	g.log.WithField("enabled", g.enabled).WithContext(ctx).Info("Starting " + g.Name() + " deriver")

	if !g.enabled {
		g.log.WithContext(ctx).Info("Deriver disabled")

		return nil
	}

	if err := g.iterator.Start(ctx, g.ActivationFork()); err != nil {
		return errors.Wrap(err, "failed to start iterator")
	}

	g.run(ctx)

	return nil
}

func (g *gloasStateDeriver) Stop(_ context.Context) error {
	return nil
}

func (g *gloasStateDeriver) run(rctx context.Context) {
	bo := backoff.NewExponentialBackOff()
	bo.MaxInterval = 3 * time.Minute

	tracer := observability.Tracer()

	for {
		select {
		case <-rctx.Done():
			return
		default:
			operation := func() (string, error) {
				ctx, span := tracer.Start(rctx, fmt.Sprintf("Derive %s", g.Name()),
					trace.WithAttributes(
						attribute.String("network", string(g.beacon.Metadata().Network.Name))),
				)
				defer span.End()

				time.Sleep(100 * time.Millisecond)

				if err := g.beacon.Synced(ctx); err != nil {
					span.SetStatus(codes.Error, err.Error())

					return "", err
				}

				position, err := g.iterator.Next(ctx)
				if err != nil {
					span.SetStatus(codes.Error, err.Error())

					return "", err
				}

				events, err := g.processEpoch(ctx, position.Next)
				if err != nil {
					g.log.WithError(err).WithField("epoch", position.Next).WithContext(ctx).Error("Failed to process epoch")

					span.SetStatus(codes.Error, err.Error())

					return "", err
				}

				for _, fn := range g.onEventsCallbacks {
					if err := fn(ctx, events); err != nil {
						span.SetStatus(codes.Error, err.Error())

						return "", errors.Wrap(err, "failed to send events")
					}
				}

				if err := g.iterator.UpdateLocation(ctx, position.Next, position.Direction); err != nil {
					span.SetStatus(codes.Error, err.Error())

					return "", err
				}

				bo.Reset()

				return "", nil
			}

			retryOpts := []backoff.RetryOption{
				backoff.WithBackOff(bo),
				backoff.WithNotify(func(err error, timer time.Duration) {
					g.log.WithError(err).WithField("next_attempt", timer).WithContext(rctx).Warn("Failed to process")
				}),
			}

			if _, err := backoff.Retry(rctx, operation, retryOpts...); err != nil {
				g.log.WithError(err).WithContext(rctx).Warn("Failed to process")
			}
		}
	}
}

func (g *gloasStateDeriver) processEpoch(ctx context.Context, epoch phase0.Epoch) ([]*xatu.DecoratedEvent, error) {
	ctx, span := observability.Tracer().Start(ctx,
		"gloasStateDeriver.processEpoch",
		trace.WithAttributes(
			attribute.Int64("epoch", int64(epoch)), //nolint:gosec // epoch fits int64
			attribute.String("cannon_type", g.Name()),
		),
	)
	defer span.End()

	slotsPerEpoch, err := g.slotsPerEpoch()
	if err != nil {
		return nil, err
	}

	boundarySlot := phase0.Slot(uint64(epoch) * slotsPerEpoch)
	stateID := xatuethv1.SlotAsString(boundarySlot)

	state, err := g.beacon.GetGloasEpochState(ctx, stateID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to fetch beacon state")
	}

	if state.Slot != boundarySlot {
		return nil, fmt.Errorf("beacon state %s is at slot %d, expected %d", stateID, state.Slot, boundarySlot)
	}

	forkEpoch, err := g.beacon.Metadata().Spec.ForkEpochs.GetByName(spec.DataVersionGloas.String())
	if err != nil {
		return nil, errors.Wrap(err, "failed to get gloas fork epoch")
	}

	return g.derive(&gloasStateInput{
		epoch:         epoch,
		stateID:       stateID,
		state:         state,
		slotsPerEpoch: slotsPerEpoch,
		forkEpoch:     forkEpoch.Epoch,
	})
}

func (g *gloasStateDeriver) slotsPerEpoch() (uint64, error) {
	sp, err := g.beacon.Node().Spec()
	if err != nil {
		return 0, errors.Wrap(err, "failed to fetch spec")
	}

	return uint64(sp.SlotsPerEpoch), nil
}

// newEvent returns an event with its own copy of the client metadata.
func (g *gloasStateDeriver) newEvent(name xatu.Event_Name) (*xatu.DecoratedEvent, error) {
	metadata, ok := proto.Clone(g.clientMeta).(*xatu.ClientMeta)
	if !ok {
		return nil, errors.New("failed to clone client metadata")
	}

	return &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     name,
			DateTime: timestamppb.New(time.Now()),
			Id:       uuid.New().String(),
		},
		Meta: &xatu.Meta{
			Client: metadata,
		},
	}, nil
}
