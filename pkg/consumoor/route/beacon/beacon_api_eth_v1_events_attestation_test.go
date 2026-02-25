package beacon

import (
	"testing"

	"github.com/ethpandaops/xatu/pkg/consumoor/route/testfixture"
	ethv1 "github.com/ethpandaops/xatu/pkg/proto/eth/v1"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func TestSnapshot_beacon_api_eth_v1_events_attestation(t *testing.T) {
	testfixture.AssertSnapshot(t, newbeaconApiEthV1EventsAttestationBatch(), &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_BEACON_API_ETH_V1_EVENTS_ATTESTATION_V2,
			DateTime: testfixture.TS(),
			Id:       "attestation-1",
		},
		Meta: testfixture.MetaWithAdditional(&xatu.ClientMeta{
			AdditionalData: &xatu.ClientMeta_EthV1EventsAttestationV2{
				EthV1EventsAttestationV2: &xatu.ClientMeta_AdditionalEthV1EventsAttestationV2Data{
					Slot:  testfixture.SlotEpochAdditional(),
					Epoch: testfixture.EpochAdditional(),
				},
			},
		}),
		Data: &xatu.DecoratedEvent_EthV1EventsAttestationV2{
			EthV1EventsAttestationV2: &ethv1.AttestationV2{
				AggregationBits: "0xff",
				Signature:       "0xsig",
				Data: &ethv1.AttestationDataV2{
					Slot:            wrapperspb.UInt64(100),
					Index:           wrapperspb.UInt64(5),
					BeaconBlockRoot: "0xbbr",
					Source: &ethv1.CheckpointV2{
						Epoch: wrapperspb.UInt64(2),
						Root:  "0xsource",
					},
					Target: &ethv1.CheckpointV2{
						Epoch: wrapperspb.UInt64(3),
						Root:  "0xtarget",
					},
				},
			},
		},
	}, 1, map[string]any{
		"slot":              uint32(100),
		"committee_index":   "5",
		"beacon_block_root": "0xbbr",
		"source_epoch":      uint32(2),
		"target_epoch":      uint32(3),
		"meta_client_name":  "test-client",
		"meta_network_name": "mainnet",
	})
}
