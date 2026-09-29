package beacon

import (
	"fmt"
	"time"

	"github.com/ethpandaops/xatu/pkg/clickhouse/route"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

var beaconApiEthV1EventsPayloadAttestationEventNames = []xatu.Event_Name{
	xatu.Event_BEACON_API_ETH_V1_EVENTS_PAYLOAD_ATTESTATION,
}

func init() {
	r, err := route.NewStaticRoute(
		beaconApiEthV1EventsPayloadAttestationTableName,
		beaconApiEthV1EventsPayloadAttestationEventNames,
		func() route.ColumnarBatch { return newbeaconApiEthV1EventsPayloadAttestationBatch() },
	)
	if err != nil {
		route.RecordError(err)

		return
	}

	if err := route.Register(r); err != nil {
		route.RecordError(err)
	}
}

func (b *beaconApiEthV1EventsPayloadAttestationBatch) FlattenTo(
	event *xatu.DecoratedEvent,
) error {
	if event == nil || event.GetEvent() == nil {
		return nil
	}

	if event.GetEthV1EventsPayloadAttestation() == nil {
		return fmt.Errorf("nil eth_v1_events_payload_attestation payload: %w", route.ErrInvalidEvent)
	}

	b.appendRuntime(event)
	b.appendMetadata(event)
	b.appendPayload(event)
	b.appendAdditionalData(event)
	b.rows++

	return nil
}

func (b *beaconApiEthV1EventsPayloadAttestationBatch) appendRuntime(event *xatu.DecoratedEvent) {
	b.UpdatedDateTime.Append(time.Now())

	if ts := event.GetEvent().GetDateTime(); ts != nil {
		b.EventDateTime.Append(ts.AsTime())
	} else {
		b.EventDateTime.Append(time.Time{})
	}
}

//nolint:gosec // G115: proto uint64 values are bounded by ClickHouse column schema
func (b *beaconApiEthV1EventsPayloadAttestationBatch) appendPayload(event *xatu.DecoratedEvent) {
	msg := event.GetEthV1EventsPayloadAttestation()

	if validatorIndex := msg.GetValidatorIndex(); validatorIndex != nil {
		b.ValidatorIndex.Append(uint32(validatorIndex.GetValue()))
	} else {
		b.ValidatorIndex.Append(0)
	}

	data := msg.GetData()
	if data != nil {
		b.BeaconBlockRoot.Append([]byte(data.GetBeaconBlockRoot()))
		b.PayloadPresent.Append(data.GetPayloadPresent())
		b.BlobDataAvailable.Append(data.GetBlobDataAvailable())
	} else {
		b.BeaconBlockRoot.Append(nil)
		b.PayloadPresent.Append(false)
		b.BlobDataAvailable.Append(false)
	}
}

func (b *beaconApiEthV1EventsPayloadAttestationBatch) appendAdditionalData(
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
	additional := extractBeaconSlotEpochPropagation(client.GetEthV1EventsPayloadAttestation())

	b.Slot.Append(uint32(additional.Slot)) //nolint:gosec // slot fits uint32
	b.SlotStartDateTime.Append(time.Unix(additional.SlotStartDateTime, 0))
	b.PropagationSlotStartDiff.Append(uint32(additional.PropagationSlotStartDiff)) //nolint:gosec // propagation diff fits uint32
	b.Epoch.Append(uint32(additional.Epoch))                                       //nolint:gosec // epoch fits uint32
	b.EpochStartDateTime.Append(time.Unix(additional.EpochStartDateTime, 0))
}
