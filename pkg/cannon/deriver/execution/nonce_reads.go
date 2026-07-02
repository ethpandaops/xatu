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

// NonceReadsDeriverName is the cannon type produced by the nonce reads deriver.
const NonceReadsDeriverName = xatu.CannonType_EXECUTION_CANONICAL_NONCE_READS

const nonceReadsDataset = "nonce_reads"

// nonceReadsColumns restricts cryo output to exactly the columns the deriver maps.
var nonceReadsColumns = []string{
	"block_number",
	"transaction_index",
	"transaction_hash",
	"address",
	"nonce",
}

// nonceReadRow mirrors the cryo `nonce_reads` parquet schema collected with --hex.
type nonceReadRow struct {
	BlockNumber      uint32 `parquet:"block_number"`
	TransactionIndex uint64 `parquet:"transaction_index"`
	TransactionHash  string `parquet:"transaction_hash"`
	Address          string `parquet:"address"`
	Nonce            uint64 `parquet:"nonce"`

	InternalIndex uint32
}

// NonceReadsDeriverConfig configures the nonce reads deriver.
type NonceReadsDeriverConfig struct {
	Enabled   bool                            `yaml:"enabled" default:"false"`
	ChunkSize int                             `yaml:"chunkSize" default:"100"`
	Iterator  iterator.BackfillingBlockConfig `yaml:"iterator"`
}

// NonceReadsDeriver derives canonical_execution_nonce_reads events via cryo.
type NonceReadsDeriver struct {
	base

	cfg        *NonceReadsDeriverConfig
	cryo       *cryo.Runner
	clientMeta *xatu.ClientMeta
}

// NewNonceReadsDeriver creates a nonce reads deriver.
func NewNonceReadsDeriver(
	log observability.ContextualLogger,
	config *NonceReadsDeriverConfig,
	iter *iterator.BackfillingBlock,
	runner *cryo.Runner,
	beacon *ethereum.BeaconNode,
	clientMeta *xatu.ClientMeta,
) *NonceReadsDeriver {
	return &NonceReadsDeriver{
		base: base{
			log: log.WithFields(logrus.Fields{
				"module": "cannon/event/execution/nonce_reads",
				"type":   NonceReadsDeriverName.String(),
			}),
			name:     NonceReadsDeriverName.String(),
			beacon:   beacon,
			iterator: iter,
		},
		cfg:        config,
		cryo:       runner,
		clientMeta: clientMeta,
	}
}

// CannonType returns the deriver's cannon type.
func (b *NonceReadsDeriver) CannonType() xatu.CannonType {
	return NonceReadsDeriverName
}

// ActivationFork returns Phase0 so the deriver starts as soon as the beacon
// node is ready; the iterator gates EL progress on CL finality.
func (b *NonceReadsDeriver) ActivationFork() spec.DataVersion {
	return spec.DataVersionPhase0
}

// Name returns the deriver name.
func (b *NonceReadsDeriver) Name() string {
	return b.name
}

// Start begins the deriver loop.
func (b *NonceReadsDeriver) Start(ctx context.Context) error {
	if !b.cfg.Enabled {
		b.log.WithContext(ctx).Info("Execution nonce reads deriver disabled")

		return nil
	}

	b.log.WithContext(ctx).Info("Execution nonce reads deriver enabled")

	if err := b.iterator.Start(ctx); err != nil {
		return errors.Wrap(err, "failed to start iterator")
	}

	b.run(ctx, b.processRange)

	return nil
}

// Stop is a no-op for the nonce reads deriver.
func (b *NonceReadsDeriver) Stop(_ context.Context) error {
	return nil
}

// processRange runs cryo for [from, to] and builds chunked DecoratedEvents.
func (b *NonceReadsDeriver) processRange(ctx context.Context, from, to uint64) ([]*xatu.DecoratedEvent, error) {
	ctx, span := observability.Tracer().Start(ctx,
		"NonceReadsDeriver.processRange",
		trace.WithAttributes(
			attribute.Int64("from", int64(from)), //nolint:gosec // block numbers are far below int64 max.
			attribute.Int64("to", int64(to)),     //nolint:gosec // block numbers are far below int64 max.
		),
	)
	defer span.End()

	collection, err := b.cryo.Collect(ctx, nonceReadsDataset, from, to, nonceReadsColumns)
	if err != nil {
		return nil, errors.Wrap(err, "failed to collect nonce reads via cryo")
	}

	defer func() {
		if cErr := collection.Close(); cErr != nil {
			b.log.WithError(cErr).WithContext(ctx).Warn("Failed to clean up cryo output")
		}
	}()

	rows, err := cryo.ReadParquet[nonceReadRow](collection.Files)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read cryo nonce reads parquet")
	}

	stampInternalIndex(rows,
		func(r *nonceReadRow) uint64 { return uint64(r.BlockNumber) },
		func(r *nonceReadRow) string { return r.TransactionHash },
		func(r *nonceReadRow, idx uint32) { r.InternalIndex = idx },
	)

	return chunkEvents(rows, b.cfg.ChunkSize, b.createEvent)
}

// createEvent builds a single DecoratedEvent carrying a chunk of nonce read rows.
func (b *NonceReadsDeriver) createEvent(rows []nonceReadRow) (*xatu.DecoratedEvent, error) {
	metadata, ok := proto.Clone(b.clientMeta).(*xatu.ClientMeta)
	if !ok {
		return nil, errors.New("failed to clone client metadata")
	}

	nonceReads := make([]*xatu.ExecutionNonceRead, 0, len(rows))

	for i := range rows {
		nonceReads = append(nonceReads, nonceReadRowToProto(&rows[i]))
	}

	return &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_EXECUTION_CANONICAL_NONCE_READS,
			DateTime: timestamppb.New(time.Now()),
			Id:       uuid.New().String(),
		},
		Meta: &xatu.Meta{Client: metadata},
		Data: &xatu.DecoratedEvent_ExecutionCanonicalNonceReads{
			ExecutionCanonicalNonceReads: &xatu.ExecutionCanonicalNonceReads{NonceReads: nonceReads},
		},
	}, nil
}

// nonceReadRowToProto converts a cryo nonce read row (collected with --hex) to proto.
func nonceReadRowToProto(row *nonceReadRow) *xatu.ExecutionNonceRead {
	return &xatu.ExecutionNonceRead{
		BlockNumber:      uint64(row.BlockNumber),
		TransactionIndex: row.TransactionIndex,
		TransactionHash:  row.TransactionHash,
		InternalIndex:    row.InternalIndex,
		Address:          row.Address,
		Nonce:            row.Nonce,
	}
}
