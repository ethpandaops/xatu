package deriver

import (
	"context"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/attestantio/go-eth2-client/api"
	"github.com/attestantio/go-eth2-client/spec"
	"github.com/attestantio/go-eth2-client/spec/deneb"
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
	BeaconBlobDeriverName = xatu.CannonType_BEACON_API_ETH_V1_BEACON_BLOB_SIDECAR
)

// BeaconBlobDeriverConfig holds the configuration for the BeaconBlobDeriver.
type BeaconBlobDeriverConfig struct {
	Enabled bool `yaml:"enabled" default:"true"`
}

// BeaconBlobDeriver derives beacon blob sidecar events from the consensus layer.
// It processes epochs and emits decorated events for each blob sidecar.
type BeaconBlobDeriver struct {
	log               logrus.FieldLogger
	cfg               *BeaconBlobDeriverConfig
	iterator          iterator.Iterator
	onEventsCallbacks []func(ctx context.Context, events []*xatu.DecoratedEvent) error
	beacon            cldata.BeaconClient
	ctx               cldata.ContextProvider
}

// NewBeaconBlobDeriver creates a new BeaconBlobDeriver instance.
func NewBeaconBlobDeriver(
	log logrus.FieldLogger,
	config *BeaconBlobDeriverConfig,
	iter iterator.Iterator,
	beacon cldata.BeaconClient,
	ctxProvider cldata.ContextProvider,
) *BeaconBlobDeriver {
	return &BeaconBlobDeriver{
		log: log.WithFields(logrus.Fields{
			"module": "cldata/deriver/beacon_blob",
			"type":   BeaconBlobDeriverName.String(),
		}),
		cfg:      config,
		iterator: iter,
		beacon:   beacon,
		ctx:      ctxProvider,
	}
}

func (d *BeaconBlobDeriver) CannonType() xatu.CannonType {
	return BeaconBlobDeriverName
}

func (d *BeaconBlobDeriver) ActivationFork() spec.DataVersion {
	return spec.DataVersionDeneb
}

func (d *BeaconBlobDeriver) Name() string {
	return BeaconBlobDeriverName.String()
}

func (d *BeaconBlobDeriver) OnEventsDerived(
	_ context.Context,
	fn func(ctx context.Context, events []*xatu.DecoratedEvent) error,
) {
	d.onEventsCallbacks = append(d.onEventsCallbacks, fn)
}

func (d *BeaconBlobDeriver) Start(ctx context.Context) error {
	if !d.cfg.Enabled {
		d.log.Info("Beacon blob deriver disabled")

		return nil
	}

	d.log.Info("Beacon blob deriver enabled")

	if err := d.iterator.Start(ctx, d.ActivationFork()); err != nil {
		return errors.Wrap(err, "failed to start iterator")
	}

	// Start our main loop
	d.run(ctx)

	return nil
}

func (d *BeaconBlobDeriver) Stop(_ context.Context) error {
	return nil
}

func (d *BeaconBlobDeriver) run(rctx context.Context) {
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

				// Get the next position
				position, err := d.iterator.Next(ctx)
				if err != nil {
					span.SetStatus(codes.Error, err.Error())

					return "", err
				}

				// Process the epoch
				events, err := d.processEpoch(ctx, position.Epoch)
				if err != nil {
					d.log.WithError(err).Error("Failed to process epoch")

					span.SetStatus(codes.Error, err.Error())

					return "", err
				}

				// Look ahead
				d.lookAhead(ctx, position.LookAheadEpochs)

				// Send the events
				for _, fn := range d.onEventsCallbacks {
					if err := fn(ctx, events); err != nil {
						span.SetStatus(codes.Error, err.Error())

						return "", errors.Wrap(err, "failed to send events")
					}
				}

				// Update our location
				if err := d.iterator.UpdateLocation(ctx, position); err != nil {
					span.SetStatus(codes.Error, err.Error())

					return "", err
				}

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

func (d *BeaconBlobDeriver) processEpoch(ctx context.Context, epoch phase0.Epoch) ([]*xatu.DecoratedEvent, error) {
	ctx, span := observability.Tracer().Start(ctx,
		"BeaconBlobDeriver.processEpoch",
		//nolint:gosec // epoch numbers won't exceed int64 max in practice
		trace.WithAttributes(attribute.Int64("epoch", int64(epoch))),
	)
	defer span.End()

	sp, err := d.beacon.Node().Spec()
	if err != nil {
		return nil, errors.Wrap(err, "failed to obtain spec")
	}

	allEvents := []*xatu.DecoratedEvent{}

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

func (d *BeaconBlobDeriver) processSlot(ctx context.Context, slot phase0.Slot) ([]*xatu.DecoratedEvent, error) {
	ctx, span := observability.Tracer().Start(ctx,
		"BeaconBlobDeriver.processSlot",
		//nolint:gosec // slot numbers won't exceed int64 max in practice
		trace.WithAttributes(attribute.Int64("slot", int64(slot))),
	)
	defer span.End()

	blobs, err := d.beacon.FetchBeaconBlockBlobs(ctx, xatuethv1.SlotAsString(slot))
	if err != nil {
		var apiErr *api.Error
		if errors.As(err, &apiErr) {
			switch apiErr.StatusCode {
			case 404:
				return []*xatu.DecoratedEvent{}, nil
			case 503:
				return nil, errors.New("beacon node is syncing")
			}
		}

		return nil, errors.Wrapf(err, "failed to get beacon blob sidecars for slot %d", slot)
	}

	if blobs == nil {
		return []*xatu.DecoratedEvent{}, nil
	}

	events := make([]*xatu.DecoratedEvent, 0, len(blobs))

	for _, blob := range blobs {
		event, err := d.createEventFromBlob(ctx, blob)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to create event from blob sidecars for slot %d", slot)
		}

		events = append(events, event)
	}

	return events, nil
}

// lookAhead takes the upcoming epochs and looks ahead to do any pre-processing that might be required.
func (d *BeaconBlobDeriver) lookAhead(ctx context.Context, epochs []phase0.Epoch) {
	_, span := observability.Tracer().Start(ctx,
		"BeaconBlobDeriver.lookAhead",
	)
	defer span.End()

	sp, err := d.beacon.Node().Spec()
	if err != nil {
		d.log.WithError(err).Warn("Failed to look ahead at epoch")

		return
	}

	for _, epoch := range epochs {
		for i := uint64(0); i <= uint64(sp.SlotsPerEpoch-1); i++ {
			slot := phase0.Slot(i + uint64(epoch)*uint64(sp.SlotsPerEpoch))

			// Add the block to the preload queue so it's available when we need it
			d.beacon.LazyLoadBeaconBlock(xatuethv1.SlotAsString(slot))
		}
	}
}

func (d *BeaconBlobDeriver) createEventFromBlob(
	ctx context.Context,
	blob *deneb.BlobSidecar,
) (*xatu.DecoratedEvent, error) {
	// Get client metadata
	clientMeta, err := d.ctx.CreateClientMeta(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create client metadata")
	}

	// Make a clone of the metadata
	metadata, ok := proto.Clone(clientMeta).(*xatu.ClientMeta)
	if !ok {
		return nil, errors.New("failed to clone client metadata")
	}

	blockRoot, err := blob.SignedBlockHeader.Message.HashTreeRoot()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get block root")
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
				Slot:            &wrapperspb.UInt64Value{Value: uint64(blob.SignedBlockHeader.Message.Slot)},
				Blob:            fmt.Sprintf("0x%s", hex.EncodeToString(blob.Blob[:])),
				Index:           &wrapperspb.UInt64Value{Value: uint64(blob.Index)},
				BlockRoot:       fmt.Sprintf("0x%s", hex.EncodeToString(blockRoot[:])),
				BlockParentRoot: blob.SignedBlockHeader.Message.ParentRoot.String(),
				ProposerIndex:   &wrapperspb.UInt64Value{Value: uint64(blob.SignedBlockHeader.Message.ProposerIndex)},
				KzgCommitment:   blob.KZGCommitment.String(),
				KzgProof:        blob.KZGProof.String(),
			},
		},
	}

	additionalData, err := d.getAdditionalData(blob)
	if err != nil {
		d.log.WithError(err).Error("Failed to get extra beacon blob data")

		return nil, err
	}

	decoratedEvent.Meta.Client.AdditionalData = &xatu.ClientMeta_EthV1BeaconBlobSidecar{
		EthV1BeaconBlobSidecar: additionalData,
	}

	return decoratedEvent, nil
}

func (d *BeaconBlobDeriver) getAdditionalData(
	blob *deneb.BlobSidecar,
) (*xatu.ClientMeta_AdditionalEthV1BeaconBlobSidecarData, error) {
	//nolint:gosec // blob sizes are bounded and count is always non-negative
	extra := &xatu.ClientMeta_AdditionalEthV1BeaconBlobSidecarData{
		DataSize:      &wrapperspb.UInt64Value{Value: uint64(len(blob.Blob))},
		DataEmptySize: &wrapperspb.UInt64Value{Value: uint64(cldata.CountConsecutiveEmptyBytes(blob.Blob[:], 4))},
		VersionedHash: cldata.ConvertKzgCommitmentToVersionedHash(blob.KZGCommitment[:]).String(),
	}

	slot := d.ctx.Wallclock().Slots().FromNumber(uint64(blob.SignedBlockHeader.Message.Slot))
	epoch := d.ctx.Wallclock().Epochs().FromSlot(uint64(blob.SignedBlockHeader.Message.Slot))

	extra.Slot = &xatu.SlotV2{
		StartDateTime: timestamppb.New(slot.TimeWindow().Start()),
		Number:        &wrapperspb.UInt64Value{Value: uint64(blob.SignedBlockHeader.Message.Slot)},
	}

	extra.Epoch = &xatu.EpochV2{
		Number:        &wrapperspb.UInt64Value{Value: epoch.Number()},
		StartDateTime: timestamppb.New(epoch.TimeWindow().Start()),
	}

	return extra, nil
}
