package execution

import (
	"time"

	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/flattener"
	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/metadata"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

var executionBlockMetricsEventNames = []xatu.Event_Name{
	xatu.Event_EXECUTION_BLOCK_METRICS,
}

func init() {
	flattener.MustRegister(flattener.NewStaticRoute(
		executionBlockMetricsTableName,
		executionBlockMetricsEventNames,
		func() flattener.ColumnarBatch { return newexecutionBlockMetricsBatch() },
	))
}

func (b *executionBlockMetricsBatch) FlattenTo(
	event *xatu.DecoratedEvent,
	meta *metadata.CommonMetadata,
) error {
	if event == nil || event.GetEvent() == nil {
		return nil
	}

	if meta == nil {
		meta = metadata.Extract(event)
	}

	b.appendRuntime(event)
	b.appendMetadata(meta)
	b.appendPayload(event)
	b.rows++

	return nil
}

func (b *executionBlockMetricsBatch) appendRuntime(event *xatu.DecoratedEvent) {
	b.UpdatedDateTime.Append(time.Now())

	if ts := event.GetEvent().GetDateTime(); ts != nil {
		b.EventDateTime.Append(ts.AsTime())
	} else {
		b.EventDateTime.Append(time.Time{})
	}
}

func (b *executionBlockMetricsBatch) appendPayload(event *xatu.DecoratedEvent) {
	payload := event.GetExecutionBlockMetrics()
	if payload == nil {
		b.Source.Append("")
		b.BlockNumber.Append(0)
		b.BlockHash.Append(nil)
		b.GasUsed.Append(0)
		b.TxCount.Append(0)
		b.appendPayloadTiming(nil)
		b.appendPayloadStateIO(nil)
		b.appendPayloadCaches(nil)

		return
	}

	b.Source.Append(payload.GetSource())

	if blockNumber := payload.GetBlockNumber(); blockNumber != nil {
		b.BlockNumber.Append(blockNumber.GetValue())
	} else {
		b.BlockNumber.Append(0)
	}

	b.BlockHash.Append([]byte(payload.GetBlockHash()))

	if gasUsed := payload.GetGasUsed(); gasUsed != nil {
		b.GasUsed.Append(gasUsed.GetValue())
	} else {
		b.GasUsed.Append(0)
	}

	if txCount := payload.GetTxCount(); txCount != nil {
		b.TxCount.Append(txCount.GetValue())
	} else {
		b.TxCount.Append(0)
	}

	b.appendPayloadTiming(payload)
	b.appendPayloadStateIO(payload)
	b.appendPayloadCaches(payload)
}

func (b *executionBlockMetricsBatch) appendPayloadTiming(payload *xatu.ExecutionBlockMetrics) {
	if payload == nil {
		b.ExecutionMs.Append(0)
		b.StateReadMs.Append(0)
		b.StateHashMs.Append(0)
		b.CommitMs.Append(0)
		b.TotalMs.Append(0)
		b.MgasPerSec.Append(0)

		return
	}

	if executionMs := payload.GetExecutionMs(); executionMs != nil {
		b.ExecutionMs.Append(executionMs.GetValue())
	} else {
		b.ExecutionMs.Append(0)
	}

	if stateReadMs := payload.GetStateReadMs(); stateReadMs != nil {
		b.StateReadMs.Append(stateReadMs.GetValue())
	} else {
		b.StateReadMs.Append(0)
	}

	if stateHashMs := payload.GetStateHashMs(); stateHashMs != nil {
		b.StateHashMs.Append(stateHashMs.GetValue())
	} else {
		b.StateHashMs.Append(0)
	}

	if commitMs := payload.GetCommitMs(); commitMs != nil {
		b.CommitMs.Append(commitMs.GetValue())
	} else {
		b.CommitMs.Append(0)
	}

	if totalMs := payload.GetTotalMs(); totalMs != nil {
		b.TotalMs.Append(totalMs.GetValue())
	} else {
		b.TotalMs.Append(0)
	}

	if mgasPerSec := payload.GetMgasPerSec(); mgasPerSec != nil {
		b.MgasPerSec.Append(mgasPerSec.GetValue())
	} else {
		b.MgasPerSec.Append(0)
	}
}

func (b *executionBlockMetricsBatch) appendPayloadStateIO(payload *xatu.ExecutionBlockMetrics) {
	if payload == nil {
		b.StateReadsAccounts.Append(0)
		b.StateReadsStorageSlots.Append(0)
		b.StateReadsCode.Append(0)
		b.StateReadsCodeBytes.Append(0)
		b.StateWritesAccounts.Append(0)
		b.StateWritesAccountsDeleted.Append(0)
		b.StateWritesStorageSlots.Append(0)
		b.StateWritesStorageSlotsDeleted.Append(0)
		b.StateWritesCode.Append(0)
		b.StateWritesCodeBytes.Append(0)

		return
	}

	if stateReads := payload.GetStateReads(); stateReads != nil {
		if accounts := stateReads.GetAccounts(); accounts != nil {
			b.StateReadsAccounts.Append(accounts.GetValue())
		} else {
			b.StateReadsAccounts.Append(0)
		}

		if storageSlots := stateReads.GetStorageSlots(); storageSlots != nil {
			b.StateReadsStorageSlots.Append(storageSlots.GetValue())
		} else {
			b.StateReadsStorageSlots.Append(0)
		}

		if code := stateReads.GetCode(); code != nil {
			b.StateReadsCode.Append(code.GetValue())
		} else {
			b.StateReadsCode.Append(0)
		}

		if codeBytes := stateReads.GetCodeBytes(); codeBytes != nil {
			b.StateReadsCodeBytes.Append(codeBytes.GetValue())
		} else {
			b.StateReadsCodeBytes.Append(0)
		}
	} else {
		b.StateReadsAccounts.Append(0)
		b.StateReadsStorageSlots.Append(0)
		b.StateReadsCode.Append(0)
		b.StateReadsCodeBytes.Append(0)
	}

	if stateWrites := payload.GetStateWrites(); stateWrites != nil {
		if accounts := stateWrites.GetAccounts(); accounts != nil {
			b.StateWritesAccounts.Append(accounts.GetValue())
		} else {
			b.StateWritesAccounts.Append(0)
		}

		if accountsDeleted := stateWrites.GetAccountsDeleted(); accountsDeleted != nil {
			b.StateWritesAccountsDeleted.Append(accountsDeleted.GetValue())
		} else {
			b.StateWritesAccountsDeleted.Append(0)
		}

		if storageSlots := stateWrites.GetStorageSlots(); storageSlots != nil {
			b.StateWritesStorageSlots.Append(storageSlots.GetValue())
		} else {
			b.StateWritesStorageSlots.Append(0)
		}

		if storageSlotsDeleted := stateWrites.GetStorageSlotsDeleted(); storageSlotsDeleted != nil {
			b.StateWritesStorageSlotsDeleted.Append(storageSlotsDeleted.GetValue())
		} else {
			b.StateWritesStorageSlotsDeleted.Append(0)
		}

		if code := stateWrites.GetCode(); code != nil {
			b.StateWritesCode.Append(code.GetValue())
		} else {
			b.StateWritesCode.Append(0)
		}

		if codeBytes := stateWrites.GetCodeBytes(); codeBytes != nil {
			b.StateWritesCodeBytes.Append(codeBytes.GetValue())
		} else {
			b.StateWritesCodeBytes.Append(0)
		}
	} else {
		b.StateWritesAccounts.Append(0)
		b.StateWritesAccountsDeleted.Append(0)
		b.StateWritesStorageSlots.Append(0)
		b.StateWritesStorageSlotsDeleted.Append(0)
		b.StateWritesCode.Append(0)
		b.StateWritesCodeBytes.Append(0)
	}
}

func (b *executionBlockMetricsBatch) appendPayloadCaches(payload *xatu.ExecutionBlockMetrics) {
	if payload == nil {
		b.AccountCacheHits.Append(0)
		b.AccountCacheMisses.Append(0)
		b.AccountCacheHitRate.Append(0)
		b.StorageCacheHits.Append(0)
		b.StorageCacheMisses.Append(0)
		b.StorageCacheHitRate.Append(0)
		b.CodeCacheHits.Append(0)
		b.CodeCacheMisses.Append(0)
		b.CodeCacheHitRate.Append(0)
		b.CodeCacheHitBytes.Append(0)
		b.CodeCacheMissBytes.Append(0)

		return
	}

	if accountCache := payload.GetAccountCache(); accountCache != nil {
		if hits := accountCache.GetHits(); hits != nil {
			b.AccountCacheHits.Append(hits.GetValue())
		} else {
			b.AccountCacheHits.Append(0)
		}

		if misses := accountCache.GetMisses(); misses != nil {
			b.AccountCacheMisses.Append(misses.GetValue())
		} else {
			b.AccountCacheMisses.Append(0)
		}

		if hitRate := accountCache.GetHitRate(); hitRate != nil {
			b.AccountCacheHitRate.Append(hitRate.GetValue())
		} else {
			b.AccountCacheHitRate.Append(0)
		}
	} else {
		b.AccountCacheHits.Append(0)
		b.AccountCacheMisses.Append(0)
		b.AccountCacheHitRate.Append(0)
	}

	if storageCache := payload.GetStorageCache(); storageCache != nil {
		if hits := storageCache.GetHits(); hits != nil {
			b.StorageCacheHits.Append(hits.GetValue())
		} else {
			b.StorageCacheHits.Append(0)
		}

		if misses := storageCache.GetMisses(); misses != nil {
			b.StorageCacheMisses.Append(misses.GetValue())
		} else {
			b.StorageCacheMisses.Append(0)
		}

		if hitRate := storageCache.GetHitRate(); hitRate != nil {
			b.StorageCacheHitRate.Append(hitRate.GetValue())
		} else {
			b.StorageCacheHitRate.Append(0)
		}
	} else {
		b.StorageCacheHits.Append(0)
		b.StorageCacheMisses.Append(0)
		b.StorageCacheHitRate.Append(0)
	}

	if codeCache := payload.GetCodeCache(); codeCache != nil {
		if hits := codeCache.GetHits(); hits != nil {
			b.CodeCacheHits.Append(hits.GetValue())
		} else {
			b.CodeCacheHits.Append(0)
		}

		if misses := codeCache.GetMisses(); misses != nil {
			b.CodeCacheMisses.Append(misses.GetValue())
		} else {
			b.CodeCacheMisses.Append(0)
		}

		if hitRate := codeCache.GetHitRate(); hitRate != nil {
			b.CodeCacheHitRate.Append(hitRate.GetValue())
		} else {
			b.CodeCacheHitRate.Append(0)
		}

		if hitBytes := codeCache.GetHitBytes(); hitBytes != nil {
			b.CodeCacheHitBytes.Append(hitBytes.GetValue())
		} else {
			b.CodeCacheHitBytes.Append(0)
		}

		if missBytes := codeCache.GetMissBytes(); missBytes != nil {
			b.CodeCacheMissBytes.Append(missBytes.GetValue())
		} else {
			b.CodeCacheMissBytes.Append(0)
		}
	} else {
		b.CodeCacheHits.Append(0)
		b.CodeCacheMisses.Append(0)
		b.CodeCacheHitRate.Append(0)
		b.CodeCacheHitBytes.Append(0)
		b.CodeCacheMissBytes.Append(0)
	}
}
