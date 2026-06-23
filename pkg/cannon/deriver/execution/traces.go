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

// TracesDeriverName is the cannon type produced by the traces deriver.
const TracesDeriverName = xatu.CannonType_EXECUTION_CANONICAL_TRACES

const tracesDataset = "traces"

// tracesColumns restricts cryo output to exactly the columns mapped. The
// UInt256 action_value is a single decimal `action_value` column (not split
// like the blocks/transactions value), parsed in the route.
var tracesColumns = []string{
	"block_number",
	"transaction_index",
	"transaction_hash",
	"action_from",
	"action_to",
	"action_value",
	"action_gas",
	"action_input",
	"action_call_type",
	"action_init",
	"action_reward_type",
	"action_type",
	"result_gas_used",
	"result_output",
	"result_code",
	"result_address",
	"trace_address",
	"subtraces",
	"error",
}

// tracesRow mirrors the cryo `traces` parquet schema (--hex).
type tracesRow struct {
	BlockNumber      uint32 `parquet:"block_number"`
	TransactionIndex uint32 `parquet:"transaction_index"`
	TransactionHash  string `parquet:"transaction_hash"`
	ActionFrom       string `parquet:"action_from"`
	ActionTo         string `parquet:"action_to"`
	ActionValue      string `parquet:"action_value"`
	ActionGas        uint32 `parquet:"action_gas"`
	ActionInput      string `parquet:"action_input"`
	ActionCallType   string `parquet:"action_call_type"`
	ActionInit       string `parquet:"action_init"`
	ActionRewardType string `parquet:"action_reward_type"`
	ActionType       string `parquet:"action_type"`
	ResultGasUsed    uint32 `parquet:"result_gas_used"`
	ResultOutput     string `parquet:"result_output"`
	ResultCode       string `parquet:"result_code"`
	ResultAddress    string `parquet:"result_address"`
	TraceAddress     string `parquet:"trace_address"`
	Subtraces        uint32 `parquet:"subtraces"`
	Error            string `parquet:"error"`

	// InternalIndex is stamped after read via stampInternalIndex; it has no
	// cryo parquet column.
	InternalIndex uint32
}

// TracesDeriverConfig configures the traces deriver.
type TracesDeriverConfig struct {
	Enabled   bool                            `yaml:"enabled" default:"false"`
	ChunkSize int                             `yaml:"chunkSize" default:"100"`
	Iterator  iterator.BackfillingBlockConfig `yaml:"iterator"`
}

// TracesDeriver derives canonical_execution_traces events via cryo.
type TracesDeriver struct {
	base

	cfg        *TracesDeriverConfig
	cryo       *cryo.Runner
	clientMeta *xatu.ClientMeta
}

// NewTracesDeriver creates a traces deriver.
func NewTracesDeriver(
	log observability.ContextualLogger,
	config *TracesDeriverConfig,
	iter *iterator.BackfillingBlock,
	runner *cryo.Runner,
	beacon *ethereum.BeaconNode,
	clientMeta *xatu.ClientMeta,
) *TracesDeriver {
	return &TracesDeriver{
		base: base{
			log: log.WithFields(logrus.Fields{
				"module": "cannon/event/execution/traces",
				"type":   TracesDeriverName.String(),
			}),
			name:     TracesDeriverName.String(),
			beacon:   beacon,
			iterator: iter,
		},
		cfg:        config,
		cryo:       runner,
		clientMeta: clientMeta,
	}
}

// CannonType returns the deriver's cannon type.
func (b *TracesDeriver) CannonType() xatu.CannonType {
	return TracesDeriverName
}

// ActivationFork returns Phase0 so the deriver starts as soon as the beacon
// node is ready; the iterator gates EL progress on CL finality.
func (b *TracesDeriver) ActivationFork() spec.DataVersion {
	return spec.DataVersionPhase0
}

// Name returns the deriver name.
func (b *TracesDeriver) Name() string {
	return b.name
}

// Start begins the deriver loop.
func (b *TracesDeriver) Start(ctx context.Context) error {
	if !b.cfg.Enabled {
		b.log.WithContext(ctx).Info("Execution traces deriver disabled")

		return nil
	}

	b.log.WithContext(ctx).Info("Execution traces deriver enabled")

	if err := b.iterator.Start(ctx); err != nil {
		return errors.Wrap(err, "failed to start iterator")
	}

	b.run(ctx, b.processRange)

	return nil
}

// Stop is a no-op for the traces deriver.
func (b *TracesDeriver) Stop(_ context.Context) error {
	return nil
}

// processRange runs cryo for [from, to] and builds chunked DecoratedEvents.
func (b *TracesDeriver) processRange(ctx context.Context, from, to uint64) ([]*xatu.DecoratedEvent, error) {
	ctx, span := observability.Tracer().Start(ctx,
		"TracesDeriver.processRange",
		trace.WithAttributes(
			attribute.Int64("from", int64(from)), //nolint:gosec // block numbers are far below int64 max.
			attribute.Int64("to", int64(to)),     //nolint:gosec // block numbers are far below int64 max.
		),
	)
	defer span.End()

	collection, err := b.cryo.Collect(ctx, tracesDataset, from, to, tracesColumns)
	if err != nil {
		return nil, errors.Wrap(err, "failed to collect traces via cryo")
	}

	defer func() {
		if cErr := collection.Close(); cErr != nil {
			b.log.WithError(cErr).WithContext(ctx).Warn("Failed to clean up cryo output")
		}
	}()

	rows, err := cryo.ReadParquet[tracesRow](collection.Files)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read cryo traces parquet")
	}

	stampInternalIndex(rows,
		func(r *tracesRow) uint64 { return uint64(r.BlockNumber) },
		func(r *tracesRow) string { return r.TransactionHash },
		func(r *tracesRow, idx uint32) { r.InternalIndex = idx },
	)

	return chunkEvents(rows, b.cfg.ChunkSize, b.createEvent)
}

// createEvent builds a single DecoratedEvent carrying a chunk of trace rows.
func (b *TracesDeriver) createEvent(rows []tracesRow) (*xatu.DecoratedEvent, error) {
	metadata, ok := proto.Clone(b.clientMeta).(*xatu.ClientMeta)
	if !ok {
		return nil, errors.New("failed to clone client metadata")
	}

	traces := make([]*xatu.ExecutionTrace, 0, len(rows))

	for i := range rows {
		traces = append(traces, tracesRowToProto(&rows[i]))
	}

	return &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_EXECUTION_CANONICAL_TRACES,
			DateTime: timestamppb.New(time.Now()),
			Id:       uuid.New().String(),
		},
		Meta: &xatu.Meta{Client: metadata},
		Data: &xatu.DecoratedEvent_ExecutionCanonicalTraces{
			ExecutionCanonicalTraces: &xatu.ExecutionCanonicalTraces{Traces: traces},
		},
	}, nil
}

// tracesRowToProto converts a cryo traces row (collected with --hex) to proto.
func tracesRowToProto(row *tracesRow) *xatu.ExecutionTrace {
	out := &xatu.ExecutionTrace{
		BlockNumber:      uint64(row.BlockNumber),
		TransactionIndex: uint64(row.TransactionIndex),
		TransactionHash:  row.TransactionHash,
		InternalIndex:    row.InternalIndex,
		ActionFrom:       row.ActionFrom,
		ActionValue:      row.ActionValue,
		ActionGas:        uint64(row.ActionGas),
		ActionCallType:   row.ActionCallType,
		ActionRewardType: row.ActionRewardType,
		ActionType:       row.ActionType,
		ResultGasUsed:    uint64(row.ResultGasUsed),
		Subtraces:        row.Subtraces,
	}

	if row.ActionTo != "" {
		out.ActionTo = wrapperspb.String(row.ActionTo)
	}

	if row.ActionInput != "" {
		out.ActionInput = wrapperspb.String(row.ActionInput)
	}

	if row.ActionInit != "" {
		out.ActionInit = wrapperspb.String(row.ActionInit)
	}

	if row.ResultOutput != "" {
		out.ResultOutput = wrapperspb.String(row.ResultOutput)
	}

	if row.ResultCode != "" {
		out.ResultCode = wrapperspb.String(row.ResultCode)
	}

	if row.ResultAddress != "" {
		out.ResultAddress = wrapperspb.String(row.ResultAddress)
	}

	if row.TraceAddress != "" {
		out.TraceAddress = wrapperspb.String(row.TraceAddress)
	}

	if row.Error != "" {
		out.Error = wrapperspb.String(row.Error)
	}

	return out
}
