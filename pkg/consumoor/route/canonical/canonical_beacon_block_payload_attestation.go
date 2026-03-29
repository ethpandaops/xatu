package canonical

import (
	"fmt"
	"time"

	"github.com/ethpandaops/xatu/pkg/consumoor/route"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

var canonicalBeaconBlockPayloadAttestationEventNames = []xatu.Event_Name{
	xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_PAYLOAD_ATTESTATION,
}

func init() {
	r, err := route.NewStaticRoute(
		canonicalBeaconBlockPayloadAttestationTableName,
		canonicalBeaconBlockPayloadAttestationEventNames,
		func() route.ColumnarBatch { return newcanonicalBeaconBlockPayloadAttestationBatch() },
	)
	if err != nil {
		route.RecordError(err)

		return
	}

	if err := route.Register(r); err != nil {
		route.RecordError(err)
	}
}

func (b *canonicalBeaconBlockPayloadAttestationBatch) FlattenTo(
	event *xatu.DecoratedEvent,
) error {
	if event == nil || event.GetEvent() == nil {
		return nil
	}

	if event.GetEthV2BeaconBlockPayloadAttestation() == nil {
		return fmt.Errorf("nil eth_v2_beacon_block_payload_attestation payload: %w", route.ErrInvalidEvent)
	}

	b.appendRuntime(event)
	b.appendMetadata(event)
	b.appendPayload(event)
	b.appendAdditionalData(event)
	b.rows++

	return nil
}

func (b *canonicalBeaconBlockPayloadAttestationBatch) appendRuntime(_ *xatu.DecoratedEvent) {
	b.UpdatedDateTime.Append(time.Now())
}

func (b *canonicalBeaconBlockPayloadAttestationBatch) appendPayload(event *xatu.DecoratedEvent) {
	attestation := event.GetEthV2BeaconBlockPayloadAttestation()

	data := attestation.GetData()
	if data != nil {
		b.BeaconBlockRoot.Append([]byte(data.GetBeaconBlockRoot()))
		b.PayloadPresent.Append(data.GetPayloadPresent())
		b.BlobDataAvailable.Append(data.GetBlobDataAvailable())
	} else {
		b.BeaconBlockRoot.Append(nil)
		b.PayloadPresent.Append(false)
		b.BlobDataAvailable.Append(false)
	}

	b.AggregationBits.Append(attestation.GetAggregationBits())

	// TODO(epbs): AttestingValidatorCount and Position are not available on the PayloadAttestation
	// proto type. These require Additional*Data proto definitions to be populated by cannon.
	// For now, zero-fill these fields.
	b.AttestingValidatorCount.Append(0)
	b.Position.Append(0)
}

func (b *canonicalBeaconBlockPayloadAttestationBatch) appendAdditionalData(
	event *xatu.DecoratedEvent,
) {
	additional := event.GetMeta().GetClient().GetEthV2BeaconBlockPayloadAttestation()
	if additional == nil {
		b.Slot.Append(0)
		b.SlotStartDateTime.Append(time.Time{})
		b.Epoch.Append(0)
		b.EpochStartDateTime.Append(time.Time{})
		b.BlockRoot.Append(nil)
		b.BlockVersion.Append("")

		return
	}

	appendBlockIdentifier(additional.GetBlock(),
		&b.Slot, &b.SlotStartDateTime, &b.Epoch, &b.EpochStartDateTime, &b.BlockVersion, &b.BlockRoot)
}
