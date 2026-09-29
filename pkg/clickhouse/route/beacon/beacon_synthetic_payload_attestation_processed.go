package beacon

import (
	"fmt"
	"time"

	"github.com/ethpandaops/xatu/pkg/clickhouse/route"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

var beaconSyntheticPayloadAttestationProcessedEventNames = []xatu.Event_Name{
	xatu.Event_BEACON_SYNTHETIC_PAYLOAD_ATTESTATION_PROCESSED,
}

func init() {
	r, err := route.NewStaticRoute(
		beaconSyntheticPayloadAttestationProcessedTableName,
		beaconSyntheticPayloadAttestationProcessedEventNames,
		func() route.ColumnarBatch { return newbeaconSyntheticPayloadAttestationProcessedBatch() },
	)
	if err != nil {
		route.RecordError(err)

		return
	}

	if err := route.Register(r); err != nil {
		route.RecordError(err)
	}
}

func (b *beaconSyntheticPayloadAttestationProcessedBatch) FlattenTo(
	event *xatu.DecoratedEvent,
) error {
	if event == nil || event.GetEvent() == nil {
		return nil
	}

	if event.GetBeaconSyntheticPayloadAttestationProcessed() == nil {
		return fmt.Errorf(
			"nil beacon_synthetic_payload_attestation_processed payload: %w",
			route.ErrInvalidEvent,
		)
	}

	b.appendRuntime(event)
	b.appendMetadata(event)
	b.appendPayload(event)
	b.appendAdditionalData(event)
	b.rows++

	return nil
}

func (b *beaconSyntheticPayloadAttestationProcessedBatch) appendRuntime(
	event *xatu.DecoratedEvent,
) {
	b.UpdatedDateTime.Append(time.Now())

	if ts := event.GetEvent().GetDateTime(); ts != nil {
		b.EventDateTime.Append(ts.AsTime())
	} else {
		b.EventDateTime.Append(time.Time{})
	}
}

func (b *beaconSyntheticPayloadAttestationProcessedBatch) appendPayload(
	event *xatu.DecoratedEvent,
) {
	payload := event.GetBeaconSyntheticPayloadAttestationProcessed()

	if slot := payload.GetSlot(); slot != nil {
		b.Slot.Append(uint32(slot.GetValue())) //nolint:gosec // slot values fit uint32
	} else {
		b.Slot.Append(0)
	}

	b.BeaconBlockRoot.Append([]byte(payload.GetBeaconBlockRoot()))

	if v := payload.GetValidatorIndex(); v != nil {
		b.ValidatorIndex.Append(uint32(v.GetValue())) //nolint:gosec // validator index fits uint32
	} else {
		b.ValidatorIndex.Append(0)
	}

	b.PayloadPresent.Append(payload.GetPayloadPresent())
	b.BlobDataAvailable.Append(payload.GetBlobDataAvailable())
	b.PeerID.Append(payload.GetPeerId())

	if v := payload.GetProcessingDurationMs(); v != nil {
		b.ProcessingDurationMs.Append(v.GetValue())
	} else {
		b.ProcessingDurationMs.Append(0)
	}

	if v := payload.GetReceivedAt(); v != nil {
		b.ReceivedAt.Append(v.AsTime())
	} else {
		b.ReceivedAt.Append(time.Time{})
	}
}

func (b *beaconSyntheticPayloadAttestationProcessedBatch) appendAdditionalData(
	event *xatu.DecoratedEvent,
) {
	if event.GetMeta() == nil || event.GetMeta().GetClient() == nil {
		b.SlotStartDateTime.Append(time.Time{})
		b.PropagationSlotStartDiff.Append(0)
		b.Epoch.Append(0)
		b.EpochStartDateTime.Append(time.Time{})

		return
	}

	additional := extractBeaconSlotEpochPropagation(event.GetMeta().GetClient().GetBeaconSyntheticPayloadAttestationProcessed())

	b.SlotStartDateTime.Append(time.Unix(additional.SlotStartDateTime, 0))
	b.PropagationSlotStartDiff.Append(uint32(additional.PropagationSlotStartDiff)) //nolint:gosec // propagation diff fits uint32
	b.Epoch.Append(uint32(additional.Epoch))                                       //nolint:gosec // epoch fits uint32
	b.EpochStartDateTime.Append(time.Unix(additional.EpochStartDateTime, 0))
}
