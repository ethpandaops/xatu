package beacon

import (
	"testing"

	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/ethpandaops/xatu/pkg/clickhouse/route/testfixture"
	ethv1 "github.com/ethpandaops/xatu/pkg/proto/eth/v1"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

func TestSnapshot_beacon_api_eth_v1_events_payload_attestation(t *testing.T) {
	if len(beaconApiEthV1EventsPayloadAttestationEventNames) == 0 {
		t.Skip("no event names registered for beacon_api_eth_v1_events_payload_attestation")
	}

	const (
		colValidatorIndex    = "validator_index"
		colBeaconBlockRoot   = "beacon_block_root"
		colPayloadPresent    = "payload_present"
		colBlobDataAvailable = "blob_data_available"
	)

	beaconBlockRoot := repeatHex("5d", 32)

	testfixture.AssertSnapshot(t, newbeaconApiEthV1EventsPayloadAttestationBatch(), &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     beaconApiEthV1EventsPayloadAttestationEventNames[0],
			DateTime: testfixture.TS(),
			Id:       testfixture.SnapshotID,
		},
		Meta: testfixture.MetaWithAdditional(&xatu.ClientMeta{
			AdditionalData: &xatu.ClientMeta_EthV1EventsPayloadAttestation{
				EthV1EventsPayloadAttestation: &xatu.ClientMeta_AdditionalEthV1EventsPayloadAttestationData{
					Slot:        testfixture.SlotEpochAdditional(),
					Epoch:       testfixture.EpochAdditional(),
					Propagation: testfixture.PropagationAdditional(),
				},
			},
		}),
		Data: &xatu.DecoratedEvent_EthV1EventsPayloadAttestation{
			EthV1EventsPayloadAttestation: &ethv1.PayloadAttestationMessage{
				ValidatorIndex: wrapperspb.UInt64(2891),
				Data: &ethv1.PayloadAttestationData{
					BeaconBlockRoot:   beaconBlockRoot,
					Slot:              wrapperspb.UInt64(48752),
					PayloadPresent:    true,
					BlobDataAvailable: true,
				},
				Signature: repeatHex("53", 96),
			},
		},
	}, 1, map[string]any{
		testfixture.MetaClientNameKey: testfixture.MetaClientName,
		colValidatorIndex:             uint32(2891),
		colBeaconBlockRoot:            beaconBlockRoot,
		colPayloadPresent:             true,
		colBlobDataAvailable:          true,
	})
}
