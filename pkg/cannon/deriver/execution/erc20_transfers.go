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

// Erc20TransfersDeriverName is the cannon type produced by the erc20_transfers deriver.
const Erc20TransfersDeriverName = xatu.CannonType_EXECUTION_CANONICAL_ERC20_TRANSFERS

const erc20TransfersDataset = "erc20_transfers"

// erc20TransfersColumns restricts cryo output to exactly the columns mapped. The
// UInt256 value is taken from value_string (decimal), parsed in the route.
var erc20TransfersColumns = []string{
	"block_number",
	"transaction_index",
	"log_index",
	"transaction_hash",
	"erc20",
	"from_address",
	"to_address",
	// cryo's `value` expands (with --hex) to value_binary/value_string/value_f64;
	// erc20TransferRow reads value_string (decimal) and parquet-go projects the rest.
	"value",
}

// erc20TransferRow mirrors the cryo `erc20_transfers` parquet schema (--hex).
type erc20TransferRow struct {
	BlockNumber      uint32 `parquet:"block_number"`
	TransactionIndex uint32 `parquet:"transaction_index"`
	LogIndex         uint32 `parquet:"log_index"`
	TransactionHash  string `parquet:"transaction_hash"`
	Erc20            string `parquet:"erc20"`
	FromAddress      string `parquet:"from_address"`
	ToAddress        string `parquet:"to_address"`
	Value            string `parquet:"value_string"`

	// InternalIndex is stamped post-read (no cryo column); see stampInternalIndex.
	InternalIndex uint32
}

// Erc20TransfersDeriverConfig configures the erc20_transfers deriver.
type Erc20TransfersDeriverConfig struct {
	Enabled   bool                            `yaml:"enabled" default:"false"`
	ChunkSize int                             `yaml:"chunkSize" default:"100"`
	Iterator  iterator.BackfillingBlockConfig `yaml:"iterator"`
}

// Erc20TransfersDeriver derives canonical_execution_erc20_transfers events via cryo.
type Erc20TransfersDeriver struct {
	base

	cfg        *Erc20TransfersDeriverConfig
	cryo       *cryo.Runner
	clientMeta *xatu.ClientMeta
}

// NewErc20TransfersDeriver creates an erc20_transfers deriver.
func NewErc20TransfersDeriver(
	log observability.ContextualLogger,
	config *Erc20TransfersDeriverConfig,
	iter *iterator.BackfillingBlock,
	runner *cryo.Runner,
	beacon *ethereum.BeaconNode,
	clientMeta *xatu.ClientMeta,
) *Erc20TransfersDeriver {
	return &Erc20TransfersDeriver{
		base: base{
			log: log.WithFields(logrus.Fields{
				"module": "cannon/event/execution/erc20_transfers",
				"type":   Erc20TransfersDeriverName.String(),
			}),
			name:     Erc20TransfersDeriverName.String(),
			beacon:   beacon,
			iterator: iter,
		},
		cfg:        config,
		cryo:       runner,
		clientMeta: clientMeta,
	}
}

// CannonType returns the deriver's cannon type.
func (b *Erc20TransfersDeriver) CannonType() xatu.CannonType {
	return Erc20TransfersDeriverName
}

// ActivationFork returns Phase0 so the deriver starts as soon as the beacon
// node is ready; the iterator gates EL progress on CL finality.
func (b *Erc20TransfersDeriver) ActivationFork() spec.DataVersion {
	return spec.DataVersionPhase0
}

// Name returns the deriver name.
func (b *Erc20TransfersDeriver) Name() string {
	return b.name
}

// Start begins the deriver loop.
func (b *Erc20TransfersDeriver) Start(ctx context.Context) error {
	if !b.cfg.Enabled {
		b.log.WithContext(ctx).Info("Execution erc20_transfers deriver disabled")

		return nil
	}

	b.log.WithContext(ctx).Info("Execution erc20_transfers deriver enabled")

	if err := b.iterator.Start(ctx); err != nil {
		return errors.Wrap(err, "failed to start iterator")
	}

	b.run(ctx, b.processRange)

	return nil
}

// Stop is a no-op for the erc20_transfers deriver.
func (b *Erc20TransfersDeriver) Stop(_ context.Context) error {
	return nil
}

// processRange runs cryo for [from, to] and builds chunked DecoratedEvents.
func (b *Erc20TransfersDeriver) processRange(ctx context.Context, from, to uint64) ([]*xatu.DecoratedEvent, error) {
	ctx, span := observability.Tracer().Start(ctx,
		"Erc20TransfersDeriver.processRange",
		trace.WithAttributes(
			attribute.Int64("from", int64(from)), //nolint:gosec // block numbers are far below int64 max.
			attribute.Int64("to", int64(to)),     //nolint:gosec // block numbers are far below int64 max.
		),
	)
	defer span.End()

	collection, err := b.cryo.Collect(ctx, erc20TransfersDataset, from, to, erc20TransfersColumns)
	if err != nil {
		return nil, errors.Wrap(err, "failed to collect erc20_transfers via cryo")
	}

	defer func() {
		if cErr := collection.Close(); cErr != nil {
			b.log.WithError(cErr).WithContext(ctx).Warn("Failed to clean up cryo output")
		}
	}()

	rows, err := cryo.ReadParquet[erc20TransferRow](collection.Files)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read cryo erc20_transfers parquet")
	}

	stampInternalIndex(rows,
		func(r *erc20TransferRow) uint64 { return uint64(r.BlockNumber) },
		func(r *erc20TransferRow) string { return r.TransactionHash },
		func(r *erc20TransferRow, idx uint32) { r.InternalIndex = idx },
	)

	return chunkEvents(rows, b.cfg.ChunkSize, b.createEvent)
}

// createEvent builds a single DecoratedEvent carrying a chunk of erc20_transfer rows.
func (b *Erc20TransfersDeriver) createEvent(rows []erc20TransferRow) (*xatu.DecoratedEvent, error) {
	metadata, ok := proto.Clone(b.clientMeta).(*xatu.ClientMeta)
	if !ok {
		return nil, errors.New("failed to clone client metadata")
	}

	transfers := make([]*xatu.ExecutionErc20Transfer, 0, len(rows))

	for i := range rows {
		transfers = append(transfers, erc20TransferRowToProto(&rows[i]))
	}

	return &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_EXECUTION_CANONICAL_ERC20_TRANSFERS,
			DateTime: timestamppb.New(time.Now()),
			Id:       uuid.New().String(),
		},
		Meta: &xatu.Meta{Client: metadata},
		Data: &xatu.DecoratedEvent_ExecutionCanonicalErc20Transfers{
			ExecutionCanonicalErc20Transfers: &xatu.ExecutionCanonicalErc20Transfers{Erc20Transfers: transfers},
		},
	}, nil
}

// erc20TransferRowToProto converts a cryo erc20_transfer row to proto.
func erc20TransferRowToProto(row *erc20TransferRow) *xatu.ExecutionErc20Transfer {
	return &xatu.ExecutionErc20Transfer{
		BlockNumber:      uint64(row.BlockNumber),
		TransactionIndex: uint64(row.TransactionIndex),
		TransactionHash:  row.TransactionHash,
		InternalIndex:    row.InternalIndex,
		LogIndex:         uint64(row.LogIndex),
		Erc20:            row.Erc20,
		FromAddress:      row.FromAddress,
		ToAddress:        row.ToAddress,
		Value:            row.Value,
	}
}
