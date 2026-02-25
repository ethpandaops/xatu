package execution

import (
	"fmt"
	"strconv"
	"time"

	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/flattener"
	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/metadata"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

func init() {
	flattener.MustRegister(flattener.NewStaticRoute(
		executionStateSizeTableName,
		[]xatu.Event_Name{xatu.Event_EXECUTION_STATE_SIZE},
		func() flattener.ColumnarBatch {
			return newexecutionStateSizeBatch()
		},
	))
}

func (b *executionStateSizeBatch) FlattenTo(
	event *xatu.DecoratedEvent,
	meta *metadata.CommonMetadata,
) error {
	if event == nil || event.GetEvent() == nil {
		return nil
	}

	payload := event.GetExecutionStateSize()
	if payload == nil {
		return fmt.Errorf("nil ExecutionStateSize payload: %w", flattener.ErrInvalidEvent)
	}

	if err := b.validate(event); err != nil {
		return err
	}

	b.UpdatedDateTime.Append(time.Now())
	b.EventDateTime.Append(event.GetEvent().GetDateTime().AsTime())
	b.BlockNumber.Append(parseUint64(payload.GetBlockNumber()))
	b.StateRoot.Append([]byte(payload.GetStateRoot()))
	b.Accounts.Append(parseUint64(payload.GetAccounts()))
	b.AccountBytes.Append(parseUint64(payload.GetAccountBytes()))
	b.AccountTrienodes.Append(parseUint64(payload.GetAccountTrienodes()))
	b.AccountTrienodeBytes.Append(parseUint64(payload.GetAccountTrienodeBytes()))
	b.ContractCodes.Append(parseUint64(payload.GetContractCodes()))
	b.ContractCodeBytes.Append(parseUint64(payload.GetContractCodeBytes()))
	b.Storages.Append(parseUint64(payload.GetStorages()))
	b.StorageBytes.Append(parseUint64(payload.GetStorageBytes()))
	b.StorageTrienodes.Append(parseUint64(payload.GetStorageTrienodes()))
	b.StorageTrienodeBytes.Append(parseUint64(payload.GetStorageTrienodeBytes()))

	b.appendMetadata(meta)
	b.rows++

	return nil
}

func (b *executionStateSizeBatch) validate(event *xatu.DecoratedEvent) error {
	payload := event.GetExecutionStateSize()

	if payload.GetBlockNumber() == "" {
		return fmt.Errorf("nil BlockNumber: %w", flattener.ErrInvalidEvent)
	}

	return nil
}

// parseUint64 parses a decimal string to uint64, returning 0 on failure.
func parseUint64(s string) uint64 {
	v, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return 0
	}

	return v
}
