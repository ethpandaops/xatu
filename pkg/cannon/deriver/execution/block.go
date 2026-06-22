// Package execution holds EL cannon derivers. Each deriver pulls a single cryo
// dataset over a block range, converts the rows into chunked DecoratedEvents,
// and advances its own block cursor — mirroring the per-deriver independence of
// the consensus-layer derivers. The shared loop lives in deriver_base.go; each
// deriver supplies its row struct, cryo columns, and row→proto mapping.
package execution

import (
	"context"
	"encoding/hex"
	"strings"
	"time"
	"unicode/utf8"

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
	"github.com/ethpandaops/xatu/pkg/cannon/execution/cryo"
	"github.com/ethpandaops/xatu/pkg/cannon/iterator"
	"github.com/ethpandaops/xatu/pkg/observability"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

// BlockDeriverName is the cannon type produced by the block deriver.
const BlockDeriverName = xatu.CannonType_EXECUTION_CANONICAL_BLOCK

const blockDataset = "blocks"

// blockColumns restricts cryo output to exactly the columns the deriver maps.
var blockColumns = []string{
	"block_number",
	"block_hash",
	"author",
	"gas_used",
	"gas_limit",
	"extra_data",
	"timestamp",
	"base_fee_per_gas",
}

// blockRow mirrors the cryo `blocks` parquet schema collected with --hex.
type blockRow struct {
	BlockNumber   uint32 `parquet:"block_number"`
	BlockHash     string `parquet:"block_hash"`
	Author        string `parquet:"author"`
	GasUsed       uint64 `parquet:"gas_used"`
	GasLimit      uint64 `parquet:"gas_limit"`
	ExtraData     string `parquet:"extra_data"`
	Timestamp     uint32 `parquet:"timestamp"`
	BaseFeePerGas uint64 `parquet:"base_fee_per_gas"`
}

// BlockDeriverConfig configures the block deriver.
type BlockDeriverConfig struct {
	Enabled   bool                            `yaml:"enabled" default:"false"`
	ChunkSize int                             `yaml:"chunkSize" default:"100"`
	Iterator  iterator.BackfillingBlockConfig `yaml:"iterator"`
}

// BlockDeriver derives canonical_execution_block events via cryo.
type BlockDeriver struct {
	base

	cfg        *BlockDeriverConfig
	cryo       *cryo.Runner
	clientMeta *xatu.ClientMeta
}

// NewBlockDeriver creates a block deriver.
func NewBlockDeriver(
	log observability.ContextualLogger,
	config *BlockDeriverConfig,
	iter *iterator.BackfillingBlock,
	runner *cryo.Runner,
	beacon *ethereum.BeaconNode,
	clientMeta *xatu.ClientMeta,
) *BlockDeriver {
	return &BlockDeriver{
		base: base{
			log: log.WithFields(logrus.Fields{
				"module": "cannon/event/execution/block",
				"type":   BlockDeriverName.String(),
			}),
			name:     BlockDeriverName.String(),
			beacon:   beacon,
			iterator: iter,
		},
		cfg:        config,
		cryo:       runner,
		clientMeta: clientMeta,
	}
}

// CannonType returns the deriver's cannon type.
func (b *BlockDeriver) CannonType() xatu.CannonType {
	return BlockDeriverName
}

// ActivationFork returns Phase0 so the deriver starts as soon as the beacon
// node is ready; the iterator gates EL progress on CL finality.
func (b *BlockDeriver) ActivationFork() spec.DataVersion {
	return spec.DataVersionPhase0
}

// Name returns the deriver name.
func (b *BlockDeriver) Name() string {
	return b.name
}

// Start begins the deriver loop.
func (b *BlockDeriver) Start(ctx context.Context) error {
	if !b.cfg.Enabled {
		b.log.WithContext(ctx).Info("Execution block deriver disabled")

		return nil
	}

	b.log.WithContext(ctx).Info("Execution block deriver enabled")

	if err := b.iterator.Start(ctx); err != nil {
		return errors.Wrap(err, "failed to start iterator")
	}

	b.run(ctx, b.processRange)

	return nil
}

// Stop is a no-op for the block deriver.
func (b *BlockDeriver) Stop(_ context.Context) error {
	return nil
}

// processRange runs cryo for [from, to] and builds chunked DecoratedEvents.
func (b *BlockDeriver) processRange(ctx context.Context, from, to uint64) ([]*xatu.DecoratedEvent, error) {
	ctx, span := observability.Tracer().Start(ctx,
		"BlockDeriver.processRange",
		trace.WithAttributes(
			attribute.Int64("from", int64(from)), //nolint:gosec // block numbers are far below int64 max.
			attribute.Int64("to", int64(to)),     //nolint:gosec // block numbers are far below int64 max.
		),
	)
	defer span.End()

	collection, err := b.cryo.Collect(ctx, blockDataset, from, to, blockColumns)
	if err != nil {
		return nil, errors.Wrap(err, "failed to collect blocks via cryo")
	}

	defer func() {
		if cErr := collection.Close(); cErr != nil {
			b.log.WithError(cErr).WithContext(ctx).Warn("Failed to clean up cryo output")
		}
	}()

	rows, err := cryo.ReadParquet[blockRow](collection.Files)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read cryo block parquet")
	}

	return chunkEvents(rows, b.cfg.ChunkSize, b.createEvent)
}

// createEvent builds a single DecoratedEvent carrying a chunk of block rows.
func (b *BlockDeriver) createEvent(rows []blockRow) (*xatu.DecoratedEvent, error) {
	metadata, ok := proto.Clone(b.clientMeta).(*xatu.ClientMeta)
	if !ok {
		return nil, errors.New("failed to clone client metadata")
	}

	blocks := make([]*xatu.ExecutionBlock, 0, len(rows))

	for i := range rows {
		blocks = append(blocks, blockRowToProto(&rows[i]))
	}

	return &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_EXECUTION_CANONICAL_BLOCK,
			DateTime: timestamppb.New(time.Now()),
			Id:       uuid.New().String(),
		},
		Meta: &xatu.Meta{Client: metadata},
		Data: &xatu.DecoratedEvent_ExecutionCanonicalBlock{
			ExecutionCanonicalBlock: &xatu.ExecutionCanonicalBlock{Blocks: blocks},
		},
	}, nil
}

// blockRowToProto converts a cryo block row (collected with --hex) to proto.
func blockRowToProto(row *blockRow) *xatu.ExecutionBlock {
	out := &xatu.ExecutionBlock{
		BlockNumber:   uint64(row.BlockNumber),
		BlockHash:     row.BlockHash,
		BlockDateTime: timestamppb.New(time.Unix(int64(row.Timestamp), 0).UTC()),
		GasUsed:       wrapperspb.UInt64(row.GasUsed),
		GasLimit:      wrapperspb.UInt64(row.GasLimit),
		BaseFeePerGas: wrapperspb.UInt64(row.BaseFeePerGas),
	}

	if row.Author != "" {
		out.Author = wrapperspb.String(row.Author)
	}

	// The legacy pipeline stores empty extra_data as '' (not NULL). cryo --hex
	// encodes empty bytes as "0x"; normalise that to "" so the route writes ''.
	extraData := row.ExtraData
	if extraData == "0x" {
		extraData = ""
	}

	out.ExtraData = wrapperspb.String(extraData)

	if extraData == "" {
		out.ExtraDataString = wrapperspb.String("")
	} else if s, ok := decodeHexUTF8(extraData); ok {
		out.ExtraDataString = wrapperspb.String(s)
	}

	return out
}

// decodeHexUTF8 decodes a 0x-prefixed hex string and returns it as a UTF-8
// string when valid. Used to populate extra_data_string.
func decodeHexUTF8(hexStr string) (string, bool) {
	raw, err := hex.DecodeString(strings.TrimPrefix(hexStr, "0x"))
	if err != nil {
		return "", false
	}

	if !utf8.Valid(raw) {
		return "", false
	}

	return string(raw), true
}
