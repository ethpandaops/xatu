package canonical

import (
	"testing"

	"github.com/ClickHouse/ch-go/proto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/ethpandaops/xatu/pkg/clickhouse/route"
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
		"hash":                      "0xtxhash",
		"from":                      "0xfrom",
		"authorization_chain_ids":   []proto.UInt256(nil),
		"authorization_addresses":   []string(nil),
		"authorization_nonces":      []uint64(nil),
		"authorization_authorities": []string(nil),
	})
}

func TestSnapshot_canonical_beacon_block_execution_transaction_set_code(t *testing.T) {
	mainnet, err := route.ParseUInt256("1")
	require.NoError(t, err)

	testfixture.AssertSnapshot(t, newcanonicalBeaconBlockExecutionTransactionBatch(), &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_EXECUTION_TRANSACTION,
			DateTime: testfixture.TS(),
			Id:       "cbtx-2",
		},
		Meta: testfixture.MetaWithAdditional(&xatu.ClientMeta{
			AdditionalData: &xatu.ClientMeta_EthV2BeaconBlockExecutionTransaction{
				EthV2BeaconBlockExecutionTransaction: &xatu.ClientMeta_AdditionalEthV2BeaconBlockExecutionTransactionData{
					Block: &xatu.BlockIdentifier{
						Slot:    testfixture.SlotEpochAdditional(),
						Epoch:   testfixture.EpochAdditional(),
						Version: "fulu",
						Root:    "0x1111111111111111111111111111111111111111111111111111111111111111",
					},
					Size: "300",
				},
			},
		}),
		Data: &xatu.DecoratedEvent_EthV2BeaconBlockExecutionTransaction{
			EthV2BeaconBlockExecutionTransaction: &ethv1.Transaction{
				Hash:     "0xtxhash",
				From:     "0xfrom",
				Nonce:    wrapperspb.UInt64(1),
				Gas:      wrapperspb.UInt64(60000),
				Type:     wrapperspb.UInt32(4),
				GasPrice: "1000000000",
				AuthorizationList: []*ethv1.SetCodeAuthorization{
					{
						ChainId:   "1",
						Address:   "0x63c0c19a282a1B52b07dD5a65b58948A07DAE32B",
						Nonce:     wrapperspb.UInt64(7),
						Authority: "0x000000000000000000000000000000000000dEaD",
					},
					{
						ChainId: "0",
						Address: "0x0000000000000000000000000000000000000001",
						Nonce:   wrapperspb.UInt64(0),
					},
				},
			},
		},
	}, 1, map[string]any{
		"type":                    uint8(4),
		"authorization_chain_ids": []proto.UInt256{mainnet, {}},
		"authorization_addresses": []string{
			"0x63c0c19a282a1B52b07dD5a65b58948A07DAE32B",
			"0x0000000000000000000000000000000000000001",
		},
		"authorization_nonces":      []uint64{7, 0},
		"authorization_authorities": []string{"0x000000000000000000000000000000000000dEaD", ""},
	})
}

func TestCanonicalBeaconBlockExecutionTransactionRejectsBadAuthorizationChainID(t *testing.T) {
	batch := newcanonicalBeaconBlockExecutionTransactionBatch()

	err := batch.FlattenTo(&xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_EXECUTION_TRANSACTION,
			DateTime: testfixture.TS(),
			Id:       "cbtx-3",
		},
		Meta: testfixture.MetaWithAdditional(&xatu.ClientMeta{
			AdditionalData: &xatu.ClientMeta_EthV2BeaconBlockExecutionTransaction{
				EthV2BeaconBlockExecutionTransaction: &xatu.ClientMeta_AdditionalEthV2BeaconBlockExecutionTransactionData{
					Block: &xatu.BlockIdentifier{
						Slot:    testfixture.SlotEpochAdditional(),
						Epoch:   testfixture.EpochAdditional(),
						Version: "fulu",
						Root:    "0x1111111111111111111111111111111111111111111111111111111111111111",
					},
				},
			},
		}),
		Data: &xatu.DecoratedEvent_EthV2BeaconBlockExecutionTransaction{
			EthV2BeaconBlockExecutionTransaction: &ethv1.Transaction{
				Hash:              "0xtxhash",
				From:              "0xfrom",
				GasPrice:          "1",
				AuthorizationList: []*ethv1.SetCodeAuthorization{{ChainId: "not-a-number"}},
			},
		},
	})
	require.ErrorIs(t, err, route.ErrInvalidEvent)
	assert.Zero(t, batch.Rows())
}
