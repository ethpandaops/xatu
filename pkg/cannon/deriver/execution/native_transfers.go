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

// NativeTransfersDeriverName is the cannon type produced by the native transfers deriver.
const NativeTransfersDeriverName = xatu.CannonType_EXECUTION_CANONICAL_NATIVE_TRANSFERS

const nativeTransfersDataset = "native_transfers"

// nativeTransfersColumns restricts cryo output to exactly the columns mapped.
// The UInt256 value is taken from value_string (decimal), parsed in the route.
var nativeTransfersColumns = []string{
	"block_number",
	"transaction_index",
	"transfer_index",
	"transaction_hash",
	"from_address",
	"to_address",
	// cryo's `value` expands (with --hex) to value_binary/value_string/value_f64;
	// nativeTransferRow reads value_string (decimal) and parquet-go projects the rest.
	"value",
}

// nativeTransferRow mirrors the cryo `native_transfers` parquet schema (--hex).
type nativeTransferRow struct {
	BlockNumber      uint32 `parquet:"block_number"`
	TransactionIndex uint32 `parquet:"transaction_index"`
	TransferIndex    uint32 `parquet:"transfer_index"`
	TransactionHash  string `parquet:"transaction_hash"`
	FromAddress      string `parquet:"from_address"`
	ToAddress        string `parquet:"to_address"`
	Value            string `parquet:"value_string"`

	InternalIndex uint32
}

// NativeTransfersDeriverConfig configures the native transfers deriver.
type NativeTransfersDeriverConfig struct {
	Enabled   bool                            `yaml:"enabled" default:"false"`
	ChunkSize int                             `yaml:"chunkSize" default:"100"`
	Iterator  iterator.BackfillingBlockConfig `yaml:"iterator"`
}

// NativeTransfersDeriver derives canonical_execution_native_transfers events via cryo.
type NativeTransfersDeriver struct {
	base

	cfg        *NativeTransfersDeriverConfig
	cryo       *cryo.Runner
	clientMeta *xatu.ClientMeta
}

// NewNativeTransfersDeriver creates a native transfers deriver.
func NewNativeTransfersDeriver(
	log observability.ContextualLogger,
	config *NativeTransfersDeriverConfig,
	iter *iterator.BackfillingBlock,
	runner *cryo.Runner,
	beacon *ethereum.BeaconNode,
	clientMeta *xatu.ClientMeta,
) *NativeTransfersDeriver {
	return &NativeTransfersDeriver{
		base: base{
			log: log.WithFields(logrus.Fields{
				"module": "cannon/event/execution/native_transfers",
				"type":   NativeTransfersDeriverName.String(),
			}),
			name:     NativeTransfersDeriverName.String(),
			beacon:   beacon,
			iterator: iter,
		},
		cfg:        config,
		cryo:       runner,
		clientMeta: clientMeta,
	}
}

// CannonType returns the deriver's cannon type.
func (b *NativeTransfersDeriver) CannonType() xatu.CannonType {
	return NativeTransfersDeriverName
}

// ActivationFork returns Phase0 so the deriver starts as soon as the beacon
// node is ready; the iterator gates EL progress on CL finality.
func (b *NativeTransfersDeriver) ActivationFork() spec.DataVersion {
	return spec.DataVersionPhase0
}

// Name returns the deriver name.
func (b *NativeTransfersDeriver) Name() string {
	return b.name
}

// Start begins the deriver loop.
func (b *NativeTransfersDeriver) Start(ctx context.Context) error {
	if !b.cfg.Enabled {
		b.log.WithContext(ctx).Info("Execution native transfers deriver disabled")

		return nil
	}

	b.log.WithContext(ctx).Info("Execution native transfers deriver enabled")

	if err := b.iterator.Start(ctx); err != nil {
		return errors.Wrap(err, "failed to start iterator")
	}

	b.run(ctx, b.processRange)

	return nil
}

// Stop is a no-op for the native transfers deriver.
func (b *NativeTransfersDeriver) Stop(_ context.Context) error {
	return nil
}

// processRange runs cryo for [from, to] and builds chunked DecoratedEvents.
func (b *NativeTransfersDeriver) processRange(ctx context.Context, from, to uint64) ([]*xatu.DecoratedEvent, error) {
	ctx, span := observability.Tracer().Start(ctx,
		"NativeTransfersDeriver.processRange",
		trace.WithAttributes(
			attribute.Int64("from", int64(from)), //nolint:gosec // block numbers are far below int64 max.
			attribute.Int64("to", int64(to)),     //nolint:gosec // block numbers are far below int64 max.
		),
	)
	defer span.End()

	collection, err := b.cryo.Collect(ctx, nativeTransfersDataset, from, to, nativeTransfersColumns)
	if err != nil {
		return nil, errors.Wrap(err, "failed to collect native transfers via cryo")
	}

	defer func() {
		if cErr := collection.Close(); cErr != nil {
			b.log.WithError(cErr).WithContext(ctx).Warn("Failed to clean up cryo output")
		}
	}()

	rows, err := cryo.ReadParquet[nativeTransferRow](collection.Files)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read cryo native transfers parquet")
	}

	stampInternalIndex(rows,
		func(r *nativeTransferRow) uint64 { return uint64(r.BlockNumber) },
		func(r *nativeTransferRow) string { return r.TransactionHash },
		func(r *nativeTransferRow, idx uint32) { r.InternalIndex = idx },
	)

	return chunkEvents(rows, b.cfg.ChunkSize, b.createEvent)
}

// createEvent builds a single DecoratedEvent carrying a chunk of native transfer rows.
func (b *NativeTransfersDeriver) createEvent(rows []nativeTransferRow) (*xatu.DecoratedEvent, error) {
	metadata, ok := proto.Clone(b.clientMeta).(*xatu.ClientMeta)
	if !ok {
		return nil, errors.New("failed to clone client metadata")
	}

	nativeTransfers := make([]*xatu.ExecutionNativeTransfer, 0, len(rows))

	for i := range rows {
		nativeTransfers = append(nativeTransfers, nativeTransferRowToProto(&rows[i]))
	}

	return &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_EXECUTION_CANONICAL_NATIVE_TRANSFERS,
			DateTime: timestamppb.New(time.Now()),
			Id:       uuid.New().String(),
		},
		Meta: &xatu.Meta{Client: metadata},
		Data: &xatu.DecoratedEvent_ExecutionCanonicalNativeTransfers{
			ExecutionCanonicalNativeTransfers: &xatu.ExecutionCanonicalNativeTransfers{NativeTransfers: nativeTransfers},
		},
	}, nil
}

// nativeTransferRowToProto converts a cryo native transfer row to proto.
func nativeTransferRowToProto(row *nativeTransferRow) *xatu.ExecutionNativeTransfer {
	return &xatu.ExecutionNativeTransfer{
		BlockNumber:      uint64(row.BlockNumber),
		TransactionIndex: uint64(row.TransactionIndex),
		TransactionHash:  row.TransactionHash,
		InternalIndex:    row.InternalIndex,
		TransferIndex:    uint64(row.TransferIndex),
		FromAddress:      row.FromAddress,
		ToAddress:        row.ToAddress,
		Value:            row.Value,
	}
}
