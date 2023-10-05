package blockprint

import (
	"context"
	"fmt"
	"time"

	"github.com/attestantio/go-eth2-client/spec/phase0"
	backoff "github.com/cenkalti/backoff/v4"
	aBlockprint "github.com/ethpandaops/xatu/pkg/cannon/blockprint"
	"github.com/ethpandaops/xatu/pkg/cannon/ethereum"
	"github.com/ethpandaops/xatu/pkg/cannon/iterator"
	"github.com/ethpandaops/xatu/pkg/observability"
	pBlockprint "github.com/ethpandaops/xatu/pkg/proto/blockprint"
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
	BlockClassificationName = xatu.CannonType_BLOCKPRINT_BLOCK_CLASSIFICATION
)

type BlockClassificationDeriverConfig struct {
	Enabled   bool              `yaml:"enabled" default:"false"`
	Endpoint  string            `yaml:"endpoint" default:"http://localhost:8080"`
	Headers   map[string]string `yaml:"headers"`
	BatchSize int               `yaml:"batchSize" default:"50"`
}

func (c *BlockClassificationDeriverConfig) Validate() error {
	if c.BatchSize < 1 {
		return errors.New("batch size must be greater than 0")
	}

	return nil
}

type BlockClassificationDeriver struct {
	log               logrus.FieldLogger
	cfg               *BlockClassificationDeriverConfig
	iterator          *iterator.BlockprintIterator
	onEventsCallbacks []func(ctx context.Context, events []*xatu.DecoratedEvent) error
	beacon            *ethereum.BeaconNode
	clientMeta        *xatu.ClientMeta
	blockprintClient  *aBlockprint.Client
}

func NewBlockClassificationDeriver(log logrus.FieldLogger, config *BlockClassificationDeriverConfig, iter *iterator.BlockprintIterator, beacon *ethereum.BeaconNode, clientMeta *xatu.ClientMeta, client *aBlockprint.Client) *BlockClassificationDeriver {
	return &BlockClassificationDeriver{
		log:              log.WithField("module", "cannon/event/blockprint/block_classification"),
		cfg:              config,
		iterator:         iter,
		beacon:           beacon,
		clientMeta:       clientMeta,
		blockprintClient: client,
	}
}

func (b *BlockClassificationDeriver) CannonType() xatu.CannonType {
	return BlockClassificationName
}

func (b *BlockClassificationDeriver) Name() string {
	return BlockClassificationName.String()
}

func (b *BlockClassificationDeriver) OnEventsDerived(ctx context.Context, fn func(ctx context.Context, events []*xatu.DecoratedEvent) error) {
	b.onEventsCallbacks = append(b.onEventsCallbacks, fn)
}

func (b *BlockClassificationDeriver) Start(ctx context.Context) error {
	if !b.cfg.Enabled {
		b.log.Info("BlockClassification deriver disabled")

		return nil
	}

	b.log.Info("BlockClassification deriver enabled")

	// Start our main loop
	go b.run(ctx)

	return nil
}

func (b *BlockClassificationDeriver) Stop(ctx context.Context) error {
	return nil
}

func (b *BlockClassificationDeriver) run(rctx context.Context) {
	bo := backoff.NewExponentialBackOff()
	bo.MaxInterval = 10 * time.Minute

	for {
		select {
		case <-rctx.Done():
			return
		default:
			operation := func() error {
				ctx, span := observability.Tracer().Start(rctx, fmt.Sprintf("Derive %s", b.Name()),
					trace.WithAttributes(
						attribute.String("network", string(b.beacon.Metadata().Network.Name))),
				)
				defer span.End()

				time.Sleep(1000 * time.Millisecond)

				// Get the next slot
				location, _, err := b.iterator.Next(ctx)
				if err != nil {
					return err
				}

				// Process the location
				data := location.GetBlockprintBlockClassification()

				currentSlot := phase0.Slot(data.Slot)
				targetEndSlot := phase0.Slot(data.TargetEndSlot)

				// Fetch the range from blockprint using our batch size
				end := currentSlot + phase0.Slot(b.cfg.BatchSize)
				if end > targetEndSlot {
					end = targetEndSlot
				}

				if currentSlot >= end {
					return errors.New("current slot is equal or larger than end slot")
				}

				events, err := b.processLocation(ctx, currentSlot, end)
				if err != nil {
					return errors.Wrapf(err, "failed to process location start_slot: %d end_slot: %d", currentSlot, end)
				}

				for _, fn := range b.onEventsCallbacks {
					if errr := fn(ctx, events); errr != nil {
						return errors.Wrapf(errr, "failed to send events")
					}
				}

				location.Data = &xatu.CannonLocation_BlockprintBlockClassification{
					BlockprintBlockClassification: &xatu.CannonLocationBlockprintBlockClassification{
						Slot:          uint64(end),
						TargetEndSlot: uint64(targetEndSlot),
					},
				}

				// Update our location
				if err := b.iterator.UpdateLocation(ctx, location); err != nil {
					return err
				}

				// Sleep if everything went well to avoid hammering the API
				time.Sleep(5000 * time.Millisecond)

				bo.Reset()

				return nil
			}

			if err := backoff.RetryNotify(operation, bo, func(err error, timer time.Duration) {
				b.log.WithError(err).WithField("next_attempt", timer).Warn("Failed to process")
			}); err != nil {
				b.log.WithError(err).Warn("Failed to process")
			}
		}
	}
}

func (b *BlockClassificationDeriver) processLocation(ctx context.Context, start, end phase0.Slot) ([]*xatu.DecoratedEvent, error) {
	// Get the blockprint data
	data, err := b.blockprintClient.BlocksRange(ctx, start, end)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get blockprint data")
	}

	diff := end - start

	if len(data) != int(diff) {
		b.log.WithField("start", start).WithField("end", end).WithField("diff", diff).WithField("data", len(data)).Warn("Blockprint data length does not match expected")
	}

	events, err := b.processBlocks(ctx, data)
	if err != nil {
		return nil, errors.Wrap(err, "failed to process blocks")
	}

	return events, nil
}

func (b *BlockClassificationDeriver) processBlocks(ctx context.Context, blocks []*aBlockprint.ProposersBlocksResponse) ([]*xatu.DecoratedEvent, error) {
	events := []*xatu.DecoratedEvent{}

	for _, block := range blocks {
		slot, epoch := GetSlotEpochTimings(phase0.Slot(block.Slot), b.beacon.Metadata().Wallclock())

		classification := &pBlockprint.BlockClassification{
			Slot:            wrapperspb.UInt64(block.Slot),
			BestGuessSingle: string(block.BestGuessSingle),
			BestGuessMulti:  block.BestGuessMulti,
			ProposerIndex:   wrapperspb.UInt64(block.ProposerIndex),
			ClientProbability: &pBlockprint.ClientProbability{
				Prysm:      wrapperspb.Float(float32(block.ProbabilityMap.Prysm())),
				Lighthouse: wrapperspb.Float(float32(block.ProbabilityMap.Lighthouse())),
				Nimbus:     wrapperspb.Float(float32(block.ProbabilityMap.Nimbus())),
				Lodestar:   wrapperspb.Float(float32(block.ProbabilityMap.Lodestar())),
				Teku:       wrapperspb.Float(float32(block.ProbabilityMap.Teku())),
				Grandine:   wrapperspb.Float(float32(block.ProbabilityMap.Grandine())),
				Uncertain:  wrapperspb.Float(float32(block.ProbabilityMap.Uncertain())),
			},
		}

		event, err := b.createEvent(ctx, classification, slot, epoch)
		if err != nil {
			return nil, errors.Wrap(err, fmt.Sprintf("failed to create event for slot %d", block.Slot))
		}

		events = append(events, event)
	}

	return events, nil
}

func (b *BlockClassificationDeriver) createEvent(ctx context.Context, classification *pBlockprint.BlockClassification, slot *xatu.SlotV2, epoch *xatu.EpochV2) (*xatu.DecoratedEvent, error) {
	// Make a clone of the metadata
	metadata, ok := proto.Clone(b.clientMeta).(*xatu.ClientMeta)
	if !ok {
		return nil, errors.New("failed to clone client metadata")
	}

	decoratedEvent := &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_BLOCKPRINT_BLOCK_CLASSIFICATION,
			DateTime: timestamppb.New(time.Now()),
			Id:       uuid.New().String(),
		},
		Meta: &xatu.Meta{
			Client: metadata,
		},
		Data: &xatu.DecoratedEvent_BlockprintBlockClassification{
			BlockprintBlockClassification: classification,
		},
	}

	decoratedEvent.Meta.Client.AdditionalData = &xatu.ClientMeta_BlockprintBlockClassification{
		BlockprintBlockClassification: &xatu.ClientMeta_AdditionalBlockprintBlockClassificationData{
			Slot:  slot,
			Epoch: epoch,
		},
	}

	return decoratedEvent, nil
}
