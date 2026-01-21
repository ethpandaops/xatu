package deriver

import (
	"context"
	"encoding/hex"
	"fmt"
	"time"

	apiv1 "github.com/attestantio/go-eth2-client/api/v1"
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
	ProposerDutyDeriverName = xatu.CannonType_BEACON_API_ETH_V1_PROPOSER_DUTY
)

// ProposerDutyDeriverConfig holds the configuration for the ProposerDutyDeriver.
type ProposerDutyDeriverConfig struct {
	Enabled bool `yaml:"enabled" default:"true"`
}

// ProposerDutyDeriver derives proposer duty events from the consensus layer.
// It processes epochs and emits decorated events for each proposer duty.
type ProposerDutyDeriver struct {
	log               logrus.FieldLogger
	cfg               *ProposerDutyDeriverConfig
	iterator          iterator.Iterator
	onEventsCallbacks []func(ctx context.Context, events []*xatu.DecoratedEvent) error
	beacon            cldata.BeaconClient
	ctx               cldata.ContextProvider
}

// NewProposerDutyDeriver creates a new ProposerDutyDeriver instance.
func NewProposerDutyDeriver(
	log logrus.FieldLogger,
	config *ProposerDutyDeriverConfig,
	iter iterator.Iterator,
	beacon cldata.BeaconClient,
	ctxProvider cldata.ContextProvider,
) *ProposerDutyDeriver {
	return &ProposerDutyDeriver{
		log: log.WithFields(logrus.Fields{
			"module": "cldata/deriver/proposer_duty",
			"type":   ProposerDutyDeriverName.String(),
		}),
		cfg:      config,
		iterator: iter,
		beacon:   beacon,
		ctx:      ctxProvider,
	}
}

func (d *ProposerDutyDeriver) CannonType() xatu.CannonType {
	return ProposerDutyDeriverName
}

func (d *ProposerDutyDeriver) ActivationFork() spec.DataVersion {
	return spec.DataVersionPhase0
}

func (d *ProposerDutyDeriver) Name() string {
	return ProposerDutyDeriverName.String()
}

func (d *ProposerDutyDeriver) OnEventsDerived(
	_ context.Context,
	fn func(ctx context.Context, events []*xatu.DecoratedEvent) error,
) {
	d.onEventsCallbacks = append(d.onEventsCallbacks, fn)
}

func (d *ProposerDutyDeriver) Start(ctx context.Context) error {
	if !d.cfg.Enabled {
		d.log.Info("Proposer duty deriver disabled")

		return nil
	}

	d.log.Info("Proposer duty deriver enabled")

	if err := d.iterator.Start(ctx, d.ActivationFork()); err != nil {
		return errors.Wrap(err, "failed to start iterator")
	}

	// Start our main loop
	d.run(ctx)

	return nil
}

func (d *ProposerDutyDeriver) Stop(_ context.Context) error {
	return nil
}

func (d *ProposerDutyDeriver) run(rctx context.Context) {
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

func (d *ProposerDutyDeriver) processEpoch(ctx context.Context, epoch phase0.Epoch) ([]*xatu.DecoratedEvent, error) {
	ctx, span := observability.Tracer().Start(ctx,
		"ProposerDutyDeriver.processEpoch",
		//nolint:gosec // epoch numbers won't exceed int64 max in practice
		trace.WithAttributes(attribute.Int64("epoch", int64(epoch))),
	)
	defer span.End()

	// Get the proposer duties for this epoch
	proposerDuties, err := d.beacon.FetchProposerDuties(ctx, epoch)
	if err != nil {
		return nil, errors.Wrap(err, "failed to fetch proposer duties")
	}

	allEvents := make([]*xatu.DecoratedEvent, 0, len(proposerDuties))

	for _, duty := range proposerDuties {
		event, err := d.createEventFromProposerDuty(ctx, duty)
		if err != nil {
			d.log.
				WithError(err).
				WithField("slot", duty.Slot).
				WithField("epoch", epoch).
				Error("Failed to create event from proposer duty")

			return nil, err
		}

		allEvents = append(allEvents, event)
	}

	return allEvents, nil
}

// lookAhead takes the upcoming epochs and looks ahead to do any pre-processing that might be required.
func (d *ProposerDutyDeriver) lookAhead(ctx context.Context, epochs []phase0.Epoch) {
	_, span := observability.Tracer().Start(ctx,
		"ProposerDutyDeriver.lookAhead",
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

func (d *ProposerDutyDeriver) createEventFromProposerDuty(
	ctx context.Context,
	duty *apiv1.ProposerDuty,
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

	decoratedEvent := &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_BEACON_API_ETH_V1_PROPOSER_DUTY,
			DateTime: timestamppb.New(time.Now()),
			Id:       uuid.New().String(),
		},
		Meta: &xatu.Meta{
			Client: metadata,
		},
		Data: &xatu.DecoratedEvent_EthV1ProposerDuty{
			EthV1ProposerDuty: &xatuethv1.ProposerDuty{
				Slot:           wrapperspb.UInt64(uint64(duty.Slot)),
				Pubkey:         fmt.Sprintf("0x%s", hex.EncodeToString(duty.PubKey[:])),
				ValidatorIndex: wrapperspb.UInt64(uint64(duty.ValidatorIndex)),
			},
		},
	}

	additionalData, err := d.getAdditionalData(duty)
	if err != nil {
		d.log.WithError(err).Error("Failed to get extra proposer duty data")

		return nil, err
	}

	decoratedEvent.Meta.Client.AdditionalData = &xatu.ClientMeta_EthV1ProposerDuty{
		EthV1ProposerDuty: additionalData,
	}

	return decoratedEvent, nil
}

func (d *ProposerDutyDeriver) getAdditionalData(
	duty *apiv1.ProposerDuty,
) (*xatu.ClientMeta_AdditionalEthV1ProposerDutyData, error) {
	extra := &xatu.ClientMeta_AdditionalEthV1ProposerDutyData{
		StateId: xatuethv1.StateIDFinalized,
	}

	slot := d.ctx.Wallclock().Slots().FromNumber(uint64(duty.Slot))
	epoch := d.ctx.Wallclock().Epochs().FromSlot(uint64(duty.Slot))

	extra.Slot = &xatu.SlotV2{
		StartDateTime: timestamppb.New(slot.TimeWindow().Start()),
		Number:        &wrapperspb.UInt64Value{Value: uint64(duty.Slot)},
	}

	extra.Epoch = &xatu.EpochV2{
		Number:        &wrapperspb.UInt64Value{Value: epoch.Number()},
		StartDateTime: timestamppb.New(epoch.TimeWindow().Start()),
	}

	return extra, nil
}
