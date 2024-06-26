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

type CheckpointIterator struct {
	log            logrus.FieldLogger
	cannonType     xatu.CannonType
	coordinator    coordinator.Client
	wallclock      *ethwallclock.EthereumBeaconChain
	networkID      string
	networkName    string
	metrics        *CheckpointMetrics
	beaconNode     *ethereum.BeaconNode
	checkpointName string
}

func NewCheckpointIterator(log logrus.FieldLogger, networkName, networkID string, cannonType xatu.CannonType, coordinatorClient *coordinator.Client, wallclock *ethwallclock.EthereumBeaconChain, metrics *CheckpointMetrics, beacon *ethereum.BeaconNode, checkpoint string) *CheckpointIterator {
	return &CheckpointIterator{
		log: log.
			WithField("module", "cannon/iterator/checkpoint_iterator").
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

func (c *CheckpointIterator) Start(ctx context.Context) error {
	return nil
}

func (c *CheckpointIterator) UpdateLocation(ctx context.Context, location *xatu.CannonLocation) error {
	return c.coordinator.UpsertCannonLocationRequest(ctx, location)
}

func (c *CheckpointIterator) Next(ctx context.Context) (next *xatu.CannonLocation, lookAhead []*xatu.CannonLocation, err error) {
	ctx, span := observability.Tracer().Start(ctx,
		"CheckpointIterator.Next",
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
			epoch, err := c.getEpochFromLocation(next)
			if err == nil {
				span.SetAttributes(attribute.Int64("next", int64(epoch)))
			}
		}

		span.End()
	}()

	for {
		// Grab the current checkpoint from the beacon node
		checkpoint, err := c.fetchLatestEpoch(ctx)
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
		// is empty, we'll start at epoch 0.
		if location == nil {
			location, err = c.createLocationFromEpochNumber(phase0.Epoch(GetDefaultSlotLocation(c.beaconNode.Metadata().Spec.ForkEpochs, c.cannonType) / 32))
			if err != nil {
				return nil, []*xatu.CannonLocation{}, errors.Wrap(err, "failed to create location from slot number 0")
			}

			return location, c.getLookAheads(ctx, location), nil
		}

		// If the location is the same as the current checkpoint, we should sleep until the next epoch
		locationEpoch, err := c.getEpochFromLocation(location)
		if err != nil {
			return nil, []*xatu.CannonLocation{}, errors.Wrap(err, "failed to get epoch from location")
		}

		c.metrics.SetTrailingEpochs(c.cannonType.String(), c.networkName, c.checkpointName, float64(checkpoint.Epoch-locationEpoch))

		if locationEpoch >= checkpoint.Epoch {
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

		nextEpoch := locationEpoch + 1

		current, err := c.createLocationFromEpochNumber(nextEpoch)
		if err != nil {
			return nil, []*xatu.CannonLocation{}, errors.Wrap(err, "failed to create location from epoch number")
		}

		c.metrics.SetCurrentEpoch(c.cannonType.String(), c.networkName, c.checkpointName, float64(nextEpoch))

		return current, c.getLookAheads(ctx, current), nil
	}
}

func (c *CheckpointIterator) getLookAheads(ctx context.Context, location *xatu.CannonLocation) []*xatu.CannonLocation {
	// Calculate if we should look ahead
	epoch, err := c.getEpochFromLocation(location)
	if err != nil {
		return []*xatu.CannonLocation{}
	}

	latestCheckpoint, err := c.fetchLatestEpoch(ctx)
	if err != nil {
		return []*xatu.CannonLocation{}
	}

	lookAheads := make([]*xatu.CannonLocation, 0)

	for _, i := range []int{1, 2, 3} {
		lookAheadEpoch := epoch + phase0.Epoch(i)

		if lookAheadEpoch > latestCheckpoint.Epoch {
			continue
		}

		loc, err := c.createLocationFromEpochNumber(lookAheadEpoch)
		if err != nil {
			return []*xatu.CannonLocation{}
		}

		lookAheads = append(lookAheads, loc)
	}

	return lookAheads
}

func (c *CheckpointIterator) fetchLatestEpoch(ctx context.Context) (*phase0.Checkpoint, error) {
	_, span := observability.Tracer().Start(ctx,
		"CheckpointIterator.FetchLatestEpoch",
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

func (c *CheckpointIterator) getEpochFromLocation(location *xatu.CannonLocation) (phase0.Epoch, error) {
	switch location.Type {
	case xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_ATTESTER_SLASHING:
		return phase0.Epoch(location.GetEthV2BeaconBlockAttesterSlashing().Epoch), nil
	case xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_PROPOSER_SLASHING:
		return phase0.Epoch(location.GetEthV2BeaconBlockProposerSlashing().Epoch), nil
	case xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_BLS_TO_EXECUTION_CHANGE:
		return phase0.Epoch(location.GetEthV2BeaconBlockBlsToExecutionChange().Epoch), nil
	case xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_EXECUTION_TRANSACTION:
		return phase0.Epoch(location.GetEthV2BeaconBlockExecutionTransaction().Epoch), nil
	case xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_VOLUNTARY_EXIT:
		return phase0.Epoch(location.GetEthV2BeaconBlockVoluntaryExit().Epoch), nil
	case xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_DEPOSIT:
		return phase0.Epoch(location.GetEthV2BeaconBlockDeposit().Epoch), nil
	case xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_WITHDRAWAL:
		return phase0.Epoch(location.GetEthV2BeaconBlockWithdrawal().Epoch), nil
	case xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK:
		return phase0.Epoch(location.GetEthV2BeaconBlock().Epoch), nil
	case xatu.CannonType_BEACON_API_ETH_V1_BEACON_BLOB_SIDECAR:
		return phase0.Epoch(location.GetEthV1BeaconBlobSidecar().Epoch), nil
	default:
		return 0, errors.Errorf("unknown cannon type %s", location.Type)
	}
}

func (c *CheckpointIterator) createLocationFromEpochNumber(epoch phase0.Epoch) (*xatu.CannonLocation, error) {
	location := &xatu.CannonLocation{
		NetworkId: c.networkID,
		Type:      c.cannonType,
	}

	switch c.cannonType {
	case xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_ATTESTER_SLASHING:
		location.Data = &xatu.CannonLocation_EthV2BeaconBlockAttesterSlashing{
			EthV2BeaconBlockAttesterSlashing: &xatu.CannonLocationEthV2BeaconBlockAttesterSlashing{
				Epoch: uint64(epoch),
			},
		}
	case xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_PROPOSER_SLASHING:
		location.Data = &xatu.CannonLocation_EthV2BeaconBlockProposerSlashing{
			EthV2BeaconBlockProposerSlashing: &xatu.CannonLocationEthV2BeaconBlockProposerSlashing{
				Epoch: uint64(epoch),
			},
		}
	case xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_BLS_TO_EXECUTION_CHANGE:
		location.Data = &xatu.CannonLocation_EthV2BeaconBlockBlsToExecutionChange{
			EthV2BeaconBlockBlsToExecutionChange: &xatu.CannonLocationEthV2BeaconBlockBlsToExecutionChange{
				Epoch: uint64(epoch),
			},
		}
	case xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_EXECUTION_TRANSACTION:
		location.Data = &xatu.CannonLocation_EthV2BeaconBlockExecutionTransaction{
			EthV2BeaconBlockExecutionTransaction: &xatu.CannonLocationEthV2BeaconBlockExecutionTransaction{
				Epoch: uint64(epoch),
			},
		}
	case xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_VOLUNTARY_EXIT:
		location.Data = &xatu.CannonLocation_EthV2BeaconBlockVoluntaryExit{
			EthV2BeaconBlockVoluntaryExit: &xatu.CannonLocationEthV2BeaconBlockVoluntaryExit{
				Epoch: uint64(epoch),
			},
		}
	case xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_DEPOSIT:
		location.Data = &xatu.CannonLocation_EthV2BeaconBlockDeposit{
			EthV2BeaconBlockDeposit: &xatu.CannonLocationEthV2BeaconBlockDeposit{
				Epoch: uint64(epoch),
			},
		}
	case xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_WITHDRAWAL:
		location.Data = &xatu.CannonLocation_EthV2BeaconBlockWithdrawal{
			EthV2BeaconBlockWithdrawal: &xatu.CannonLocationEthV2BeaconBlockWithdrawal{
				Epoch: uint64(epoch),
			},
		}
	case xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK:
		location.Data = &xatu.CannonLocation_EthV2BeaconBlock{
			EthV2BeaconBlock: &xatu.CannonLocationEthV2BeaconBlock{
				Epoch: uint64(epoch),
			},
		}
	case xatu.CannonType_BEACON_API_ETH_V1_BEACON_BLOB_SIDECAR:
		location.Data = &xatu.CannonLocation_EthV1BeaconBlobSidecar{
			EthV1BeaconBlobSidecar: &xatu.CannonLocationEthV1BeaconBlobSidecar{
				Epoch: uint64(epoch),
			},
		}
	default:
		return location, errors.Errorf("unknown cannon type %s", location.Type)
	}

	return location, nil
}
