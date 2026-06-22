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

// StorageDiffsDeriverName is the cannon type produced by the storage diffs deriver.
const StorageDiffsDeriverName = xatu.CannonType_EXECUTION_CANONICAL_STORAGE_DIFFS

const storageDiffsDataset = "storage_diffs"

// storageDiffsColumns restricts cryo output to exactly the columns mapped.
var storageDiffsColumns = []string{
	"block_number",
	"transaction_index",
	"transaction_hash",
	"address",
	"slot",
	"from_value",
	"to_value",
}

// storageDiffRow mirrors the cryo `storage_diffs` parquet schema (--hex). slot,
// from_value and to_value arrive as 0x-prefixed hex strings.
type storageDiffRow struct {
	BlockNumber      uint32 `parquet:"block_number"`
	TransactionIndex uint64 `parquet:"transaction_index"`
	TransactionHash  string `parquet:"transaction_hash"`
	Address          string `parquet:"address"`
	Slot             string `parquet:"slot"`
	FromValue        string `parquet:"from_value"`
	ToValue          string `parquet:"to_value"`
	InternalIndex    uint32
}

// StorageDiffsDeriverConfig configures the storage diffs deriver.
type StorageDiffsDeriverConfig struct {
	Enabled   bool                            `yaml:"enabled" default:"false"`
	ChunkSize int                             `yaml:"chunkSize" default:"100"`
	Iterator  iterator.BackfillingBlockConfig `yaml:"iterator"`
}

// StorageDiffsDeriver derives canonical_execution_storage_diffs events via cryo.
type StorageDiffsDeriver struct {
	base

	cfg        *StorageDiffsDeriverConfig
	cryo       *cryo.Runner
	clientMeta *xatu.ClientMeta
}

// NewStorageDiffsDeriver creates a storage diffs deriver.
func NewStorageDiffsDeriver(
	log observability.ContextualLogger,
	config *StorageDiffsDeriverConfig,
	iter *iterator.BackfillingBlock,
	runner *cryo.Runner,
	beacon *ethereum.BeaconNode,
	clientMeta *xatu.ClientMeta,
) *StorageDiffsDeriver {
	return &StorageDiffsDeriver{
		base: base{
			log: log.WithFields(logrus.Fields{
				"module": "cannon/event/execution/storage_diffs",
				"type":   StorageDiffsDeriverName.String(),
			}),
			name:     StorageDiffsDeriverName.String(),
			beacon:   beacon,
			iterator: iter,
		},
		cfg:        config,
		cryo:       runner,
		clientMeta: clientMeta,
	}
}

// CannonType returns the deriver's cannon type.
func (b *StorageDiffsDeriver) CannonType() xatu.CannonType {
	return StorageDiffsDeriverName
}

// ActivationFork returns Phase0 so the deriver starts as soon as the beacon
// node is ready; the iterator gates EL progress on CL finality.
func (b *StorageDiffsDeriver) ActivationFork() spec.DataVersion {
	return spec.DataVersionPhase0
}

// Name returns the deriver name.
func (b *StorageDiffsDeriver) Name() string {
	return b.name
}

// Start begins the deriver loop.
func (b *StorageDiffsDeriver) Start(ctx context.Context) error {
	if !b.cfg.Enabled {
		b.log.WithContext(ctx).Info("Execution storage diffs deriver disabled")

		return nil
	}

	b.log.WithContext(ctx).Info("Execution storage diffs deriver enabled")

	if err := b.iterator.Start(ctx); err != nil {
		return errors.Wrap(err, "failed to start iterator")
	}

	b.run(ctx, b.processRange)

	return nil
}

// Stop is a no-op for the storage diffs deriver.
func (b *StorageDiffsDeriver) Stop(_ context.Context) error {
	return nil
}

// processRange runs cryo for [from, to] and builds chunked DecoratedEvents.
func (b *StorageDiffsDeriver) processRange(ctx context.Context, from, to uint64) ([]*xatu.DecoratedEvent, error) {
	ctx, span := observability.Tracer().Start(ctx,
		"StorageDiffsDeriver.processRange",
		trace.WithAttributes(
			attribute.Int64("from", int64(from)), //nolint:gosec // block numbers are far below int64 max.
			attribute.Int64("to", int64(to)),     //nolint:gosec // block numbers are far below int64 max.
		),
	)
	defer span.End()

	collection, err := b.cryo.Collect(ctx, storageDiffsDataset, from, to, storageDiffsColumns)
	if err != nil {
		return nil, errors.Wrap(err, "failed to collect storage diffs via cryo")
	}

	defer func() {
		if cErr := collection.Close(); cErr != nil {
			b.log.WithError(cErr).WithContext(ctx).Warn("Failed to clean up cryo output")
		}
	}()

	rows, err := cryo.ReadParquet[storageDiffRow](collection.Files)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read cryo storage diffs parquet")
	}

	stampInternalIndex(rows,
		func(r *storageDiffRow) uint64 { return uint64(r.BlockNumber) },
		func(r *storageDiffRow) string { return r.TransactionHash },
		func(r *storageDiffRow, idx uint32) { r.InternalIndex = idx },
	)

	return chunkEvents(rows, b.cfg.ChunkSize, b.createEvent)
}

// createEvent builds a single DecoratedEvent carrying a chunk of storage diff rows.
func (b *StorageDiffsDeriver) createEvent(rows []storageDiffRow) (*xatu.DecoratedEvent, error) {
	metadata, ok := proto.Clone(b.clientMeta).(*xatu.ClientMeta)
	if !ok {
		return nil, errors.New("failed to clone client metadata")
	}

	storageDiffs := make([]*xatu.ExecutionStorageDiff, 0, len(rows))

	for i := range rows {
		storageDiffs = append(storageDiffs, storageDiffRowToProto(&rows[i]))
	}

	return &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_EXECUTION_CANONICAL_STORAGE_DIFFS,
			DateTime: timestamppb.New(time.Now()),
			Id:       uuid.New().String(),
		},
		Meta: &xatu.Meta{Client: metadata},
		Data: &xatu.DecoratedEvent_ExecutionCanonicalStorageDiffs{
			ExecutionCanonicalStorageDiffs: &xatu.ExecutionCanonicalStorageDiffs{StorageDiffs: storageDiffs},
		},
	}, nil
}

// storageDiffRowToProto converts a cryo storage diff row to proto. slot,
// from_value and to_value are passed through as plain 0x-hex strings.
func storageDiffRowToProto(row *storageDiffRow) *xatu.ExecutionStorageDiff {
	return &xatu.ExecutionStorageDiff{
		BlockNumber:      uint64(row.BlockNumber),
		TransactionIndex: row.TransactionIndex,
		TransactionHash:  row.TransactionHash,
		InternalIndex:    row.InternalIndex,
		Address:          row.Address,
		Slot:             row.Slot,
		FromValue:        row.FromValue,
		ToValue:          row.ToValue,
	}
}
