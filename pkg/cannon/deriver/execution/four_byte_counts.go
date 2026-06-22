package execution

import (
	"context"
	"time"

	"github.com/ethpandaops/go-eth2-client/spec"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/ethpandaops/xatu/pkg/cannon/ethereum"
	"github.com/ethpandaops/xatu/pkg/cannon/iterator"
	"github.com/ethpandaops/xatu/pkg/cryo"
	"github.com/ethpandaops/xatu/pkg/observability"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

// FourByteCountsDeriverName is the cannon type produced by the four_byte_counts deriver.
const FourByteCountsDeriverName = xatu.CannonType_EXECUTION_CANONICAL_FOUR_BYTE_COUNTS

const fourByteCountsDataset = "four_byte_counts"

// fourByteCountsColumns restricts cryo output to exactly the columns mapped.
var fourByteCountsColumns = []string{
	"block_number",
	"transaction_index",
	"transaction_hash",
	"signature",
	"size",
	"count",
}

// fourByteCountRow mirrors the cryo `four_byte_counts` parquet schema (--hex).
type fourByteCountRow struct {
	BlockNumber      uint32 `parquet:"block_number"`
	TransactionIndex uint32 `parquet:"transaction_index"`
	TransactionHash  string `parquet:"transaction_hash"`
	Signature        string `parquet:"signature"`
	Size             uint64 `parquet:"size"`
	Count            uint64 `parquet:"count"`
}

// FourByteCountsDeriverConfig configures the four_byte_counts deriver.
type FourByteCountsDeriverConfig struct {
	Enabled   bool                            `yaml:"enabled" default:"false"`
	ChunkSize int                             `yaml:"chunkSize" default:"100"`
	Iterator  iterator.BackfillingBlockConfig `yaml:"iterator"`
}

// FourByteCountsDeriver derives canonical_execution_four_byte_counts events via cryo.
type FourByteCountsDeriver struct {
	base

	cfg        *FourByteCountsDeriverConfig
	cryo       *cryo.Runner
	clientMeta *xatu.ClientMeta
}

// NewFourByteCountsDeriver creates a four_byte_counts deriver.
func NewFourByteCountsDeriver(
	log observability.ContextualLogger,
	config *FourByteCountsDeriverConfig,
	iter *iterator.BackfillingBlock,
	runner *cryo.Runner,
	beacon *ethereum.BeaconNode,
	clientMeta *xatu.ClientMeta,
) *FourByteCountsDeriver {
	return &FourByteCountsDeriver{
		base: base{
			log: log.WithFields(logrus.Fields{
				"module": "cannon/event/execution/four_byte_counts",
				"type":   FourByteCountsDeriverName.String(),
			}),
			name:     FourByteCountsDeriverName.String(),
			beacon:   beacon,
			iterator: iter,
		},
		cfg:        config,
		cryo:       runner,
		clientMeta: clientMeta,
	}
}

// CannonType returns the deriver's cannon type.
func (b *FourByteCountsDeriver) CannonType() xatu.CannonType {
	return FourByteCountsDeriverName
}

// ActivationFork returns Phase0 so the deriver starts as soon as the beacon
// node is ready; the iterator gates EL progress on CL finality.
func (b *FourByteCountsDeriver) ActivationFork() spec.DataVersion {
	return spec.DataVersionPhase0
}

// Name returns the deriver name.
func (b *FourByteCountsDeriver) Name() string {
	return b.name
}

// Start begins the deriver loop.
func (b *FourByteCountsDeriver) Start(ctx context.Context) error {
	if !b.cfg.Enabled {
		b.log.WithContext(ctx).Info("Execution four_byte_counts deriver disabled")

		return nil
	}

	b.log.WithContext(ctx).Info("Execution four_byte_counts deriver enabled")

	if err := b.iterator.Start(ctx); err != nil {
		return errors.Wrap(err, "failed to start iterator")
	}

	b.run(ctx, b.processRange)

	return nil
}

// Stop is a no-op for the four_byte_counts deriver.
func (b *FourByteCountsDeriver) Stop(_ context.Context) error {
	return nil
}

// processRange runs cryo for [from, to] and builds chunked DecoratedEvents.
func (b *FourByteCountsDeriver) processRange(ctx context.Context, from, to uint64) ([]*xatu.DecoratedEvent, error) {
	ctx, span := observability.Tracer().Start(ctx,
		"FourByteCountsDeriver.processRange",
		trace.WithAttributes(
			attribute.Int64("from", int64(from)), //nolint:gosec // block numbers are far below int64 max.
			attribute.Int64("to", int64(to)),     //nolint:gosec // block numbers are far below int64 max.
		),
	)
	defer span.End()

	collection, err := b.cryo.Collect(ctx, fourByteCountsDataset, from, to, fourByteCountsColumns)
	if err != nil {
		return nil, errors.Wrap(err, "failed to collect four_byte_counts via cryo")
	}

	defer func() {
		if cErr := collection.Close(); cErr != nil {
			b.log.WithError(cErr).WithContext(ctx).Warn("Failed to clean up cryo output")
		}
	}()

	rows, err := cryo.ReadParquet[fourByteCountRow](collection.Files)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read cryo four_byte_counts parquet")
	}

	return chunkEvents(rows, b.cfg.ChunkSize, b.createEvent)
}

// createEvent builds a single DecoratedEvent carrying a chunk of four_byte_count rows.
func (b *FourByteCountsDeriver) createEvent(rows []fourByteCountRow) (*xatu.DecoratedEvent, error) {
	metadata, ok := proto.Clone(b.clientMeta).(*xatu.ClientMeta)
	if !ok {
		return nil, errors.New("failed to clone client metadata")
	}

	fourByteCounts := make([]*xatu.ExecutionFourByteCount, 0, len(rows))

	for i := range rows {
		fourByteCounts = append(fourByteCounts, fourByteCountRowToProto(&rows[i]))
	}

	return &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_EXECUTION_CANONICAL_FOUR_BYTE_COUNTS,
			DateTime: timestamppb.New(time.Now()),
			Id:       uuid.New().String(),
		},
		Meta: &xatu.Meta{Client: metadata},
		Data: &xatu.DecoratedEvent_ExecutionCanonicalFourByteCounts{
			ExecutionCanonicalFourByteCounts: &xatu.ExecutionCanonicalFourByteCounts{FourByteCounts: fourByteCounts},
		},
	}, nil
}

// fourByteCountRowToProto converts a cryo four_byte_count row (collected with --hex) to proto.
func fourByteCountRowToProto(row *fourByteCountRow) *xatu.ExecutionFourByteCount {
	return &xatu.ExecutionFourByteCount{
		BlockNumber:      uint64(row.BlockNumber),
		TransactionIndex: uint64(row.TransactionIndex),
		TransactionHash:  row.TransactionHash,
		Signature:        row.Signature,
		Size:             row.Size,
		Count:            row.Count,
	}
}
