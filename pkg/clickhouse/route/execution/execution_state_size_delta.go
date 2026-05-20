package execution

import (
	"fmt"
	"time"

	"github.com/ethpandaops/xatu/pkg/clickhouse/route"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

var executionStateSizeDeltaEventNames = []xatu.Event_Name{
	xatu.Event_EXECUTION_STATE_SIZE_DELTA,
}

func init() {
	r, err := route.NewStaticRoute(
		executionStateSizeDeltaTableName,
		executionStateSizeDeltaEventNames,
		func() route.ColumnarBatch { return newexecutionStateSizeDeltaBatch() },
	)
	if err != nil {
		route.RecordError(err)

		return
	}

	if err := route.Register(r); err != nil {
		route.RecordError(err)
	}
}

func (b *executionStateSizeDeltaBatch) FlattenTo(event *xatu.DecoratedEvent) error {
	if event == nil || event.GetEvent() == nil {
		return nil
	}

	if event.GetExecutionStateSizeDelta() == nil {
		return fmt.Errorf("nil execution_state_size_delta payload: %w", route.ErrInvalidEvent)
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

func (b *executionStateSizeDeltaBatch) validate(event *xatu.DecoratedEvent) error {
	payload := event.GetExecutionStateSizeDelta()

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

func (b *executionStateSizeDeltaBatch) appendRuntime(_ *xatu.DecoratedEvent) {
	b.UpdatedDateTime.Append(time.Now())
}

func (b *executionStateSizeDeltaBatch) appendPayload(event *xatu.DecoratedEvent) {
	payload := event.GetExecutionStateSizeDelta()

	b.BlockNumber.Append(payload.GetBlockNumber().GetValue())
	b.StateRoot.Append([]byte(payload.GetStateRoot()))
	b.ParentStateRoot.Append([]byte(payload.GetParentStateRoot()))

	b.AccountWrites.Append(payload.GetAccountWrites().GetValue())
	b.AccountWriteBytes.Append(payload.GetAccountWriteBytes().GetValue())
	b.AccountTrienodeWrites.Append(payload.GetAccountTrienodeWrites().GetValue())
	b.AccountTrienodeWriteBytes.Append(payload.GetAccountTrienodeWriteBytes().GetValue())
	b.ContractCodeWrites.Append(payload.GetContractCodeWrites().GetValue())
	b.ContractCodeWriteBytes.Append(payload.GetContractCodeWriteBytes().GetValue())
	b.StorageWrites.Append(payload.GetStorageWrites().GetValue())
	b.StorageWriteBytes.Append(payload.GetStorageWriteBytes().GetValue())
	b.StorageTrienodeWrites.Append(payload.GetStorageTrienodeWrites().GetValue())
	b.StorageTrienodeWriteBytes.Append(payload.GetStorageTrienodeWriteBytes().GetValue())

	b.AccountDeletes.Append(payload.GetAccountDeletes().GetValue())
	b.AccountDeleteBytes.Append(payload.GetAccountDeleteBytes().GetValue())
	b.AccountTrienodeDeletes.Append(payload.GetAccountTrienodeDeletes().GetValue())
	b.AccountTrienodeDeleteBytes.Append(payload.GetAccountTrienodeDeleteBytes().GetValue())
	b.ContractCodeDeletes.Append(payload.GetContractCodeDeletes().GetValue())
	b.ContractCodeDeleteBytes.Append(payload.GetContractCodeDeleteBytes().GetValue())
	b.StorageDeletes.Append(payload.GetStorageDeletes().GetValue())
	b.StorageDeleteBytes.Append(payload.GetStorageDeleteBytes().GetValue())
	b.StorageTrienodeDeletes.Append(payload.GetStorageTrienodeDeletes().GetValue())
	b.StorageTrienodeDeleteBytes.Append(payload.GetStorageTrienodeDeleteBytes().GetValue())
}
