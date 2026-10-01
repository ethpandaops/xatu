package canonical

import (
	"fmt"
	"time"

	"github.com/ethpandaops/xatu/pkg/clickhouse/route"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

var canonicalBeaconBlockAccessListSummaryEventNames = []xatu.Event_Name{
	xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_ACCESS_LIST_SUMMARY,
}

func init() {
	r, err := route.NewStaticRoute(
		canonicalBeaconBlockAccessListSummaryTableName,
		canonicalBeaconBlockAccessListSummaryEventNames,
		func() route.ColumnarBatch { return newcanonicalBeaconBlockAccessListSummaryBatch() },
	)
	if err != nil {
		route.RecordError(err)

		return
	}

	if err := route.Register(r); err != nil {
		route.RecordError(err)
	}
}

func (b *canonicalBeaconBlockAccessListSummaryBatch) FlattenTo(event *xatu.DecoratedEvent) error {
	if event == nil || event.GetEvent() == nil {
		return nil
	}

	if event.GetEthV2BeaconBlockAccessListSummary() == nil {
		return fmt.Errorf("nil eth_v2_beacon_block_access_list_summary payload: %w", route.ErrInvalidEvent)
	}

	if err := b.validate(event); err != nil {
		return err
	}

	b.appendRuntime(event)
	b.appendMetadata(event)
	b.appendPayload(event)
	b.appendAdditionalData(event)
	b.rows++

	return nil
}

func (b *canonicalBeaconBlockAccessListSummaryBatch) validate(event *xatu.DecoratedEvent) error {
	payload := event.GetEthV2BeaconBlockAccessListSummary()

	counts := []struct {
		name  string
		isNil bool
	}{
		{"AccountsTouched", payload.GetAccountsTouched() == nil},
		{"StorageSlotsChanged", payload.GetStorageSlotsChanged() == nil},
		{"StorageChanges", payload.GetStorageChanges() == nil},
		{"StorageReads", payload.GetStorageReads() == nil},
		{"BalanceChanges", payload.GetBalanceChanges() == nil},
		{"NonceChanges", payload.GetNonceChanges() == nil},
		{"CodeChanges", payload.GetCodeChanges() == nil},
		{"TotalChanges", payload.GetTotalChanges() == nil},
		{"BalSizeBytes", payload.GetBalSizeBytes() == nil},
		{"BalHash", payload.GetBalHash() == nil},
	}

	for _, count := range counts {
		if count.isNil {
			return fmt.Errorf("nil %s: %w", count.name, route.ErrInvalidEvent)
		}
	}

	additional := event.GetMeta().GetClient().GetEthV2BeaconBlockAccessListSummary()
	if additional == nil || additional.GetBlockNumber() == nil {
		return fmt.Errorf("nil BlockNumber: %w", route.ErrInvalidEvent)
	}

	if additional.GetBlockHash() == "" {
		return fmt.Errorf("empty BlockHash: %w", route.ErrInvalidEvent)
	}

	return nil
}

func (b *canonicalBeaconBlockAccessListSummaryBatch) appendRuntime(_ *xatu.DecoratedEvent) {
	b.UpdatedDateTime.Append(time.Now())
}

func (b *canonicalBeaconBlockAccessListSummaryBatch) appendPayload(event *xatu.DecoratedEvent) {
	summary := event.GetEthV2BeaconBlockAccessListSummary()

	b.AccountsTouched.Append(summary.GetAccountsTouched().GetValue())
	b.StorageSlotsChanged.Append(summary.GetStorageSlotsChanged().GetValue())
	b.StorageChanges.Append(summary.GetStorageChanges().GetValue())
	b.StorageReads.Append(summary.GetStorageReads().GetValue())
	b.BalanceChanges.Append(summary.GetBalanceChanges().GetValue())
	b.NonceChanges.Append(summary.GetNonceChanges().GetValue())
	b.CodeChanges.Append(summary.GetCodeChanges().GetValue())
	b.TotalChanges.Append(summary.GetTotalChanges().GetValue())
	b.BalSizeBytes.Append(summary.GetBalSizeBytes().GetValue())
	b.BalHash.Append([]byte(summary.GetBalHash().GetValue()))
}

func (b *canonicalBeaconBlockAccessListSummaryBatch) appendAdditionalData(event *xatu.DecoratedEvent) {
	additional := event.GetMeta().GetClient().GetEthV2BeaconBlockAccessListSummary()
	appendBlockIdentifier(additional.GetBlock(),
		&b.Slot, &b.SlotStartDateTime, &b.Epoch, &b.EpochStartDateTime, &b.BlockVersion, &b.BlockRoot)

	b.BlockNumber.Append(additional.GetBlockNumber().GetValue())
	b.BlockHash.Append([]byte(additional.GetBlockHash()))
}
