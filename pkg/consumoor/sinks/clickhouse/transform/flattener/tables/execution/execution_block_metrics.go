package execution

import (
	"fmt"
	"time"

	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/flattener"
	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/metadata"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

func init() {
	flattener.MustRegister(flattener.NewStaticRoute(
		executionBlockMetricsTableName,
		[]xatu.Event_Name{xatu.Event_EXECUTION_BLOCK_METRICS},
		func() flattener.ColumnarBatch {
			return newexecutionBlockMetricsBatch()
		},
	))
}

func (b *executionBlockMetricsBatch) FlattenTo(
	event *xatu.DecoratedEvent,
	meta *metadata.CommonMetadata,
) error {
	if event == nil || event.GetEvent() == nil {
		return nil
	}

	payload := event.GetExecutionBlockMetrics()
	if payload == nil {
		return fmt.Errorf("nil ExecutionBlockMetrics payload: %w", flattener.ErrInvalidEvent)
	}

	if err := b.validate(event); err != nil {
		return err
	}

	b.UpdatedDateTime.Append(time.Now())
	b.EventDateTime.Append(event.GetEvent().GetDateTime().AsTime())
	b.Source.Append(payload.GetSource())
	b.BlockNumber.Append(payload.GetBlockNumber().GetValue())
	b.BlockHash.Append([]byte(payload.GetBlockHash()))
	b.GasUsed.Append(payload.GetGasUsed().GetValue())
	b.TxCount.Append(payload.GetTxCount().GetValue())

	// Timing metrics (DoubleValue -> Float64, default 0).
	b.ExecutionMs.Append(payload.GetExecutionMs().GetValue())
	b.StateReadMs.Append(payload.GetStateReadMs().GetValue())
	b.StateHashMs.Append(payload.GetStateHashMs().GetValue())
	b.CommitMs.Append(payload.GetCommitMs().GetValue())
	b.TotalMs.Append(payload.GetTotalMs().GetValue())
	b.MgasPerSec.Append(payload.GetMgasPerSec().GetValue())

	// State reads.
	sr := payload.GetStateReads()
	b.StateReadsAccounts.Append(sr.GetAccounts().GetValue())
	b.StateReadsStorageSlots.Append(sr.GetStorageSlots().GetValue())
	b.StateReadsCode.Append(sr.GetCode().GetValue())
	b.StateReadsCodeBytes.Append(sr.GetCodeBytes().GetValue())

	// State writes.
	sw := payload.GetStateWrites()
	b.StateWritesAccounts.Append(sw.GetAccounts().GetValue())
	b.StateWritesAccountsDeleted.Append(sw.GetAccountsDeleted().GetValue())
	b.StateWritesStorageSlots.Append(sw.GetStorageSlots().GetValue())
	b.StateWritesStorageSlotsDeleted.Append(sw.GetStorageSlotsDeleted().GetValue())
	b.StateWritesCode.Append(sw.GetCode().GetValue())
	b.StateWritesCodeBytes.Append(sw.GetCodeBytes().GetValue())

	// Account cache.
	ac := payload.GetAccountCache()
	b.AccountCacheHits.Append(ac.GetHits().GetValue())
	b.AccountCacheMisses.Append(ac.GetMisses().GetValue())
	b.AccountCacheHitRate.Append(ac.GetHitRate().GetValue())

	// Storage cache.
	sc := payload.GetStorageCache()
	b.StorageCacheHits.Append(sc.GetHits().GetValue())
	b.StorageCacheMisses.Append(sc.GetMisses().GetValue())
	b.StorageCacheHitRate.Append(sc.GetHitRate().GetValue())

	// Code cache.
	cc := payload.GetCodeCache()
	b.CodeCacheHits.Append(cc.GetHits().GetValue())
	b.CodeCacheMisses.Append(cc.GetMisses().GetValue())
	b.CodeCacheHitRate.Append(cc.GetHitRate().GetValue())
	b.CodeCacheHitBytes.Append(cc.GetHitBytes().GetValue())
	b.CodeCacheMissBytes.Append(cc.GetMissBytes().GetValue())

	b.appendMetadata(meta)
	b.rows++

	return nil
}

func (b *executionBlockMetricsBatch) validate(event *xatu.DecoratedEvent) error {
	payload := event.GetExecutionBlockMetrics()

	if payload.GetBlockNumber() == nil {
		return fmt.Errorf("nil BlockNumber: %w", flattener.ErrInvalidEvent)
	}

	return nil
}
