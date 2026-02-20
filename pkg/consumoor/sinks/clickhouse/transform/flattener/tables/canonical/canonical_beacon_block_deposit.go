package canonical

import (
	"fmt"
	"time"

	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/flattener"
	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/metadata"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

var canonicalBeaconBlockDepositEventNames = []xatu.Event_Name{
	xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_DEPOSIT,
}

func init() {
	flattener.MustRegister(flattener.NewStaticRoute(
		canonicalBeaconBlockDepositTableName,
		canonicalBeaconBlockDepositEventNames,
		func() flattener.ColumnarBatch { return newcanonicalBeaconBlockDepositBatch() },
	))
}

func (b *canonicalBeaconBlockDepositBatch) FlattenTo(
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
	b.appendAdditionalData(event)
	b.rows++

	return nil
}

func (b *canonicalBeaconBlockDepositBatch) appendRuntime(_ *xatu.DecoratedEvent) {
	b.UpdatedDateTime.Append(time.Now())
}

func (b *canonicalBeaconBlockDepositBatch) appendPayload(event *xatu.DecoratedEvent) {
	deposit := event.GetEthV2BeaconBlockDeposit()
	if deposit == nil {
		b.DepositProof.Append([]string{})
		b.DepositDataPubkey.Append("")
		b.DepositDataWithdrawalCredentials.Append(nil)
		b.DepositDataSignature.Append("")
		b.DepositDataAmount.Append(flattener.ParseUInt128("0"))

		return
	}

	b.DepositProof.Append(deposit.GetProof())

	if data := deposit.GetData(); data != nil {
		b.DepositDataPubkey.Append(data.GetPubkey())
		b.DepositDataWithdrawalCredentials.Append([]byte(data.GetWithdrawalCredentials()))
		b.DepositDataSignature.Append(data.GetSignature())

		if amount := data.GetAmount(); amount != nil {
			b.DepositDataAmount.Append(flattener.ParseUInt128(fmt.Sprintf("%d", amount.GetValue())))
		} else {
			b.DepositDataAmount.Append(flattener.ParseUInt128("0"))
		}
	} else {
		b.DepositDataPubkey.Append("")
		b.DepositDataWithdrawalCredentials.Append(nil)
		b.DepositDataSignature.Append("")
		b.DepositDataAmount.Append(flattener.ParseUInt128("0"))
	}
}

func (b *canonicalBeaconBlockDepositBatch) appendAdditionalData(event *xatu.DecoratedEvent) {
	additional := event.GetMeta().GetClient().GetEthV2BeaconBlockDeposit()
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
