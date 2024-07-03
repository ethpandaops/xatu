package iterator

import (
	"context"
	"slices"
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

func (c *BackfillingCheckpoint) Start(ctx context.Context) error {
	return nil
}

func (c *BackfillingCheckpoint) migrateFromCheckpointIterator(ctx context.Context) error {
	checkpointTypes := []xatu.CannonType{
		xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_ATTESTER_SLASHING,
		xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_PROPOSER_SLASHING,
		xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_BLS_TO_EXECUTION_CHANGE,
		xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_EXECUTION_TRANSACTION,
		xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_VOLUNTARY_EXIT,
		xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_DEPOSIT,
		xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_WITHDRAWAL,
		xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK,
		xatu.CannonType_BEACON_API_ETH_V1_BEACON_BLOB_SIDECAR,
	}

	if !slices.Contains(checkpointTypes, c.cannonType) {
		return nil
	}

	// If the cannon type is the checkpoint type, we need to migrate the cannon location to the backfilling checkpoint iterator.
	// We'll need to get the current location, and then create a new checkpoint iterator with the current location as the 'head' point.
	location, err := c.coordinator.GetCannonLocation(ctx, c.cannonType, c.networkID)
	if err != nil {
		return errors.Wrap(err, "failed to get cannon location")
	}

	// We need to get the epoch from the location
	currentEpoch := phase0.Epoch(0)
	var currentBackfill *xatu.BackfillingCheckpointMarker

	switch location.Type {
	case xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_ATTESTER_SLASHING:
		currentEpoch = phase0.Epoch(location.GetEthV2BeaconBlockAttesterSlashing().GetEpoch())
		currentBackfill = location.GetEthV2BeaconBlockAttesterSlashing().GetBackfillingCheckpointMarker()
	case xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_PROPOSER_SLASHING:
		currentEpoch = phase0.Epoch(location.GetEthV2BeaconBlockProposerSlashing().GetEpoch())
		currentBackfill = location.GetEthV2BeaconBlockProposerSlashing().GetBackfillingCheckpointMarker()
	case xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_BLS_TO_EXECUTION_CHANGE:
		currentEpoch = phase0.Epoch(location.GetEthV2BeaconBlockBlsToExecutionChange().GetEpoch())
		currentBackfill = location.GetEthV2BeaconBlockBlsToExecutionChange().GetBackfillingCheckpointMarker()
	case xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_EXECUTION_TRANSACTION:
		currentEpoch = phase0.Epoch(location.GetEthV2BeaconBlockExecutionTransaction().GetEpoch())
		currentBackfill = location.GetEthV2BeaconBlockExecutionTransaction().GetBackfillingCheckpointMarker()
	case xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_VOLUNTARY_EXIT:
		currentEpoch = phase0.Epoch(location.GetEthV2BeaconBlockVoluntaryExit().GetEpoch())
		currentBackfill = location.GetEthV2BeaconBlockVoluntaryExit().GetBackfillingCheckpointMarker()
	case xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_DEPOSIT:
		currentEpoch = phase0.Epoch(location.GetEthV2BeaconBlockDeposit().GetEpoch())
		currentBackfill = location.GetEthV2BeaconBlockDeposit().GetBackfillingCheckpointMarker()
	case xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_WITHDRAWAL:
		currentEpoch = phase0.Epoch(location.GetEthV2BeaconBlockWithdrawal().GetEpoch())
		currentBackfill = location.GetEthV2BeaconBlockWithdrawal().GetBackfillingCheckpointMarker()
	case xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK:
		currentEpoch = phase0.Epoch(location.GetEthV2BeaconBlock().GetEpoch())
		currentBackfill = location.GetEthV2BeaconBlock().GetBackfillingCheckpointMarker()
	case xatu.CannonType_BEACON_API_ETH_V1_BEACON_BLOB_SIDECAR:
		currentEpoch = phase0.Epoch(location.GetEthV1BeaconBlobSidecar().GetEpoch())
		currentBackfill = location.GetEthV1BeaconBlobSidecar().GetBackfillingCheckpointMarker()
	default:
		return errors.Errorf("unknown cannon type (%s) when migrating from checkpoint iterator", location.Type)
	}

	if currentEpoch == 0 && currentBackfill != nil {
		// We've already migrated, so we don't need to do anything.
		return nil
	}

	// Create a new location with the current epoch as the 'head' point.
	newLocation, err := c.createLocationFromEpochNumber(currentEpoch, currentEpoch)
	if err != nil {
		return errors.Wrap(err, "failed to create location from epoch number")
	}

	c.log.WithField("new_location", newLocation).Info("Migrated cannon location to backfilling checkpoint iterator")

	// Update the coordinator with the new location.
	return c.coordinator.UpsertCannonLocationRequest(ctx, newLocation)
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
	// Not supported in backfilling checkpoint iterator. Consumers must calculate this themselves.
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
	case xatu.CannonType_BEACON_API_ETH_V1_BEACON_VALIDATORS:
		return location.GetEthV1BeaconValidators().GetBackfillingCheckpointMarker(), nil
	case xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_ATTESTER_SLASHING:
		return location.GetEthV2BeaconBlockAttesterSlashing().GetBackfillingCheckpointMarker(), nil
	case xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_PROPOSER_SLASHING:
		return location.GetEthV2BeaconBlockProposerSlashing().GetBackfillingCheckpointMarker(), nil
	case xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_BLS_TO_EXECUTION_CHANGE:
		return location.GetEthV2BeaconBlockBlsToExecutionChange().GetBackfillingCheckpointMarker(), nil
	case xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_EXECUTION_TRANSACTION:
		return location.GetEthV2BeaconBlockExecutionTransaction().GetBackfillingCheckpointMarker(), nil
	case xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_VOLUNTARY_EXIT:
		return location.GetEthV2BeaconBlockVoluntaryExit().GetBackfillingCheckpointMarker(), nil
	case xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_DEPOSIT:
		return location.GetEthV2BeaconBlockDeposit().GetBackfillingCheckpointMarker(), nil
	case xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_WITHDRAWAL:
		return location.GetEthV2BeaconBlockWithdrawal().GetBackfillingCheckpointMarker(), nil
	case xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK:
		return location.GetEthV2BeaconBlock().GetBackfillingCheckpointMarker(), nil
	case xatu.CannonType_BEACON_API_ETH_V1_BEACON_BLOB_SIDECAR:
		return location.GetEthV1BeaconBlobSidecar().GetBackfillingCheckpointMarker(), nil
	default:
		return nil, errors.Errorf("unknown cannon type %s", location.Type)
	}
}

func (c *BackfillingCheckpoint) createLocationFromEpochNumber(finalized, backfill phase0.Epoch) (*xatu.CannonLocation, error) {
	location := &xatu.CannonLocation{
		NetworkId: c.networkID,
		Type:      c.cannonType,
	}

	marker := &xatu.BackfillingCheckpointMarker{
		FinalizedEpoch: uint64(finalized),
		BackfillEpoch:  int64(backfill),
	}

	switch c.cannonType {
	case xatu.CannonType_BEACON_API_ETH_V1_PROPOSER_DUTY:
		location.Data = &xatu.CannonLocation_EthV1BeaconProposerDuty{
			EthV1BeaconProposerDuty: &xatu.CannonLocationEthV1BeaconProposerDuty{
				BackfillingCheckpointMarker: marker,
			},
		}
	case xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_ELABORATED_ATTESTATION:
		location.Data = &xatu.CannonLocation_EthV2BeaconBlockElaboratedAttestation{
			EthV2BeaconBlockElaboratedAttestation: &xatu.CannonLocationEthV2BeaconBlockElaboratedAttestation{
				BackfillingCheckpointMarker: marker,
			},
		}
	case xatu.CannonType_BEACON_API_ETH_V1_BEACON_VALIDATORS:
		location.Data = &xatu.CannonLocation_EthV1BeaconValidators{
			EthV1BeaconValidators: &xatu.CannonLocationEthV1BeaconValidators{
				BackfillingCheckpointMarker: marker,
			},
		}
	case xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_ATTESTER_SLASHING:
		location.Data = &xatu.CannonLocation_EthV2BeaconBlockAttesterSlashing{
			EthV2BeaconBlockAttesterSlashing: &xatu.CannonLocationEthV2BeaconBlockAttesterSlashing{
				BackfillingCheckpointMarker: marker,
			},
		}
	case xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_PROPOSER_SLASHING:
		location.Data = &xatu.CannonLocation_EthV2BeaconBlockProposerSlashing{
			EthV2BeaconBlockProposerSlashing: &xatu.CannonLocationEthV2BeaconBlockProposerSlashing{
				BackfillingCheckpointMarker: marker,
			},
		}
	case xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_BLS_TO_EXECUTION_CHANGE:
		location.Data = &xatu.CannonLocation_EthV2BeaconBlockBlsToExecutionChange{
			EthV2BeaconBlockBlsToExecutionChange: &xatu.CannonLocationEthV2BeaconBlockBlsToExecutionChange{
				BackfillingCheckpointMarker: marker,
			},
		}
	case xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_EXECUTION_TRANSACTION:
		location.Data = &xatu.CannonLocation_EthV2BeaconBlockExecutionTransaction{
			EthV2BeaconBlockExecutionTransaction: &xatu.CannonLocationEthV2BeaconBlockExecutionTransaction{
				BackfillingCheckpointMarker: marker,
			},
		}
	case xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_VOLUNTARY_EXIT:
		location.Data = &xatu.CannonLocation_EthV2BeaconBlockVoluntaryExit{
			EthV2BeaconBlockVoluntaryExit: &xatu.CannonLocationEthV2BeaconBlockVoluntaryExit{
				BackfillingCheckpointMarker: marker,
			},
		}
	case xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_DEPOSIT:
		location.Data = &xatu.CannonLocation_EthV2BeaconBlockDeposit{
			EthV2BeaconBlockDeposit: &xatu.CannonLocationEthV2BeaconBlockDeposit{
				BackfillingCheckpointMarker: marker,
			},
		}
	case xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_WITHDRAWAL:
		location.Data = &xatu.CannonLocation_EthV2BeaconBlockWithdrawal{
			EthV2BeaconBlockWithdrawal: &xatu.CannonLocationEthV2BeaconBlockWithdrawal{
				BackfillingCheckpointMarker: marker,
			},
		}
	case xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK:
		location.Data = &xatu.CannonLocation_EthV2BeaconBlock{
			EthV2BeaconBlock: &xatu.CannonLocationEthV2BeaconBlock{
				BackfillingCheckpointMarker: marker,
			},
		}
	case xatu.CannonType_BEACON_API_ETH_V1_BEACON_BLOB_SIDECAR:
		location.Data = &xatu.CannonLocation_EthV1BeaconBlobSidecar{
			EthV1BeaconBlobSidecar: &xatu.CannonLocationEthV1BeaconBlobSidecar{
				BackfillingCheckpointMarker: marker,
			},
		}
	default:
		return location, errors.Errorf("unknown cannon type %s", location.Type)
	}

	return location, nil
}
