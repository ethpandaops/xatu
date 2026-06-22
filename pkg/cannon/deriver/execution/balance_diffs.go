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

// BalanceDiffsDeriverName is the cannon type produced by the balance diffs deriver.
const BalanceDiffsDeriverName = xatu.CannonType_EXECUTION_CANONICAL_BALANCE_DIFFS

const balanceDiffsDataset = "balance_diffs"

// balanceDiffsColumns restricts cryo output to exactly the columns mapped. The
// UInt256 values are taken from from_value_string/to_value_string (decimal),
// parsed in the route.
var balanceDiffsColumns = []string{
	"block_number",
	"transaction_index",
	"transaction_hash",
	"address",
	// cryo's `from_value`/`to_value` expand (with --hex) to *_binary/*_string/*_f64;
	// balanceDiffRow reads from_value_string/to_value_string (decimal).
	"from_value",
	"to_value",
}

// balanceDiffRow mirrors the cryo `balance_diffs` parquet schema (--hex).
type balanceDiffRow struct {
	BlockNumber      uint32 `parquet:"block_number"`
	TransactionIndex uint32 `parquet:"transaction_index"`
	TransactionHash  string `parquet:"transaction_hash"`
	Address          string `parquet:"address"`
	FromValue        string `parquet:"from_value_string"`
	ToValue          string `parquet:"to_value_string"`

	InternalIndex uint32
}

// BalanceDiffsDeriverConfig configures the balance diffs deriver.
type BalanceDiffsDeriverConfig struct {
	Enabled   bool                            `yaml:"enabled" default:"false"`
	ChunkSize int                             `yaml:"chunkSize" default:"100"`
	Iterator  iterator.BackfillingBlockConfig `yaml:"iterator"`
}

// BalanceDiffsDeriver derives canonical_execution_balance_diffs events via cryo.
type BalanceDiffsDeriver struct {
	base

	cfg        *BalanceDiffsDeriverConfig
	cryo       *cryo.Runner
	clientMeta *xatu.ClientMeta
}

// NewBalanceDiffsDeriver creates a balance diffs deriver.
func NewBalanceDiffsDeriver(
	log observability.ContextualLogger,
	config *BalanceDiffsDeriverConfig,
	iter *iterator.BackfillingBlock,
	runner *cryo.Runner,
	beacon *ethereum.BeaconNode,
	clientMeta *xatu.ClientMeta,
) *BalanceDiffsDeriver {
	return &BalanceDiffsDeriver{
		base: base{
			log: log.WithFields(logrus.Fields{
				"module": "cannon/event/execution/balance_diffs",
				"type":   BalanceDiffsDeriverName.String(),
			}),
			name:     BalanceDiffsDeriverName.String(),
			beacon:   beacon,
			iterator: iter,
		},
		cfg:        config,
		cryo:       runner,
		clientMeta: clientMeta,
	}
}

// CannonType returns the deriver's cannon type.
func (b *BalanceDiffsDeriver) CannonType() xatu.CannonType {
	return BalanceDiffsDeriverName
}

// ActivationFork returns Phase0 so the deriver starts as soon as the beacon
// node is ready; the iterator gates EL progress on CL finality.
func (b *BalanceDiffsDeriver) ActivationFork() spec.DataVersion {
	return spec.DataVersionPhase0
}

// Name returns the deriver name.
func (b *BalanceDiffsDeriver) Name() string {
	return b.name
}

// Start begins the deriver loop.
func (b *BalanceDiffsDeriver) Start(ctx context.Context) error {
	if !b.cfg.Enabled {
		b.log.WithContext(ctx).Info("Execution balance diffs deriver disabled")

		return nil
	}

	b.log.WithContext(ctx).Info("Execution balance diffs deriver enabled")

	if err := b.iterator.Start(ctx); err != nil {
		return errors.Wrap(err, "failed to start iterator")
	}

	b.run(ctx, b.processRange)

	return nil
}

// Stop is a no-op for the balance diffs deriver.
func (b *BalanceDiffsDeriver) Stop(_ context.Context) error {
	return nil
}

// processRange runs cryo for [from, to] and builds chunked DecoratedEvents.
func (b *BalanceDiffsDeriver) processRange(ctx context.Context, from, to uint64) ([]*xatu.DecoratedEvent, error) {
	ctx, span := observability.Tracer().Start(ctx,
		"BalanceDiffsDeriver.processRange",
		trace.WithAttributes(
			attribute.Int64("from", int64(from)), //nolint:gosec // block numbers are far below int64 max.
			attribute.Int64("to", int64(to)),     //nolint:gosec // block numbers are far below int64 max.
		),
	)
	defer span.End()

	collection, err := b.cryo.Collect(ctx, balanceDiffsDataset, from, to, balanceDiffsColumns)
	if err != nil {
		return nil, errors.Wrap(err, "failed to collect balance diffs via cryo")
	}

	defer func() {
		if cErr := collection.Close(); cErr != nil {
			b.log.WithError(cErr).WithContext(ctx).Warn("Failed to clean up cryo output")
		}
	}()

	rows, err := cryo.ReadParquet[balanceDiffRow](collection.Files)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read cryo balance diffs parquet")
	}

	stampInternalIndex(rows,
		func(r *balanceDiffRow) uint64 { return uint64(r.BlockNumber) },
		func(r *balanceDiffRow) string { return r.TransactionHash },
		func(r *balanceDiffRow, idx uint32) { r.InternalIndex = idx },
	)

	return chunkEvents(rows, b.cfg.ChunkSize, b.createEvent)
}

// createEvent builds a single DecoratedEvent carrying a chunk of balance diff rows.
func (b *BalanceDiffsDeriver) createEvent(rows []balanceDiffRow) (*xatu.DecoratedEvent, error) {
	metadata, ok := proto.Clone(b.clientMeta).(*xatu.ClientMeta)
	if !ok {
		return nil, errors.New("failed to clone client metadata")
	}

	balanceDiffs := make([]*xatu.ExecutionBalanceDiff, 0, len(rows))

	for i := range rows {
		balanceDiffs = append(balanceDiffs, balanceDiffRowToProto(&rows[i]))
	}

	return &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_EXECUTION_CANONICAL_BALANCE_DIFFS,
			DateTime: timestamppb.New(time.Now()),
			Id:       uuid.New().String(),
		},
		Meta: &xatu.Meta{Client: metadata},
		Data: &xatu.DecoratedEvent_ExecutionCanonicalBalanceDiffs{
			ExecutionCanonicalBalanceDiffs: &xatu.ExecutionCanonicalBalanceDiffs{BalanceDiffs: balanceDiffs},
		},
	}, nil
}

// balanceDiffRowToProto converts a cryo balance diff row to proto.
func balanceDiffRowToProto(row *balanceDiffRow) *xatu.ExecutionBalanceDiff {
	return &xatu.ExecutionBalanceDiff{
		BlockNumber:      uint64(row.BlockNumber),
		TransactionIndex: uint64(row.TransactionIndex),
		TransactionHash:  row.TransactionHash,
		InternalIndex:    row.InternalIndex,
		Address:          row.Address,
		FromValue:        row.FromValue,
		ToValue:          row.ToValue,
	}
}
