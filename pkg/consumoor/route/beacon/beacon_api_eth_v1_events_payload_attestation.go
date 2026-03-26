package beacon

import (
	"fmt"

	"github.com/ethpandaops/xatu/pkg/consumoor/route"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

// TODO: Add the xatu.Event_* name(s) that route events to the beacon_api_eth_v1_events_payload_attestation table.
var beaconApiEthV1EventsPayloadAttestationEventNames = []xatu.Event_Name{}

func init() {
	route, err := route.NewStaticRoute(
		beaconApiEthV1EventsPayloadAttestationTableName,
		beaconApiEthV1EventsPayloadAttestationEventNames,
		func() route.ColumnarBatch { return newbeaconApiEthV1EventsPayloadAttestationBatch() },
	)
	if err != nil {
		route.RecordError(err)

		return
	}

	if err := route.Register(route); err != nil {
		route.RecordError(err)
	}
}

func (b *beaconApiEthV1EventsPayloadAttestationBatch) FlattenTo(
	event *xatu.DecoratedEvent,
) error {
	// TODO: Implement this method to flatten the event into columnar batch columns.
	// The generated .gen.go file contains the available column fields for this table.
	//
	// Typical structure:
	//   b.appendRuntime(event)
	//   b.appendMetadata(event)
	//   b.appendPayload(event)
	//   b.rows++
	//   return nil
	return fmt.Errorf("beaconApiEthV1EventsPayloadAttestation: FlattenTo not implemented")
}
