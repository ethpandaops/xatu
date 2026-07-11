package beacon

import (
	"fmt"
	"time"

	"github.com/ethpandaops/xatu/pkg/clickhouse/route"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

var beaconApiEthV1EventsProposerPreferencesEventNames = []xatu.Event_Name{
	xatu.Event_BEACON_API_ETH_V1_EVENTS_PROPOSER_PREFERENCES,
}

func init() {
	r, err := route.NewStaticRoute(
		beaconApiEthV1EventsProposerPreferencesTableName,
		beaconApiEthV1EventsProposerPreferencesEventNames,
		func() route.ColumnarBatch { return newbeaconApiEthV1EventsProposerPreferencesBatch() },
	)
	if err != nil {
		route.RecordError(err)

		return
	}

	if err := route.Register(r); err != nil {
		route.RecordError(err)
	}
}

func (b *beaconApiEthV1EventsProposerPreferencesBatch) FlattenTo(
	event *xatu.DecoratedEvent,
) error {
	if event == nil || event.GetEvent() == nil {
		return nil
	}

	if event.GetEthV1EventsProposerPreferences() == nil {
		return fmt.Errorf("nil eth_v1_events_proposer_preferences payload: %w", route.ErrInvalidEvent)
	}

	b.appendRuntime(event)
	b.appendMetadata(event)
	b.appendPayload(event)
	b.appendAdditionalData(event)
	b.rows++

	return nil
}

func (b *beaconApiEthV1EventsProposerPreferencesBatch) appendRuntime(event *xatu.DecoratedEvent) {
	b.UpdatedDateTime.Append(time.Now())

	if ts := event.GetEvent().GetDateTime(); ts != nil {
		b.EventDateTime.Append(ts.AsTime())
	} else {
		b.EventDateTime.Append(time.Time{})
	}
}

//nolint:gosec // G115: proto uint64 values are bounded by ClickHouse column schema
func (b *beaconApiEthV1EventsProposerPreferencesBatch) appendPayload(event *xatu.DecoratedEvent) {
	signed := event.GetEthV1EventsProposerPreferences()
	prefs := signed.GetMessage()

	if validatorIndex := prefs.GetValidatorIndex(); validatorIndex != nil {
		b.ValidatorIndex.Append(uint32(validatorIndex.GetValue()))
	} else {
		b.ValidatorIndex.Append(0)
	}

	b.FeeRecipient.Append([]byte(prefs.GetFeeRecipient()))

	if gasLimit := prefs.GetTargetGasLimit(); gasLimit != nil {
		b.TargetGasLimit.Append(gasLimit.GetValue())
	} else {
		b.TargetGasLimit.Append(0)
	}
}

func (b *beaconApiEthV1EventsProposerPreferencesBatch) appendAdditionalData(
	event *xatu.DecoratedEvent,
) {
	if event.GetMeta() == nil || event.GetMeta().GetClient() == nil {
		b.Slot.Append(0)
		b.SlotStartDateTime.Append(time.Time{})
		b.PropagationSlotStartDiff.Append(0)
		b.Epoch.Append(0)
		b.EpochStartDateTime.Append(time.Time{})

		return
	}

	client := event.GetMeta().GetClient()
	additional := extractBeaconSlotEpochPropagation(client.GetEthV1EventsProposerPreferences())

	b.Slot.Append(uint32(additional.Slot)) //nolint:gosec // slot fits uint32
	b.SlotStartDateTime.Append(time.Unix(additional.SlotStartDateTime, 0))
	// Signed: proposer preferences propagate before slot start, so the diff can be
	// negative. It is carried through the proto as a two's-complement uint64;
	// reinterpret to int32.
	//nolint:gosec // intentional two's-complement reinterpret of a signed ms diff
	b.PropagationSlotStartDiff.Append(int32(int64(additional.PropagationSlotStartDiff)))
	b.Epoch.Append(uint32(additional.Epoch)) //nolint:gosec // epoch fits uint32
	b.EpochStartDateTime.Append(time.Unix(additional.EpochStartDateTime, 0))
}
