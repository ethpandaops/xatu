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
	BlockAccessListDeriverName = xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_ACCESS_LIST
)

// BlockAccessListDeriverConfig holds the configuration for the BAL deriver.
type BlockAccessListDeriverConfig struct {
	Enabled  bool                                 `yaml:"enabled" default:"true"`
	Iterator iterator.BackfillingCheckpointConfig `yaml:"iterator"`
}

// BlockAccessListDeriver extracts block access list data from Gloas beacon blocks.
// BAL (Block Access List) is introduced in EIP-7928 and only exists from the
// Gloas fork onwards. The BAL data is embedded in the ExecutionPayload as the
// BlockAccessList field.
type BlockAccessListDeriver struct {
	log               logrus.FieldLogger
	cfg               *BlockAccessListDeriverConfig
	iterator          *iterator.BackfillingCheckpoint
	onEventsCallbacks []func(ctx context.Context, events []*xatu.DecoratedEvent) error
	beacon            *ethereum.BeaconNode
	clientMeta        *xatu.ClientMeta
}

// NewBlockAccessListDeriver creates a new BlockAccessListDeriver.
func NewBlockAccessListDeriver(
	log logrus.FieldLogger,
	config *BlockAccessListDeriverConfig,
	iter *iterator.BackfillingCheckpoint,
	beacon *ethereum.BeaconNode,
	clientMeta *xatu.ClientMeta,
) *BlockAccessListDeriver {
	return &BlockAccessListDeriver{
		log: log.WithFields(logrus.Fields{
			"module": "cannon/event/beacon/eth/v2/block_access_list",
			"type":   BlockAccessListDeriverName.String(),
		}),
		cfg:        config,
		iterator:   iter,
		beacon:     beacon,
		clientMeta: clientMeta,
	}
}

// CannonType returns the cannon type for this deriver.
func (b *BlockAccessListDeriver) CannonType() xatu.CannonType {
	return BlockAccessListDeriverName
}

// Name returns the human-readable name for this deriver.
func (b *BlockAccessListDeriver) Name() string {
	return BlockAccessListDeriverName.String()
}

// ActivationFork returns the fork at which this deriver activates.
func (b *BlockAccessListDeriver) ActivationFork() spec.DataVersion {
	return spec.DataVersionGloas
}

// OnEventsDerived registers a callback for when events are derived.
func (b *BlockAccessListDeriver) OnEventsDerived(
	ctx context.Context,
	fn func(ctx context.Context, events []*xatu.DecoratedEvent) error,
) {
	b.onEventsCallbacks = append(b.onEventsCallbacks, fn)
}

// Start begins the deriver's main processing loop.
func (b *BlockAccessListDeriver) Start(ctx context.Context) error {
	if !b.cfg.Enabled {
		b.log.Info("Block access list deriver disabled")

		return nil
	}

	b.log.Info("Block access list deriver enabled")

	if err := b.iterator.Start(ctx, b.ActivationFork()); err != nil {
		return errors.Wrap(err, "failed to start iterator")
	}

	// Start our main loop
	b.run(ctx)

	return nil
}

// Stop gracefully stops the deriver.
func (b *BlockAccessListDeriver) Stop(ctx context.Context) error {
	return nil
}

func (b *BlockAccessListDeriver) run(rctx context.Context) {
	bo := backoff.NewExponentialBackOff()
	bo.MaxInterval = 3 * time.Minute

	for {
		select {
		case <-rctx.Done():
			return
		default:
			operation := func() (string, error) {
				ctx, span := observability.Tracer().Start(rctx,
					fmt.Sprintf("Derive %s", b.Name()),
					trace.WithAttributes(
						attribute.String("network",
							string(b.beacon.Metadata().Network.Name))),
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

				for _, fn := range b.onEventsCallbacks {
					if errr := fn(ctx, events); errr != nil {
						return "", errors.Wrapf(errr, "failed to send events")
					}
				}

				// Update our location
				if err := b.iterator.UpdateLocation(ctx, position.Next,
					position.Direction); err != nil {
					return "", err
				}

				bo.Reset()

				return "", nil
			}

			retryOpts := []backoff.RetryOption{
				backoff.WithBackOff(bo),
				backoff.WithNotify(func(err error, timer time.Duration) {
					b.log.WithError(err).WithField("next_attempt", timer).
						Warn("Failed to process")
				}),
			}
			if _, err := backoff.Retry(rctx, operation, retryOpts...); err != nil {
				b.log.WithError(err).Warn("Failed to process")
			}
		}
	}
}

func (b *BlockAccessListDeriver) processEpoch(
	ctx context.Context,
	epoch phase0.Epoch,
) ([]*xatu.DecoratedEvent, error) {
	ctx, span := observability.Tracer().Start(ctx,
		"BlockAccessListDeriver.processEpoch",
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

func (b *BlockAccessListDeriver) processSlot(
	ctx context.Context,
	slot phase0.Slot,
) ([]*xatu.DecoratedEvent, error) {
	ctx, span := observability.Tracer().Start(ctx,
		"BlockAccessListDeriver.processSlot",
		//nolint:gosec // slot value will never overflow int64
		trace.WithAttributes(attribute.Int64("slot", int64(slot))),
	)
	defer span.End()

	// Get the block
	block, err := b.beacon.GetBeaconBlock(ctx, xatuethv1.SlotAsString(slot))
	if err != nil {
		return nil, errors.Wrapf(err,
			"failed to get beacon block for slot %d", slot)
	}

	if block == nil {
		return []*xatu.DecoratedEvent{}, nil
	}

	// BAL only exists from Gloas onwards. Pre-Gloas blocks have no BAL data.
	if block.Version < spec.DataVersionGloas {
		return []*xatu.DecoratedEvent{}, nil
	}

	if block.Gloas == nil || block.Gloas.Message == nil ||
		block.Gloas.Message.Body == nil ||
		block.Gloas.Message.Body.ExecutionPayload == nil {
		return []*xatu.DecoratedEvent{}, nil
	}

	blockIdentifier, err := GetBlockIdentifier(
		block, b.beacon.Metadata().Wallclock(),
	)
	if err != nil {
		return nil, errors.Wrapf(err,
			"failed to get block identifier for slot %d", slot)
	}

	execPayload := block.Gloas.Message.Body.ExecutionPayload
	execBlockNumber := execPayload.BlockNumber
	execBlockHash := fmt.Sprintf("%#x", execPayload.BlockHash)

	// Decode the BAL from the ExecutionPayload
	rawBAL := execPayload.BlockAccessList
	b.log.WithField("slot", slot).WithField("raw_bal_len", len(rawBAL)).Debug("Processing BAL data")

	bal := xatuethv1.NewBlockAccessListFromGloas(rawBAL)

	b.log.WithField("slot", slot).WithField("entries", len(bal.GetEntries())).Debug("Decoded BAL entries")

	if bal == nil || len(bal.GetEntries()) == 0 {
		return []*xatu.DecoratedEvent{}, nil
	}

	events := make([]*xatu.DecoratedEvent, 0)

	// Iterate over entries and their changes, creating one event per change
	for _, entry := range bal.GetEntries() {
		address := entry.GetAddress()

		for _, sc := range entry.GetStorageChanges() {
			change := &xatuethv1.BlockAccessListChange{
				Address:          address,
				ChangeType:       "storage",
				BlockAccessIndex: sc.GetBlockAccessIndex(),
				StorageKey:       sc.GetKey(),
				NewValue:         sc.GetNewValue(),
			}

			event, err := b.createEvent(ctx, change, blockIdentifier, execBlockNumber, execBlockHash)
			if err != nil {
				return nil, errors.Wrap(err, "failed to create storage change event")
			}

			events = append(events, event)
		}

		for _, bc := range entry.GetBalanceChanges() {
			change := &xatuethv1.BlockAccessListChange{
				Address:          address,
				ChangeType:       "balance",
				BlockAccessIndex: bc.GetBlockAccessIndex(),
				NewValue:         bc.GetPostBalance(),
			}

			event, err := b.createEvent(ctx, change, blockIdentifier, execBlockNumber, execBlockHash)
			if err != nil {
				return nil, errors.Wrap(err, "failed to create balance change event")
			}

			events = append(events, event)
		}

		for _, nc := range entry.GetNonceChanges() {
			var newValue *wrapperspb.StringValue
			if nc.GetNewNonce() != nil {
				newValue = &wrapperspb.StringValue{
					Value: fmt.Sprintf("%d", nc.GetNewNonce().GetValue()),
				}
			}

			change := &xatuethv1.BlockAccessListChange{
				Address:          address,
				ChangeType:       "nonce",
				BlockAccessIndex: nc.GetBlockAccessIndex(),
				NewValue:         newValue,
			}

			event, err := b.createEvent(ctx, change, blockIdentifier, execBlockNumber, execBlockHash)
			if err != nil {
				return nil, errors.Wrap(err, "failed to create nonce change event")
			}

			events = append(events, event)
		}

		for _, cc := range entry.GetCodeChanges() {
			change := &xatuethv1.BlockAccessListChange{
				Address:          address,
				ChangeType:       "code",
				BlockAccessIndex: cc.GetBlockAccessIndex(),
				NewValue:         cc.GetNewCode(),
			}

			event, err := b.createEvent(ctx, change, blockIdentifier, execBlockNumber, execBlockHash)
			if err != nil {
				return nil, errors.Wrap(err, "failed to create code change event")
			}

			events = append(events, event)
		}

		for _, readKey := range entry.GetStorageReads() {
			change := &xatuethv1.BlockAccessListChange{
				Address:    address,
				ChangeType: "storage_read",
				StorageKey: readKey.GetKey(),
			}

			event, err := b.createEvent(ctx, change, blockIdentifier, execBlockNumber, execBlockHash)
			if err != nil {
				return nil, errors.Wrap(err, "failed to create storage read event")
			}

			events = append(events, event)
		}

		// Emit a "touched" event for accounts with no state changes and no storage
		// reads. These are accounts accessed via BALANCE, EXTCODESIZE, calls, etc.
		// that had no state interactions. Valuable for parallel execution analysis.
		if len(entry.GetStorageChanges()) == 0 &&
			len(entry.GetStorageReads()) == 0 &&
			len(entry.GetBalanceChanges()) == 0 &&
			len(entry.GetNonceChanges()) == 0 &&
			len(entry.GetCodeChanges()) == 0 {
			change := &xatuethv1.BlockAccessListChange{
				Address:    address,
				ChangeType: "touched",
			}

			event, err := b.createEvent(ctx, change, blockIdentifier, execBlockNumber, execBlockHash)
			if err != nil {
				return nil, errors.Wrap(err, "failed to create touched account event")
			}

			events = append(events, event)
		}
	}

	return events, nil
}

func (b *BlockAccessListDeriver) createEvent(
	ctx context.Context,
	change *xatuethv1.BlockAccessListChange,
	blockIdentifier *xatu.BlockIdentifier,
	execBlockNumber uint64,
	execBlockHash string,
) (*xatu.DecoratedEvent, error) {
	metadata, ok := proto.Clone(b.clientMeta).(*xatu.ClientMeta)
	if !ok {
		return nil, errors.New("failed to clone client metadata")
	}

	decoratedEvent := &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_ACCESS_LIST,
			DateTime: timestamppb.New(time.Now()),
			Id:       uuid.New().String(),
		},
		Meta: &xatu.Meta{
			Client: metadata,
		},
		Data: &xatu.DecoratedEvent_EthV2BeaconBlockAccessList{
			EthV2BeaconBlockAccessList: change,
		},
	}

	decoratedEvent.Meta.Client.AdditionalData = &xatu.ClientMeta_EthV2BeaconBlockAccessList{
		EthV2BeaconBlockAccessList: &xatu.ClientMeta_AdditionalEthV2BeaconBlockAccessListData{
			Block:       blockIdentifier,
			BlockNumber: &wrapperspb.UInt64Value{Value: execBlockNumber},
			BlockHash:   execBlockHash,
		},
	}

	return decoratedEvent, nil
}

// lookAhead attempts to pre-load any blocks that might be required for
// the epochs that are coming up.
func (b *BlockAccessListDeriver) lookAhead(
	ctx context.Context,
	epochs []phase0.Epoch,
) {
	_, span := observability.Tracer().Start(ctx,
		"BlockAccessListDeriver.lookAhead",
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

			// Add the block to the preload queue so it's available when
			// we need it
			b.beacon.LazyLoadBeaconBlock(xatuethv1.SlotAsString(slot))
		}
	}
}
