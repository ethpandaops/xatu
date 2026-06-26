package canonical

import (
	"testing"

	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/ethpandaops/xatu/pkg/clickhouse/route/testfixture"
	ethv1 "github.com/ethpandaops/xatu/pkg/proto/eth/v1"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

func TestSnapshot_canonical_beacon_block_execution_transaction(t *testing.T) {
	testfixture.AssertSnapshot(t, newcanonicalBeaconBlockExecutionTransactionBatch(), &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_EXECUTION_TRANSACTION,
			DateTime: testfixture.TS(),
			Id:       "cbtx-1",
		},
		Meta: testfixture.MetaWithAdditional(&xatu.ClientMeta{
			AdditionalData: &xatu.ClientMeta_EthV2BeaconBlockExecutionTransaction{
				EthV2BeaconBlockExecutionTransaction: &xatu.ClientMeta_AdditionalEthV2BeaconBlockExecutionTransactionData{
					Block: &xatu.BlockIdentifier{
						Slot:    testfixture.SlotEpochAdditional(),
						Epoch:   testfixture.EpochAdditional(),
						Version: "deneb",
						Root:    "0x1111111111111111111111111111111111111111111111111111111111111111",
					},
					Size: "150",
				},
			},
		}),
		Data: &xatu.DecoratedEvent_EthV2BeaconBlockExecutionTransaction{
			EthV2BeaconBlockExecutionTransaction: &ethv1.Transaction{
				Hash:     "0xtxhash",
				From:     "0xfrom",
				To:       "0xto",
				Nonce:    wrapperspb.UInt64(1),
				Gas:      wrapperspb.UInt64(21000),
				Type:     wrapperspb.UInt32(2),
				GasPrice: "1000000000",
			},
		},
	}, 1, map[string]any{
		"hash": "0xtxhash",
		"from": "0xfrom",
	})
}
