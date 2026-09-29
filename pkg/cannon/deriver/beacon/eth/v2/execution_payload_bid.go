package v2

import (
	"context"
	"fmt"
	"time"

	backoff "github.com/cenkalti/backoff/v5"
	"github.com/ethpandaops/go-eth2-client/spec"
	"github.com/ethpandaops/go-eth2-client/spec/phase0"
	"github.com/ethpandaops/xatu/pkg/cannon/ethereum"
	"github.com/ethpandaops/xatu/pkg/cannon/iterator"
	"github.com/ethpandaops/xatu/pkg/observability"
	xatuethv1 "github.com/ethpandaops/xatu/pkg/proto/eth/v1"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	ExecutionPayloadBidDeriverName = xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_EXECUTION_PAYLOAD_BID
)

// ExecutionPayloadBidDeriverConfig configures the cannon deriver that emits
// the winning execution payload bid from each Gloas+ beacon block.
type ExecutionPayloadBidDeriverConfig struct {
	Enabled  bool                                 `yaml:"enabled" default:"true"`
	Iterator iterator.BackfillingCheckpointConfig `yaml:"iterator"`
}

// ExecutionPayloadBidDeriver derives BEACON_API_ETH_V2_BEACON_BLOCK_EXECUTION_PAYLOAD_BID
// events from `block.Body.SignedExecutionPayloadBid` on Gloas+ blocks. One
// bid per block (the proposer's chosen builder).
type ExecutionPayloadBidDeriver struct {
	log               observability.ContextualLogger
	cfg               *ExecutionPayloadBidDeriverConfig
	iterator          *iterator.BackfillingCheckpoint
	onEventsCallbacks []func(ctx context.Context, events []*xatu.DecoratedEvent) error
	beacon            *ethereum.BeaconNode
	clientMeta        *xatu.ClientMeta
}

func NewExecutionPayloadBidDeriver(log observability.ContextualLogger, config *ExecutionPayloadBidDeriverConfig, iter *iterator.BackfillingCheckpoint, beacon *ethereum.BeaconNode, clientMeta *xatu.ClientMeta) *ExecutionPayloadBidDeriver {
	return &ExecutionPayloadBidDeriver{
		log: log.WithFields(logrus.Fields{
			moduleLogField: "cannon/event/beacon/eth/v2/execution_payload_bid",
			typeLogField:   ExecutionPayloadBidDeriverName.String(),
		}),
		cfg:        config,
		iterator:   iter,
		beacon:     beacon,
		clientMeta: clientMeta,
	}
}

func (b *ExecutionPayloadBidDeriver) CannonType() xatu.CannonType {
	return ExecutionPayloadBidDeriverName
}

func (b *ExecutionPayloadBidDeriver) Name() string {
	return ExecutionPayloadBidDeriverName.String()
}

// ActivationFork pegs this deriver to Gloas — execution payload bids didn't
// exist on prior forks.
func (b *ExecutionPayloadBidDeriver) ActivationFork() spec.DataVersion {
	return spec.DataVersionGloas
}

func (b *ExecutionPayloadBidDeriver) OnEventsDerived(ctx context.Context, fn func(ctx context.Context, events []*xatu.DecoratedEvent) error) {
	b.onEventsCallbacks = append(b.onEventsCallbacks, fn)
}

func (b *ExecutionPayloadBidDeriver) Start(ctx context.Context) error {
	if !b.cfg.Enabled {
		b.log.WithContext(ctx).Info("Execution payload bid deriver disabled")

		return nil
	}

	b.log.WithContext(ctx).Info("Execution payload bid deriver enabled")

	if err := b.iterator.Start(ctx, b.ActivationFork()); err != nil {
		return errors.Wrap(err, "failed to start iterator")
	}

	b.run(ctx)

	return nil
}

func (b *ExecutionPayloadBidDeriver) Stop(ctx context.Context) error {
	return nil
}

func (b *ExecutionPayloadBidDeriver) run(rctx context.Context) {
	bo := backoff.NewExponentialBackOff()
	bo.MaxInterval = 3 * time.Minute

	for {
		select {
		case <-rctx.Done():
			return
		default:
			operation := func() (string, error) {
				ctx, span := observability.Tracer().Start(rctx, fmt.Sprintf("Derive %s", b.Name()),
					trace.WithAttributes(
						attribute.String("network", string(b.beacon.Metadata().Network.Name))),
				)
				defer span.End()

				time.Sleep(100 * time.Millisecond)

				if err := b.beacon.Synced(ctx); err != nil {
					return "", err
				}

				position, err := b.iterator.Next(ctx)
				if err != nil {
					return "", err
				}

				events, err := b.processEpoch(ctx, position.Next)
				if err != nil {
					b.log.WithError(err).WithContext(ctx).Error("Failed to process epoch")

					return "", err
				}

				b.lookAhead(ctx, position.LookAheads)

				for _, fn := range b.onEventsCallbacks {
					if err := fn(ctx, events); err != nil {
						return "", errors.Wrap(err, "failed to send events")
					}
				}

				if err := b.iterator.UpdateLocation(ctx, position.Next, position.Direction); err != nil {
					return "", err
				}

				bo.Reset()

				return "", nil
			}

			retryOpts := []backoff.RetryOption{
				backoff.WithBackOff(bo),
				backoff.WithNotify(func(err error, timer time.Duration) {
					b.log.WithError(err).WithField("next_attempt", timer).WithContext(rctx).Warn("Failed to process")
				}),
			}
			if _, err := backoff.Retry(rctx, operation, retryOpts...); err != nil {
				b.log.WithError(err).WithContext(rctx).Warn("Failed to process")
			}
		}
	}
}

func (b *ExecutionPayloadBidDeriver) processEpoch(ctx context.Context, epoch phase0.Epoch) ([]*xatu.DecoratedEvent, error) {
	ctx, span := observability.Tracer().Start(ctx,
		"ExecutionPayloadBidDeriver.processEpoch",
		//nolint:gosec // epoch fits int64
		trace.WithAttributes(attribute.Int64("epoch", int64(epoch))),
	)
	defer span.End()

	sp, err := b.beacon.Node().Spec()
	if err != nil {
		return nil, errors.Wrap(err, "failed to obtain spec")
	}

	allEvents := make([]*xatu.DecoratedEvent, 0, sp.SlotsPerEpoch)

	for i := uint64(0); i <= uint64(sp.SlotsPerEpoch-1); i++ {
		slot := phase0.Slot(i + uint64(epoch)*uint64(sp.SlotsPerEpoch))

		events, err := b.processSlot(ctx, slot)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to process slot %d", slot)
		}

		allEvents = append(allEvents, events...)
	}

	return allEvents, nil
}

func (b *ExecutionPayloadBidDeriver) processSlot(ctx context.Context, slot phase0.Slot) ([]*xatu.DecoratedEvent, error) {
	ctx, span := observability.Tracer().Start(ctx,
		"ExecutionPayloadBidDeriver.processSlot",
		//nolint:gosec // slot fits int64
		trace.WithAttributes(attribute.Int64("slot", int64(slot))),
	)
	defer span.End()

	block, err := b.beacon.GetBeaconBlock(ctx, xatuethv1.SlotAsString(slot))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get beacon block for slot %d", slot)
	}

	if block == nil {
		return []*xatu.DecoratedEvent{}, nil
	}

	// Execution payload bids only exist on Gloas+ blocks.
	if block.Version < spec.DataVersionGloas {
		return []*xatu.DecoratedEvent{}, nil
	}

	if block.Gloas == nil || block.Gloas.Message == nil || block.Gloas.Message.Body == nil {
		return []*xatu.DecoratedEvent{}, nil
	}

	rawBid := block.Gloas.Message.Body.SignedExecutionPayloadBid
	if rawBid == nil || rawBid.Message == nil {
		return []*xatu.DecoratedEvent{}, nil
	}

	blockIdentifier, err := GetBlockIdentifier(block, b.beacon.Metadata().Wallclock())
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get block identifier for slot %d", slot)
	}

	bid := xatuethv1.NewSignedExecutionPayloadBidFromGloas(rawBid)
	if bid == nil {
		return []*xatu.DecoratedEvent{}, nil
	}

	event, err := b.createEvent(ctx, bid, blockIdentifier)
	if err != nil {
		b.log.WithError(err).WithContext(ctx).Error("Failed to create event")

		return nil, errors.Wrapf(err, "failed to create execution payload bid event for slot %d", slot)
	}

	return []*xatu.DecoratedEvent{event}, nil
}

// lookAhead pre-loads upcoming epoch blocks so the iterator's next pass has them ready.
func (b *ExecutionPayloadBidDeriver) lookAhead(ctx context.Context, epochs []phase0.Epoch) {
	_, span := observability.Tracer().Start(ctx,
		"ExecutionPayloadBidDeriver.lookAhead",
	)
	defer span.End()

	if epochs == nil {
		return
	}

	sp, err := b.beacon.Node().Spec()
	if err != nil {
		b.log.WithError(err).WithContext(ctx).Warn("Failed to look ahead at epoch")

		return
	}

	for _, epoch := range epochs {
		for i := uint64(0); i <= uint64(sp.SlotsPerEpoch-1); i++ {
			slot := phase0.Slot(i + uint64(epoch)*uint64(sp.SlotsPerEpoch))

			b.beacon.LazyLoadBeaconBlock(xatuethv1.SlotAsString(slot))
		}
	}
}

func (b *ExecutionPayloadBidDeriver) createEvent(ctx context.Context, bid *xatuethv1.SignedExecutionPayloadBid, identifier *xatu.BlockIdentifier) (*xatu.DecoratedEvent, error) {
	metadata, ok := proto.Clone(b.clientMeta).(*xatu.ClientMeta)
	if !ok {
		return nil, errors.New("failed to clone client metadata")
	}

	decoratedEvent := &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_EXECUTION_PAYLOAD_BID,
			DateTime: timestamppb.New(time.Now()),
			Id:       uuid.New().String(),
		},
		Meta: &xatu.Meta{
			Client: metadata,
		},
		Data: &xatu.DecoratedEvent_EthV2BeaconBlockExecutionPayloadBid{
			EthV2BeaconBlockExecutionPayloadBid: bid,
		},
	}

	decoratedEvent.Meta.Client.AdditionalData = &xatu.ClientMeta_EthV2BeaconBlockExecutionPayloadBid{
		EthV2BeaconBlockExecutionPayloadBid: &xatu.ClientMeta_AdditionalEthV2BeaconBlockExecutionPayloadBidData{
			Block: identifier,
		},
	}

	return decoratedEvent, nil
}
