package v2

import (
	"context"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"
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
	BeaconBlockSyncAggregateDeriverName = xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_SYNC_AGGREGATE
)

type BeaconBlockSyncAggregateDeriverConfig struct {
	Enabled  bool                                 `yaml:"enabled" default:"true"`
	Iterator iterator.BackfillingCheckpointConfig `yaml:"iterator"`
}

type BeaconBlockSyncAggregateDeriver struct {
	log               logrus.FieldLogger
	cfg               *BeaconBlockSyncAggregateDeriverConfig
	iterator          *iterator.BackfillingCheckpoint
	onEventsCallbacks []func(ctx context.Context, events []*xatu.DecoratedEvent) error
	beacon            *ethereum.BeaconNode
	clientMeta        *xatu.ClientMeta

	// Cache sync committee composition per period (changes every 256 epochs)
	syncCommitteeCache   map[uint64]*xatuethv1.SyncCommittee
	syncCommitteeCacheMu sync.RWMutex
}

func NewBeaconBlockSyncAggregateDeriver(
	log logrus.FieldLogger,
	config *BeaconBlockSyncAggregateDeriverConfig,
	iter *iterator.BackfillingCheckpoint,
	beacon *ethereum.BeaconNode,
	clientMeta *xatu.ClientMeta,
) *BeaconBlockSyncAggregateDeriver {
	return &BeaconBlockSyncAggregateDeriver{
		log: log.WithFields(logrus.Fields{
			"module": "cannon/event/beacon/eth/v2/beacon_block_sync_aggregate",
			"type":   BeaconBlockSyncAggregateDeriverName.String(),
		}),
		cfg:                config,
		iterator:           iter,
		beacon:             beacon,
		clientMeta:         clientMeta,
		syncCommitteeCache: make(map[uint64]*xatuethv1.SyncCommittee, 2),
	}
}

func (b *BeaconBlockSyncAggregateDeriver) CannonType() xatu.CannonType {
	return BeaconBlockSyncAggregateDeriverName
}

func (b *BeaconBlockSyncAggregateDeriver) ActivationFork() spec.DataVersion {
	return spec.DataVersionAltair
}

func (b *BeaconBlockSyncAggregateDeriver) Name() string {
	return BeaconBlockSyncAggregateDeriverName.String()
}

func (b *BeaconBlockSyncAggregateDeriver) OnEventsDerived(
	ctx context.Context,
	fn func(ctx context.Context, events []*xatu.DecoratedEvent) error,
) {
	b.onEventsCallbacks = append(b.onEventsCallbacks, fn)
}

func (b *BeaconBlockSyncAggregateDeriver) Start(ctx context.Context) error {
	if !b.cfg.Enabled {
		b.log.Info("Beacon block sync aggregate deriver disabled")

		return nil
	}

	b.log.Info("Beacon block sync aggregate deriver enabled")

	if err := b.iterator.Start(ctx, b.ActivationFork()); err != nil {
		return errors.Wrap(err, "failed to start iterator")
	}

	// Start our main loop
	b.run(ctx)

	return nil
}

func (b *BeaconBlockSyncAggregateDeriver) Stop(ctx context.Context) error {
	return nil
}

func (b *BeaconBlockSyncAggregateDeriver) run(rctx context.Context) {
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

				// Get the next slot
				position, err := b.iterator.Next(ctx)
				if err != nil {
					span.SetStatus(codes.Error, err.Error())

					return "", err
				}

				// Look ahead
				b.lookAhead(ctx, position.LookAheads)

				// Process the epoch
				events, err := b.processEpoch(ctx, position.Next)
				if err != nil {
					b.log.WithError(err).Error("Failed to process epoch")

					span.SetStatus(codes.Error, err.Error())

					return "", err
				}

				span.AddEvent("Epoch processing complete. Sending events...")

				// Send the events
				for _, fn := range b.onEventsCallbacks {
					if err := fn(ctx, events); err != nil {
						span.SetStatus(codes.Error, err.Error())

						return "", errors.Wrap(err, "failed to send events")
					}
				}

				span.AddEvent("Events sent. Updating location...")

				// Update our location
				if err := b.iterator.UpdateLocation(ctx, position.Next, position.Direction); err != nil {
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
					b.log.WithError(err).WithField("next_attempt", timer).Warn("Failed to process")
				}),
			}
			if _, err := backoff.Retry(rctx, operation, retryOpts...); err != nil {
				b.log.WithError(err).Warn("Failed to process")
			}
		}
	}
}

// lookAhead takes the upcoming epochs and looks ahead to do any pre-processing that might be required.
func (b *BeaconBlockSyncAggregateDeriver) lookAhead(ctx context.Context, epochs []phase0.Epoch) {
	_, span := observability.Tracer().Start(ctx,
		"BeaconBlockSyncAggregateDeriver.lookAhead",
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

func (b *BeaconBlockSyncAggregateDeriver) processEpoch(
	ctx context.Context,
	epoch phase0.Epoch,
) ([]*xatu.DecoratedEvent, error) {
	ctx, span := observability.Tracer().Start(ctx,
		"BeaconBlockSyncAggregateDeriver.processEpoch",
		//nolint:gosec // epoch value will never overflow int64
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

func (b *BeaconBlockSyncAggregateDeriver) processSlot(
	ctx context.Context,
	slot phase0.Slot,
) ([]*xatu.DecoratedEvent, error) {
	ctx, span := observability.Tracer().Start(ctx,
		"BeaconBlockSyncAggregateDeriver.processSlot",
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

	// Only process blocks from Altair onwards (sync aggregates were introduced in Altair)
	if block.Version < spec.DataVersionAltair {
		return []*xatu.DecoratedEvent{}, nil
	}

	// Get the sync aggregate from the block
	syncAggregate, err := b.getSyncAggregate(block)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get sync aggregate from block")
	}

	if syncAggregate == nil {
		return []*xatu.DecoratedEvent{}, nil
	}

	// Get the sync committee for this slot's period
	syncCommittee, err := b.getSyncCommitteeForSlot(ctx, slot)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get sync committee for slot")
	}

	// Expand the sync aggregate bits to validator indices
	participated, missed, err := b.expandSyncAggregateBits(syncAggregate.SyncCommitteeBits, syncCommittee)
	if err != nil {
		return nil, errors.Wrap(err, "failed to expand sync aggregate bits")
	}

	event, err := b.createEvent(ctx, block, syncAggregate, participated, missed)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create event")
	}

	return []*xatu.DecoratedEvent{event}, nil
}

// getSyncAggregate extracts the sync aggregate from a versioned beacon block.
func (b *BeaconBlockSyncAggregateDeriver) getSyncAggregate(
	block *spec.VersionedSignedBeaconBlock,
) (*xatuethv1.SyncAggregate, error) {
	var (
		bits      []byte
		signature []byte
	)

	switch block.Version {
	case spec.DataVersionAltair:
		if block.Altair == nil || block.Altair.Message == nil || block.Altair.Message.Body == nil {
			return nil, nil //nolint:nilnil // nil indicates no sync aggregate available
		}

		sa := block.Altair.Message.Body.SyncAggregate
		bits = sa.SyncCommitteeBits[:]
		signature = sa.SyncCommitteeSignature[:]
	case spec.DataVersionBellatrix:
		if block.Bellatrix == nil || block.Bellatrix.Message == nil || block.Bellatrix.Message.Body == nil {
			return nil, nil //nolint:nilnil // nil indicates no sync aggregate available
		}

		sa := block.Bellatrix.Message.Body.SyncAggregate
		bits = sa.SyncCommitteeBits[:]
		signature = sa.SyncCommitteeSignature[:]
	case spec.DataVersionCapella:
		if block.Capella == nil || block.Capella.Message == nil || block.Capella.Message.Body == nil {
			return nil, nil //nolint:nilnil // nil indicates no sync aggregate available
		}

		sa := block.Capella.Message.Body.SyncAggregate
		bits = sa.SyncCommitteeBits[:]
		signature = sa.SyncCommitteeSignature[:]
	case spec.DataVersionDeneb:
		if block.Deneb == nil || block.Deneb.Message == nil || block.Deneb.Message.Body == nil {
			return nil, nil //nolint:nilnil // nil indicates no sync aggregate available
		}

		sa := block.Deneb.Message.Body.SyncAggregate
		bits = sa.SyncCommitteeBits[:]
		signature = sa.SyncCommitteeSignature[:]
	case spec.DataVersionElectra:
		if block.Electra == nil || block.Electra.Message == nil || block.Electra.Message.Body == nil {
			return nil, nil //nolint:nilnil // nil indicates no sync aggregate available
		}

		sa := block.Electra.Message.Body.SyncAggregate
		bits = sa.SyncCommitteeBits[:]
		signature = sa.SyncCommitteeSignature[:]
	case spec.DataVersionFulu:
		if block.Fulu == nil || block.Fulu.Message == nil || block.Fulu.Message.Body == nil {
			return nil, nil //nolint:nilnil // nil indicates no sync aggregate available
		}

		sa := block.Fulu.Message.Body.SyncAggregate
		bits = sa.SyncCommitteeBits[:]
		signature = sa.SyncCommitteeSignature[:]
	default:
		return nil, fmt.Errorf("unsupported block version: %s", block.Version)
	}

	return &xatuethv1.SyncAggregate{
		SyncCommitteeBits:      fmt.Sprintf("0x%s", hex.EncodeToString(bits)),
		SyncCommitteeSignature: fmt.Sprintf("0x%s", hex.EncodeToString(signature)),
	}, nil
}

// getSyncCommitteeForSlot gets the sync committee for a given slot, using a cache.
func (b *BeaconBlockSyncAggregateDeriver) getSyncCommitteeForSlot(
	ctx context.Context,
	slot phase0.Slot,
) (*xatuethv1.SyncCommittee, error) {
	sp, err := b.beacon.Node().Spec()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get beacon spec")
	}

	// Calculate the sync committee period for this slot
	epoch := phase0.Epoch(uint64(slot) / uint64(sp.SlotsPerEpoch))
	epochsPerPeriod := uint64(sp.EpochsPerSyncCommitteePeriod)
	period := uint64(epoch) / epochsPerPeriod

	// Check cache first
	b.syncCommitteeCacheMu.RLock()
	syncCommittee, exists := b.syncCommitteeCache[period]
	b.syncCommitteeCacheMu.RUnlock()

	if exists {
		return syncCommittee, nil
	}

	// Fetch from beacon node
	syncCommittee, err = b.fetchSyncCommittee(ctx, slot)
	if err != nil {
		return nil, err
	}

	// Update cache - keep only current and next period to limit memory usage
	b.syncCommitteeCacheMu.Lock()
	// Clear old entries if cache is getting large
	if len(b.syncCommitteeCache) >= 3 {
		for k := range b.syncCommitteeCache {
			if k < period-1 {
				delete(b.syncCommitteeCache, k)
			}
		}
	}

	b.syncCommitteeCache[period] = syncCommittee
	b.syncCommitteeCacheMu.Unlock()

	return syncCommittee, nil
}

func (b *BeaconBlockSyncAggregateDeriver) fetchSyncCommittee(
	ctx context.Context,
	slot phase0.Slot,
) (*xatuethv1.SyncCommittee, error) {
	provider, isProvider := b.beacon.Node().Service().(client.SyncCommitteesProvider)
	if !isProvider {
		return nil, errors.New("beacon node does not support sync committees provider")
	}

	sp, err := b.beacon.Node().Spec()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get beacon spec")
	}

	// Calculate the sync committee period boundary slot.
	// The sync committee is the same for all slots in a period, so we use the
	// boundary slot which is more likely to have state available on beacon nodes.
	epoch := phase0.Epoch(uint64(slot) / uint64(sp.SlotsPerEpoch))
	epochsPerPeriod := uint64(sp.EpochsPerSyncCommitteePeriod)
	boundaryEpoch := phase0.Epoch((uint64(epoch) / epochsPerPeriod) * epochsPerPeriod)
	boundarySlot := phase0.Slot(uint64(boundaryEpoch) * uint64(sp.SlotsPerEpoch))

	// Use the SyncCommittee method from go-eth2-client
	resp, err := provider.SyncCommittee(ctx, &api.SyncCommitteeOpts{
		State: fmt.Sprintf("%d", boundarySlot),
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

	aggregates := make([]*xatuethv1.SyncCommitteeValidatorAggregate, len(resp.Data.ValidatorAggregates))
	for i, agg := range resp.Data.ValidatorAggregates {
		aggValidators := make([]*wrapperspb.UInt64Value, len(agg))
		for j, v := range agg {
			aggValidators[j] = wrapperspb.UInt64(uint64(v))
		}

		aggregates[i] = &xatuethv1.SyncCommitteeValidatorAggregate{
			Validators: aggValidators,
		}
	}

	return &xatuethv1.SyncCommittee{
		Validators:          validators,
		ValidatorAggregates: aggregates,
	}, nil
}

// expandSyncAggregateBits expands the sync committee bitvector to validator indices.
// Returns participated and missed validator indices.
func (b *BeaconBlockSyncAggregateDeriver) expandSyncAggregateBits(
	bitsHex string,
	syncCommittee *xatuethv1.SyncCommittee,
) (participated, missed []uint64, err error) {
	// Remove 0x prefix if present
	bitsHex = strings.TrimPrefix(bitsHex, "0x")

	bits, err := hex.DecodeString(bitsHex)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to decode sync committee bits")
	}

	// Get sync committee size from spec
	sp, err := b.beacon.Node().Spec()
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to get beacon spec")
	}

	//nolint:gosec // SyncCommitteeSize is a small protocol constant, will never overflow int
	syncCommitteeSize := int(sp.SyncCommitteeSize)

	const numAggregates = 4 // Fixed: always 4 subcommittees in protocol

	validatorsPerAggregate := syncCommitteeSize / numAggregates

	// Pre-allocate slices with reasonable capacity
	participated = make([]uint64, 0, syncCommitteeSize)
	missed = make([]uint64, 0, syncCommitteeSize)

	// Verify sync committee structure
	if len(syncCommittee.ValidatorAggregates) != numAggregates {
		return nil, nil, errors.Errorf(
			"expected %d validator aggregates, got %d",
			numAggregates,
			len(syncCommittee.ValidatorAggregates),
		)
	}

	for i := range syncCommitteeSize {
		aggregateIndex := i / validatorsPerAggregate
		positionInAggregate := i % validatorsPerAggregate

		if aggregateIndex >= len(syncCommittee.ValidatorAggregates) {
			return nil, nil, errors.Errorf("aggregate index %d out of range", aggregateIndex)
		}

		aggregate := syncCommittee.ValidatorAggregates[aggregateIndex]
		if positionInAggregate >= len(aggregate.Validators) {
			return nil, nil, errors.Errorf(
				"position %d out of range in aggregate %d",
				positionInAggregate,
				aggregateIndex,
			)
		}

		validatorIndex := aggregate.Validators[positionInAggregate].GetValue()

		// Check if this validator participated (bit is set)
		byteIndex := i / 8
		bitIndex := i % 8

		if byteIndex >= len(bits) {
			return nil, nil, errors.Errorf("byte index %d out of range", byteIndex)
		}

		if (bits[byteIndex] & (1 << bitIndex)) != 0 {
			participated = append(participated, validatorIndex)
		} else {
			missed = append(missed, validatorIndex)
		}
	}

	return participated, missed, nil
}

func (b *BeaconBlockSyncAggregateDeriver) createEvent(
	ctx context.Context,
	block *spec.VersionedSignedBeaconBlock,
	syncAggregate *xatuethv1.SyncAggregate,
	participated, missed []uint64,
) (*xatu.DecoratedEvent, error) {
	// Make a clone of the metadata
	metadata, ok := proto.Clone(b.clientMeta).(*xatu.ClientMeta)
	if !ok {
		return nil, errors.New("failed to clone client metadata")
	}

	// Convert participated and missed to wrapped values
	participatedWrapped := make([]*wrapperspb.UInt64Value, len(participated))
	for i, v := range participated {
		participatedWrapped[i] = wrapperspb.UInt64(v)
	}

	missedWrapped := make([]*wrapperspb.UInt64Value, len(missed))
	for i, v := range missed {
		missedWrapped[i] = wrapperspb.UInt64(v)
	}

	data := &xatu.SyncAggregateData{
		SyncCommitteeBits:      syncAggregate.SyncCommitteeBits,
		SyncCommitteeSignature: syncAggregate.SyncCommitteeSignature,
		ValidatorsParticipated: participatedWrapped,
		ValidatorsMissed:       missedWrapped,
		ParticipationCount:     wrapperspb.UInt64(uint64(len(participated))),
	}

	decoratedEvent := &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_SYNC_AGGREGATE,
			DateTime: timestamppb.New(time.Now()),
			Id:       uuid.New().String(),
		},
		Meta: &xatu.Meta{
			Client: metadata,
		},
		Data: &xatu.DecoratedEvent_EthV2BeaconBlockSyncAggregate{
			EthV2BeaconBlockSyncAggregate: data,
		},
	}

	additionalData, err := b.getAdditionalData(ctx, block)
	if err != nil {
		b.log.WithError(err).Error("Failed to get additional sync aggregate data")

		return nil, err
	}

	decoratedEvent.Meta.Client.AdditionalData = &xatu.ClientMeta_EthV2BeaconBlockSyncAggregate{
		EthV2BeaconBlockSyncAggregate: additionalData,
	}

	return decoratedEvent, nil
}

func (b *BeaconBlockSyncAggregateDeriver) getAdditionalData(
	_ context.Context,
	block *spec.VersionedSignedBeaconBlock,
) (*xatu.ClientMeta_AdditionalEthV2BeaconBlockSyncAggregateData, error) {
	sp, err := b.beacon.Node().Spec()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get beacon spec")
	}

	blockIdentifier, err := GetBlockIdentifier(block, b.beacon.Metadata().Wallclock())
	if err != nil {
		return nil, errors.Wrap(err, "failed to get block identifier")
	}

	slotNum, err := block.Slot()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get slot from block")
	}

	// Calculate sync committee period
	epoch := uint64(slotNum) / uint64(sp.SlotsPerEpoch)
	epochsPerPeriod := uint64(sp.EpochsPerSyncCommitteePeriod)
	syncCommitteePeriod := epoch / epochsPerPeriod

	return &xatu.ClientMeta_AdditionalEthV2BeaconBlockSyncAggregateData{
		Block:               blockIdentifier,
		SyncCommitteePeriod: wrapperspb.UInt64(syncCommitteePeriod),
	}, nil
}
