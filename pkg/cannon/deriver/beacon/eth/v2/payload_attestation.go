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
	"google.golang.org/protobuf/types/known/wrapperspb"
)

const (
	PayloadAttestationDeriverName = xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_PAYLOAD_ATTESTATION
)

// PayloadAttestationDeriverConfig configures the cannon deriver that emits
// aggregated PTC payload attestations from Gloas+ beacon blocks.
type PayloadAttestationDeriverConfig struct {
	Enabled  bool                                 `yaml:"enabled" default:"true"`
	Iterator iterator.BackfillingCheckpointConfig `yaml:"iterator"`
}

// PayloadAttestationDeriver derives BEACON_API_ETH_V2_BEACON_BLOCK_PAYLOAD_ATTESTATION
// events from `block.Body.PayloadAttestations` on Gloas+ blocks. Up to
// MAX_PAYLOAD_ATTESTATIONS=4 entries per block.
type PayloadAttestationDeriver struct {
	log               logrus.FieldLogger
	cfg               *PayloadAttestationDeriverConfig
	iterator          *iterator.BackfillingCheckpoint
	onEventsCallbacks []func(ctx context.Context, events []*xatu.DecoratedEvent) error
	beacon            *ethereum.BeaconNode
	clientMeta        *xatu.ClientMeta
}

func NewPayloadAttestationDeriver(log logrus.FieldLogger, config *PayloadAttestationDeriverConfig, iter *iterator.BackfillingCheckpoint, beacon *ethereum.BeaconNode, clientMeta *xatu.ClientMeta) *PayloadAttestationDeriver {
	return &PayloadAttestationDeriver{
		log: log.WithFields(logrus.Fields{
			"module": "cannon/event/beacon/eth/v2/payload_attestation",
			"type":   PayloadAttestationDeriverName.String(),
		}),
		cfg:        config,
		iterator:   iter,
		beacon:     beacon,
		clientMeta: clientMeta,
	}
}

func (b *PayloadAttestationDeriver) CannonType() xatu.CannonType {
	return PayloadAttestationDeriverName
}

func (b *PayloadAttestationDeriver) Name() string {
	return PayloadAttestationDeriverName.String()
}

// ActivationFork pegs this deriver to Gloas — payload attestations didn't
// exist on prior forks, so the iterator starts at the Gloas activation epoch.
func (b *PayloadAttestationDeriver) ActivationFork() spec.DataVersion {
	return spec.DataVersionGloas
}

func (b *PayloadAttestationDeriver) OnEventsDerived(ctx context.Context, fn func(ctx context.Context, events []*xatu.DecoratedEvent) error) {
	b.onEventsCallbacks = append(b.onEventsCallbacks, fn)
}

func (b *PayloadAttestationDeriver) Start(ctx context.Context) error {
	if !b.cfg.Enabled {
		b.log.Info("Payload attestation deriver disabled")

		return nil
	}

	b.log.Info("Payload attestation deriver enabled")

	if err := b.iterator.Start(ctx, b.ActivationFork()); err != nil {
		return errors.Wrap(err, "failed to start iterator")
	}

	b.run(ctx)

	return nil
}

func (b *PayloadAttestationDeriver) Stop(ctx context.Context) error {
	return nil
}

func (b *PayloadAttestationDeriver) run(rctx context.Context) {
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
					b.log.WithError(err).Error("Failed to process epoch")

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
					b.log.WithError(err).WithField("next_attempt", timer).Warn("Failed to process")
				}),
			}
			if _, err := backoff.Retry(rctx, operation, retryOpts...); err != nil {
				b.log.WithError(err).Warn("Failed to process")
			}
		}
	}
}

func (b *PayloadAttestationDeriver) processEpoch(ctx context.Context, epoch phase0.Epoch) ([]*xatu.DecoratedEvent, error) {
	ctx, span := observability.Tracer().Start(ctx,
		"PayloadAttestationDeriver.processEpoch",
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

func (b *PayloadAttestationDeriver) processSlot(ctx context.Context, slot phase0.Slot) ([]*xatu.DecoratedEvent, error) {
	ctx, span := observability.Tracer().Start(ctx,
		"PayloadAttestationDeriver.processSlot",
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

	// Payload attestations only exist on Gloas+ blocks.
	if block.Version < spec.DataVersionGloas {
		return []*xatu.DecoratedEvent{}, nil
	}

	if block.Gloas == nil || block.Gloas.Message == nil || block.Gloas.Message.Body == nil {
		return []*xatu.DecoratedEvent{}, nil
	}

	attestations := block.Gloas.Message.Body.PayloadAttestations
	if len(attestations) == 0 {
		return []*xatu.DecoratedEvent{}, nil
	}

	blockIdentifier, err := GetBlockIdentifier(block, b.beacon.Metadata().Wallclock())
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get block identifier for slot %d", slot)
	}

	converted := xatuethv1.NewPayloadAttestationsFromGloas(attestations)

	events := make([]*xatu.DecoratedEvent, 0, len(converted))

	var position uint32

	for _, att := range converted {
		event, err := b.createEvent(ctx, att, blockIdentifier, position)
		if err != nil {
			b.log.WithError(err).Error("Failed to create event")

			return nil, errors.Wrapf(err, "failed to create payload attestation event for slot %d", slot)
		}

		events = append(events, event)
		position++
	}

	return events, nil
}

// lookAhead pre-loads upcoming epoch blocks so the iterator's next pass has them ready.
func (b *PayloadAttestationDeriver) lookAhead(ctx context.Context, epochs []phase0.Epoch) {
	_, span := observability.Tracer().Start(ctx,
		"PayloadAttestationDeriver.lookAhead",
	)
	defer span.End()

	if epochs == nil {
		return
	}

	sp, err := b.beacon.Node().Spec()
	if err != nil {
		b.log.WithError(err).Warn("Failed to look ahead at epoch")

		return
	}

	for _, epoch := range epochs {
		for i := uint64(0); i <= uint64(sp.SlotsPerEpoch-1); i++ {
			slot := phase0.Slot(i + uint64(epoch)*uint64(sp.SlotsPerEpoch))

			b.beacon.LazyLoadBeaconBlock(xatuethv1.SlotAsString(slot))
		}
	}
}

func (b *PayloadAttestationDeriver) createEvent(ctx context.Context, attestation *xatuethv1.PayloadAttestation, identifier *xatu.BlockIdentifier, position uint32) (*xatu.DecoratedEvent, error) {
	metadata, ok := proto.Clone(b.clientMeta).(*xatu.ClientMeta)
	if !ok {
		return nil, errors.New("failed to clone client metadata")
	}

	decoratedEvent := &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_PAYLOAD_ATTESTATION,
			DateTime: timestamppb.New(time.Now()),
			Id:       uuid.New().String(),
		},
		Meta: &xatu.Meta{
			Client: metadata,
		},
		Data: &xatu.DecoratedEvent_EthV2BeaconBlockPayloadAttestation{
			EthV2BeaconBlockPayloadAttestation: attestation,
		},
	}

	decoratedEvent.Meta.Client.AdditionalData = &xatu.ClientMeta_EthV2BeaconBlockPayloadAttestation{
		EthV2BeaconBlockPayloadAttestation: &xatu.ClientMeta_AdditionalEthV2BeaconBlockPayloadAttestationData{
			Block:    identifier,
			Position: &wrapperspb.UInt32Value{Value: position},
		},
	}

	return decoratedEvent, nil
}
