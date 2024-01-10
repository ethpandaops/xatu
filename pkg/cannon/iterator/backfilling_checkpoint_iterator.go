package iterator

import (
	"context"
	"time"

	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/ethpandaops/ethwallclock"
	"github.com/ethpandaops/xatu/pkg/cannon/coordinator"
	"github.com/ethpandaops/xatu/pkg/cannon/ethereum"
	"github.com/ethpandaops/xatu/pkg/observability"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type BackfillingCheckpoint struct {
	log            logrus.FieldLogger
	cannonType     xatu.CannonType
	coordinator    coordinator.Client
	wallclock      *ethwallclock.EthereumBeaconChain
	networkID      string
	networkName    string
	metrics        *BackfillingCheckpointMetrics
	beaconNode     *ethereum.BeaconNode
	checkpointName string
}

func NewBackfillingCheckpoint(log logrus.FieldLogger, networkName, networkID string, cannonType xatu.CannonType, coordinatorClient *coordinator.Client, wallclock *ethwallclock.EthereumBeaconChain, metrics *BackfillingCheckpointMetrics, beacon *ethereum.BeaconNode, checkpoint string) *BackfillingCheckpoint {
	return &BackfillingCheckpoint{
		log: log.
			WithField("module", "cannon/iterator/backfilling_checkpoint_iterator").
			WithField("cannon_type", cannonType.String()),
		networkName:    networkName,
		networkID:      networkID,
		cannonType:     cannonType,
		coordinator:    *coordinatorClient,
		wallclock:      wallclock,
		beaconNode:     beacon,
		metrics:        metrics,
		checkpointName: checkpoint,
	}
}

func (c *BackfillingCheckpoint) UpdateLocation(ctx context.Context, location *xatu.CannonLocation) error {
	return c.coordinator.UpsertCannonLocationRequest(ctx, location)
}

func (c *BackfillingCheckpoint) Next(ctx context.Context) (next *xatu.CannonLocation, lookAhead []*xatu.CannonLocation, err error) {
	ctx, span := observability.Tracer().Start(ctx,
		"BackfillingCheckpoint.Next",
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
			marker, err := c.getMarker(next)
			if err == nil {
				span.SetAttributes(attribute.Int64("finalized_epoch", int64(marker.FinalizedEpoch)))
				span.SetAttributes(attribute.Int64("backfill_epoch", marker.BackfillEpoch))
				c.metrics.SetBackfillEpoch(c.cannonType.String(), c.networkName, c.checkpointName, float64(marker.BackfillEpoch))
				c.metrics.SetFinalizedEpoch(c.cannonType.String(), c.networkName, c.checkpointName, float64(marker.FinalizedEpoch))
			}
		}

		span.End()
	}()

	for {
		// Grab the current checkpoint from the beacon node
		checkpoint, err := c.fetchLatestCheckpoint(ctx)
		if err != nil {
			return nil, []*xatu.CannonLocation{}, errors.Wrap(err, "failed to fetch latest checkpoint")
		}

		if checkpoint == nil {
			return nil, []*xatu.CannonLocation{}, errors.New("checkpoint is nil")
		}

		// Check where we are at from the coordinator
		location, err := c.coordinator.GetCannonLocation(ctx, c.cannonType, c.networkID)
		if err != nil {
			return nil, []*xatu.CannonLocation{}, errors.Wrap(err, "failed to get cannon location")
		}

		// If location is empty we haven't started yet, start at the network default for the type. If the network default
		// is empty, we'll start at the checkpoint.
		if location == nil {
			location, err = c.createLocationFromEpochNumber(checkpoint.Epoch, checkpoint.Epoch)
			if err != nil {
				return nil, []*xatu.CannonLocation{}, errors.Wrap(err, "failed to create location from slot number 0")
			}

			return location, c.getLookAheads(ctx, location), nil
		}

		// If the backfill is at zero, and the finalized epoch is up to date, we should sleep until the next epoch
		marker, err := c.getMarker(location)
		if err != nil {
			return nil, []*xatu.CannonLocation{}, errors.Wrap(err, "failed to get marker from location")
		}

		if marker.BackfillEpoch == -1 && checkpoint.Epoch == phase0.Epoch(marker.FinalizedEpoch) {
			// Sleep until the next epoch
			epoch := c.wallclock.Epochs().Current()

			sleepFor := time.Until(epoch.TimeWindow().End())

			c.log.WithFields(logrus.Fields{
				"current_epoch":    epoch.Number(),
				"sleep_for":        sleepFor.String(),
				"checkpoint_epoch": checkpoint.Epoch,
			}).Trace("Sleeping until next epoch")

			time.Sleep(sleepFor)

			// Sleep for an additional 5 seconds to give the beacon node time to do epoch processing.
			time.Sleep(5 * time.Second)

			continue
		}

		// Return the current location -- consumers of this iterator will have to update the location themselves.
		return location, c.getLookAheads(ctx, location), nil
	}
}

func (c *BackfillingCheckpoint) getLookAheads(ctx context.Context, location *xatu.CannonLocation) []*xatu.CannonLocation {
	// Not supported
	return []*xatu.CannonLocation{}
}

func (c *BackfillingCheckpoint) fetchLatestCheckpoint(ctx context.Context) (*phase0.Checkpoint, error) {
	_, span := observability.Tracer().Start(ctx,
		"BackfillingCheckpoint.FetchLatestEpoch",
	)
	defer span.End()

	finality, err := c.beaconNode.Node().Finality()
	if err != nil {
		return nil, errors.Wrap(err, "failed to fetch finality")
	}

	if c.checkpointName == "justified" {
		return finality.Justified, nil
	}

	if c.checkpointName == "finalized" {
		return finality.Finalized, nil
	}

	return nil, errors.Errorf("unknown checkpoint name %s", c.checkpointName)
}

func (c *BackfillingCheckpoint) getMarker(location *xatu.CannonLocation) (*xatu.BackfillingCheckpointMarker, error) {
	switch location.Type {
	case xatu.CannonType_BEACON_API_ETH_V1_PROPOSER_DUTY:
		return location.GetEthV1BeaconProposerDuty().GetBackfillingCheckpointMarker(), nil
	case xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_ELABORATED_ATTESTATION:
		return location.GetEthV2BeaconBlockElaboratedAttestation().GetBackfillingCheckpointMarker(), nil
	default:
		return nil, errors.Errorf("unknown cannon type %s", location.Type)
	}
}

func (c *BackfillingCheckpoint) createLocationFromEpochNumber(finalized, backfill phase0.Epoch) (*xatu.CannonLocation, error) {
	location := &xatu.CannonLocation{
		NetworkId: c.networkID,
		Type:      c.cannonType,
	}

	switch c.cannonType {
	case xatu.CannonType_BEACON_API_ETH_V1_PROPOSER_DUTY:
		location.Data = &xatu.CannonLocation_EthV1BeaconProposerDuty{
			EthV1BeaconProposerDuty: &xatu.CannonLocationEthV1BeaconProposerDuty{
				BackfillingCheckpointMarker: &xatu.BackfillingCheckpointMarker{
					FinalizedEpoch: uint64(finalized),
					BackfillEpoch:  int64(backfill),
				},
			},
		}
	case xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_ELABORATED_ATTESTATION:
		location.Data = &xatu.CannonLocation_EthV2BeaconBlockElaboratedAttestation{
			EthV2BeaconBlockElaboratedAttestation: &xatu.CannonLocationEthV2BeaconBlockElaboratedAttestation{
				BackfillingCheckpointMarker: &xatu.BackfillingCheckpointMarker{
					FinalizedEpoch: uint64(finalized),
					BackfillEpoch:  int64(backfill),
				},
			},
		}
	default:
		return location, errors.Errorf("unknown cannon type %s", location.Type)
	}

	return location, nil
}
