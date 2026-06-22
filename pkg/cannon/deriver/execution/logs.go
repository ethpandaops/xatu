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

// LogsDeriverName is the cannon type produced by the logs deriver.
const LogsDeriverName = xatu.CannonType_EXECUTION_CANONICAL_LOGS

const logsDataset = "logs"

// logsColumns restricts cryo output to exactly the columns the deriver maps.
var logsColumns = []string{
	"block_number",
	"transaction_index",
	"log_index",
	"transaction_hash",
	"address",
	"topic0",
	"topic1",
	"topic2",
	"topic3",
	"data",
}

// logRow mirrors the cryo `logs` parquet schema collected with --hex.
type logRow struct {
	BlockNumber      uint32 `parquet:"block_number"`
	TransactionIndex uint32 `parquet:"transaction_index"`
	LogIndex         uint32 `parquet:"log_index"`
	TransactionHash  string `parquet:"transaction_hash"`
	Address          string `parquet:"address"`
	Topic0           string `parquet:"topic0"`
	Topic1           string `parquet:"topic1"`
	Topic2           string `parquet:"topic2"`
	Topic3           string `parquet:"topic3"`
	Data             string `parquet:"data"`

	// InternalIndex is assigned by stampInternalIndex (no cryo column).
	InternalIndex uint32
}

// LogsDeriverConfig configures the logs deriver.
type LogsDeriverConfig struct {
	Enabled   bool                            `yaml:"enabled" default:"false"`
	ChunkSize int                             `yaml:"chunkSize" default:"100"`
	Iterator  iterator.BackfillingBlockConfig `yaml:"iterator"`
}

// LogsDeriver derives canonical_execution_logs events via cryo.
type LogsDeriver struct {
	base

	cfg        *LogsDeriverConfig
	cryo       *cryo.Runner
	clientMeta *xatu.ClientMeta
}

// NewLogsDeriver creates a logs deriver.
func NewLogsDeriver(
	log observability.ContextualLogger,
	config *LogsDeriverConfig,
	iter *iterator.BackfillingBlock,
	runner *cryo.Runner,
	beacon *ethereum.BeaconNode,
	clientMeta *xatu.ClientMeta,
) *LogsDeriver {
	return &LogsDeriver{
		base: base{
			log: log.WithFields(logrus.Fields{
				"module": "cannon/event/execution/logs",
				"type":   LogsDeriverName.String(),
			}),
			name:     LogsDeriverName.String(),
			beacon:   beacon,
			iterator: iter,
		},
		cfg:        config,
		cryo:       runner,
		clientMeta: clientMeta,
	}
}

// CannonType returns the deriver's cannon type.
func (b *LogsDeriver) CannonType() xatu.CannonType {
	return LogsDeriverName
}

// ActivationFork returns Phase0 so the deriver starts as soon as the beacon
// node is ready; the iterator gates EL progress on CL finality.
func (b *LogsDeriver) ActivationFork() spec.DataVersion {
	return spec.DataVersionPhase0
}

// Name returns the deriver name.
func (b *LogsDeriver) Name() string {
	return b.name
}

// Start begins the deriver loop.
func (b *LogsDeriver) Start(ctx context.Context) error {
	if !b.cfg.Enabled {
		b.log.WithContext(ctx).Info("Execution logs deriver disabled")

		return nil
	}

	b.log.WithContext(ctx).Info("Execution logs deriver enabled")

	if err := b.iterator.Start(ctx); err != nil {
		return errors.Wrap(err, "failed to start iterator")
	}

	b.run(ctx, b.processRange)

	return nil
}

// Stop is a no-op for the logs deriver.
func (b *LogsDeriver) Stop(_ context.Context) error {
	return nil
}

// processRange runs cryo for [from, to] and builds chunked DecoratedEvents.
func (b *LogsDeriver) processRange(ctx context.Context, from, to uint64) ([]*xatu.DecoratedEvent, error) {
	ctx, span := observability.Tracer().Start(ctx,
		"LogsDeriver.processRange",
		trace.WithAttributes(
			attribute.Int64("from", int64(from)), //nolint:gosec // block numbers are far below int64 max.
			attribute.Int64("to", int64(to)),     //nolint:gosec // block numbers are far below int64 max.
		),
	)
	defer span.End()

	collection, err := b.cryo.Collect(ctx, logsDataset, from, to, logsColumns)
	if err != nil {
		return nil, errors.Wrap(err, "failed to collect logs via cryo")
	}

	defer func() {
		if cErr := collection.Close(); cErr != nil {
			b.log.WithError(cErr).WithContext(ctx).Warn("Failed to clean up cryo output")
		}
	}()

	rows, err := cryo.ReadParquet[logRow](collection.Files)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read cryo logs parquet")
	}

	stampInternalIndex(rows,
		func(r *logRow) uint64 { return uint64(r.BlockNumber) },
		func(r *logRow) string { return r.TransactionHash },
		func(r *logRow, idx uint32) { r.InternalIndex = idx },
	)

	return chunkEvents(rows, b.cfg.ChunkSize, b.createEvent)
}

// createEvent builds a single DecoratedEvent carrying a chunk of log rows.
func (b *LogsDeriver) createEvent(rows []logRow) (*xatu.DecoratedEvent, error) {
	metadata, ok := proto.Clone(b.clientMeta).(*xatu.ClientMeta)
	if !ok {
		return nil, errors.New("failed to clone client metadata")
	}

	logs := make([]*xatu.ExecutionLog, 0, len(rows))

	for i := range rows {
		logs = append(logs, logRowToProto(&rows[i]))
	}

	return &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_EXECUTION_CANONICAL_LOGS,
			DateTime: timestamppb.New(time.Now()),
			Id:       uuid.New().String(),
		},
		Meta: &xatu.Meta{Client: metadata},
		Data: &xatu.DecoratedEvent_ExecutionCanonicalLogs{
			ExecutionCanonicalLogs: &xatu.ExecutionCanonicalLogs{Logs: logs},
		},
	}, nil
}

// logRowToProto converts a cryo log row (collected with --hex) to proto.
func logRowToProto(row *logRow) *xatu.ExecutionLog {
	out := &xatu.ExecutionLog{
		BlockNumber:      uint64(row.BlockNumber),
		TransactionIndex: uint64(row.TransactionIndex),
		TransactionHash:  row.TransactionHash,
		InternalIndex:    row.InternalIndex,
		LogIndex:         row.LogIndex,
		Address:          row.Address,
		Topic0:           zeroHexIfEmpty(row.Topic0),
	}

	if row.Topic1 != "" {
		out.Topic1 = wrapperspb.String(row.Topic1)
	}

	if row.Topic2 != "" {
		out.Topic2 = wrapperspb.String(row.Topic2)
	}

	if row.Topic3 != "" {
		out.Topic3 = wrapperspb.String(row.Topic3)
	}

	if row.Data != "" {
		out.Data = wrapperspb.String(row.Data)
	}

	return out
}
