package canonical

import (
	"testing"

	"github.com/ethpandaops/xatu/pkg/consumoor/route/testfixture"
	ethv1 "github.com/ethpandaops/xatu/pkg/proto/eth/v1"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func TestSnapshot_canonical_beacon_sync_committee(t *testing.T) {
	testfixture.AssertSnapshot(t, newcanonicalBeaconSyncCommitteeBatch(), &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_BEACON_API_ETH_V1_BEACON_SYNC_COMMITTEE,
			DateTime: testfixture.TS(),
			Id:       "csc-1",
		},
		Meta: testfixture.MetaWithAdditional(&xatu.ClientMeta{
			AdditionalData: &xatu.ClientMeta_EthV1BeaconSyncCommittee{
				EthV1BeaconSyncCommittee: &xatu.ClientMeta_AdditionalEthV1BeaconSyncCommitteeData{
					Epoch: testfixture.EpochAdditional(),
				},
			},
		}),
		Data: &xatu.DecoratedEvent_EthV1BeaconSyncCommittee{
			EthV1BeaconSyncCommittee: &xatu.SyncCommitteeData{
				SyncCommittee: &ethv1.SyncCommittee{
					ValidatorAggregates: []*ethv1.SyncCommitteeValidatorAggregate{
						{
							Validators: []*wrapperspb.UInt64Value{
								wrapperspb.UInt64(1),
							},
						},
					},
				},
			},
		},
	}, 1, map[string]any{
		"meta_client_name": "test-client",
	})
}
