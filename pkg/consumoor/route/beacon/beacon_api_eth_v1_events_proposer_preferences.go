package beacon

import (
	"fmt"
	"time"

	"github.com/ethpandaops/xatu/pkg/consumoor/route"
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

	if gasLimit := prefs.GetGasLimit(); gasLimit != nil {
		b.GasLimit.Append(gasLimit.GetValue())
	} else {
		b.GasLimit.Append(0)
	}
}

// TODO: Define AdditionalEthV1EventsProposerPreferencesData proto message to extract
// slot/epoch/propagation from event.GetMeta().GetClient(). For now, zero-fill these fields.
func (b *beaconApiEthV1EventsProposerPreferencesBatch) appendAdditionalData(
	_ *xatu.DecoratedEvent,
) {
	b.Slot.Append(0)
	b.SlotStartDateTime.Append(time.Time{})
	b.PropagationSlotStartDiff.Append(0)
	b.Epoch.Append(0)
	b.EpochStartDateTime.Append(time.Time{})
}
