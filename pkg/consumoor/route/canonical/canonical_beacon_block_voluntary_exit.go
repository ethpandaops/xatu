package canonical

import (
	"time"

	"github.com/ethpandaops/xatu/pkg/consumoor/route"
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

	b.appendRuntime(event)
	b.appendMetadata(event)
	b.appendPayload(event)
	b.appendAdditionalData(event)
	b.rows++

	return nil
}

func (b *canonicalBeaconBlockVoluntaryExitBatch) appendRuntime(_ *xatu.DecoratedEvent) {
	b.UpdatedDateTime.Append(time.Now())
}

//nolint:gosec // G115: proto uint64 values are bounded by ClickHouse uint32 column schema
func (b *canonicalBeaconBlockVoluntaryExitBatch) appendPayload(event *xatu.DecoratedEvent) {
	exit := event.GetEthV2BeaconBlockVoluntaryExit()
	if exit == nil {
		b.VoluntaryExitSignature.Append("")
		b.VoluntaryExitMessageEpoch.Append(0)
		b.VoluntaryExitMessageValidatorIndex.Append(0)

		return
	}

	b.VoluntaryExitSignature.Append(exit.GetSignature())

	if msg := exit.GetMessage(); msg != nil {
		if epoch := msg.GetEpoch(); epoch != nil {
			b.VoluntaryExitMessageEpoch.Append(uint32(epoch.GetValue()))
		} else {
			b.VoluntaryExitMessageEpoch.Append(0)
		}

		if validatorIndex := msg.GetValidatorIndex(); validatorIndex != nil {
			b.VoluntaryExitMessageValidatorIndex.Append(uint32(validatorIndex.GetValue()))
		} else {
			b.VoluntaryExitMessageValidatorIndex.Append(0)
		}
	} else {
		b.VoluntaryExitMessageEpoch.Append(0)
		b.VoluntaryExitMessageValidatorIndex.Append(0)
	}
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
