package v2

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

const (
	ExecutionRequestBuilderDepositDeriverName = xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_EXECUTION_REQUEST_BUILDER_DEPOSIT
)

// ExecutionRequestBuilderDepositDeriverConfig configures the cannon deriver
// that emits the EIP-8282 builder deposit requests of Gloas+ blocks.
type ExecutionRequestBuilderDepositDeriverConfig struct {
	Enabled  bool                                 `yaml:"enabled" default:"true"`
	Iterator iterator.BackfillingCheckpointConfig `yaml:"iterator"`
}

// ExecutionRequestBuilderDepositDeriver derives
// BEACON_API_ETH_V2_BEACON_BLOCK_EXECUTION_REQUEST_BUILDER_DEPOSIT events from
// the builder deposit requests in a Gloas+ payload envelope. A withheld
// payload carries no requests and produces no events.
type ExecutionRequestBuilderDepositDeriver struct {
	log               observability.ContextualLogger
	cfg               *ExecutionRequestBuilderDepositDeriverConfig
	iterator          *iterator.BackfillingCheckpoint
	onEventsCallbacks []func(ctx context.Context, events []*xatu.DecoratedEvent) error
	beacon            *ethereum.BeaconNode
	clientMeta        *xatu.ClientMeta
}

func NewExecutionRequestBuilderDepositDeriver(log observability.ContextualLogger, config *ExecutionRequestBuilderDepositDeriverConfig, iter *iterator.BackfillingCheckpoint, beacon *ethereum.BeaconNode, clientMeta *xatu.ClientMeta) *ExecutionRequestBuilderDepositDeriver {
	return &ExecutionRequestBuilderDepositDeriver{
		log: log.WithFields(logrus.Fields{
			moduleLogField: "cannon/event/beacon/eth/v2/execution_request_builder_deposit",
			typeLogField:   ExecutionRequestBuilderDepositDeriverName.String(),
		}),
		cfg:        config,
		iterator:   iter,
		beacon:     beacon,
		clientMeta: clientMeta,
	}
}

func (b *ExecutionRequestBuilderDepositDeriver) CannonType() xatu.CannonType {
	return ExecutionRequestBuilderDepositDeriverName
}

func (b *ExecutionRequestBuilderDepositDeriver) Name() string {
	return ExecutionRequestBuilderDepositDeriverName.String()
}

// ActivationFork pegs this deriver to Gloas, builder requests (EIP-8282) did
// not exist on prior forks.
func (b *ExecutionRequestBuilderDepositDeriver) ActivationFork() spec.DataVersion {
	return spec.DataVersionGloas
}

func (b *ExecutionRequestBuilderDepositDeriver) OnEventsDerived(ctx context.Context, fn func(ctx context.Context, events []*xatu.DecoratedEvent) error) {
	b.onEventsCallbacks = append(b.onEventsCallbacks, fn)
}

func (b *ExecutionRequestBuilderDepositDeriver) Start(ctx context.Context) error {
	if !b.cfg.Enabled {
		b.log.WithContext(ctx).Info("Execution request builder deposit deriver disabled")

		return nil
	}

	b.log.WithContext(ctx).Info("Execution request builder deposit deriver enabled")

	if err := b.iterator.Start(ctx, b.ActivationFork()); err != nil {
		return errors.Wrap(err, "failed to start iterator")
	}

	// Start our main loop
	b.run(ctx)

	return nil
}

func (b *ExecutionRequestBuilderDepositDeriver) Stop(ctx context.Context) error {
	return nil
}

func (b *ExecutionRequestBuilderDepositDeriver) run(rctx context.Context) {
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

				// Get the next position
				position, err := b.iterator.Next(ctx)
				if err != nil {
					return "", err
				}

				// Process the epoch
				events, err := b.processEpoch(ctx, position.Next)
				if err != nil {
					b.log.WithError(err).WithContext(ctx).Error("Failed to process epoch")

					return "", err
				}

				// Look ahead
				b.lookAhead(ctx, position.LookAheads)

				for _, fn := range b.onEventsCallbacks {
					if errr := fn(ctx, events); errr != nil {
						return "", errors.Wrapf(errr, "failed to send events")
					}
				}

				// Update our location
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

func (b *ExecutionRequestBuilderDepositDeriver) processEpoch(ctx context.Context, epoch phase0.Epoch) ([]*xatu.DecoratedEvent, error) {
	ctx, span := observability.Tracer().Start(ctx,
		"ExecutionRequestBuilderDepositDeriver.processEpoch",
		//nolint:gosec // epoch value will never overflow int64
		trace.WithAttributes(attribute.Int64("epoch", int64(epoch))),
	)
	defer span.End()

	sp, err := b.beacon.Node().Spec()
	if err != nil {
		return nil, errors.Wrap(err, "failed to obtain spec")
	}

	allEvents := []*xatu.DecoratedEvent{}

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

func (b *ExecutionRequestBuilderDepositDeriver) processSlot(ctx context.Context, slot phase0.Slot) ([]*xatu.DecoratedEvent, error) {
	ctx, span := observability.Tracer().Start(ctx,
		"ExecutionRequestBuilderDepositDeriver.processSlot",
		//nolint:gosec // slot value will never overflow int64
		trace.WithAttributes(attribute.Int64("slot", int64(slot))),
	)
	defer span.End()

	// Get the block
	block, err := b.beacon.GetBeaconBlock(ctx, xatuethv1.SlotAsString(slot))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get beacon block for slot %d", slot)
	}

	if block == nil {
		return []*xatu.DecoratedEvent{}, nil
	}

	blockIdentifier, err := GetBlockIdentifier(block, b.beacon.Metadata().Wallclock())
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get block identifier for slot %d", slot)
	}

	return b.deriveEvents(ctx, b.beacon, block, blockIdentifier)
}

// deriveEvents turns the builder deposit requests of a block's payload
// envelope into events.
func (b *ExecutionRequestBuilderDepositDeriver) deriveEvents(
	ctx context.Context,
	envelopes executionPayloadEnvelopeProvider,
	block *spec.VersionedSignedBeaconBlock,
	identifier *xatu.BlockIdentifier,
) ([]*xatu.DecoratedEvent, error) {
	payload, err := getEnvelopeRequests(ctx, envelopes, block)
	if err != nil {
		return nil, err
	}

	events := []*xatu.DecoratedEvent{}

	if payload.requests == nil {
		return events, nil
	}

	for index, deposit := range payload.requests.BuilderDeposits {
		event, err := b.createEvent(ctx, xatuethv1.NewGloasBuilderDepositRequestFromGloas(deposit), uint64(index), identifier, payload)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to create event for execution request builder deposit %d", index)
		}

		events = append(events, event)
	}

	return events, nil
}

// lookAhead attempts to pre-load any blocks that might be required for the epochs that are coming up.
func (b *ExecutionRequestBuilderDepositDeriver) lookAhead(ctx context.Context, epochs []phase0.Epoch) {
	sp, err := b.beacon.Node().Spec()
	if err != nil {
		b.log.WithError(err).WithContext(ctx).Warn("Failed to look ahead at epoch")

		return
	}

	for _, epoch := range epochs {
		for i := uint64(0); i <= uint64(sp.SlotsPerEpoch-1); i++ {
			slot := phase0.Slot(i + uint64(epoch)*uint64(sp.SlotsPerEpoch))

			// Add the block to the preload queue so it's available when we need it
			b.beacon.LazyLoadBeaconBlock(xatuethv1.SlotAsString(slot))
		}
	}
}

func (b *ExecutionRequestBuilderDepositDeriver) createEvent(_ context.Context, deposit *xatuethv1.GloasBuilderDepositRequest, positionInBlock uint64, identifier *xatu.BlockIdentifier, payload envelopeRequests) (*xatu.DecoratedEvent, error) {
	// Make a clone of the metadata
	metadata, ok := proto.Clone(b.clientMeta).(*xatu.ClientMeta)
	if !ok {
		return nil, errors.New("failed to clone client metadata")
	}

	decoratedEvent := &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_EXECUTION_REQUEST_BUILDER_DEPOSIT,
			DateTime: timestamppb.New(time.Now()),
			Id:       uuid.New().String(),
		},
		Meta: &xatu.Meta{
			Client: metadata,
		},
		Data: &xatu.DecoratedEvent_EthV2BeaconBlockExecutionRequestBuilderDeposit{
			EthV2BeaconBlockExecutionRequestBuilderDeposit: deposit,
		},
	}

	decoratedEvent.Meta.Client.AdditionalData = &xatu.ClientMeta_EthV2BeaconBlockExecutionRequestBuilderDeposit{
		EthV2BeaconBlockExecutionRequestBuilderDeposit: &xatu.ClientMeta_AdditionalEthV2BeaconBlockExecutionRequestBuilderDepositData{
			Block:           identifier,
			PositionInBlock: &wrapperspb.UInt64Value{Value: positionInBlock},
			BlockNumber:     &wrapperspb.UInt64Value{Value: payload.blockNumber},
			BlockHash:       payload.blockHash,
		},
	}

	return decoratedEvent, nil
}
