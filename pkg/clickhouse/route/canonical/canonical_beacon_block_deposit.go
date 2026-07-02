package canonical

import (
	"fmt"
	"strconv"
	"time"

	"github.com/ethpandaops/xatu/pkg/clickhouse/route"
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

	if err := b.validate(event); err != nil {
		return err
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

func (b *canonicalBeaconBlockDepositBatch) validate(event *xatu.DecoratedEvent) error {
	payload := event.GetEthV2BeaconBlockDeposit()

	if len(payload.GetProof()) == 0 {
		return fmt.Errorf("empty deposit_proof: %w", route.ErrInvalidEvent)
	}

	data := payload.GetData()
	if data == nil {
		return fmt.Errorf("nil deposit_data: %w", route.ErrInvalidEvent)
	}

	if data.GetPubkey() == "" {
		return fmt.Errorf("empty deposit_data_pubkey: %w", route.ErrInvalidEvent)
	}

	if data.GetWithdrawalCredentials() == "" {
		return fmt.Errorf("empty deposit_data_withdrawal_credentials: %w", route.ErrInvalidEvent)
	}

	if data.GetSignature() == "" {
		return fmt.Errorf("empty deposit_data_signature: %w", route.ErrInvalidEvent)
	}

	if data.GetAmount() == nil {
		return fmt.Errorf("nil deposit_data_amount: %w", route.ErrInvalidEvent)
	}

	additional := event.GetMeta().GetClient().GetEthV2BeaconBlockDeposit()
	if additional == nil {
		return fmt.Errorf("nil eth_v2_beacon_block_deposit additional data: %w", route.ErrInvalidEvent)
	}

	block := additional.GetBlock()
	if block == nil {
		return fmt.Errorf("nil block identifier: %w", route.ErrInvalidEvent)
	}

	if block.GetVersion() == "" {
		return fmt.Errorf("empty block_version: %w", route.ErrInvalidEvent)
	}

	if block.GetRoot() == "" {
		return fmt.Errorf("empty block_root: %w", route.ErrInvalidEvent)
	}

	if block.GetSlot().GetStartDateTime() == nil {
		return fmt.Errorf("nil slot_start_date_time: %w", route.ErrInvalidEvent)
	}

	if block.GetEpoch().GetStartDateTime() == nil {
		return fmt.Errorf("nil epoch_start_date_time: %w", route.ErrInvalidEvent)
	}

	if event.GetMeta().GetClient().GetEthereum().GetNetwork().GetName() == "" {
		return fmt.Errorf("empty meta_network_name: %w", route.ErrInvalidEvent)
	}

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
			parsedAmount, err := route.ParseUInt128(strconv.FormatUint(amount.GetValue(), 10))
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
