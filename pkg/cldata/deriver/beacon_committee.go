package deriver

import (
	"context"
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
	BeaconCommitteeDeriverName = xatu.CannonType_BEACON_API_ETH_V1_BEACON_COMMITTEE
)

// BeaconCommitteeDeriverConfig holds the configuration for the BeaconCommitteeDeriver.
type BeaconCommitteeDeriverConfig struct {
	Enabled bool `yaml:"enabled" default:"true"`
}

// BeaconCommitteeDeriver derives beacon committee events from the consensus layer.
// It processes epochs and emits decorated events for each committee.
type BeaconCommitteeDeriver struct {
	log               logrus.FieldLogger
	cfg               *BeaconCommitteeDeriverConfig
	iterator          iterator.Iterator
	onEventsCallbacks []func(ctx context.Context, events []*xatu.DecoratedEvent) error
	beacon            cldata.BeaconClient
	ctx               cldata.ContextProvider
}

// NewBeaconCommitteeDeriver creates a new BeaconCommitteeDeriver instance.
func NewBeaconCommitteeDeriver(
	log logrus.FieldLogger,
	config *BeaconCommitteeDeriverConfig,
	iter iterator.Iterator,
	beacon cldata.BeaconClient,
	ctxProvider cldata.ContextProvider,
) *BeaconCommitteeDeriver {
	return &BeaconCommitteeDeriver{
		log: log.WithFields(logrus.Fields{
			"module": "cldata/deriver/beacon_committee",
			"type":   BeaconCommitteeDeriverName.String(),
		}),
		cfg:      config,
		iterator: iter,
		beacon:   beacon,
		ctx:      ctxProvider,
	}
}

func (d *BeaconCommitteeDeriver) CannonType() xatu.CannonType {
	return BeaconCommitteeDeriverName
}

func (d *BeaconCommitteeDeriver) ActivationFork() spec.DataVersion {
	return spec.DataVersionPhase0
}

func (d *BeaconCommitteeDeriver) Name() string {
	return BeaconCommitteeDeriverName.String()
}

func (d *BeaconCommitteeDeriver) OnEventsDerived(
	_ context.Context,
	fn func(ctx context.Context, events []*xatu.DecoratedEvent) error,
) {
	d.onEventsCallbacks = append(d.onEventsCallbacks, fn)
}

func (d *BeaconCommitteeDeriver) Start(ctx context.Context) error {
	if !d.cfg.Enabled {
		d.log.Info("Beacon committee deriver disabled")

		return nil
	}

	d.log.Info("Beacon committee deriver enabled")

	if err := d.iterator.Start(ctx, d.ActivationFork()); err != nil {
		return errors.Wrap(err, "failed to start iterator")
	}

	// Start our main loop
	d.run(ctx)

	return nil
}

func (d *BeaconCommitteeDeriver) Stop(_ context.Context) error {
	return nil
}

func (d *BeaconCommitteeDeriver) run(rctx context.Context) {
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
					d.log.WithError(err).WithField("epoch", position.Epoch).Error("Failed to process epoch")

					span.SetStatus(codes.Error, err.Error())

					return "", err
				}

				// Look ahead (not supported for beacon committees)
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

// lookAhead takes the upcoming epochs and looks ahead to do any pre-processing that might be required.
// Not supported for beacon committees.
func (d *BeaconCommitteeDeriver) lookAhead(_ context.Context, _ []phase0.Epoch) {
	// Not supported.
}

func (d *BeaconCommitteeDeriver) processEpoch(
	ctx context.Context,
	epoch phase0.Epoch,
) ([]*xatu.DecoratedEvent, error) {
	ctx, span := observability.Tracer().Start(ctx,
		"BeaconCommitteeDeriver.processEpoch",
		//nolint:gosec // epoch numbers won't exceed int64 max in practice
		trace.WithAttributes(attribute.Int64("epoch", int64(epoch))),
	)
	defer span.End()

	sp, err := d.beacon.Node().Spec()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get beacon spec")
	}

	// Get the beacon committees for this epoch
	beaconCommittees, err := d.beacon.FetchBeaconCommittee(ctx, epoch)
	if err != nil {
		return nil, errors.Wrap(err, "failed to fetch beacon committees")
	}

	allEvents := make([]*xatu.DecoratedEvent, 0, len(beaconCommittees))
	uniqueEpochs := make(map[phase0.Epoch]struct{}, 1)
	uniqueSlots := make(map[phase0.Slot]struct{}, sp.SlotsPerEpoch)
	uniqueCommittees := make(map[phase0.CommitteeIndex]struct{}, len(beaconCommittees))

	for _, committee := range beaconCommittees {
		uniqueEpochs[epoch] = struct{}{}
		uniqueSlots[committee.Slot] = struct{}{}
		uniqueCommittees[committee.Index] = struct{}{}
	}

	if len(uniqueEpochs) > 1 {
		d.log.WithField("epochs", uniqueEpochs).Warn("Multiple epochs found")

		return nil, errors.New("multiple epochs found")
	}

	minSlot := phase0.Slot(epoch) * sp.SlotsPerEpoch
	maxSlot := (phase0.Slot(epoch) * sp.SlotsPerEpoch) + sp.SlotsPerEpoch - 1

	for _, committee := range beaconCommittees {
		if committee.Slot < minSlot || committee.Slot > maxSlot {
			return nil, fmt.Errorf(
				"beacon committee slot outside of epoch. (epoch: %d, slot: %d, min: %d, max: %d)",
				epoch, committee.Slot, minSlot, maxSlot,
			)
		}

		event, err := d.createEventFromBeaconCommittee(ctx, committee)
		if err != nil {
			d.log.
				WithError(err).
				WithField("slot", committee.Slot).
				WithField("epoch", epoch).
				Error("Failed to create event from beacon committee")

			return nil, err
		}

		allEvents = append(allEvents, event)
	}

	return allEvents, nil
}

func (d *BeaconCommitteeDeriver) createEventFromBeaconCommittee(
	ctx context.Context,
	committee *apiv1.BeaconCommittee,
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

	validators := make([]*wrapperspb.UInt64Value, 0, len(committee.Validators))
	for _, validator := range committee.Validators {
		validators = append(validators, wrapperspb.UInt64(uint64(validator)))
	}

	decoratedEvent := &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_BEACON_API_ETH_V1_BEACON_COMMITTEE,
			DateTime: timestamppb.New(time.Now()),
			Id:       uuid.New().String(),
		},
		Meta: &xatu.Meta{
			Client: metadata,
		},
		Data: &xatu.DecoratedEvent_EthV1BeaconCommittee{
			EthV1BeaconCommittee: &xatuethv1.Committee{
				Slot:       wrapperspb.UInt64(uint64(committee.Slot)),
				Index:      wrapperspb.UInt64(uint64(committee.Index)),
				Validators: validators,
			},
		},
	}

	additionalData, err := d.getAdditionalData(committee)
	if err != nil {
		d.log.WithError(err).Error("Failed to get extra beacon committee data")

		return nil, err
	}

	decoratedEvent.Meta.Client.AdditionalData = &xatu.ClientMeta_EthV1BeaconCommittee{
		EthV1BeaconCommittee: additionalData,
	}

	return decoratedEvent, nil
}

func (d *BeaconCommitteeDeriver) getAdditionalData(
	committee *apiv1.BeaconCommittee,
) (*xatu.ClientMeta_AdditionalEthV1BeaconCommitteeData, error) {
	extra := &xatu.ClientMeta_AdditionalEthV1BeaconCommitteeData{
		StateId: xatuethv1.StateIDFinalized,
	}

	slot := d.ctx.Wallclock().Slots().FromNumber(uint64(committee.Slot))
	epoch := d.ctx.Wallclock().Epochs().FromSlot(uint64(committee.Slot))

	extra.Slot = &xatu.SlotV2{
		StartDateTime: timestamppb.New(slot.TimeWindow().Start()),
		Number:        &wrapperspb.UInt64Value{Value: uint64(committee.Slot)},
	}

	extra.Epoch = &xatu.EpochV2{
		Number:        &wrapperspb.UInt64Value{Value: epoch.Number()},
		StartDateTime: timestamppb.New(epoch.TimeWindow().Start()),
	}

	return extra, nil
}
