package canonical

import (
	"time"

	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/flattener"
	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/metadata"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

var canonicalBeaconBlockBlsToExecutionChangeEventNames = []xatu.Event_Name{
	xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_BLS_TO_EXECUTION_CHANGE,
}

func init() {
	flattener.MustRegister(flattener.NewStaticRoute(
		canonicalBeaconBlockBlsToExecutionChangeTableName,
		canonicalBeaconBlockBlsToExecutionChangeEventNames,
		func() flattener.ColumnarBatch { return newcanonicalBeaconBlockBlsToExecutionChangeBatch() },
	))
}

func (b *canonicalBeaconBlockBlsToExecutionChangeBatch) FlattenTo(
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

func (b *canonicalBeaconBlockBlsToExecutionChangeBatch) appendRuntime(_ *xatu.DecoratedEvent) {
	b.UpdatedDateTime.Append(time.Now())
}

//nolint:gosec // G115: proto uint64 values are bounded by ClickHouse uint32 column schema
func (b *canonicalBeaconBlockBlsToExecutionChangeBatch) appendPayload(event *xatu.DecoratedEvent) {
	change := event.GetEthV2BeaconBlockBlsToExecutionChange()
	if change == nil {
		b.ExchangingSignature.Append("")
		b.ExchangingMessageValidatorIndex.Append(0)
		b.ExchangingMessageFromBlsPubkey.Append("")
		b.ExchangingMessageToExecutionAddress.Append(nil)

		return
	}

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
