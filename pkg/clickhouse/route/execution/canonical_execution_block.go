package execution

import (
	"fmt"
	"time"

	"github.com/ClickHouse/ch-go/proto"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/ethpandaops/xatu/pkg/clickhouse/route"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

var canonicalExecutionBlockEventNames = []xatu.Event_Name{
	xatu.Event_EXECUTION_CANONICAL_BLOCK,
}

func init() {
	r, err := route.NewStaticRoute(
		canonicalExecutionBlockTableName,
		canonicalExecutionBlockEventNames,
		func() route.ColumnarBatch { return newCanonicalExecutionBlockBatch() },
	)
	if err != nil {
		route.RecordError(err)

		return
	}

	if err := route.Register(r); err != nil {
		route.RecordError(err)
	}
}

// FlattenTo flattens one DecoratedEvent (a chunk of blocks) into one row per
// block in canonical_execution_block.
func (b *canonicalExecutionBlockBatch) FlattenTo(event *xatu.DecoratedEvent) error {
	if event == nil || event.GetEvent() == nil {
		return nil
	}

	payload := event.GetExecutionCanonicalBlock()
	if payload == nil {
		return fmt.Errorf("nil execution_canonical_block payload: %w", route.ErrInvalidEvent)
	}

	network := event.GetMeta().GetClient().GetEthereum().GetNetwork().GetName()
	now := time.Now()

	for _, blk := range payload.GetBlocks() {
		b.UpdatedDateTime.Append(now)

		if ts := blk.GetBlockDateTime(); ts != nil {
			b.BlockDateTime.Append(ts.AsTime())
		} else {
			b.BlockDateTime.Append(time.Time{})
		}

		b.BlockNumber.Append(blk.GetBlockNumber())
		b.BlockHash.Append([]byte(blk.GetBlockHash()))
		b.Author.Append(ceNullStr(blk.GetAuthor()))
		b.GasUsed.Append(ceNullU64(blk.GetGasUsed()))
		b.GasLimit.Append(blk.GetGasLimit().GetValue())
		b.ExtraData.Append(ceNullStr(blk.GetExtraData()))
		b.ExtraDataString.Append(ceNullStr(blk.GetExtraDataString()))
		b.BaseFeePerGas.Append(ceNullU64(blk.GetBaseFeePerGas()))
		b.MetaNetworkName.Append(network)

		b.rows++
	}

	return nil
}

// ceNullStr converts a protobuf StringValue into a nullable ch-go string value.
// A bare "0x" (cryo's --hex encoding of empty bytes) is treated as NULL, matching
// the legacy pipeline which mapped empty byte columns to NULL.
func ceNullStr(v *wrapperspb.StringValue) proto.Nullable[string] {
	if v == nil || v.GetValue() == "0x" {
		return proto.Nullable[string]{}
	}

	return proto.NewNullable(v.GetValue())
}

// ceNullU64 converts a protobuf UInt64Value into a nullable ch-go uint64 value.
func ceNullU64(v *wrapperspb.UInt64Value) proto.Nullable[uint64] {
	if v == nil {
		return proto.Nullable[uint64]{}
	}

	return proto.NewNullable(v.GetValue())
}
