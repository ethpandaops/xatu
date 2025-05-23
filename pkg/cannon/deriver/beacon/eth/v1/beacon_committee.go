package v1

import (
	"context"
	"fmt"
	"time"

	apiv1 "github.com/attestantio/go-eth2-client/api/v1"
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
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

const (
	BeaconCommitteeDeriverName = xatu.CannonType_BEACON_API_ETH_V1_BEACON_COMMITTEE
)

type BeaconCommitteeDeriverConfig struct {
	Enabled  bool                                 `yaml:"enabled" default:"true"`
	Iterator iterator.BackfillingCheckpointConfig `yaml:"iterator"`
}

type BeaconCommitteeDeriver struct {
	log               logrus.FieldLogger
	cfg               *BeaconCommitteeDeriverConfig
	iterator          *iterator.BackfillingCheckpoint
	onEventsCallbacks []func(ctx context.Context, events []*xatu.DecoratedEvent) error
	beacon            *ethereum.BeaconNode
	clientMeta        *xatu.ClientMeta
}

func NewBeaconCommitteeDeriver(log logrus.FieldLogger, config *BeaconCommitteeDeriverConfig, iter *iterator.BackfillingCheckpoint, beacon *ethereum.BeaconNode, clientMeta *xatu.ClientMeta) *BeaconCommitteeDeriver {
	return &BeaconCommitteeDeriver{
		log: log.WithFields(logrus.Fields{
			"module": "cannon/event/beacon/eth/v1/beacon_committee",
			"type":   BeaconCommitteeDeriverName.String(),
		}),
		cfg:        config,
		iterator:   iter,
		beacon:     beacon,
		clientMeta: clientMeta,
	}
}

func (b *BeaconCommitteeDeriver) CannonType() xatu.CannonType {
	return BeaconCommitteeDeriverName
}

func (b *BeaconCommitteeDeriver) ActivationFork() spec.DataVersion {
	return spec.DataVersionPhase0
}

func (b *BeaconCommitteeDeriver) Name() string {
	return BeaconCommitteeDeriverName.String()
}

func (b *BeaconCommitteeDeriver) OnEventsDerived(ctx context.Context, fn func(ctx context.Context, events []*xatu.DecoratedEvent) error) {
	b.onEventsCallbacks = append(b.onEventsCallbacks, fn)
}

func (b *BeaconCommitteeDeriver) Start(ctx context.Context) error {
	if !b.cfg.Enabled {
		b.log.Info("Beacon committee deriver disabled")

		return nil
	}

	b.log.Info("Beacon committee deriver enabled")

	if err := b.iterator.Start(ctx, b.ActivationFork()); err != nil {
		return errors.Wrap(err, "failed to start iterator")
	}

	// Start our main loop
	b.run(ctx)

	return nil
}

func (b *BeaconCommitteeDeriver) Stop(ctx context.Context) error {
	return nil
}

func (b *BeaconCommitteeDeriver) run(rctx context.Context) {
	bo := backoff.NewExponentialBackOff()
	bo.MaxInterval = 3 * time.Minute

	tracer := observability.Tracer()

	for {
		select {
		case <-rctx.Done():
			return
		default:
			operation := func() (string, error) {
				ctx, span := tracer.Start(rctx, fmt.Sprintf("Derive %s", b.Name()),
					trace.WithAttributes(
						attribute.String("network", string(b.beacon.Metadata().Network.Name))),
				)
				defer span.End()

				time.Sleep(100 * time.Millisecond)

				if err := b.beacon.Synced(ctx); err != nil {
					span.SetStatus(codes.Error, err.Error())

					return "", err
				}

				// Get the next position.
				position, err := b.iterator.Next(ctx)
				if err != nil {
					span.SetStatus(codes.Error, err.Error())

					return "", err
				}

				// Process the epoch
				events, err := b.processEpoch(ctx, position.Next)
				if err != nil {
					b.log.WithError(err).Error("Failed to process epoch")
					span.SetStatus(codes.Error, err.Error())

					return "", err
				}

				// Look ahead
				b.lookAhead(ctx, position.LookAheads)

				// Send the events
				for _, fn := range b.onEventsCallbacks {
					if err := fn(ctx, events); err != nil {
						span.SetStatus(codes.Error, err.Error())

						return "", errors.Wrap(err, "failed to send events")
					}
				}

				// Update our location
				if err := b.iterator.UpdateLocation(ctx, position.Next, position.Direction); err != nil {
					span.SetStatus(codes.Error, err.Error())

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

func (b *BeaconCommitteeDeriver) processEpoch(ctx context.Context, epoch phase0.Epoch) ([]*xatu.DecoratedEvent, error) {
	ctx, span := observability.Tracer().Start(ctx,
		"BeaconCommitteeDeriver.processEpoch",
		trace.WithAttributes(attribute.Int64("epoch", int64(epoch))),
	)
	defer span.End()

	sp, err := b.beacon.Node().Spec()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get beacon spec")
	}

	// Get the beacon committees for this epoch
	beaconCommittees, err := b.beacon.Node().FetchBeaconCommittees(ctx, fmt.Sprintf("%d", phase0.Slot(epoch)*sp.SlotsPerEpoch), nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to fetch beacon committees")
	}

	allEvents := []*xatu.DecoratedEvent{}
	uniqueEpochs := make(map[phase0.Epoch]struct{})
	uniqueSlots := make(map[phase0.Slot]struct{})
	uniqueCommittees := make(map[phase0.CommitteeIndex]struct{})

	for _, committee := range beaconCommittees {
		uniqueEpochs[epoch] = struct{}{}
		uniqueSlots[committee.Slot] = struct{}{}
		uniqueCommittees[committee.Index] = struct{}{}
	}

	if len(uniqueEpochs) > 1 {
		b.log.WithField("epochs", uniqueEpochs).Warn("Multiple epochs found")

		return nil, errors.New("multiple epochs found")
	}

	minSlot := phase0.Slot(epoch) * sp.SlotsPerEpoch
	maxSlot := (phase0.Slot(epoch) * sp.SlotsPerEpoch) + sp.SlotsPerEpoch - 1

	for _, committee := range beaconCommittees {
		if committee.Slot < minSlot || committee.Slot > maxSlot {
			return nil, fmt.Errorf("beacon committee slot outside of epoch. (epoch: %d, slot: %d, min: %d, max: %d)", epoch, committee.Slot, minSlot, maxSlot)
		}

		event, err := b.createEventFromBeaconCommittee(ctx, committee)
		if err != nil {
			b.log.
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

func (b *BeaconCommitteeDeriver) lookAhead(ctx context.Context, epochs []phase0.Epoch) {
	// Not supported.
}

func (b *BeaconCommitteeDeriver) createEventFromBeaconCommittee(ctx context.Context, committee *apiv1.BeaconCommittee) (*xatu.DecoratedEvent, error) {
	// Make a clone of the metadata
	metadata, ok := proto.Clone(b.clientMeta).(*xatu.ClientMeta)
	if !ok {
		return nil, errors.New("failed to clone client metadata")
	}

	validators := []*wrapperspb.UInt64Value{}
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

	additionalData, err := b.getAdditionalData(ctx, committee)
	if err != nil {
		b.log.WithError(err).Error("Failed to get extra beacon committee data")

		return nil, err
	} else {
		decoratedEvent.Meta.Client.AdditionalData = &xatu.ClientMeta_EthV1BeaconCommittee{
			EthV1BeaconCommittee: additionalData,
		}
	}

	return decoratedEvent, nil
}

func (b *BeaconCommitteeDeriver) getAdditionalData(_ context.Context, committee *apiv1.BeaconCommittee) (*xatu.ClientMeta_AdditionalEthV1BeaconCommitteeData, error) {
	extra := &xatu.ClientMeta_AdditionalEthV1BeaconCommitteeData{
		StateId: xatuethv1.StateIDFinalized,
	}

	slot := b.beacon.Metadata().Wallclock().Slots().FromNumber(uint64(committee.Slot))
	epoch := b.beacon.Metadata().Wallclock().Epochs().FromSlot(uint64(committee.Slot))

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
