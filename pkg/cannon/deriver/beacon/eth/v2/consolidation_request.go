package v2

import (
	"context"
	"fmt"
	"time"

	"github.com/attestantio/go-eth2-client/spec"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	backoff "github.com/cenkalti/backoff/v5"
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
	"google.golang.org/protobuf/types/known/wrapperspb"
)

const (
	ConsolidationRequestDeriverName = xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_CONSOLIDATION_REQUEST
)

type ConsolidationRequestDeriverConfig struct {
	Enabled  bool                                 `yaml:"enabled" default:"true"`
	Iterator iterator.BackfillingCheckpointConfig `yaml:"iterator"`
}

type ConsolidationRequestDeriver struct {
	log               logrus.FieldLogger
	cfg               *ConsolidationRequestDeriverConfig
	iterator          *iterator.BackfillingCheckpoint
	onEventsCallbacks []func(ctx context.Context, events []*xatu.DecoratedEvent) error
	beacon            *ethereum.BeaconNode
	clientMeta        *xatu.ClientMeta
}

func NewConsolidationRequestDeriver(log logrus.FieldLogger, config *ConsolidationRequestDeriverConfig, iter *iterator.BackfillingCheckpoint, beacon *ethereum.BeaconNode, clientMeta *xatu.ClientMeta) *ConsolidationRequestDeriver {
	return &ConsolidationRequestDeriver{
		log: log.WithFields(logrus.Fields{
			"module": "cannon/event/beacon/eth/v2/consolidation_request",
			"type":   ConsolidationRequestDeriverName.String(),
		}),
		cfg:        config,
		iterator:   iter,
		beacon:     beacon,
		clientMeta: clientMeta,
	}
}

func (b *ConsolidationRequestDeriver) CannonType() xatu.CannonType {
	return ConsolidationRequestDeriverName
}

func (b *ConsolidationRequestDeriver) Name() string {
	return ConsolidationRequestDeriverName.String()
}

func (b *ConsolidationRequestDeriver) ActivationFork() spec.DataVersion {
	return spec.DataVersionElectra
}

func (b *ConsolidationRequestDeriver) OnEventsDerived(ctx context.Context, fn func(ctx context.Context, events []*xatu.DecoratedEvent) error) {
	b.onEventsCallbacks = append(b.onEventsCallbacks, fn)
}

func (b *ConsolidationRequestDeriver) Start(ctx context.Context) error {
	if !b.cfg.Enabled {
		b.log.Info("ConsolidationRequest deriver disabled")

		return nil
	}

	b.log.Info("ConsolidationRequest deriver enabled")

	if err := b.iterator.Start(ctx, b.ActivationFork()); err != nil {
		return errors.Wrap(err, "failed to start iterator")
	}

	// Start our main loop
	b.run(ctx)

	return nil
}

func (b *ConsolidationRequestDeriver) Stop(ctx context.Context) error {
	return nil
}

func (b *ConsolidationRequestDeriver) run(rctx context.Context) {
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
					b.log.WithError(err).Error("Failed to process epoch")

					return "", err
				}

				// Look ahead
				b.lookAhead(ctx, position.LookAheads)

				// Send the events
				for _, fn := range b.onEventsCallbacks {
					if err := fn(ctx, events); err != nil {
						return "", errors.Wrap(err, "failed to send events")
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
					b.log.WithError(err).WithField("next_attempt", timer).Warn("Failed to process")
				}),
			}

			if _, err := backoff.Retry(rctx, operation, retryOpts...); err != nil {
				b.log.WithError(err).Warn("Failed to process")
			}
		}
	}
}

// lookAhead attempts to pre-load any blocks that might be required for the epochs that are coming up.
func (b *ConsolidationRequestDeriver) lookAhead(ctx context.Context, epochs []phase0.Epoch) {
	_, span := observability.Tracer().Start(ctx,
		"ConsolidationRequestDeriver.lookAhead",
	)
	defer span.End()

	sp, err := b.beacon.Node().Spec()
	if err != nil {
		b.log.WithError(err).Warn("Failed to look ahead at epoch")

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

func (b *ConsolidationRequestDeriver) processEpoch(ctx context.Context, epoch phase0.Epoch) ([]*xatu.DecoratedEvent, error) {
	ctx, span := observability.Tracer().Start(ctx,
		"ConsolidationRequestDeriver.processEpoch",
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

func (b *ConsolidationRequestDeriver) processSlot(ctx context.Context, slot phase0.Slot) ([]*xatu.DecoratedEvent, error) {
	ctx, span := observability.Tracer().Start(ctx,
		"ConsolidationRequestDeriver.processSlot",
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

	events := []*xatu.DecoratedEvent{}

	consolidationRequests, err := b.getConsolidationRequests(ctx, block)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get consolidation requests for block %s", blockIdentifier.String())
	}

	for i, consolidationRequest := range consolidationRequests {
		position := uint32(i)
		event, err := b.createEvent(ctx, consolidationRequest, blockIdentifier, position)
		if err != nil {
			b.log.WithError(err).Error("Failed to create event")

			return nil, errors.Wrapf(err, "failed to create event for consolidation request %d", i)
		}

		events = append(events, event)
	}

	return events, nil
}

func (b *ConsolidationRequestDeriver) getConsolidationRequests(ctx context.Context, block *spec.VersionedSignedBeaconBlock) ([]*xatuethv1.ElectraExecutionRequestConsolidation, error) {
	consolidationRequests := []*xatuethv1.ElectraExecutionRequestConsolidation{}

	// Get execution requests from the beacon block
	executionRequests, err := block.ExecutionRequests()
	if err != nil {
		// If execution requests are not available (pre-Electra), return empty slice
		return consolidationRequests, nil
	}

	// Extract consolidation requests from execution requests
	for _, request := range executionRequests.Consolidations {
		consolidationRequests = append(consolidationRequests, &xatuethv1.ElectraExecutionRequestConsolidation{
			SourceAddress: &wrapperspb.StringValue{Value: request.SourceAddress.String()},
			SourcePubkey:  &wrapperspb.StringValue{Value: request.SourcePubkey.String()},
			TargetPubkey:  &wrapperspb.StringValue{Value: request.TargetPubkey.String()},
		})
	}

	return consolidationRequests, nil
}

func (b *ConsolidationRequestDeriver) createEvent(ctx context.Context, consolidationRequest *xatuethv1.ElectraExecutionRequestConsolidation, identifier *xatu.BlockIdentifier, position uint32) (*xatu.DecoratedEvent, error) {
	// Make a clone of the metadata
	metadata, ok := proto.Clone(b.clientMeta).(*xatu.ClientMeta)
	if !ok {
		return nil, errors.New("failed to clone client metadata")
	}

	decoratedEvent := &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_CONSOLIDATION_REQUEST,
			DateTime: timestamppb.New(time.Now()),
			Id:       uuid.New().String(),
		},
		Meta: &xatu.Meta{
			Client: metadata,
		},
		Data: &xatu.DecoratedEvent_EthV2BeaconBlockConsolidationRequest{
			EthV2BeaconBlockConsolidationRequest: consolidationRequest,
		},
	}

	decoratedEvent.Meta.Client.AdditionalData = &xatu.ClientMeta_EthV2BeaconBlockConsolidationRequest{
		EthV2BeaconBlockConsolidationRequest: &xatu.ClientMeta_AdditionalEthV2BeaconBlockConsolidationRequestData{
			Block:           identifier,
			PositionInBlock: position,
		},
	}

	return decoratedEvent, nil
}
