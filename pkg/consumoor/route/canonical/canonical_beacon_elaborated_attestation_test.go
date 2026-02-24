package canonical

import (
	"testing"

	"github.com/ethpandaops/xatu/pkg/consumoor/route/testfixture"
	ethv1 "github.com/ethpandaops/xatu/pkg/proto/eth/v1"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func TestSnapshot_canonical_beacon_elaborated_attestation(t *testing.T) {
	testfixture.AssertSnapshot(t, newcanonicalBeaconElaboratedAttestationBatch(), &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_ELABORATED_ATTESTATION,
			DateTime: testfixture.TS(),
			Id:       "cea-1",
		},
		Meta: testfixture.MetaWithAdditional(&xatu.ClientMeta{
			AdditionalData: &xatu.ClientMeta_EthV2BeaconBlockElaboratedAttestation{
				EthV2BeaconBlockElaboratedAttestation: &xatu.ClientMeta_AdditionalEthV2BeaconBlockElaboratedAttestationData{
					Epoch: testfixture.EpochAdditional(),
				},
			},
		}),
		Data: &xatu.DecoratedEvent_EthV2BeaconBlockElaboratedAttestation{
			EthV2BeaconBlockElaboratedAttestation: &ethv1.ElaboratedAttestation{
				ValidatorIndexes: []*wrapperspb.UInt64Value{
					wrapperspb.UInt64(11),
					wrapperspb.UInt64(22),
				},
			},
		},
	}, 1, map[string]any{
		"meta_client_name": "test-client",
	})
}
