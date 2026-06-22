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
	"github.com/ethpandaops/xatu/pkg/cannon/execution/cryo"
	"github.com/ethpandaops/xatu/pkg/cannon/iterator"
	"github.com/ethpandaops/xatu/pkg/observability"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

// BalanceReadsDeriverName is the cannon type produced by the balance reads deriver.
const BalanceReadsDeriverName = xatu.CannonType_EXECUTION_CANONICAL_BALANCE_READS

const balanceReadsDataset = "balance_reads"

// balanceReadsColumns restricts cryo output to exactly the columns mapped. The
// UInt256 balance is taken from balance_string (decimal), parsed in the route.
var balanceReadsColumns = []string{
	"block_number",
	"transaction_index",
	"transaction_hash",
	"address",
	// cryo's `balance` expands (with --hex) to balance_binary/balance_string/balance_f64;
	// balanceReadRow reads balance_string (decimal) and parquet-go projects the rest.
	"balance",
}

// balanceReadRow mirrors the cryo `balance_reads` parquet schema (--hex).
type balanceReadRow struct {
	BlockNumber      uint32 `parquet:"block_number"`
	TransactionIndex uint32 `parquet:"transaction_index"`
	TransactionHash  string `parquet:"transaction_hash"`
	Address          string `parquet:"address"`
	Balance          string `parquet:"balance_string"`

	InternalIndex uint32
}

// BalanceReadsDeriverConfig configures the balance reads deriver.
type BalanceReadsDeriverConfig struct {
	Enabled   bool                            `yaml:"enabled" default:"false"`
	ChunkSize int                             `yaml:"chunkSize" default:"100"`
	Iterator  iterator.BackfillingBlockConfig `yaml:"iterator"`
}

// BalanceReadsDeriver derives canonical_execution_balance_reads events via cryo.
type BalanceReadsDeriver struct {
	base

	cfg        *BalanceReadsDeriverConfig
	cryo       *cryo.Runner
	clientMeta *xatu.ClientMeta
}

// NewBalanceReadsDeriver creates a balance reads deriver.
func NewBalanceReadsDeriver(
	log observability.ContextualLogger,
	config *BalanceReadsDeriverConfig,
	iter *iterator.BackfillingBlock,
	runner *cryo.Runner,
	beacon *ethereum.BeaconNode,
	clientMeta *xatu.ClientMeta,
) *BalanceReadsDeriver {
	return &BalanceReadsDeriver{
		base: base{
			log: log.WithFields(logrus.Fields{
				"module": "cannon/event/execution/balance_reads",
				"type":   BalanceReadsDeriverName.String(),
			}),
			name:     BalanceReadsDeriverName.String(),
			beacon:   beacon,
			iterator: iter,
		},
		cfg:        config,
		cryo:       runner,
		clientMeta: clientMeta,
	}
}

// CannonType returns the deriver's cannon type.
func (b *BalanceReadsDeriver) CannonType() xatu.CannonType {
	return BalanceReadsDeriverName
}

// ActivationFork returns Phase0 so the deriver starts as soon as the beacon
// node is ready; the iterator gates EL progress on CL finality.
func (b *BalanceReadsDeriver) ActivationFork() spec.DataVersion {
	return spec.DataVersionPhase0
}

// Name returns the deriver name.
func (b *BalanceReadsDeriver) Name() string {
	return b.name
}

// Start begins the deriver loop.
func (b *BalanceReadsDeriver) Start(ctx context.Context) error {
	if !b.cfg.Enabled {
		b.log.WithContext(ctx).Info("Execution balance reads deriver disabled")

		return nil
	}

	b.log.WithContext(ctx).Info("Execution balance reads deriver enabled")

	if err := b.iterator.Start(ctx); err != nil {
		return errors.Wrap(err, "failed to start iterator")
	}

	b.run(ctx, b.processRange)

	return nil
}

// Stop is a no-op for the balance reads deriver.
func (b *BalanceReadsDeriver) Stop(_ context.Context) error {
	return nil
}

// processRange runs cryo for [from, to] and builds chunked DecoratedEvents.
func (b *BalanceReadsDeriver) processRange(ctx context.Context, from, to uint64) ([]*xatu.DecoratedEvent, error) {
	ctx, span := observability.Tracer().Start(ctx,
		"BalanceReadsDeriver.processRange",
		trace.WithAttributes(
			attribute.Int64("from", int64(from)), //nolint:gosec // block numbers are far below int64 max.
			attribute.Int64("to", int64(to)),     //nolint:gosec // block numbers are far below int64 max.
		),
	)
	defer span.End()

	collection, err := b.cryo.Collect(ctx, balanceReadsDataset, from, to, balanceReadsColumns)
	if err != nil {
		return nil, errors.Wrap(err, "failed to collect balance reads via cryo")
	}

	defer func() {
		if cErr := collection.Close(); cErr != nil {
			b.log.WithError(cErr).WithContext(ctx).Warn("Failed to clean up cryo output")
		}
	}()

	rows, err := cryo.ReadParquet[balanceReadRow](collection.Files)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read cryo balance reads parquet")
	}

	stampInternalIndex(rows,
		func(r *balanceReadRow) uint64 { return uint64(r.BlockNumber) },
		func(r *balanceReadRow) string { return r.TransactionHash },
		func(r *balanceReadRow, idx uint32) { r.InternalIndex = idx },
	)

	return chunkEvents(rows, b.cfg.ChunkSize, b.createEvent)
}

// createEvent builds a single DecoratedEvent carrying a chunk of balance read rows.
func (b *BalanceReadsDeriver) createEvent(rows []balanceReadRow) (*xatu.DecoratedEvent, error) {
	metadata, ok := proto.Clone(b.clientMeta).(*xatu.ClientMeta)
	if !ok {
		return nil, errors.New("failed to clone client metadata")
	}

	balanceReads := make([]*xatu.ExecutionBalanceRead, 0, len(rows))

	for i := range rows {
		balanceReads = append(balanceReads, balanceReadRowToProto(&rows[i]))
	}

	return &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_EXECUTION_CANONICAL_BALANCE_READS,
			DateTime: timestamppb.New(time.Now()),
			Id:       uuid.New().String(),
		},
		Meta: &xatu.Meta{Client: metadata},
		Data: &xatu.DecoratedEvent_ExecutionCanonicalBalanceReads{
			ExecutionCanonicalBalanceReads: &xatu.ExecutionCanonicalBalanceReads{BalanceReads: balanceReads},
		},
	}, nil
}

// balanceReadRowToProto converts a cryo balance read row to proto.
func balanceReadRowToProto(row *balanceReadRow) *xatu.ExecutionBalanceRead {
	return &xatu.ExecutionBalanceRead{
		BlockNumber:      uint64(row.BlockNumber),
		TransactionIndex: uint64(row.TransactionIndex),
		TransactionHash:  row.TransactionHash,
		InternalIndex:    row.InternalIndex,
		Address:          row.Address,
		Balance:          row.Balance,
	}
}
