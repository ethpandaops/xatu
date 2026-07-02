package iterator

import (
	"context"
	"time"

	"github.com/ethpandaops/ethwallclock"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/ethpandaops/xatu/pkg/cannon/coordinator"
	"github.com/ethpandaops/xatu/pkg/cannon/ethereum"
	"github.com/ethpandaops/xatu/pkg/observability"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

// BackfillingBlockConfig configures an EL cannon block iterator.
type BackfillingBlockConfig struct {
	// MaxRangeSize is the maximum number of blocks returned by a single Next().
	MaxRangeSize uint64 `yaml:"maxRangeSize" default:"50"`
	Backfill     struct {
		Enabled bool   `yaml:"enabled" default:"false"`
		ToBlock uint64 `yaml:"toBlock" default:"0"`
	} `yaml:"backfill"`
}

// BackfillingBlockDirection is the direction an EL cannon iterator is moving.
type BackfillingBlockDirection string

const (
	BackfillingBlockDirectionHead     BackfillingBlockDirection = "head"
	BackfillingBlockDirectionBackfill BackfillingBlockDirection = "backfill"
)

const logKeySleepFor = "sleep_for"

// BackFillingBlockNextResponse is an inclusive block range to process.
type BackFillingBlockNextResponse struct {
	From      uint64
	To        uint64
	Direction BackfillingBlockDirection
}

// BackfillingBlock walks execution-block-number space, gated on the consensus
// layer: the head frontier never runs past the execution block embedded in the
// CL-finalized beacon block. Backfill walks immutable history down to a floor.
type BackfillingBlock struct {
	log         observability.ContextualLogger
	cannonType  xatu.CannonType
	coordinator coordinator.Client
	wallclock   *ethwallclock.EthereumBeaconChain
	networkID   string
	networkName string
	metrics     *BackfillingBlockMetrics
	beaconNode  *ethereum.BeaconNode
	config      *BackfillingBlockConfig
}

// NewBackfillingBlock creates an EL cannon block iterator.
func NewBackfillingBlock(
	log observability.ContextualLogger, networkName, networkID string,
	cannonType xatu.CannonType,
	coordinatorClient *coordinator.Client,
	wallclock *ethwallclock.EthereumBeaconChain,
	metrics *BackfillingBlockMetrics,
	beacon *ethereum.BeaconNode,
	config *BackfillingBlockConfig,
) *BackfillingBlock {
	return &BackfillingBlock{
		log: log.
			WithField("module", "cannon/iterator/backfilling_block_iterator").
			WithField("cannon_type", cannonType.String()),
		networkName: networkName,
		networkID:   networkID,
		cannonType:  cannonType,
		coordinator: *coordinatorClient,
		wallclock:   wallclock,
		beaconNode:  beacon,
		metrics:     metrics,
		config:      config,
	}
}

// Start logs the iterator configuration.
func (b *BackfillingBlock) Start(ctx context.Context) error {
	b.log.WithFields(logrus.Fields{
		"backfill_enabled":  b.config.Backfill.Enabled,
		"backfill_to_block": b.config.Backfill.ToBlock,
		"max_range_size":    b.config.MaxRangeSize,
	}).WithContext(ctx).Info("Starting EL cannon block iterator")

	return nil
}

// maxRange returns the effective max range size (never zero).
func (b *BackfillingBlock) maxRange() uint64 {
	if b.config.MaxRangeSize == 0 {
		return 1
	}

	return b.config.MaxRangeSize
}

// minBackfillBlock is the lowest block a dataset can be collected from. The
// state-read datasets (balance/storage/nonce reads) cannot be traced at the
// genesis block — block 0 has no parent state, so cryo's state-access tracer
// fails on it. Clamping their floor to 1 avoids requesting (and then retrying
// forever) an untraceable range. Block 0 carries no state reads anyway, so this
// loses no data. All other datasets keep a floor of 0 (genesis is valid for
// e.g. the block row and the genesis premine balance_diffs).
func minBackfillBlock(cannonType xatu.CannonType) uint64 {
	switch cannonType {
	case xatu.CannonType_EXECUTION_CANONICAL_BALANCE_READS,
		xatu.CannonType_EXECUTION_CANONICAL_STORAGE_READS,
		xatu.CannonType_EXECUTION_CANONICAL_NONCE_READS:
		return 1
	default:
		return 0
	}
}

// fetchExecutionCeiling resolves the execution block number of the CL-finalized
// beacon block. This is the highest block EL cannon may process on the head.
func (b *BackfillingBlock) fetchExecutionCeiling(ctx context.Context) (uint64, error) {
	block, err := b.beaconNode.GetBeaconBlock(ctx, "finalized")
	if err != nil {
		return 0, errors.Wrap(err, "failed to fetch finalized beacon block")
	}

	if block == nil {
		return 0, errors.New("finalized beacon block is nil")
	}

	blockNumber, err := block.ExecutionBlockNumber()
	if err != nil {
		return 0, errors.Wrap(err, "failed to read execution block number from finalized beacon block")
	}

	return blockNumber, nil
}

// Next returns the next inclusive block range to process. It blocks (sleeping
// until the next epoch) when caught up to the CL-finalized ceiling and backfill
// is complete.
func (b *BackfillingBlock) Next(ctx context.Context) (*BackFillingBlockNextResponse, error) {
	for {
		ceiling, err := b.fetchExecutionCeiling(ctx)
		if err != nil {
			return nil, err
		}

		b.metrics.SetCeilingBlock(b.networkName, float64(ceiling))

		location, err := b.coordinator.GetCannonLocation(ctx, b.cannonType, b.networkID)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get cannon location")
		}

		// Fresh start: seed at the ceiling and process that single block. The
		// head walks up from here, backfill walks down.
		if location == nil {
			return &BackFillingBlockNextResponse{From: ceiling, To: ceiling, Direction: BackfillingBlockDirectionHead}, nil
		}

		marker, err := b.GetMarker(location)
		if err != nil {
			return nil, err
		}

		finalized := marker.GetFinalizedBlock()
		backfill := marker.GetBackfillBlock()

		if finalized == 0 && backfill <= 0 {
			return &BackFillingBlockNextResponse{From: ceiling, To: ceiling, Direction: BackfillingBlockDirectionHead}, nil
		}

		b.metrics.SetFinalizedBlock(b.cannonType.String(), b.networkName, float64(finalized))
		b.metrics.SetBackfillBlock(b.cannonType.String(), b.networkName, float64(backfill))

		// Head: catch up to the ceiling.
		if finalized < ceiling {
			from := finalized + 1
			to := min(from+b.maxRange()-1, ceiling)

			b.metrics.SetLag(b.cannonType.String(), b.networkName, BackfillingBlockDirectionHead, float64(ceiling-finalized))

			return &BackFillingBlockNextResponse{From: from, To: to, Direction: BackfillingBlockDirectionHead}, nil
		}

		// Backfill: walk down toward the floor. The effective floor never drops
		// below the dataset's hard minimum (e.g. 1 for state-read datasets,
		// whose genesis block is untraceable).
		floor := max(b.config.Backfill.ToBlock, minBackfillBlock(b.cannonType))
		//nolint:gosec // block numbers are far below int64 max.
		if b.config.Backfill.Enabled && backfill > int64(floor) {
			//nolint:gosec // backfill > floor >= 0 here, so the conversion is safe.
			to := uint64(backfill - 1)

			var from uint64
			if to+1 > b.maxRange() {
				from = to + 1 - b.maxRange()
			}

			if from < floor {
				from = floor
			}

			//nolint:gosec // block numbers are far below int64 max.
			b.metrics.SetLag(b.cannonType.String(), b.networkName, BackfillingBlockDirectionBackfill, float64(backfill-int64(floor)))

			return &BackFillingBlockNextResponse{From: from, To: to, Direction: BackfillingBlockDirectionBackfill}, nil
		}

		// Caught up and backfill complete: sleep until the next epoch, when CL
		// finality (and therefore the ceiling) advances.
		epoch := b.wallclock.Epochs().Current()
		sleepFor := time.Until(epoch.TimeWindow().End()) + 5*time.Second

		b.log.WithFields(logrus.Fields{
			"finalized_block": finalized,
			"backfill_block":  backfill,
			"ceiling_block":   ceiling,
			logKeySleepFor:    sleepFor.String(),
		}).WithContext(ctx).Info("Caught up to CL-finalized ceiling, sleeping until next epoch")

		time.Sleep(sleepFor)
	}
}

// UpdateLocation advances the cursor after a range has been processed. Head
// advances finalized_block to the top of the range; backfill lowers
// backfill_block to the bottom of the range.
func (b *BackfillingBlock) UpdateLocation(ctx context.Context, from, to uint64, direction BackfillingBlockDirection) error {
	location, err := b.coordinator.GetCannonLocation(ctx, b.cannonType, b.networkID)
	if err != nil {
		return errors.Wrap(err, "failed to get cannon location")
	}

	if location == nil {
		//nolint:gosec // block numbers are far below int64 max.
		location, err = b.createLocation(to, int64(from))
		if err != nil {
			return errors.Wrap(err, "failed to create fresh location")
		}
	}

	marker, err := b.GetMarker(location)
	if err != nil {
		return err
	}

	switch direction {
	case BackfillingBlockDirectionHead:
		marker.FinalizedBlock = to
		// Seed the backfill cursor on the very first head advance.
		if marker.GetBackfillBlock() <= 0 {
			//nolint:gosec // block numbers are far below int64 max.
			marker.BackfillBlock = int64(from)
		}
	case BackfillingBlockDirectionBackfill:
		//nolint:gosec // block numbers are far below int64 max.
		marker.BackfillBlock = int64(from)
	default:
		return errors.Errorf("unknown direction (%s) when updating cannon location", direction)
	}

	newLocation, err := b.createLocation(marker.GetFinalizedBlock(), marker.GetBackfillBlock())
	if err != nil {
		return err
	}

	if err := b.coordinator.UpsertCannonLocationRequest(ctx, newLocation); err != nil {
		return errors.Wrap(err, "failed to update cannon location")
	}

	b.metrics.SetFinalizedBlock(b.cannonType.String(), b.networkName, float64(marker.GetFinalizedBlock()))
	b.metrics.SetBackfillBlock(b.cannonType.String(), b.networkName, float64(marker.GetBackfillBlock()))

	return nil
}

// GetMarker extracts the block marker from a location.
func (b *BackfillingBlock) GetMarker(location *xatu.CannonLocation) (*xatu.BackfillingBlockMarker, error) {
	if location == nil {
		return nil, errors.New("location is nil")
	}

	var marker *xatu.BackfillingBlockMarker

	switch location.GetType() {
	case xatu.CannonType_EXECUTION_CANONICAL_BLOCK:
		marker = location.GetExecutionCanonicalBlock().GetBackfillingBlockMarker()
	case xatu.CannonType_EXECUTION_CANONICAL_TRANSACTION:
		marker = location.GetExecutionCanonicalTransaction().GetBackfillingBlockMarker()
	case xatu.CannonType_EXECUTION_CANONICAL_LOGS:
		marker = location.GetExecutionCanonicalLogs().GetBackfillingBlockMarker()
	case xatu.CannonType_EXECUTION_CANONICAL_TRACES:
		marker = location.GetExecutionCanonicalTraces().GetBackfillingBlockMarker()
	case xatu.CannonType_EXECUTION_CANONICAL_NATIVE_TRANSFERS:
		marker = location.GetExecutionCanonicalNativeTransfers().GetBackfillingBlockMarker()
	case xatu.CannonType_EXECUTION_CANONICAL_ERC20_TRANSFERS:
		marker = location.GetExecutionCanonicalErc20Transfers().GetBackfillingBlockMarker()
	case xatu.CannonType_EXECUTION_CANONICAL_ERC721_TRANSFERS:
		marker = location.GetExecutionCanonicalErc721Transfers().GetBackfillingBlockMarker()
	case xatu.CannonType_EXECUTION_CANONICAL_CONTRACTS:
		marker = location.GetExecutionCanonicalContracts().GetBackfillingBlockMarker()
	case xatu.CannonType_EXECUTION_CANONICAL_BALANCE_DIFFS:
		marker = location.GetExecutionCanonicalBalanceDiffs().GetBackfillingBlockMarker()
	case xatu.CannonType_EXECUTION_CANONICAL_STORAGE_DIFFS:
		marker = location.GetExecutionCanonicalStorageDiffs().GetBackfillingBlockMarker()
	case xatu.CannonType_EXECUTION_CANONICAL_NONCE_DIFFS:
		marker = location.GetExecutionCanonicalNonceDiffs().GetBackfillingBlockMarker()
	case xatu.CannonType_EXECUTION_CANONICAL_BALANCE_READS:
		marker = location.GetExecutionCanonicalBalanceReads().GetBackfillingBlockMarker()
	case xatu.CannonType_EXECUTION_CANONICAL_STORAGE_READS:
		marker = location.GetExecutionCanonicalStorageReads().GetBackfillingBlockMarker()
	case xatu.CannonType_EXECUTION_CANONICAL_NONCE_READS:
		marker = location.GetExecutionCanonicalNonceReads().GetBackfillingBlockMarker()
	case xatu.CannonType_EXECUTION_CANONICAL_FOUR_BYTE_COUNTS:
		marker = location.GetExecutionCanonicalFourByteCounts().GetBackfillingBlockMarker()
	case xatu.CannonType_EXECUTION_CANONICAL_ADDRESS_APPEARANCES:
		marker = location.GetExecutionCanonicalAddressAppearances().GetBackfillingBlockMarker()
	default:
		return nil, errors.Errorf("unknown cannon type %s", location.GetType())
	}

	if marker == nil {
		marker = &xatu.BackfillingBlockMarker{BackfillBlock: -1}
	}

	return marker, nil
}

// createLocation builds a CannonLocation for the iterator's type.
func (b *BackfillingBlock) createLocation(finalized uint64, backfill int64) (*xatu.CannonLocation, error) {
	location := &xatu.CannonLocation{
		NetworkId: b.networkID,
		Type:      b.cannonType,
	}

	marker := &xatu.BackfillingBlockMarker{
		FinalizedBlock: finalized,
		BackfillBlock:  backfill,
	}

	switch b.cannonType {
	case xatu.CannonType_EXECUTION_CANONICAL_BLOCK:
		location.Data = &xatu.CannonLocation_ExecutionCanonicalBlock{
			ExecutionCanonicalBlock: &xatu.CannonLocationExecutionCanonicalBlock{
				BackfillingBlockMarker: marker,
			},
		}
	case xatu.CannonType_EXECUTION_CANONICAL_TRANSACTION:
		location.Data = &xatu.CannonLocation_ExecutionCanonicalTransaction{
			ExecutionCanonicalTransaction: &xatu.CannonLocationExecutionCanonicalTransaction{
				BackfillingBlockMarker: marker,
			},
		}
	case xatu.CannonType_EXECUTION_CANONICAL_LOGS:
		location.Data = &xatu.CannonLocation_ExecutionCanonicalLogs{
			ExecutionCanonicalLogs: &xatu.CannonLocationExecutionCanonicalLogs{BackfillingBlockMarker: marker},
		}
	case xatu.CannonType_EXECUTION_CANONICAL_TRACES:
		location.Data = &xatu.CannonLocation_ExecutionCanonicalTraces{
			ExecutionCanonicalTraces: &xatu.CannonLocationExecutionCanonicalTraces{BackfillingBlockMarker: marker},
		}
	case xatu.CannonType_EXECUTION_CANONICAL_NATIVE_TRANSFERS:
		location.Data = &xatu.CannonLocation_ExecutionCanonicalNativeTransfers{
			ExecutionCanonicalNativeTransfers: &xatu.CannonLocationExecutionCanonicalNativeTransfers{BackfillingBlockMarker: marker},
		}
	case xatu.CannonType_EXECUTION_CANONICAL_ERC20_TRANSFERS:
		location.Data = &xatu.CannonLocation_ExecutionCanonicalErc20Transfers{
			ExecutionCanonicalErc20Transfers: &xatu.CannonLocationExecutionCanonicalErc20Transfers{BackfillingBlockMarker: marker},
		}
	case xatu.CannonType_EXECUTION_CANONICAL_ERC721_TRANSFERS:
		location.Data = &xatu.CannonLocation_ExecutionCanonicalErc721Transfers{
			ExecutionCanonicalErc721Transfers: &xatu.CannonLocationExecutionCanonicalErc721Transfers{BackfillingBlockMarker: marker},
		}
	case xatu.CannonType_EXECUTION_CANONICAL_CONTRACTS:
		location.Data = &xatu.CannonLocation_ExecutionCanonicalContracts{
			ExecutionCanonicalContracts: &xatu.CannonLocationExecutionCanonicalContracts{BackfillingBlockMarker: marker},
		}
	case xatu.CannonType_EXECUTION_CANONICAL_BALANCE_DIFFS:
		location.Data = &xatu.CannonLocation_ExecutionCanonicalBalanceDiffs{
			ExecutionCanonicalBalanceDiffs: &xatu.CannonLocationExecutionCanonicalBalanceDiffs{BackfillingBlockMarker: marker},
		}
	case xatu.CannonType_EXECUTION_CANONICAL_STORAGE_DIFFS:
		location.Data = &xatu.CannonLocation_ExecutionCanonicalStorageDiffs{
			ExecutionCanonicalStorageDiffs: &xatu.CannonLocationExecutionCanonicalStorageDiffs{BackfillingBlockMarker: marker},
		}
	case xatu.CannonType_EXECUTION_CANONICAL_NONCE_DIFFS:
		location.Data = &xatu.CannonLocation_ExecutionCanonicalNonceDiffs{
			ExecutionCanonicalNonceDiffs: &xatu.CannonLocationExecutionCanonicalNonceDiffs{BackfillingBlockMarker: marker},
		}
	case xatu.CannonType_EXECUTION_CANONICAL_BALANCE_READS:
		location.Data = &xatu.CannonLocation_ExecutionCanonicalBalanceReads{
			ExecutionCanonicalBalanceReads: &xatu.CannonLocationExecutionCanonicalBalanceReads{BackfillingBlockMarker: marker},
		}
	case xatu.CannonType_EXECUTION_CANONICAL_STORAGE_READS:
		location.Data = &xatu.CannonLocation_ExecutionCanonicalStorageReads{
			ExecutionCanonicalStorageReads: &xatu.CannonLocationExecutionCanonicalStorageReads{BackfillingBlockMarker: marker},
		}
	case xatu.CannonType_EXECUTION_CANONICAL_NONCE_READS:
		location.Data = &xatu.CannonLocation_ExecutionCanonicalNonceReads{
			ExecutionCanonicalNonceReads: &xatu.CannonLocationExecutionCanonicalNonceReads{BackfillingBlockMarker: marker},
		}
	case xatu.CannonType_EXECUTION_CANONICAL_FOUR_BYTE_COUNTS:
		location.Data = &xatu.CannonLocation_ExecutionCanonicalFourByteCounts{
			ExecutionCanonicalFourByteCounts: &xatu.CannonLocationExecutionCanonicalFourByteCounts{BackfillingBlockMarker: marker},
		}
	case xatu.CannonType_EXECUTION_CANONICAL_ADDRESS_APPEARANCES:
		location.Data = &xatu.CannonLocation_ExecutionCanonicalAddressAppearances{
			ExecutionCanonicalAddressAppearances: &xatu.CannonLocationExecutionCanonicalAddressAppearances{BackfillingBlockMarker: marker},
		}
	default:
		return location, errors.Errorf("unknown cannon type %s", location.GetType())
	}

	return location, nil
}
