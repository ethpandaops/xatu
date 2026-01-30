package v1

import (
	"context"
	"fmt"
	"time"

	client "github.com/attestantio/go-eth2-client"
	"github.com/attestantio/go-eth2-client/api"
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
	BeaconSyncCommitteeDeriverName = xatu.CannonType_BEACON_API_ETH_V1_BEACON_SYNC_COMMITTEE
)

type BeaconSyncCommitteeDeriverConfig struct {
	Enabled  bool                                 `yaml:"enabled" default:"true"`
	Iterator iterator.BackfillingCheckpointConfig `yaml:"iterator"`
}

type BeaconSyncCommitteeDeriver struct {
	log               logrus.FieldLogger
	cfg               *BeaconSyncCommitteeDeriverConfig
	iterator          *iterator.BackfillingCheckpoint
	onEventsCallbacks []func(ctx context.Context, events []*xatu.DecoratedEvent) error
	beacon            *ethereum.BeaconNode
	clientMeta        *xatu.ClientMeta
}

func NewBeaconSyncCommitteeDeriver(
	log logrus.FieldLogger,
	config *BeaconSyncCommitteeDeriverConfig,
	iter *iterator.BackfillingCheckpoint,
	beacon *ethereum.BeaconNode,
	clientMeta *xatu.ClientMeta,
) *BeaconSyncCommitteeDeriver {
	return &BeaconSyncCommitteeDeriver{
		log: log.WithFields(logrus.Fields{
			"module": "cannon/event/beacon/eth/v1/beacon_sync_committee",
			"type":   BeaconSyncCommitteeDeriverName.String(),
		}),
		cfg:        config,
		iterator:   iter,
		beacon:     beacon,
		clientMeta: clientMeta,
	}
}

func (b *BeaconSyncCommitteeDeriver) CannonType() xatu.CannonType {
	return BeaconSyncCommitteeDeriverName
}

func (b *BeaconSyncCommitteeDeriver) ActivationFork() spec.DataVersion {
	return spec.DataVersionAltair
}

func (b *BeaconSyncCommitteeDeriver) Name() string {
	return BeaconSyncCommitteeDeriverName.String()
}

func (b *BeaconSyncCommitteeDeriver) OnEventsDerived(
	ctx context.Context,
	fn func(ctx context.Context, events []*xatu.DecoratedEvent) error,
) {
	b.onEventsCallbacks = append(b.onEventsCallbacks, fn)
}

func (b *BeaconSyncCommitteeDeriver) Start(ctx context.Context) error {
	if !b.cfg.Enabled {
		b.log.Info("Beacon sync committee deriver disabled")

		return nil
	}

	b.log.Info("Beacon sync committee deriver enabled")

	if err := b.iterator.Start(ctx, b.ActivationFork()); err != nil {
		return errors.Wrap(err, "failed to start iterator")
	}

	// Start our main loop
	b.run(ctx)

	return nil
}

func (b *BeaconSyncCommitteeDeriver) Stop(ctx context.Context) error {
	return nil
}

func (b *BeaconSyncCommitteeDeriver) run(rctx context.Context) {
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

				// Look ahead (not supported for sync committee)
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

			if _, err := backoff.Retry(rctx, operation,
				backoff.WithBackOff(bo),
				backoff.WithNotify(func(err error, timer time.Duration) {
					b.log.WithError(err).WithField("next_attempt", timer).Warn("Failed to process")
				}),
			); err != nil {
				b.log.WithError(err).Warn("Failed to process")
			}
		}
	}
}

func (b *BeaconSyncCommitteeDeriver) processEpoch(
	ctx context.Context,
	epoch phase0.Epoch,
) ([]*xatu.DecoratedEvent, error) {
	ctx, span := observability.Tracer().Start(ctx,
		"BeaconSyncCommitteeDeriver.processEpoch",
		//nolint:gosec // epoch value will never overflow int64
		trace.WithAttributes(attribute.Int64("epoch", int64(epoch))),
	)
	defer span.End()

	sp, err := b.beacon.Node().Spec()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get beacon spec")
	}

	// Sync committees change every EPOCHS_PER_SYNC_COMMITTEE_PERIOD epochs.
	// We only need to fetch at period boundaries to avoid redundant API calls.
	epochsPerPeriod := uint64(sp.EpochsPerSyncCommitteePeriod)
	if uint64(epoch)%epochsPerPeriod != 0 {
		b.log.WithFields(logrus.Fields{
			"epoch":             epoch,
			"epochs_per_period": epochsPerPeriod,
		}).Debug("Skipping non-period-boundary epoch for sync committee")

		return []*xatu.DecoratedEvent{}, nil
	}

	syncCommitteePeriod := uint64(epoch) / epochsPerPeriod

	b.log.WithFields(logrus.Fields{
		"epoch":                 epoch,
		"sync_committee_period": syncCommitteePeriod,
	}).Debug("Fetching sync committee")

	// Calculate the slot at the epoch boundary
	boundarySlot := phase0.Slot(uint64(epoch) * uint64(sp.SlotsPerEpoch))

	// Fetch sync committee from beacon API
	syncCommittee, err := b.fetchSyncCommittee(ctx, boundarySlot)
	if err != nil {
		return nil, errors.Wrap(err, "failed to fetch sync committee")
	}

	event, err := b.createEventFromSyncCommittee(ctx, syncCommittee, epoch, syncCommitteePeriod)
	if err != nil {
		b.log.
			WithError(err).
			WithField("epoch", epoch).
			WithField("sync_committee_period", syncCommitteePeriod).
			Error("Failed to create event from sync committee")

		return nil, err
	}

	return []*xatu.DecoratedEvent{event}, nil
}

func (b *BeaconSyncCommitteeDeriver) fetchSyncCommittee(
	ctx context.Context,
	slot phase0.Slot,
) (*xatuethv1.SyncCommitteeV2, error) {
	provider, isProvider := b.beacon.Node().Service().(client.SyncCommitteesProvider)
	if !isProvider {
		return nil, errors.New("beacon node does not support sync committees provider")
	}

	// Use the SyncCommittee method from go-eth2-client
	resp, err := provider.SyncCommittee(ctx, &api.SyncCommitteeOpts{
		State: fmt.Sprintf("%d", slot),
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to get sync committee from beacon node")
	}

	if resp == nil || resp.Data == nil {
		return nil, errors.New("sync committee response is nil")
	}

	// Convert to proto format
	validators := make([]*wrapperspb.UInt64Value, len(resp.Data.Validators))
	for i, v := range resp.Data.Validators {
		validators[i] = wrapperspb.UInt64(uint64(v))
	}

	aggregates := make([]*xatuethv1.SyncCommitteeValidatorAggregateV2, len(resp.Data.ValidatorAggregates))
	for i, agg := range resp.Data.ValidatorAggregates {
		aggValidators := make([]*wrapperspb.UInt64Value, len(agg))
		for j, v := range agg {
			aggValidators[j] = wrapperspb.UInt64(uint64(v))
		}

		aggregates[i] = &xatuethv1.SyncCommitteeValidatorAggregateV2{
			Validators: aggValidators,
		}
	}

	return &xatuethv1.SyncCommitteeV2{
		Validators:          validators,
		ValidatorAggregates: aggregates,
	}, nil
}

func (b *BeaconSyncCommitteeDeriver) lookAhead(ctx context.Context, epochs []phase0.Epoch) {
	// Sync committee look-ahead is not supported as there's no preload mechanism
	// and the data changes infrequently (every ~27 hours).
}

func (b *BeaconSyncCommitteeDeriver) createEventFromSyncCommittee(
	ctx context.Context,
	syncCommittee *xatuethv1.SyncCommitteeV2,
	epoch phase0.Epoch,
	syncCommitteePeriod uint64,
) (*xatu.DecoratedEvent, error) {
	// Make a clone of the metadata
	metadata, ok := proto.Clone(b.clientMeta).(*xatu.ClientMeta)
	if !ok {
		return nil, errors.New("failed to clone client metadata")
	}

	decoratedEvent := &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_BEACON_API_ETH_V1_BEACON_SYNC_COMMITTEE,
			DateTime: timestamppb.New(time.Now()),
			Id:       uuid.New().String(),
		},
		Meta: &xatu.Meta{
			Client: metadata,
		},
		Data: &xatu.DecoratedEvent_EthV1BeaconSyncCommittee{
			EthV1BeaconSyncCommittee: &xatu.SyncCommitteeData{
				SyncCommittee: syncCommittee,
			},
		},
	}

	additionalData, err := b.getAdditionalData(ctx, epoch, syncCommitteePeriod)
	if err != nil {
		b.log.WithError(err).Error("Failed to get extra sync committee data")

		return nil, err
	}

	decoratedEvent.Meta.Client.AdditionalData = &xatu.ClientMeta_EthV1BeaconSyncCommittee{
		EthV1BeaconSyncCommittee: additionalData,
	}

	return decoratedEvent, nil
}

func (b *BeaconSyncCommitteeDeriver) getAdditionalData(
	_ context.Context,
	epoch phase0.Epoch,
	syncCommitteePeriod uint64,
) (*xatu.ClientMeta_AdditionalEthV1BeaconSyncCommitteeData, error) {
	epochInfo := b.beacon.Metadata().Wallclock().Epochs().FromNumber(uint64(epoch))

	return &xatu.ClientMeta_AdditionalEthV1BeaconSyncCommitteeData{
		Epoch: &xatu.EpochV2{
			Number:        &wrapperspb.UInt64Value{Value: uint64(epoch)},
			StartDateTime: timestamppb.New(epochInfo.TimeWindow().Start()),
		},
		SyncCommitteePeriod: &wrapperspb.UInt64Value{Value: syncCommitteePeriod},
	}, nil
}
