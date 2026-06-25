package canonical

import (
	"fmt"
	"time"

	"github.com/ethpandaops/xatu/pkg/clickhouse/route"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

var canonicalBeaconBlockVoluntaryExitEventNames = []xatu.Event_Name{
	xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_VOLUNTARY_EXIT,
}

func init() {
	r, err := route.NewStaticRoute(
		canonicalBeaconBlockVoluntaryExitTableName,
		canonicalBeaconBlockVoluntaryExitEventNames,
		func() route.ColumnarBatch { return newcanonicalBeaconBlockVoluntaryExitBatch() },
	)
	if err != nil {
		route.RecordError(err)

		return
	}

	if err := route.Register(r); err != nil {
		route.RecordError(err)
	}
}

func (b *canonicalBeaconBlockVoluntaryExitBatch) FlattenTo(event *xatu.DecoratedEvent) error {
	if event == nil || event.GetEvent() == nil {
		return nil
	}

	if event.GetEthV2BeaconBlockVoluntaryExit() == nil {
		return fmt.Errorf("nil eth_v2_beacon_block_voluntary_exit payload: %w", route.ErrInvalidEvent)
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

func (b *canonicalBeaconBlockVoluntaryExitBatch) validate(event *xatu.DecoratedEvent) error {
	msg := event.GetEthV2BeaconBlockVoluntaryExit().GetMessage()
	if msg == nil {
		return fmt.Errorf("nil Message: %w", route.ErrInvalidEvent)
	}

	if msg.GetEpoch() == nil {
		return fmt.Errorf("nil Message.Epoch: %w", route.ErrInvalidEvent)
	}

	if msg.GetValidatorIndex() == nil {
		return fmt.Errorf("nil Message.ValidatorIndex: %w", route.ErrInvalidEvent)
	}

	return nil
}

func (b *canonicalBeaconBlockVoluntaryExitBatch) appendRuntime(_ *xatu.DecoratedEvent) {
	b.UpdatedDateTime.Append(time.Now())
}

//nolint:gosec // G115: proto uint64 values are bounded by ClickHouse uint32 column schema
func (b *canonicalBeaconBlockVoluntaryExitBatch) appendPayload(event *xatu.DecoratedEvent) {
	exit := event.GetEthV2BeaconBlockVoluntaryExit()
	b.VoluntaryExitSignature.Append(exit.GetSignature())

	msg := exit.GetMessage()
	b.VoluntaryExitMessageEpoch.Append(uint32(msg.GetEpoch().GetValue()))
	b.VoluntaryExitMessageValidatorIndex.Append(uint32(msg.GetValidatorIndex().GetValue()))
}

func (b *canonicalBeaconBlockVoluntaryExitBatch) appendAdditionalData(event *xatu.DecoratedEvent) {
	additional := event.GetMeta().GetClient().GetEthV2BeaconBlockVoluntaryExit()
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
