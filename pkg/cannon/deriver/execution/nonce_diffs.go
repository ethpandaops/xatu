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

// NonceDiffsDeriverName is the cannon type produced by the nonce_diffs deriver.
const NonceDiffsDeriverName = xatu.CannonType_EXECUTION_CANONICAL_NONCE_DIFFS

const nonceDiffsDataset = "nonce_diffs"

// nonceDiffsColumns restricts cryo output to exactly the columns the deriver maps.
var nonceDiffsColumns = []string{
	"block_number",
	"transaction_index",
	"transaction_hash",
	"address",
	"from_value",
	"to_value",
}

// nonceDiffRow mirrors the cryo `nonce_diffs` parquet schema collected with --hex.
type nonceDiffRow struct {
	BlockNumber      uint32 `parquet:"block_number"`
	TransactionIndex uint64 `parquet:"transaction_index"`
	TransactionHash  string `parquet:"transaction_hash"`
	Address          string `parquet:"address"`
	FromValue        uint64 `parquet:"from_value"`
	ToValue          uint64 `parquet:"to_value"`

	InternalIndex uint32
}

// NonceDiffsDeriverConfig configures the nonce_diffs deriver.
type NonceDiffsDeriverConfig struct {
	Enabled   bool                            `yaml:"enabled" default:"false"`
	ChunkSize int                             `yaml:"chunkSize" default:"100"`
	Iterator  iterator.BackfillingBlockConfig `yaml:"iterator"`
}

// NonceDiffsDeriver derives canonical_execution_nonce_diffs events via cryo.
type NonceDiffsDeriver struct {
	base

	cfg        *NonceDiffsDeriverConfig
	cryo       *cryo.Runner
	clientMeta *xatu.ClientMeta
}

// NewNonceDiffsDeriver creates a nonce_diffs deriver.
func NewNonceDiffsDeriver(
	log observability.ContextualLogger,
	config *NonceDiffsDeriverConfig,
	iter *iterator.BackfillingBlock,
	runner *cryo.Runner,
	beacon *ethereum.BeaconNode,
	clientMeta *xatu.ClientMeta,
) *NonceDiffsDeriver {
	return &NonceDiffsDeriver{
		base: base{
			log: log.WithFields(logrus.Fields{
				"module": "cannon/event/execution/nonce_diffs",
				"type":   NonceDiffsDeriverName.String(),
			}),
			name:     NonceDiffsDeriverName.String(),
			beacon:   beacon,
			iterator: iter,
		},
		cfg:        config,
		cryo:       runner,
		clientMeta: clientMeta,
	}
}

// CannonType returns the deriver's cannon type.
func (b *NonceDiffsDeriver) CannonType() xatu.CannonType {
	return NonceDiffsDeriverName
}

// ActivationFork returns Phase0 so the deriver starts as soon as the beacon
// node is ready; the iterator gates EL progress on CL finality.
func (b *NonceDiffsDeriver) ActivationFork() spec.DataVersion {
	return spec.DataVersionPhase0
}

// Name returns the deriver name.
func (b *NonceDiffsDeriver) Name() string {
	return b.name
}

// Start begins the deriver loop.
func (b *NonceDiffsDeriver) Start(ctx context.Context) error {
	if !b.cfg.Enabled {
		b.log.WithContext(ctx).Info("Execution nonce_diffs deriver disabled")

		return nil
	}

	b.log.WithContext(ctx).Info("Execution nonce_diffs deriver enabled")

	if err := b.iterator.Start(ctx); err != nil {
		return errors.Wrap(err, "failed to start iterator")
	}

	b.run(ctx, b.processRange)

	return nil
}

// Stop is a no-op for the nonce_diffs deriver.
func (b *NonceDiffsDeriver) Stop(_ context.Context) error {
	return nil
}

// processRange runs cryo for [from, to] and builds chunked DecoratedEvents.
func (b *NonceDiffsDeriver) processRange(ctx context.Context, from, to uint64) ([]*xatu.DecoratedEvent, error) {
	ctx, span := observability.Tracer().Start(ctx,
		"NonceDiffsDeriver.processRange",
		trace.WithAttributes(
			attribute.Int64("from", int64(from)), //nolint:gosec // block numbers are far below int64 max.
			attribute.Int64("to", int64(to)),     //nolint:gosec // block numbers are far below int64 max.
		),
	)
	defer span.End()

	collection, err := b.cryo.Collect(ctx, nonceDiffsDataset, from, to, nonceDiffsColumns)
	if err != nil {
		return nil, errors.Wrap(err, "failed to collect nonce_diffs via cryo")
	}

	defer func() {
		if cErr := collection.Close(); cErr != nil {
			b.log.WithError(cErr).WithContext(ctx).Warn("Failed to clean up cryo output")
		}
	}()

	rows, err := cryo.ReadParquet[nonceDiffRow](collection.Files)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read cryo nonce_diffs parquet")
	}

	stampInternalIndex(rows,
		func(r *nonceDiffRow) uint64 { return uint64(r.BlockNumber) },
		func(r *nonceDiffRow) string { return r.TransactionHash },
		func(r *nonceDiffRow, idx uint32) { r.InternalIndex = idx },
	)

	return chunkEvents(rows, b.cfg.ChunkSize, b.createEvent)
}

// createEvent builds a single DecoratedEvent carrying a chunk of nonce_diff rows.
func (b *NonceDiffsDeriver) createEvent(rows []nonceDiffRow) (*xatu.DecoratedEvent, error) {
	metadata, ok := proto.Clone(b.clientMeta).(*xatu.ClientMeta)
	if !ok {
		return nil, errors.New("failed to clone client metadata")
	}

	nonceDiffs := make([]*xatu.ExecutionNonceDiff, 0, len(rows))

	for i := range rows {
		nonceDiffs = append(nonceDiffs, nonceDiffRowToProto(&rows[i]))
	}

	return &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_EXECUTION_CANONICAL_NONCE_DIFFS,
			DateTime: timestamppb.New(time.Now()),
			Id:       uuid.New().String(),
		},
		Meta: &xatu.Meta{Client: metadata},
		Data: &xatu.DecoratedEvent_ExecutionCanonicalNonceDiffs{
			ExecutionCanonicalNonceDiffs: &xatu.ExecutionCanonicalNonceDiffs{NonceDiffs: nonceDiffs},
		},
	}, nil
}

// nonceDiffRowToProto converts a cryo nonce_diff row to proto.
func nonceDiffRowToProto(row *nonceDiffRow) *xatu.ExecutionNonceDiff {
	return &xatu.ExecutionNonceDiff{
		BlockNumber:      uint64(row.BlockNumber),
		TransactionIndex: row.TransactionIndex,
		TransactionHash:  row.TransactionHash,
		InternalIndex:    row.InternalIndex,
		Address:          row.Address,
		FromValue:        row.FromValue,
		ToValue:          row.ToValue,
	}
}
