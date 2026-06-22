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
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/ethpandaops/xatu/pkg/cannon/ethereum"
	"github.com/ethpandaops/xatu/pkg/cannon/iterator"
	"github.com/ethpandaops/xatu/pkg/cryo"
	"github.com/ethpandaops/xatu/pkg/observability"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

// TransactionDeriverName is the cannon type produced by the transaction deriver.
const TransactionDeriverName = xatu.CannonType_EXECUTION_CANONICAL_TRANSACTION

const transactionDataset = "transactions"

// transactionColumns restricts cryo output to exactly the columns mapped. The
// UInt256 value is taken from value_string (decimal), parsed in the route.
var transactionColumns = []string{
	"block_number",
	"transaction_index",
	"transaction_hash",
	"nonce",
	"from_address",
	"to_address",
	// cryo's `value` expands (with --hex) to value_binary/value_string/value_f64;
	// transactionRow reads value_string (decimal) and parquet-go projects the rest.
	"value",
	"input",
	"gas_limit",
	"gas_used",
	"gas_price",
	"transaction_type",
	"max_priority_fee_per_gas",
	"max_fee_per_gas",
	"success",
	"n_input_bytes",
	"n_input_zero_bytes",
	"n_input_nonzero_bytes",
}

// transactionRow mirrors the cryo `transactions` parquet schema (--hex).
type transactionRow struct {
	BlockNumber          uint32 `parquet:"block_number"`
	TransactionIndex     uint64 `parquet:"transaction_index"`
	TransactionHash      string `parquet:"transaction_hash"`
	Nonce                uint64 `parquet:"nonce"`
	FromAddress          string `parquet:"from_address"`
	ToAddress            string `parquet:"to_address"`
	Value                string `parquet:"value_string"`
	Input                string `parquet:"input"`
	GasLimit             uint64 `parquet:"gas_limit"`
	GasUsed              uint64 `parquet:"gas_used"`
	GasPrice             uint64 `parquet:"gas_price"`
	TransactionType      uint32 `parquet:"transaction_type"`
	MaxPriorityFeePerGas uint64 `parquet:"max_priority_fee_per_gas"`
	MaxFeePerGas         uint64 `parquet:"max_fee_per_gas"`
	Success              bool   `parquet:"success"`
	NInputBytes          uint32 `parquet:"n_input_bytes"`
	NInputZeroBytes      uint32 `parquet:"n_input_zero_bytes"`
	NInputNonzeroBytes   uint32 `parquet:"n_input_nonzero_bytes"`
}

// TransactionDeriverConfig configures the transaction deriver.
type TransactionDeriverConfig struct {
	Enabled   bool                            `yaml:"enabled" default:"false"`
	ChunkSize int                             `yaml:"chunkSize" default:"100"`
	Iterator  iterator.BackfillingBlockConfig `yaml:"iterator"`
}

// TransactionDeriver derives canonical_execution_transaction events via cryo.
type TransactionDeriver struct {
	base

	cfg        *TransactionDeriverConfig
	cryo       *cryo.Runner
	clientMeta *xatu.ClientMeta
}

// NewTransactionDeriver creates a transaction deriver.
func NewTransactionDeriver(
	log observability.ContextualLogger,
	config *TransactionDeriverConfig,
	iter *iterator.BackfillingBlock,
	runner *cryo.Runner,
	beacon *ethereum.BeaconNode,
	clientMeta *xatu.ClientMeta,
) *TransactionDeriver {
	return &TransactionDeriver{
		base: base{
			log: log.WithFields(logrus.Fields{
				"module": "cannon/event/execution/transaction",
				"type":   TransactionDeriverName.String(),
			}),
			name:     TransactionDeriverName.String(),
			beacon:   beacon,
			iterator: iter,
		},
		cfg:        config,
		cryo:       runner,
		clientMeta: clientMeta,
	}
}

// CannonType returns the deriver's cannon type.
func (b *TransactionDeriver) CannonType() xatu.CannonType {
	return TransactionDeriverName
}

// ActivationFork returns Phase0 so the deriver starts as soon as the beacon
// node is ready; the iterator gates EL progress on CL finality.
func (b *TransactionDeriver) ActivationFork() spec.DataVersion {
	return spec.DataVersionPhase0
}

// Name returns the deriver name.
func (b *TransactionDeriver) Name() string {
	return b.name
}

// Start begins the deriver loop.
func (b *TransactionDeriver) Start(ctx context.Context) error {
	if !b.cfg.Enabled {
		b.log.WithContext(ctx).Info("Execution transaction deriver disabled")

		return nil
	}

	b.log.WithContext(ctx).Info("Execution transaction deriver enabled")

	if err := b.iterator.Start(ctx); err != nil {
		return errors.Wrap(err, "failed to start iterator")
	}

	b.run(ctx, b.processRange)

	return nil
}

// Stop is a no-op for the transaction deriver.
func (b *TransactionDeriver) Stop(_ context.Context) error {
	return nil
}

// processRange runs cryo for [from, to] and builds chunked DecoratedEvents.
func (b *TransactionDeriver) processRange(ctx context.Context, from, to uint64) ([]*xatu.DecoratedEvent, error) {
	ctx, span := observability.Tracer().Start(ctx,
		"TransactionDeriver.processRange",
		trace.WithAttributes(
			attribute.Int64("from", int64(from)), //nolint:gosec // block numbers are far below int64 max.
			attribute.Int64("to", int64(to)),     //nolint:gosec // block numbers are far below int64 max.
		),
	)
	defer span.End()

	collection, err := b.cryo.Collect(ctx, transactionDataset, from, to, transactionColumns)
	if err != nil {
		return nil, errors.Wrap(err, "failed to collect transactions via cryo")
	}

	defer func() {
		if cErr := collection.Close(); cErr != nil {
			b.log.WithError(cErr).WithContext(ctx).Warn("Failed to clean up cryo output")
		}
	}()

	rows, err := cryo.ReadParquet[transactionRow](collection.Files)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read cryo transaction parquet")
	}

	return chunkEvents(rows, b.cfg.ChunkSize, b.createEvent)
}

// createEvent builds a single DecoratedEvent carrying a chunk of transaction rows.
func (b *TransactionDeriver) createEvent(rows []transactionRow) (*xatu.DecoratedEvent, error) {
	metadata, ok := proto.Clone(b.clientMeta).(*xatu.ClientMeta)
	if !ok {
		return nil, errors.New("failed to clone client metadata")
	}

	transactions := make([]*xatu.ExecutionTransaction, 0, len(rows))

	for i := range rows {
		transactions = append(transactions, transactionRowToProto(&rows[i]))
	}

	return &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_EXECUTION_CANONICAL_TRANSACTION,
			DateTime: timestamppb.New(time.Now()),
			Id:       uuid.New().String(),
		},
		Meta: &xatu.Meta{Client: metadata},
		Data: &xatu.DecoratedEvent_ExecutionCanonicalTransaction{
			ExecutionCanonicalTransaction: &xatu.ExecutionCanonicalTransaction{Transactions: transactions},
		},
	}, nil
}

// transactionRowToProto converts a cryo transaction row to proto.
func transactionRowToProto(row *transactionRow) *xatu.ExecutionTransaction {
	out := &xatu.ExecutionTransaction{
		BlockNumber:          uint64(row.BlockNumber),
		TransactionIndex:     row.TransactionIndex,
		TransactionHash:      row.TransactionHash,
		Nonce:                row.Nonce,
		FromAddress:          row.FromAddress,
		Value:                row.Value,
		GasLimit:             row.GasLimit,
		GasUsed:              row.GasUsed,
		GasPrice:             row.GasPrice,
		TransactionType:      row.TransactionType,
		MaxPriorityFeePerGas: row.MaxPriorityFeePerGas,
		MaxFeePerGas:         row.MaxFeePerGas,
		Success:              row.Success,
		NInputBytes:          row.NInputBytes,
		NInputZeroBytes:      row.NInputZeroBytes,
		NInputNonzeroBytes:   row.NInputNonzeroBytes,
	}

	if row.ToAddress != "" {
		out.ToAddress = wrapperspb.String(row.ToAddress)
	}

	if row.Input != "" {
		out.Input = wrapperspb.String(row.Input)
	}

	return out
}
