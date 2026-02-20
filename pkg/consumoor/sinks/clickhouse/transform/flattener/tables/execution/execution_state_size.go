package execution

import (
	"strconv"
	"time"

	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/flattener"
	catalog "github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/flattener/tables/catalog"
	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/metadata"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

var executionStateSizeEventNames = []xatu.Event_Name{
	xatu.Event_EXECUTION_STATE_SIZE,
}

func init() {
	catalog.MustRegister(flattener.NewStaticRoute(
		executionStateSizeTableName,
		executionStateSizeEventNames,
		func() flattener.ColumnarBatch { return newexecutionStateSizeBatch() },
	))
}

func (b *executionStateSizeBatch) FlattenTo(
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

func (b *executionStateSizeBatch) appendRuntime(event *xatu.DecoratedEvent) {
	b.UpdatedDateTime.Append(time.Now())

	if ts := event.GetEvent().GetDateTime(); ts != nil {
		b.EventDateTime.Append(ts.AsTime())
	} else {
		b.EventDateTime.Append(time.Time{})
	}
}

func (b *executionStateSizeBatch) appendPayload(event *xatu.DecoratedEvent) {
	payload := event.GetExecutionStateSize()
	if payload == nil {
		b.BlockNumber.Append(0)
		b.StateRoot.Append(nil)
		b.Accounts.Append(0)
		b.AccountBytes.Append(0)
		b.AccountTrienodes.Append(0)
		b.AccountTrienodeBytes.Append(0)
		b.ContractCodes.Append(0)
		b.ContractCodeBytes.Append(0)
		b.Storages.Append(0)
		b.StorageBytes.Append(0)
		b.StorageTrienodes.Append(0)
		b.StorageTrienodeBytes.Append(0)

		return
	}

	blockNumber, _ := strconv.ParseUint(payload.GetBlockNumber(), 10, 64)
	b.BlockNumber.Append(blockNumber)

	b.StateRoot.Append([]byte(payload.GetStateRoot()))

	accounts, _ := strconv.ParseUint(payload.GetAccounts(), 10, 64)
	b.Accounts.Append(accounts)

	accountBytes, _ := strconv.ParseUint(payload.GetAccountBytes(), 10, 64)
	b.AccountBytes.Append(accountBytes)

	accountTrienodes, _ := strconv.ParseUint(payload.GetAccountTrienodes(), 10, 64)
	b.AccountTrienodes.Append(accountTrienodes)

	accountTrienodeBytes, _ := strconv.ParseUint(payload.GetAccountTrienodeBytes(), 10, 64)
	b.AccountTrienodeBytes.Append(accountTrienodeBytes)

	contractCodes, _ := strconv.ParseUint(payload.GetContractCodes(), 10, 64)
	b.ContractCodes.Append(contractCodes)

	contractCodeBytes, _ := strconv.ParseUint(payload.GetContractCodeBytes(), 10, 64)
	b.ContractCodeBytes.Append(contractCodeBytes)

	storages, _ := strconv.ParseUint(payload.GetStorages(), 10, 64)
	b.Storages.Append(storages)

	storageBytes, _ := strconv.ParseUint(payload.GetStorageBytes(), 10, 64)
	b.StorageBytes.Append(storageBytes)

	storageTrienodes, _ := strconv.ParseUint(payload.GetStorageTrienodes(), 10, 64)
	b.StorageTrienodes.Append(storageTrienodes)

	storageTrienodeBytes, _ := strconv.ParseUint(payload.GetStorageTrienodeBytes(), 10, 64)
	b.StorageTrienodeBytes.Append(storageTrienodeBytes)
}
