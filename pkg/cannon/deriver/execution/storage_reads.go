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

// StorageReadsDeriverName is the cannon type produced by the storage reads deriver.
const StorageReadsDeriverName = xatu.CannonType_EXECUTION_CANONICAL_STORAGE_READS

const storageReadsDataset = "storage_reads"

// storageReadsColumns restricts cryo output to exactly the columns mapped.
var storageReadsColumns = []string{
	"block_number",
	"transaction_index",
	"transaction_hash",
	"contract_address",
	"slot",
	"value",
}

// storageReadRow mirrors the cryo `storage_reads` parquet schema (--hex).
type storageReadRow struct {
	BlockNumber      uint32 `parquet:"block_number"`
	TransactionIndex uint32 `parquet:"transaction_index"`
	TransactionHash  string `parquet:"transaction_hash"`
	ContractAddress  string `parquet:"contract_address"`
	Slot             string `parquet:"slot"`
	Value            string `parquet:"value"`

	InternalIndex uint32
}

// StorageReadsDeriverConfig configures the storage reads deriver.
type StorageReadsDeriverConfig struct {
	Enabled   bool                            `yaml:"enabled" default:"false"`
	ChunkSize int                             `yaml:"chunkSize" default:"100"`
	Iterator  iterator.BackfillingBlockConfig `yaml:"iterator"`
}

// StorageReadsDeriver derives canonical_execution_storage_reads events via cryo.
type StorageReadsDeriver struct {
	base

	cfg        *StorageReadsDeriverConfig
	cryo       *cryo.Runner
	clientMeta *xatu.ClientMeta
}

// NewStorageReadsDeriver creates a storage reads deriver.
func NewStorageReadsDeriver(
	log observability.ContextualLogger,
	config *StorageReadsDeriverConfig,
	iter *iterator.BackfillingBlock,
	runner *cryo.Runner,
	beacon *ethereum.BeaconNode,
	clientMeta *xatu.ClientMeta,
) *StorageReadsDeriver {
	return &StorageReadsDeriver{
		base: base{
			log: log.WithFields(logrus.Fields{
				"module": "cannon/event/execution/storage_reads",
				"type":   StorageReadsDeriverName.String(),
			}),
			name:     StorageReadsDeriverName.String(),
			beacon:   beacon,
			iterator: iter,
		},
		cfg:        config,
		cryo:       runner,
		clientMeta: clientMeta,
	}
}

// CannonType returns the deriver's cannon type.
func (b *StorageReadsDeriver) CannonType() xatu.CannonType {
	return StorageReadsDeriverName
}

// ActivationFork returns Phase0 so the deriver starts as soon as the beacon
// node is ready; the iterator gates EL progress on CL finality.
func (b *StorageReadsDeriver) ActivationFork() spec.DataVersion {
	return spec.DataVersionPhase0
}

// Name returns the deriver name.
func (b *StorageReadsDeriver) Name() string {
	return b.name
}

// Start begins the deriver loop.
func (b *StorageReadsDeriver) Start(ctx context.Context) error {
	if !b.cfg.Enabled {
		b.log.WithContext(ctx).Info("Execution storage reads deriver disabled")

		return nil
	}

	b.log.WithContext(ctx).Info("Execution storage reads deriver enabled")

	if err := b.iterator.Start(ctx); err != nil {
		return errors.Wrap(err, "failed to start iterator")
	}

	b.run(ctx, b.processRange)

	return nil
}

// Stop is a no-op for the storage reads deriver.
func (b *StorageReadsDeriver) Stop(_ context.Context) error {
	return nil
}

// processRange runs cryo for [from, to] and builds chunked DecoratedEvents.
func (b *StorageReadsDeriver) processRange(ctx context.Context, from, to uint64) ([]*xatu.DecoratedEvent, error) {
	ctx, span := observability.Tracer().Start(ctx,
		"StorageReadsDeriver.processRange",
		trace.WithAttributes(
			attribute.Int64("from", int64(from)), //nolint:gosec // block numbers are far below int64 max.
			attribute.Int64("to", int64(to)),     //nolint:gosec // block numbers are far below int64 max.
		),
	)
	defer span.End()

	collection, err := b.cryo.Collect(ctx, storageReadsDataset, from, to, storageReadsColumns)
	if err != nil {
		return nil, errors.Wrap(err, "failed to collect storage reads via cryo")
	}

	defer func() {
		if cErr := collection.Close(); cErr != nil {
			b.log.WithError(cErr).WithContext(ctx).Warn("Failed to clean up cryo output")
		}
	}()

	rows, err := cryo.ReadParquet[storageReadRow](collection.Files)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read cryo storage reads parquet")
	}

	stampInternalIndex(rows,
		func(r *storageReadRow) uint64 { return uint64(r.BlockNumber) },
		func(r *storageReadRow) string { return r.TransactionHash },
		func(r *storageReadRow, idx uint32) { r.InternalIndex = idx },
	)

	return chunkEvents(rows, b.cfg.ChunkSize, b.createEvent)
}

// createEvent builds a single DecoratedEvent carrying a chunk of storage read rows.
func (b *StorageReadsDeriver) createEvent(rows []storageReadRow) (*xatu.DecoratedEvent, error) {
	metadata, ok := proto.Clone(b.clientMeta).(*xatu.ClientMeta)
	if !ok {
		return nil, errors.New("failed to clone client metadata")
	}

	storageReads := make([]*xatu.ExecutionStorageRead, 0, len(rows))

	for i := range rows {
		storageReads = append(storageReads, storageReadRowToProto(&rows[i]))
	}

	return &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_EXECUTION_CANONICAL_STORAGE_READS,
			DateTime: timestamppb.New(time.Now()),
			Id:       uuid.New().String(),
		},
		Meta: &xatu.Meta{Client: metadata},
		Data: &xatu.DecoratedEvent_ExecutionCanonicalStorageReads{
			ExecutionCanonicalStorageReads: &xatu.ExecutionCanonicalStorageReads{StorageReads: storageReads},
		},
	}, nil
}

// storageReadRowToProto converts a cryo storage read row (collected with --hex) to proto.
func storageReadRowToProto(row *storageReadRow) *xatu.ExecutionStorageRead {
	return &xatu.ExecutionStorageRead{
		BlockNumber:      uint64(row.BlockNumber),
		TransactionIndex: uint64(row.TransactionIndex),
		TransactionHash:  row.TransactionHash,
		InternalIndex:    row.InternalIndex,
		ContractAddress:  row.ContractAddress,
		Slot:             row.Slot,
		Value:            row.Value,
	}
}
