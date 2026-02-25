package canonical

import (
	"fmt"
	"time"

	"github.com/ethpandaops/xatu/pkg/consumoor/route"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

var canonicalBeaconBlockDepositEventNames = []xatu.Event_Name{
	xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_DEPOSIT,
}

func init() {
	r, err := route.NewStaticRoute(
		canonicalBeaconBlockDepositTableName,
		canonicalBeaconBlockDepositEventNames,
		func() route.ColumnarBatch { return newcanonicalBeaconBlockDepositBatch() },
	)
	if err != nil {
		route.RecordError(err)

		return
	}

	if err := route.Register(r); err != nil {
		route.RecordError(err)
	}
}

func (b *canonicalBeaconBlockDepositBatch) FlattenTo(event *xatu.DecoratedEvent) error {
	if event == nil || event.GetEvent() == nil {
		return nil
	}

	if event.GetEthV2BeaconBlockDeposit() == nil {
		return fmt.Errorf("nil eth_v2_beacon_block_deposit payload: %w", route.ErrInvalidEvent)
	}

	b.appendRuntime(event)
	b.appendMetadata(event)

	if err := b.appendPayload(event); err != nil {
		return err
	}

	b.appendAdditionalData(event)
	b.rows++

	return nil
}

func (b *canonicalBeaconBlockDepositBatch) appendRuntime(_ *xatu.DecoratedEvent) {
	b.UpdatedDateTime.Append(time.Now())
}

func (b *canonicalBeaconBlockDepositBatch) appendPayload(event *xatu.DecoratedEvent) error {
	deposit := event.GetEthV2BeaconBlockDeposit()
	b.DepositProof.Append(deposit.GetProof())

	if data := deposit.GetData(); data != nil {
		b.DepositDataPubkey.Append(data.GetPubkey())
		b.DepositDataWithdrawalCredentials.Append([]byte(data.GetWithdrawalCredentials()))
		b.DepositDataSignature.Append(data.GetSignature())

		if amount := data.GetAmount(); amount != nil {
			parsedAmount, err := route.ParseUInt128(fmt.Sprintf("%d", amount.GetValue()))
			if err != nil {
				return fmt.Errorf("parsing deposit_data_amount: %w", err)
			}

			b.DepositDataAmount.Append(parsedAmount)
		} else {
			zeroAmount, err := route.ParseUInt128("0")
			if err != nil {
				return fmt.Errorf("parsing deposit_data_amount: %w", err)
			}

			b.DepositDataAmount.Append(zeroAmount)
		}
	} else {
		zeroAmount, err := route.ParseUInt128("0")
		if err != nil {
			return fmt.Errorf("parsing deposit_data_amount: %w", err)
		}

		b.DepositDataPubkey.Append("")
		b.DepositDataWithdrawalCredentials.Append(nil)
		b.DepositDataSignature.Append("")
		b.DepositDataAmount.Append(zeroAmount)
	}

	return nil
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
