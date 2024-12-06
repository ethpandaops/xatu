package iterator

import (
	"context"
	"fmt"
	"time"

	"github.com/attestantio/go-eth2-client/spec"
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
	log               logrus.FieldLogger
	cannonType        xatu.CannonType
	coordinator       coordinator.Client
	wallclock         *ethwallclock.EthereumBeaconChain
	networkID         string
	networkName       string
	metrics           *BackfillingCheckpointMetrics
	beaconNode        *ethereum.BeaconNode
	checkpointName    string
	lookAheadDistance int
	config            *BackfillingCheckpointConfig
	activationFork    spec.DataVersion
}

type BackfillingCheckpointDirection string

const (
	BackfillingCheckpointDirectionBackfill BackfillingCheckpointDirection = "backfill"
	BackfillingCheckpointDirectionHead     BackfillingCheckpointDirection = "head"
)

type BackFillingCheckpointNextResponse struct {
	Next       phase0.Epoch
	LookAheads []phase0.Epoch
	Direction  BackfillingCheckpointDirection
}

func NewBackfillingCheckpoint(
	log logrus.FieldLogger,
	networkName, networkID string,
	cannonType xatu.CannonType,
	coordinatorClient *coordinator.Client,
	wallclock *ethwallclock.EthereumBeaconChain,
	metrics *BackfillingCheckpointMetrics,
	beacon *ethereum.BeaconNode,
	checkpoint string,
	lookAheadDistance int,
	config *BackfillingCheckpointConfig,
) *BackfillingCheckpoint {
	return &BackfillingCheckpoint{
		log: log.
			WithField("module", "cannon/iterator/backfilling_checkpoint_iterator").
			WithField("cannon_type", cannonType.String()),
		networkName:       networkName,
		networkID:         networkID,
		cannonType:        cannonType,
		coordinator:       *coordinatorClient,
		wallclock:         wallclock,
		beaconNode:        beacon,
		metrics:           metrics,
		checkpointName:    checkpoint,
		lookAheadDistance: lookAheadDistance,
		config:            config,
	}
}

func (c *BackfillingCheckpoint) Start(ctx context.Context, activationFork spec.DataVersion) error {
	c.activationFork = activationFork

	// Check the backfill epoch is ok
	if c.shouldBackfill(ctx) {
		epoch, err := c.getEarliestPossibleBackfillEpoch()
		if err != nil {
			return errors.Wrap(err, "failed to calculate target backfill epoch")
		}

		c.log.WithFields(logrus.Fields{
			"backfill_target_epoch": epoch,
		}).Info("Backfilling is enabled")
	} else {
		c.log.Info("Backfilling is disabled")
	}

	return nil
}

func (c *BackfillingCheckpoint) UpdateLocation(ctx context.Context, epoch phase0.Epoch, direction BackfillingCheckpointDirection) error {
	location, err := c.coordinator.GetCannonLocation(ctx, c.cannonType, c.networkID)
	if err != nil {
		return errors.Wrap(err, "failed to get cannon location")
	}

	if location == nil {
		location, err = c.createLocationFromEpochNumber(epoch, epoch)
		if err != nil {
			return errors.Wrap(err, "failed to create fresh location from epoch number")
		}
	}

	marker, err := c.GetMarker(location)
	if err != nil {
		return errors.Wrap(err, "failed to get marker from location")
	}

	switch direction {
	case BackfillingCheckpointDirectionHead:
		marker.FinalizedEpoch = uint64(epoch)
	case BackfillingCheckpointDirectionBackfill:
		marker.BackfillEpoch = int64(epoch)
	default:
		return errors.Errorf("unknown direction (%s) when updating cannon location", direction)
	}

	newLocation, err := c.createLocationFromEpochNumber(phase0.Epoch(marker.FinalizedEpoch), phase0.Epoch(marker.BackfillEpoch))
	if err != nil {
		return errors.Wrap(err, "failed to create location from epoch number")
	}

	c.log.WithFields(logrus.Fields{
		"direction": direction,
		"epoch":     epoch,
	}).Debug("Updating cannon location")

	err = c.coordinator.UpsertCannonLocationRequest(ctx, newLocation)
	if err != nil {
		return errors.Wrap(err, "failed to update cannon location")
	}

	c.log.WithFields(logrus.Fields{
		"direction": direction,
		"epoch":     epoch,
	}).Debug("Updated cannon location")

	c.metrics.SetBackfillEpoch(c.cannonType.String(), c.networkName, c.checkpointName, float64(marker.BackfillEpoch))
	c.metrics.SetFinalizedEpoch(c.cannonType.String(), c.networkName, c.checkpointName, float64(marker.FinalizedEpoch))

	return nil
}

func (c *BackfillingCheckpoint) Next(ctx context.Context) (rsp *BackFillingCheckpointNextResponse, err error) {
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
		} else if rsp != nil {
			c.log.WithFields(logrus.Fields{
				"next_epoch":  rsp.Next,
				"direction":   rsp.Direction,
				"look_aheads": rsp.LookAheads,
			}).Debug("Returning next epoch")

			span.SetAttributes(attribute.Int64("next_epoch", int64(rsp.Next)))
			span.SetAttributes(attribute.String("direction", string(rsp.Direction)))

			if rsp.LookAheads != nil {
				lookAheads := make([]int64, len(rsp.LookAheads))
				for i, epoch := range rsp.LookAheads {
					lookAheads[i] = int64(epoch)
				}

				span.SetAttributes(attribute.Int64Slice("look_aheads", lookAheads))
			}
		}

		span.End()
	}()

	for {
		// Grab the current checkpoint from the beacon node
		checkpoint, err := c.fetchLatestCheckpoint(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "failed to fetch latest checkpoint")
		}

		if checkpoint == nil {
			return nil, errors.New("checkpoint is nil")
		}

		c.metrics.SetFinalizedCheckpointEpoch(c.networkName, float64(checkpoint.Epoch))

		// Check where we are at from the coordinator
		location, err := c.coordinator.GetCannonLocation(ctx, c.cannonType, c.networkID)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get cannon location")
		}

		// If location is empty we haven't started yet, start at the network default for the type. If the network default
		// is empty, we'll start at the checkpoint.

		// Default the backfill target epoch to the checkpoint epoch.
		backfillTargetEpoch := checkpoint.Epoch

		if c.shouldBackfill(ctx) {
			backfillTargetEpoch, err = c.getEarliestPossibleBackfillEpoch()
			if err != nil {
				return nil, errors.Wrap(err, "failed to get earliest possible backfill epoch")
			}
		}

		targetEpoch := phase0.Epoch(0)

		if c.activationFork != spec.DataVersionPhase0 {
			forkEpoch, errr := c.beaconNode.Metadata().Spec.ForkEpochs.GetByName(c.activationFork.String())
			if errr != nil {
				return nil, errors.Wrap(errr, fmt.Sprintf("failed to get epoch for fork: %s", c.activationFork))
			}

			targetEpoch = forkEpoch.Epoch
		}

		if checkpoint.Epoch < targetEpoch {
			// The current finalized checkpoint is before the activation of this cannon, so we should sleep until the next epoch.
			epoch := c.wallclock.Epochs().Current()

			sleepFor := time.Until(epoch.TimeWindow().End())

			c.log.WithFields(logrus.Fields{
				"current_epoch":         epoch.Number(),
				"sleep_for":             sleepFor.String(),
				"checkpoint_epoch":      checkpoint.Epoch,
				"backfill_target_epoch": backfillTargetEpoch,
			}).Info("Sleeping until next epoch as the fork for the iterator is not yet active")

			time.Sleep(sleepFor)

			// Sleep for an additional 5 seconds to give the beacon node time to do epoch processing.
			time.Sleep(5 * time.Second)

			continue
		}

		if location == nil {
			// If the location is empty, we haven't started yet, so we should return the current checkpoint epoch.
			return &BackFillingCheckpointNextResponse{
				Next:       checkpoint.Epoch,
				LookAheads: c.calculateFinalizedLookAheads(checkpoint.Epoch, checkpoint.Epoch),
				Direction:  BackfillingCheckpointDirectionHead,
			}, nil
		}

		marker, err := c.GetMarker(location)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get marker from location")
		}

		c.metrics.SetLag(c.cannonType.String(), c.networkName, BackfillingCheckpointDirectionHead, float64(checkpoint.Epoch-phase0.Epoch(marker.FinalizedEpoch)))
		//nolint:gosec // Only used for metrics
		c.metrics.SetLag(c.cannonType.String(), c.networkName, BackfillingCheckpointDirectionBackfill, float64(phase0.Epoch(marker.BackfillEpoch)-backfillTargetEpoch))

		if marker.FinalizedEpoch == 0 {
			// If the marker is empty, we haven't started yet, so we should return the current checkpoint.
			return &BackFillingCheckpointNextResponse{
				Next:       checkpoint.Epoch,
				LookAheads: c.calculateFinalizedLookAheads(checkpoint.Epoch, checkpoint.Epoch),
				Direction:  BackfillingCheckpointDirectionHead,
			}, nil
		}

		// If the head isn't up to date, we can return the next finalized epoch to process.
		if marker.FinalizedEpoch < uint64(checkpoint.Epoch) {
			next := phase0.Epoch(marker.FinalizedEpoch + 1)

			return &BackFillingCheckpointNextResponse{
				Next:       next,
				LookAheads: c.calculateFinalizedLookAheads(next, checkpoint.Epoch),
				Direction:  BackfillingCheckpointDirectionHead,
			}, nil
		}

		// If the backfill hasn't completed, we can return the next backfill epoch to process.
		//nolint:gosec // marker.BackfillEpoch is an int64
		if c.shouldBackfill(ctx) && phase0.Epoch(marker.BackfillEpoch) > backfillTargetEpoch {
			next := phase0.Epoch(marker.BackfillEpoch - 1)

			c.log.WithFields(logrus.Fields{
				"next_epoch":   next,
				"target_epoch": backfillTargetEpoch,
			}).Info("Derived next backfill epoch to process")

			return &BackFillingCheckpointNextResponse{
				Next:       next,
				LookAheads: c.calculateBackfillingLookAheads(next),
				Direction:  BackfillingCheckpointDirectionBackfill,
			}, nil
		}

		// The backfill is done, and the finalized epoch is up to date - we can sleep until the next epoch.
		if checkpoint.Epoch == phase0.Epoch(marker.FinalizedEpoch) {
			// Sleep until the next epoch
			epoch := c.wallclock.Epochs().Current()

			sleepFor := time.Until(epoch.TimeWindow().End())

			c.log.WithFields(logrus.Fields{
				"wallclock_epoch":       epoch.Number(),
				"sleep_for":             sleepFor.String(),
				"finalized_epoch":       checkpoint.Epoch,
				"backfill_epoch_marker": marker.BackfillEpoch,
				"head_epoch_marker":     marker.FinalizedEpoch,
				"backfill_epoch_target": backfillTargetEpoch,
			}).Info("Sleeping until next epoch")

			time.Sleep(sleepFor)

			// Sleep for an additional 5 seconds to give the beacon node time to do epoch processing.
			time.Sleep(5 * time.Second)

			continue
		}

		// Log the current state for debugging
		c.log.WithFields(logrus.Fields{
			"marker_finalized_epoch": marker.FinalizedEpoch,
			"marker_backfill_epoch":  marker.BackfillEpoch,
			"checkpoint_epoch":       checkpoint.Epoch,
			"backfill_target_epoch":  backfillTargetEpoch,
		}).Info("Current state before returning unknown state")

		return nil, errors.New("unknown state")
	}
}

func (c *BackfillingCheckpoint) shouldBackfill(ctx context.Context) bool {
	return c.config.Backfill.Enabled
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

func (c *BackfillingCheckpoint) calculateBackfillingLookAheads(epoch phase0.Epoch) []phase0.Epoch {
	epochs := []phase0.Epoch{}

	for i := 0; i < c.lookAheadDistance; i++ {
		e := epoch - phase0.Epoch(i)

		epochs = append(epochs, e)
	}

	return epochs
}

func (c *BackfillingCheckpoint) calculateFinalizedLookAheads(epoch, finalizedEpoch phase0.Epoch) []phase0.Epoch {
	epochs := []phase0.Epoch{}

	for i := 0; i < c.lookAheadDistance; i++ {
		e := epoch + phase0.Epoch(i)
		if e >= finalizedEpoch {
			break
		}

		epochs = append(epochs, e)
	}

	return epochs
}

func (c *BackfillingCheckpoint) GetMarker(location *xatu.CannonLocation) (*xatu.BackfillingCheckpointMarker, error) {
	if location == nil {
		return nil, errors.New("location is nil")
	}

	var marker *xatu.BackfillingCheckpointMarker

	switch location.Type {
	case xatu.CannonType_BEACON_API_ETH_V1_PROPOSER_DUTY:
		marker = location.GetEthV1BeaconProposerDuty().GetBackfillingCheckpointMarker()
	case xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_ELABORATED_ATTESTATION:
		marker = location.GetEthV2BeaconBlockElaboratedAttestation().GetBackfillingCheckpointMarker()
	case xatu.CannonType_BEACON_API_ETH_V1_BEACON_VALIDATORS:
		marker = location.GetEthV1BeaconValidators().GetBackfillingCheckpointMarker()
	case xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_ATTESTER_SLASHING:
		marker = location.GetEthV2BeaconBlockAttesterSlashing().GetBackfillingCheckpointMarker()
	case xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_PROPOSER_SLASHING:
		marker = location.GetEthV2BeaconBlockProposerSlashing().GetBackfillingCheckpointMarker()
	case xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_BLS_TO_EXECUTION_CHANGE:
		marker = location.GetEthV2BeaconBlockBlsToExecutionChange().GetBackfillingCheckpointMarker()
	case xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_EXECUTION_TRANSACTION:
		marker = location.GetEthV2BeaconBlockExecutionTransaction().GetBackfillingCheckpointMarker()
	case xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_VOLUNTARY_EXIT:
		marker = location.GetEthV2BeaconBlockVoluntaryExit().GetBackfillingCheckpointMarker()
	case xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_DEPOSIT:
		marker = location.GetEthV2BeaconBlockDeposit().GetBackfillingCheckpointMarker()
	case xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_WITHDRAWAL:
		marker = location.GetEthV2BeaconBlockWithdrawal().GetBackfillingCheckpointMarker()
	case xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK:
		marker = location.GetEthV2BeaconBlock().GetBackfillingCheckpointMarker()
	case xatu.CannonType_BEACON_API_ETH_V1_BEACON_BLOB_SIDECAR:
		marker = location.GetEthV1BeaconBlobSidecar().GetBackfillingCheckpointMarker()
	case xatu.CannonType_BEACON_API_ETH_V1_BEACON_COMMITTEE:
		marker = location.GetEthV1BeaconCommittee().GetBackfillingCheckpointMarker()
	default:
		return nil, errors.Errorf("unknown cannon type %s", location.Type)
	}

	if marker == nil {
		marker = &xatu.BackfillingCheckpointMarker{
			BackfillEpoch: -1,
		}
	}

	return marker, nil
}

func (c *BackfillingCheckpoint) getEarliestPossibleBackfillEpoch() (phase0.Epoch, error) {
	// earliestEpochForType is the earliest epoch for the type based on the fork epochs.
	// For example, the blob_sidecar cannon type will have an earliest epoch of the DENEB fork.
	if c.activationFork == spec.DataVersionPhase0 {
		return 0, nil
	}

	forkEpoch, err := c.beaconNode.Metadata().Spec.ForkEpochs.GetByName(c.activationFork.String())
	if err != nil {
		return 0, errors.Wrap(err, "failed to get fork epoch")
	}

	if c.config.Backfill.ToEpoch == 0 {
		// Use the default starting epoch for the type.
		return forkEpoch.Epoch, nil
	}

	return c.config.Backfill.ToEpoch, nil
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
	case xatu.CannonType_BEACON_API_ETH_V1_BEACON_COMMITTEE:
		location.Data = &xatu.CannonLocation_EthV1BeaconCommittee{
			EthV1BeaconCommittee: &xatu.CannonLocationEthV1BeaconCommittee{
				BackfillingCheckpointMarker: marker,
			},
		}
	default:
		return location, errors.Errorf("unknown cannon type %s", location.Type)
	}

	return location, nil
}
