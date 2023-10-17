package v1

import (
	"context"
	"fmt"
	"time"

	"github.com/attestantio/go-eth2-client/spec/deneb"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	backoff "github.com/cenkalti/backoff/v4"
	"github.com/ethpandaops/xatu/pkg/cannon/ethereum"
	"github.com/ethpandaops/xatu/pkg/cannon/iterator"
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
	BeaconBlobDeriverName = xatu.CannonType_BEACON_API_ETH_V1_BEACON_BLOB_SIDECAR
)

type BeaconBlobDeriverConfig struct {
	Enabled bool `yaml:"enabled" default:"false"`
}

type BeaconBlobDeriver struct {
	log               logrus.FieldLogger
	cfg               *BeaconBlobDeriverConfig
	iterator          *iterator.CheckpointIterator
	onEventsCallbacks []func(ctx context.Context, events []*xatu.DecoratedEvent) error
	beacon            *ethereum.BeaconNode
	clientMeta        *xatu.ClientMeta
}

func NewBeaconBlobDeriver(log logrus.FieldLogger, config *BeaconBlobDeriverConfig, iter *iterator.CheckpointIterator, beacon *ethereum.BeaconNode, clientMeta *xatu.ClientMeta) *BeaconBlobDeriver {
	return &BeaconBlobDeriver{
		log:        log.WithField("module", "cannon/event/beacon/eth/v1/beacon_blob"),
		cfg:        config,
		iterator:   iter,
		beacon:     beacon,
		clientMeta: clientMeta,
	}
}

func (b *BeaconBlobDeriver) CannonType() xatu.CannonType {
	return BeaconBlobDeriverName
}

func (b *BeaconBlobDeriver) Name() string {
	return BeaconBlobDeriverName.String()
}

func (b *BeaconBlobDeriver) OnEventsDerived(ctx context.Context, fn func(ctx context.Context, events []*xatu.DecoratedEvent) error) {
	b.onEventsCallbacks = append(b.onEventsCallbacks, fn)
}

func (b *BeaconBlobDeriver) Start(ctx context.Context) error {
	if !b.cfg.Enabled {
		b.log.Info("Beacon blob deriver disabled")

		return nil
	}

	b.log.Info("Beacon blob deriver enabled")

	// Start our main loop
	go b.run(ctx)

	return nil
}

func (b *BeaconBlobDeriver) Stop(ctx context.Context) error {
	return nil
}

func (b *BeaconBlobDeriver) run(rctx context.Context) {
	bo := backoff.NewExponentialBackOff()
	bo.MaxInterval = 3 * time.Minute

	tracer := observability.Tracer()

	for {
		select {
		case <-rctx.Done():
			return
		default:
			operation := func() error {
				ctx, span := tracer.Start(rctx, fmt.Sprintf("Derive %s", b.Name()),
					trace.WithAttributes(
						attribute.String("network", string(b.beacon.Metadata().Network.Name))),
				)
				defer span.End()

				time.Sleep(100 * time.Millisecond)

				if err := b.beacon.Synced(ctx); err != nil {
					span.SetStatus(codes.Error, err.Error())

					return err
				}

				// Get the next slot
				location, _, err := b.iterator.Next(ctx)
				if err != nil {
					span.SetStatus(codes.Error, err.Error())

					return err
				}

				// Process the epoch
				events, err := b.processEpoch(ctx, phase0.Epoch(location.GetEthV1BeaconBlobSidecar().GetEpoch()))
				if err != nil {
					b.log.WithError(err).Error("Failed to process epoch")

					span.SetStatus(codes.Error, err.Error())

					return err
				}

				// Send the events
				for _, fn := range b.onEventsCallbacks {
					if err := fn(ctx, events); err != nil {
						span.SetStatus(codes.Error, err.Error())

						return errors.Wrap(err, "failed to send events")
					}
				}

				// Update our location
				if err := b.iterator.UpdateLocation(ctx, location); err != nil {
					span.SetStatus(codes.Error, err.Error())

					return err
				}

				bo.Reset()

				return nil
			}

			if err := backoff.Retry(operation, bo); err != nil {
				b.log.WithError(err).Error("Failed to process location")
			}
		}
	}
}

func (b *BeaconBlobDeriver) processEpoch(ctx context.Context, epoch phase0.Epoch) ([]*xatu.DecoratedEvent, error) {
	ctx, span := observability.Tracer().Start(ctx,
		"BeaconBlobDeriver.processEpoch",
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

func (b *BeaconBlobDeriver) processSlot(ctx context.Context, slot phase0.Slot) ([]*xatu.DecoratedEvent, error) {
	ctx, span := observability.Tracer().Start(ctx,
		"BeaconBlobDeriver.processSlot",
		trace.WithAttributes(attribute.Int64("slot", int64(slot))),
	)
	defer span.End()

	// Get the block
	blobs, err := b.beacon.Node().FetchBeaconBlockBlobs(ctx, xatuethv1.SlotAsString(slot))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get beacon block for slot %d", slot)
	}

	if blobs == nil {
		return []*xatu.DecoratedEvent{}, nil
	}

	events := []*xatu.DecoratedEvent{}

	for _, blob := range blobs {
		event, err := b.createEventFromBlob(ctx, blob)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to create event from block for slot %d", slot)
		}

		events = append(events, event)
	}

	return events, nil
}

func (b *BeaconBlobDeriver) createEventFromBlob(ctx context.Context, blob *deneb.BlobSidecar) (*xatu.DecoratedEvent, error) {
	// Make a clone of the metadata
	metadata, ok := proto.Clone(b.clientMeta).(*xatu.ClientMeta)
	if !ok {
		return nil, errors.New("failed to clone client metadata")
	}

	decoratedEvent := &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_BEACON_API_ETH_V1_BEACON_BLOB_SIDECAR,
			DateTime: timestamppb.New(time.Now()),
			Id:       uuid.New().String(),
		},
		Meta: &xatu.Meta{
			Client: metadata,
		},
		Data: &xatu.DecoratedEvent_EthV1BeaconBlockBlobSidecar{
			EthV1BeaconBlockBlobSidecar: &xatuethv1.BlobSidecar{
				Slot:            &wrapperspb.UInt64Value{Value: uint64(blob.Slot)},
				Blob:            blob.Blob[:],
				Index:           &wrapperspb.UInt64Value{Value: uint64(blob.Index)},
				BlockRoot:       blob.BlockRoot.String(),
				BlockParentRoot: blob.BlockParentRoot.String(),
				ProposerIndex:   &wrapperspb.UInt64Value{Value: uint64(blob.ProposerIndex)},
				KzgCommitment:   blob.KzgCommitment.String(),
				KzgProof:        blob.KzgProof.String(),
			},
		},
	}

	additionalData, err := b.getAdditionalData(ctx, blob)
	if err != nil {
		b.log.WithError(err).Error("Failed to get extra beacon blob data")

		return nil, err
	} else {
		decoratedEvent.Meta.Client.AdditionalData = &xatu.ClientMeta_EthV1BeaconBlobSidecar{
			EthV1BeaconBlobSidecar: additionalData,
		}
	}

	return decoratedEvent, nil
}

func (b *BeaconBlobDeriver) getAdditionalData(_ context.Context, blob *deneb.BlobSidecar) (*xatu.ClientMeta_AdditionalEthV1BeaconBlobSidecarData, error) {
	extra := &xatu.ClientMeta_AdditionalEthV1BeaconBlobSidecarData{
		DataSize:      &wrapperspb.UInt64Value{Value: uint64(len(blob.Blob))},
		VersionedHash: ethereum.ConvertKzgCommitmentToVersionedHash(blob.KzgCommitment[:]).String(),
	}

	slot := b.beacon.Metadata().Wallclock().Slots().FromNumber(uint64(blob.Slot))
	epoch := b.beacon.Metadata().Wallclock().Epochs().FromSlot(uint64(blob.Slot))

	extra.Slot = &xatu.SlotV2{
		StartDateTime: timestamppb.New(slot.TimeWindow().Start()),
		Number:        &wrapperspb.UInt64Value{Value: uint64(blob.Slot)},
	}

	extra.Epoch = &xatu.EpochV2{
		Number:        &wrapperspb.UInt64Value{Value: epoch.Number()},
		StartDateTime: timestamppb.New(epoch.TimeWindow().Start()),
	}

	return extra, nil
}
