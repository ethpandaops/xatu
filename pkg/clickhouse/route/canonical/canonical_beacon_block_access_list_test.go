package canonical

import (
	"testing"

	"github.com/ethpandaops/xatu/pkg/clickhouse/route/testfixture"
	ethv1 "github.com/ethpandaops/xatu/pkg/proto/eth/v1"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func TestSnapshot_canonical_beacon_block_access_list(t *testing.T) {
	testfixture.AssertSnapshot(t, newcanonicalBeaconBlockAccessListBatch(), &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_ACCESS_LIST,
			DateTime: testfixture.TS(),
			Id:       "cbal-1",
		},
		Meta: testfixture.MetaWithAdditional(&xatu.ClientMeta{
			AdditionalData: &xatu.ClientMeta_EthV2BeaconBlockAccessList{
				EthV2BeaconBlockAccessList: &xatu.ClientMeta_AdditionalEthV2BeaconBlockAccessListData{
					Block: &xatu.BlockIdentifier{
						Epoch: testfixture.EpochAdditional(),
					},
				},
			},
		}),
		Data: &xatu.DecoratedEvent_EthV2BeaconBlockAccessList{
			EthV2BeaconBlockAccessList: &ethv1.BlockAccessListChange{
				Address:          wrapperspb.String("0x1234567890abcdef1234567890abcdef12345678"),
				ChangeType:       "storage",
				BlockAccessIndex: wrapperspb.UInt32(5),
				StorageKey:       wrapperspb.String("0xabcdef"),
				NewValue:         wrapperspb.String("0xdeadbeef"),
			},
		},
	}, 1, map[string]any{
		"address":            "0x1234567890abcdef1234567890abcdef12345678",
		"change_type":        "storage",
		"block_access_index": uint32(5),
		"storage_key":        "0xabcdef",
		"new_value":          "0xdeadbeef",
	})
}

func TestSnapshot_canonical_beacon_block_access_list_balance(t *testing.T) {
	testfixture.AssertSnapshot(t, newcanonicalBeaconBlockAccessListBatch(), &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_ACCESS_LIST,
			DateTime: testfixture.TS(),
			Id:       "cbal-2",
		},
		Meta: testfixture.MetaWithAdditional(&xatu.ClientMeta{
			AdditionalData: &xatu.ClientMeta_EthV2BeaconBlockAccessList{
				EthV2BeaconBlockAccessList: &xatu.ClientMeta_AdditionalEthV2BeaconBlockAccessListData{
					Block: &xatu.BlockIdentifier{
						Epoch: testfixture.EpochAdditional(),
					},
				},
			},
		}),
		Data: &xatu.DecoratedEvent_EthV2BeaconBlockAccessList{
			EthV2BeaconBlockAccessList: &ethv1.BlockAccessListChange{
				Address:          wrapperspb.String("0xaabbccddee112233445566778899aabbccddeeff"),
				ChangeType:       "balance",
				BlockAccessIndex: wrapperspb.UInt32(2),
				NewValue:         wrapperspb.String("1000000000000000000"),
			},
		},
	}, 1, map[string]any{
		"address":            "0xaabbccddee112233445566778899aabbccddeeff",
		"change_type":        "balance",
		"block_access_index": uint32(2),
		"storage_key":        "",
		"new_value":          "1000000000000000000",
	})
}
