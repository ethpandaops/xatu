package canonical

import (
	"strings"
	"testing"

	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/ethpandaops/xatu/pkg/clickhouse/route/testfixture"
	ethv1 "github.com/ethpandaops/xatu/pkg/proto/eth/v1"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

func TestSnapshot_canonical_beacon_block_payload_attestation(t *testing.T) {
	if len(canonicalBeaconBlockPayloadAttestationEventNames) == 0 {
		t.Skip("no event names registered for canonical_beacon_block_payload_attestation")
	}

	var (
		blockRoot       = "0x" + strings.Repeat("1a", 32)
		beaconBlockRoot = "0x" + strings.Repeat("6a", 32)
		// 0xff01 has nine bits set: eight PTC members in the first byte plus
		// one in the second.
		aggregationBits = "0xff01"
	)

	testfixture.AssertSnapshot(t, newcanonicalBeaconBlockPayloadAttestationBatch(), &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     canonicalBeaconBlockPayloadAttestationEventNames[0],
			DateTime: testfixture.TS(),
			Id:       testfixture.SnapshotID,
		},
		Meta: testfixture.MetaWithAdditional(&xatu.ClientMeta{
			AdditionalData: &xatu.ClientMeta_EthV2BeaconBlockPayloadAttestation{
				EthV2BeaconBlockPayloadAttestation: &xatu.ClientMeta_AdditionalEthV2BeaconBlockPayloadAttestationData{
					Block: &xatu.BlockIdentifier{
						Slot:    testfixture.SlotEpochAdditional(),
						Epoch:   testfixture.EpochAdditional(),
						Version: "gloas",
						Root:    blockRoot,
					},
					Position: wrapperspb.UInt32(1),
				},
			},
		}),
		Data: &xatu.DecoratedEvent_EthV2BeaconBlockPayloadAttestation{
			EthV2BeaconBlockPayloadAttestation: &ethv1.PayloadAttestation{
				AggregationBits: aggregationBits,
				Data: &ethv1.PayloadAttestationData{
					BeaconBlockRoot:   beaconBlockRoot,
					Slot:              wrapperspb.UInt64(48752),
					PayloadPresent:    true,
					BlobDataAvailable: true,
				},
				Signature: "0x" + strings.Repeat("5e", 96),
			},
		},
	}, 1, map[string]any{
		testfixture.MetaClientNameKey: testfixture.MetaClientName,
		"beacon_block_root":           beaconBlockRoot,
		"payload_present":             true,
		"blob_data_available":         true,
		"aggregation_bits":            aggregationBits,
		"attesting_validator_count":   uint32(9),
		"position":                    uint32(1),
		"block_root":                  blockRoot,
		"block_version":               "gloas",
		"slot":                        uint32(100),
		"epoch":                       uint32(3),
	})
}
