package beacon

import (
	"fmt"
	"time"

	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/flattener"
	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/metadata"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

func init() {
	flattener.MustRegister(flattener.NewStaticRoute(
		beaconApiEthV1EventsVoluntaryExitTableName,
		[]xatu.Event_Name{xatu.Event_BEACON_API_ETH_V1_EVENTS_VOLUNTARY_EXIT_V2},
		func() flattener.ColumnarBatch {
			return newbeaconApiEthV1EventsVoluntaryExitBatch()
		},
	))
}

func (b *beaconApiEthV1EventsVoluntaryExitBatch) FlattenTo(
	event *xatu.DecoratedEvent,
	meta *metadata.CommonMetadata,
) error {
	if event == nil || event.GetEvent() == nil {
		return nil
	}

	payload := event.GetEthV1EventsVoluntaryExitV2()
	if payload == nil {
		return fmt.Errorf("nil EthV1EventsVoluntaryExitV2 payload: %w", flattener.ErrInvalidEvent)
	}

	if err := b.validate(event); err != nil {
		return err
	}

	addl := event.GetMeta().GetClient().GetEthV1EventsVoluntaryExitV2()

	b.UpdatedDateTime.Append(time.Now())
	b.EventDateTime.Append(event.GetEvent().GetDateTime().AsTime())
	b.Epoch.Append(uint32(payload.GetMessage().GetEpoch().GetValue())) //nolint:gosec // G115: epoch fits uint32.
	b.EpochStartDateTime.Append(addl.GetEpoch().GetStartDateTime().AsTime())
	b.WallclockSlot.Append(uint32(addl.GetWallclockSlot().GetNumber().GetValue())) //nolint:gosec // G115: wallclock slot fits uint32.
	b.WallclockSlotStartDateTime.Append(addl.GetWallclockSlot().GetStartDateTime().AsTime())
	b.WallclockEpoch.Append(uint32(addl.GetWallclockEpoch().GetNumber().GetValue())) //nolint:gosec // G115: wallclock epoch fits uint32.
	b.WallclockEpochStartDateTime.Append(addl.GetWallclockEpoch().GetStartDateTime().AsTime())
	b.ValidatorIndex.Append(uint32(payload.GetMessage().GetValidatorIndex().GetValue())) //nolint:gosec // G115: validator index fits uint32.
	b.Signature.Append(payload.GetSignature())

	b.appendMetadata(meta)
	b.rows++

	return nil
}

func (b *beaconApiEthV1EventsVoluntaryExitBatch) validate(event *xatu.DecoratedEvent) error {
	payload := event.GetEthV1EventsVoluntaryExitV2()

	if payload.GetMessage() == nil {
		return fmt.Errorf("nil voluntary exit Message: %w", flattener.ErrInvalidEvent)
	}

	if payload.GetMessage().GetEpoch() == nil {
		return fmt.Errorf("nil Epoch: %w", flattener.ErrInvalidEvent)
	}

	if payload.GetMessage().GetValidatorIndex() == nil {
		return fmt.Errorf("nil ValidatorIndex: %w", flattener.ErrInvalidEvent)
	}

	addl := event.GetMeta().GetClient().GetEthV1EventsVoluntaryExitV2()
	if addl == nil {
		return fmt.Errorf("nil EthV1EventsVoluntaryExitV2 additional data: %w", flattener.ErrInvalidEvent)
	}

	if addl.GetEpoch() == nil || addl.GetEpoch().GetNumber() == nil {
		return fmt.Errorf("nil additional Epoch: %w", flattener.ErrInvalidEvent)
	}

	return nil
}
