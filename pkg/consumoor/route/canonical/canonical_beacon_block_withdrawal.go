package canonical

import (
	"fmt"
	"time"

	"github.com/ethpandaops/xatu/pkg/consumoor/route"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

var canonicalBeaconBlockWithdrawalEventNames = []xatu.Event_Name{
	xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_WITHDRAWAL,
}

func init() {
	r, err := route.NewStaticRoute(
		canonicalBeaconBlockWithdrawalTableName,
		canonicalBeaconBlockWithdrawalEventNames,
		func() route.ColumnarBatch { return newcanonicalBeaconBlockWithdrawalBatch() },
	)
	if err != nil {
		route.RecordError(err)

		return
	}

	if err := route.Register(r); err != nil {
		route.RecordError(err)
	}
}

func (b *canonicalBeaconBlockWithdrawalBatch) FlattenTo(event *xatu.DecoratedEvent) error {
	if event == nil || event.GetEvent() == nil {
		return nil
	}

	b.appendRuntime(event)
	b.appendMetadata(event)
	b.appendPayload(event)
	b.appendAdditionalData(event)
	b.rows++

	return nil
}

func (b *canonicalBeaconBlockWithdrawalBatch) appendRuntime(_ *xatu.DecoratedEvent) {
	b.UpdatedDateTime.Append(time.Now())
}

//nolint:gosec // G115: proto uint64 values are bounded by ClickHouse uint32 column schema
func (b *canonicalBeaconBlockWithdrawalBatch) appendPayload(event *xatu.DecoratedEvent) {
	withdrawal := event.GetEthV2BeaconBlockWithdrawal()
	if withdrawal == nil {
		b.WithdrawalAddress.Append(nil)
		b.WithdrawalIndex.Append(0)
		b.WithdrawalValidatorIndex.Append(0)
		b.WithdrawalAmount.Append(route.ParseUInt128("0"))

		return
	}

	b.WithdrawalAddress.Append([]byte(withdrawal.GetAddress()))

	if index := withdrawal.GetIndex(); index != nil {
		b.WithdrawalIndex.Append(uint32(index.GetValue()))
	} else {
		b.WithdrawalIndex.Append(0)
	}

	if validatorIndex := withdrawal.GetValidatorIndex(); validatorIndex != nil {
		b.WithdrawalValidatorIndex.Append(uint32(validatorIndex.GetValue()))
	} else {
		b.WithdrawalValidatorIndex.Append(0)
	}

	if amount := withdrawal.GetAmount(); amount != nil {
		b.WithdrawalAmount.Append(route.ParseUInt128(fmt.Sprintf("%d", amount.GetValue())))
	} else {
		b.WithdrawalAmount.Append(route.ParseUInt128("0"))
	}
}

func (b *canonicalBeaconBlockWithdrawalBatch) appendAdditionalData(event *xatu.DecoratedEvent) {
	additional := event.GetMeta().GetClient().GetEthV2BeaconBlockWithdrawal()
	if additional == nil {
		b.Slot.Append(0)
		b.SlotStartDateTime.Append(time.Time{})
		b.Epoch.Append(0)
		b.EpochStartDateTime.Append(time.Time{})
		b.BlockVersion.Append("")
		b.BlockRoot.Append(nil)

		return
	}

	appendBlockIdentifier(additional.GetBlock(),
		&b.Slot, &b.SlotStartDateTime, &b.Epoch, &b.EpochStartDateTime, &b.BlockVersion, &b.BlockRoot)
}
