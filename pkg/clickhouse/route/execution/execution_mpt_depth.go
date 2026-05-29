package execution

import (
	"fmt"
	"time"

	"github.com/ethpandaops/xatu/pkg/clickhouse/route"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

var executionMptDepthEventNames = []xatu.Event_Name{
	xatu.Event_EXECUTION_MPT_DEPTH,
}

func init() {
	r, err := route.NewStaticRoute(
		executionMptDepthTableName,
		executionMptDepthEventNames,
		func() route.ColumnarBatch { return newexecutionMptDepthBatch() },
	)
	if err != nil {
		route.RecordError(err)

		return
	}

	if err := route.Register(r); err != nil {
		route.RecordError(err)
	}
}

func (b *executionMptDepthBatch) FlattenTo(event *xatu.DecoratedEvent) error {
	if event == nil || event.GetEvent() == nil {
		return nil
	}

	if event.GetExecutionMptDepth() == nil {
		return fmt.Errorf("nil execution_mpt_depth payload: %w", route.ErrInvalidEvent)
	}

	if err := b.validate(event); err != nil {
		return err
	}

	b.appendRuntime(event)
	b.appendMetadata(event)
	b.appendPayload(event)
	b.rows++

	return nil
}

func (b *executionMptDepthBatch) validate(event *xatu.DecoratedEvent) error {
	payload := event.GetExecutionMptDepth()

	if payload.GetBlockNumber() == nil {
		return fmt.Errorf("nil BlockNumber: %w", route.ErrInvalidEvent)
	}

	if payload.GetStateRoot() == "" {
		return fmt.Errorf("nil StateRoot: %w", route.ErrInvalidEvent)
	}

	if payload.GetParentStateRoot() == "" {
		return fmt.Errorf("nil ParentStateRoot: %w", route.ErrInvalidEvent)
	}

	return nil
}

func (b *executionMptDepthBatch) appendRuntime(_ *xatu.DecoratedEvent) {
	b.UpdatedDateTime.Append(time.Now())
}

func (b *executionMptDepthBatch) appendPayload(event *xatu.DecoratedEvent) {
	payload := event.GetExecutionMptDepth()

	b.BlockNumber.Append(payload.GetBlockNumber().GetValue())
	b.StateRoot.Append([]byte(payload.GetStateRoot()))
	b.ParentStateRoot.Append([]byte(payload.GetParentStateRoot()))

	b.TotalAccountWrittenNodes.Append(payload.GetTotalAccountWrittenNodes().GetValue())
	b.TotalAccountWrittenBytes.Append(payload.GetTotalAccountWrittenBytes().GetValue())
	b.TotalAccountDeletedNodes.Append(payload.GetTotalAccountDeletedNodes().GetValue())
	b.TotalAccountDeletedBytes.Append(payload.GetTotalAccountDeletedBytes().GetValue())
	b.TotalStorageWrittenNodes.Append(payload.GetTotalStorageWrittenNodes().GetValue())
	b.TotalStorageWrittenBytes.Append(payload.GetTotalStorageWrittenBytes().GetValue())
	b.TotalStorageDeletedNodes.Append(payload.GetTotalStorageDeletedNodes().GetValue())
	b.TotalStorageDeletedBytes.Append(payload.GetTotalStorageDeletedBytes().GetValue())

	b.AccountWrittenNodes.Append(toDepthMap(payload.GetAccountWrittenNodes()))
	b.AccountWrittenBytes.Append(toDepthMap(payload.GetAccountWrittenBytes()))
	b.AccountDeletedNodes.Append(toDepthMap(payload.GetAccountDeletedNodes()))
	b.AccountDeletedBytes.Append(toDepthMap(payload.GetAccountDeletedBytes()))
	b.StorageWrittenNodes.Append(toDepthMap(payload.GetStorageWrittenNodes()))
	b.StorageWrittenBytes.Append(toDepthMap(payload.GetStorageWrittenBytes()))
	b.StorageDeletedNodes.Append(toDepthMap(payload.GetStorageDeletedNodes()))
	b.StorageDeletedBytes.Append(toDepthMap(payload.GetStorageDeletedBytes()))
}

// toDepthMap converts the proto's map<uint32, uint64> (proto3 has no uint8) into
// the ClickHouse Map(UInt8, UInt64) form. Trie depths in the geth tracer are
// bounded by [0, 64]; any key outside that range indicates an upstream bug and
// is dropped to keep the column type safe.
func toDepthMap(src map[uint32]uint64) map[uint8]uint64 {
	if len(src) == 0 {
		return map[uint8]uint64{}
	}

	out := make(map[uint8]uint64, len(src))
	for k, v := range src {
		if k > 64 {
			continue
		}

		out[uint8(k)] = v
	}

	return out
}
