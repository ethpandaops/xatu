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
	AttesterSlashingDeriverName = xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_ATTESTER_SLASHING
)

// AttesterSlashingDeriverConfig holds the configuration for the AttesterSlashingDeriver.
type AttesterSlashingDeriverConfig struct {
	Enabled bool `yaml:"enabled" default:"true"`
}

// AttesterSlashingDeriver derives attester slashing events from the consensus layer.
// It processes epochs of blocks and emits decorated events for each attester slashing.
type AttesterSlashingDeriver struct {
	log               logrus.FieldLogger
	cfg               *AttesterSlashingDeriverConfig
	iterator          iterator.Iterator
	onEventsCallbacks []func(ctx context.Context, events []*xatu.DecoratedEvent) error
	beacon            cldata.BeaconClient
	ctx               cldata.ContextProvider
}

// NewAttesterSlashingDeriver creates a new AttesterSlashingDeriver instance.
func NewAttesterSlashingDeriver(
	log logrus.FieldLogger,
	config *AttesterSlashingDeriverConfig,
	iter iterator.Iterator,
	beacon cldata.BeaconClient,
	ctxProvider cldata.ContextProvider,
) *AttesterSlashingDeriver {
	return &AttesterSlashingDeriver{
		log: log.WithFields(logrus.Fields{
			"module": "cldata/deriver/attester_slashing",
			"type":   AttesterSlashingDeriverName.String(),
		}),
		cfg:      config,
		iterator: iter,
		beacon:   beacon,
		ctx:      ctxProvider,
	}
}

func (a *AttesterSlashingDeriver) CannonType() xatu.CannonType {
	return AttesterSlashingDeriverName
}

func (a *AttesterSlashingDeriver) ActivationFork() spec.DataVersion {
	return spec.DataVersionPhase0
}

func (a *AttesterSlashingDeriver) Name() string {
	return AttesterSlashingDeriverName.String()
}

func (a *AttesterSlashingDeriver) OnEventsDerived(
	ctx context.Context,
	fn func(ctx context.Context, events []*xatu.DecoratedEvent) error,
) {
	a.onEventsCallbacks = append(a.onEventsCallbacks, fn)
}

func (a *AttesterSlashingDeriver) Start(ctx context.Context) error {
	if !a.cfg.Enabled {
		a.log.Info("Attester slashing deriver disabled")

		return nil
	}

	a.log.Info("Attester slashing deriver enabled")

	if err := a.iterator.Start(ctx, a.ActivationFork()); err != nil {
		return errors.Wrap(err, "failed to start iterator")
	}

	// Start our main loop
	a.run(ctx)

	return nil
}

func (a *AttesterSlashingDeriver) Stop(ctx context.Context) error {
	return nil
}

func (a *AttesterSlashingDeriver) run(rctx context.Context) {
	bo := backoff.NewExponentialBackOff()
	bo.MaxInterval = 3 * time.Minute

	tracer := observability.Tracer()

	for {
		select {
		case <-rctx.Done():
			return
		default:
			operation := func() (string, error) {
				ctx, span := tracer.Start(rctx, fmt.Sprintf("Derive %s", a.Name()),
					trace.WithAttributes(
						attribute.String("network", a.ctx.NetworkName())),
				)
				defer span.End()

				time.Sleep(100 * time.Millisecond)

				if err := a.beacon.Synced(ctx); err != nil {
					span.SetStatus(codes.Error, err.Error())

					return "", err
				}

				// Get the next position
				position, err := a.iterator.Next(ctx)
				if err != nil {
					span.SetStatus(codes.Error, err.Error())

					return "", err
				}

				// Look ahead
				a.lookAhead(ctx, position.LookAheadEpochs)

				// Process the epoch
				events, err := a.processEpoch(ctx, position.Epoch)
				if err != nil {
					a.log.WithError(err).Error("Failed to process epoch")

					span.SetStatus(codes.Error, err.Error())

					return "", err
				}

				span.AddEvent("Epoch processing complete. Sending events...")

				// Send the events
				for _, fn := range a.onEventsCallbacks {
					if err := fn(ctx, events); err != nil {
						span.SetStatus(codes.Error, err.Error())

						return "", errors.Wrap(err, "failed to send events")
					}
				}

				span.AddEvent("Events sent. Updating location...")

				// Update our location
				if err := a.iterator.UpdateLocation(ctx, position); err != nil {
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
					a.log.WithError(err).WithField("next_attempt", timer).Warn("Failed to process")
				}),
			}
			if _, err := backoff.Retry(rctx, operation, retryOpts...); err != nil {
				a.log.WithError(err).Warn("Failed to process")
			}
		}
	}
}

// lookAhead takes the upcoming epochs and looks ahead to do any pre-processing that might be required.
func (a *AttesterSlashingDeriver) lookAhead(ctx context.Context, epochs []phase0.Epoch) {
	_, span := observability.Tracer().Start(ctx,
		"AttesterSlashingDeriver.lookAhead",
	)
	defer span.End()

	sp, err := a.beacon.Node().Spec()
	if err != nil {
		a.log.WithError(err).Warn("Failed to look ahead at epoch")

		return
	}

	for _, epoch := range epochs {
		for i := uint64(0); i <= uint64(sp.SlotsPerEpoch-1); i++ {
			slot := phase0.Slot(i + uint64(epoch)*uint64(sp.SlotsPerEpoch))

			// Add the block to the preload queue so it's available when we need it
			a.beacon.LazyLoadBeaconBlock(xatuethv1.SlotAsString(slot))
		}
	}
}

func (a *AttesterSlashingDeriver) processEpoch(ctx context.Context, epoch phase0.Epoch) ([]*xatu.DecoratedEvent, error) {
	ctx, span := observability.Tracer().Start(ctx,
		"AttesterSlashingDeriver.processEpoch",
		//nolint:gosec // epoch numbers won't exceed int64 max in practice
		trace.WithAttributes(attribute.Int64("epoch", int64(epoch))),
	)
	defer span.End()

	sp, err := a.beacon.Node().Spec()
	if err != nil {
		return nil, errors.Wrap(err, "failed to obtain spec")
	}

	allEvents := make([]*xatu.DecoratedEvent, 0)

	for i := uint64(0); i <= uint64(sp.SlotsPerEpoch-1); i++ {
		slot := phase0.Slot(i + uint64(epoch)*uint64(sp.SlotsPerEpoch))

		events, err := a.processSlot(ctx, slot)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to process slot %d", slot)
		}

		allEvents = append(allEvents, events...)
	}

	return allEvents, nil
}

func (a *AttesterSlashingDeriver) processSlot(ctx context.Context, slot phase0.Slot) ([]*xatu.DecoratedEvent, error) {
	ctx, span := observability.Tracer().Start(ctx,
		"AttesterSlashingDeriver.processSlot",
		//nolint:gosec // slot numbers won't exceed int64 max in practice
		trace.WithAttributes(attribute.Int64("slot", int64(slot))),
	)
	defer span.End()

	// Get the block
	block, err := a.beacon.GetBeaconBlock(ctx, xatuethv1.SlotAsString(slot))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get beacon block for slot %d", slot)
	}

	if block == nil {
		return []*xatu.DecoratedEvent{}, nil
	}

	blockIdentifier, err := GetBlockIdentifier(block, a.ctx.Wallclock())
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get block identifier for slot %d", slot)
	}

	events := make([]*xatu.DecoratedEvent, 0)

	slashings, err := a.getAttesterSlashings(ctx, block)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get attester slashings for slot %d", slot)
	}

	for _, slashing := range slashings {
		event, err := a.createEvent(ctx, slashing, blockIdentifier)
		if err != nil {
			a.log.WithError(err).Error("Failed to create event")

			return nil, errors.Wrapf(err, "failed to create event for attester slashing %s", slashing.String())
		}

		events = append(events, event)
	}

	return events, nil
}

func (a *AttesterSlashingDeriver) getAttesterSlashings(
	_ context.Context,
	block *spec.VersionedSignedBeaconBlock,
) ([]*xatuethv1.AttesterSlashingV2, error) {
	slashings := make([]*xatuethv1.AttesterSlashingV2, 0)

	attesterSlashings, err := block.AttesterSlashings()
	if err != nil {
		return nil, errors.Wrap(err, "failed to obtain attester slashings")
	}

	for _, slashing := range attesterSlashings {
		att1, err := slashing.Attestation1()
		if err != nil {
			return nil, errors.Wrap(err, "failed to obtain attestation 1")
		}

		indexedAttestation1, err := ConvertIndexedAttestation(att1)
		if err != nil {
			return nil, errors.Wrap(err, "failed to convert indexed attestation 1")
		}

		att2, err := slashing.Attestation2()
		if err != nil {
			return nil, errors.Wrap(err, "failed to obtain attestation 2")
		}

		indexedAttestation2, err := ConvertIndexedAttestation(att2)
		if err != nil {
			return nil, errors.Wrap(err, "failed to convert indexed attestation 2")
		}

		slashings = append(slashings, &xatuethv1.AttesterSlashingV2{
			Attestation_1: indexedAttestation1,
			Attestation_2: indexedAttestation2,
		})
	}

	return slashings, nil
}

func (a *AttesterSlashingDeriver) createEvent(
	ctx context.Context,
	slashing *xatuethv1.AttesterSlashingV2,
	identifier *xatu.BlockIdentifier,
) (*xatu.DecoratedEvent, error) {
	// Get client metadata
	clientMeta, err := a.ctx.CreateClientMeta(ctx)
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
			Name:     xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_ATTESTER_SLASHING,
			DateTime: timestamppb.New(time.Now()),
			Id:       uuid.New().String(),
		},
		Meta: &xatu.Meta{
			Client: metadata,
		},
		Data: &xatu.DecoratedEvent_EthV2BeaconBlockAttesterSlashing{
			EthV2BeaconBlockAttesterSlashing: slashing,
		},
	}

	decoratedEvent.Meta.Client.AdditionalData = &xatu.ClientMeta_EthV2BeaconBlockAttesterSlashing{
		EthV2BeaconBlockAttesterSlashing: &xatu.ClientMeta_AdditionalEthV2BeaconBlockAttesterSlashingData{
			Block: identifier,
		},
	}

	return decoratedEvent, nil
}

// ConvertIndexedAttestation converts a VersionedIndexedAttestation to an IndexedAttestationV2.
func ConvertIndexedAttestation(attestation *spec.VersionedIndexedAttestation) (*xatuethv1.IndexedAttestationV2, error) {
	indicies := make([]*wrapperspb.UInt64Value, 0)

	atIndicies, err := attestation.AttestingIndices()
	if err != nil {
		return nil, errors.Wrap(err, "failed to obtain attesting indices")
	}

	for _, index := range atIndicies {
		indicies = append(indicies, &wrapperspb.UInt64Value{Value: index})
	}

	data, err := attestation.Data()
	if err != nil {
		return nil, errors.Wrap(err, "failed to obtain attestation data")
	}

	sig, err := attestation.Signature()
	if err != nil {
		return nil, errors.Wrap(err, "failed to obtain attestation signature")
	}

	return &xatuethv1.IndexedAttestationV2{
		AttestingIndices: indicies,
		Data: &xatuethv1.AttestationDataV2{
			Slot:            &wrapperspb.UInt64Value{Value: uint64(data.Slot)},
			Index:           &wrapperspb.UInt64Value{Value: uint64(data.Index)},
			BeaconBlockRoot: data.BeaconBlockRoot.String(),
			Source: &xatuethv1.CheckpointV2{
				Epoch: &wrapperspb.UInt64Value{Value: uint64(data.Source.Epoch)},
				Root:  data.Source.Root.String(),
			},
			Target: &xatuethv1.CheckpointV2{
				Epoch: &wrapperspb.UInt64Value{Value: uint64(data.Target.Epoch)},
				Root:  data.Target.Root.String(),
			},
		},
		Signature: sig.String(),
	}, nil
}
