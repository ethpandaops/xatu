package iterator

import (
	"context"
	"time"

	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/ethpandaops/xatu/pkg/cannon/blockprint"
	"github.com/ethpandaops/xatu/pkg/cannon/coordinator"
	"github.com/ethpandaops/xatu/pkg/observability"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type BlockprintIterator struct {
	log              logrus.FieldLogger
	blockprintClient *blockprint.Client
	cannonType       xatu.CannonType
	coordinator      coordinator.Client
	networkID        string
	networkName      string
	metrics          *BlockprintMetrics
}

func NewBlockprintIterator(log logrus.FieldLogger, networkName, networkID string, cannonType xatu.CannonType, coordinatorClient *coordinator.Client, metrics *BlockprintMetrics, client *blockprint.Client) *BlockprintIterator {
	return &BlockprintIterator{
		log: log.
			WithField("module", "cannon/iterator/blockprint_iterator").
			WithField("cannon_type", cannonType.String()),
		networkName:      networkName,
		networkID:        networkID,
		cannonType:       cannonType,
		coordinator:      *coordinatorClient,
		metrics:          metrics,
		blockprintClient: client,
	}
}

func (c *BlockprintIterator) WithBlockprintClient(client *blockprint.Client) {
	c.blockprintClient = client
}

func (c *BlockprintIterator) UpdateLocation(ctx context.Context, location *xatu.CannonLocation) error {
	return c.coordinator.UpsertCannonLocationRequest(ctx, location)
}

func (c *BlockprintIterator) Next(ctx context.Context) (next *xatu.CannonLocation, err error) {
	ctx, span := observability.Tracer().Start(ctx,
		"BlockprintIterator.Next",
		trace.WithAttributes(
			attribute.String("network", c.networkName),
			attribute.String("cannon_type", c.cannonType.String()),
			attribute.String("network_id", c.networkID),
		),
	)
	defer func() {
		if err != nil {
			span.RecordError(err)
		} else {
			slot, target, err := c.getSlotsFromLocation(next)
			if err == nil {
				c.metrics.SetTargetSlot(c.cannonType.String(), c.networkName, float64(target))
				c.metrics.SetCurrentSlot(c.cannonType.String(), c.networkName, float64(slot))
				span.SetAttributes(
					attribute.Int64("slot", int64(slot)),
					attribute.Int64("target", int64(target)))
			}
		}

		span.End()
	}()

	for {
		// Check where we are at from the coordinator
		location, err := c.coordinator.GetCannonLocation(ctx, c.cannonType, c.networkID)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get cannon location")
		}

		status, err := c.blockprintClient.SyncStatus(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "failed to fetch sync status")
		}

		if !status.Synced {
			return nil, errors.New("blockprint is not synced")
		}

		// If location is empty we haven't started yet. This means we should set our `target_end_slot` to the latest synced slot.
		// We can then let the deriver bulk fetch data between the current slot and the `target_end_slot`.
		if location == nil {
			next, errr := c.createLocation(0, phase0.Slot(status.GreatestBlockSlot))
			if errr != nil {
				return nil, errors.Wrap(errr, "failed to create location")
			}

			return next, nil
		}

		slot, _, err := c.getSlotsFromLocation(location)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get slots from location")
		}

		if slot >= phase0.Slot(status.GreatestBlockSlot) {
			// Sleep for 30s to give blockprint time to sync new blocks
			time.Sleep(30 * time.Second)

			continue
		}

		next, err := c.createLocation(slot, phase0.Slot(status.GreatestBlockSlot))
		if err != nil {
			return nil, errors.Wrap(err, "failed to create location")
		}

		return next, nil
	}
}

func (c *BlockprintIterator) getSlotsFromLocation(location *xatu.CannonLocation) (slot, target phase0.Slot, err error) {
	switch location.Type {
	case xatu.CannonType_BLOCKPRINT_BLOCK_CLASSIFICATION:
		data := location.GetBlockprintBlockClassification()

		return phase0.Slot(data.GetSlot()), phase0.Slot(data.GetTargetEndSlot()), nil
	default:
		return 0, 0, errors.Errorf("unknown cannon type %s", location.Type)
	}
}

func (c *BlockprintIterator) createLocation(slot, target phase0.Slot) (*xatu.CannonLocation, error) {
	location := &xatu.CannonLocation{
		NetworkId: c.networkID,
		Type:      c.cannonType,
	}

	switch c.cannonType {
	case xatu.CannonType_BLOCKPRINT_BLOCK_CLASSIFICATION:
		location.Data = &xatu.CannonLocation_BlockprintBlockClassification{
			BlockprintBlockClassification: &xatu.CannonLocationBlockprintBlockClassification{
				Slot:          uint64(slot),
				TargetEndSlot: uint64(target),
			},
		}
	default:
		return location, errors.Errorf("unknown cannon type %s", location.Type)
	}

	return location, nil
}
