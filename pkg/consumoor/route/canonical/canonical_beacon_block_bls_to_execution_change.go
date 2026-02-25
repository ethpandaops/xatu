package canonical

import (
	"fmt"
	"time"

	"github.com/ethpandaops/xatu/pkg/consumoor/route"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

var canonicalBeaconBlockBlsToExecutionChangeEventNames = []xatu.Event_Name{
	xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_BLS_TO_EXECUTION_CHANGE,
}

func init() {
	r, err := route.NewStaticRoute(
		canonicalBeaconBlockBlsToExecutionChangeTableName,
		canonicalBeaconBlockBlsToExecutionChangeEventNames,
		func() route.ColumnarBatch { return newcanonicalBeaconBlockBlsToExecutionChangeBatch() },
	)
	if err != nil {
		route.RecordError(err)

		return
	}

	if err := route.Register(r); err != nil {
		route.RecordError(err)
	}
}

func (b *canonicalBeaconBlockBlsToExecutionChangeBatch) FlattenTo(event *xatu.DecoratedEvent) error {
	if event == nil || event.GetEvent() == nil {
		return nil
	}

	if event.GetEthV2BeaconBlockBlsToExecutionChange() == nil {
		return fmt.Errorf("nil eth_v2_beacon_block_bls_to_execution_change payload: %w", route.ErrInvalidEvent)
	}

	b.appendRuntime(event)
	b.appendMetadata(event)
	b.appendPayload(event)
	b.appendAdditionalData(event)
	b.rows++

	return nil
}

func (b *canonicalBeaconBlockBlsToExecutionChangeBatch) appendRuntime(_ *xatu.DecoratedEvent) {
	b.UpdatedDateTime.Append(time.Now())
}

//nolint:gosec // G115: proto uint64 values are bounded by ClickHouse uint32 column schema
func (b *canonicalBeaconBlockBlsToExecutionChangeBatch) appendPayload(event *xatu.DecoratedEvent) {
	change := event.GetEthV2BeaconBlockBlsToExecutionChange()
	b.ExchangingSignature.Append(change.GetSignature())

	if msg := change.GetMessage(); msg != nil {
		if validatorIndex := msg.GetValidatorIndex(); validatorIndex != nil {
			b.ExchangingMessageValidatorIndex.Append(uint32(validatorIndex.GetValue()))
		} else {
			b.ExchangingMessageValidatorIndex.Append(0)
		}

		b.ExchangingMessageFromBlsPubkey.Append(msg.GetFromBlsPubkey())
		b.ExchangingMessageToExecutionAddress.Append([]byte(msg.GetToExecutionAddress()))
	} else {
		b.ExchangingMessageValidatorIndex.Append(0)
		b.ExchangingMessageFromBlsPubkey.Append("")
		b.ExchangingMessageToExecutionAddress.Append(nil)
	}
}

func (b *canonicalBeaconBlockBlsToExecutionChangeBatch) appendAdditionalData(event *xatu.DecoratedEvent) {
	additional := event.GetMeta().GetClient().GetEthV2BeaconBlockBlsToExecutionChange()
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
