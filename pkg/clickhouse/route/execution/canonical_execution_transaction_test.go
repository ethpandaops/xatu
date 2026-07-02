package execution

import (
	"testing"

	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/ethpandaops/xatu/pkg/clickhouse/route/testfixture"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

const (
	colValue    = "value"
	colGasPrice = "gas_price"
	colSuccess  = "success"
	colTxType   = "transaction_type"
	colTxHash   = "transaction_hash"
)

func TestSnapshot_canonical_execution_transaction(t *testing.T) {
	testfixture.AssertSnapshot(t, newCanonicalExecutionTransactionBatch(), &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_EXECUTION_CANONICAL_TRANSACTION,
			DateTime: testfixture.TS(),
			Id:       "cet-1",
		},
		Meta: testfixture.MetaWithAdditional(&xatu.ClientMeta{}),
		Data: &xatu.DecoratedEvent_ExecutionCanonicalTransaction{
			ExecutionCanonicalTransaction: &xatu.ExecutionCanonicalTransaction{
				Transactions: []*xatu.ExecutionTransaction{
					{
						BlockNumber:          22000000,
						TransactionIndex:     7,
						TransactionHash:      testBlockHashHex,
						Nonce:                42,
						FromAddress:          "0x4838b106fce9647bdf1e7877bf73ce8b0bad5f97",
						ToAddress:            wrapperspb.String("0x95222290dd7278aa3ddd389cc1e1d165cc4bafe5"),
						Value:                "1000000000000000000",
						Input:                wrapperspb.String("0xabcdef"),
						GasLimit:             21000,
						GasUsed:              21000,
						GasPrice:             30000000000,
						TransactionType:      2,
						MaxPriorityFeePerGas: 1500000000,
						MaxFeePerGas:         40000000000,
						Success:              true,
						NInputBytes:          3,
						NInputZeroBytes:      0,
						NInputNonzeroBytes:   3,
					},
				},
			},
		},
	}, 1, map[string]any{
		colBlockNumber: uint64(22000000),
		colTxHash:      testBlockHashHex,
		colValue:       "1000000000000000000",
		colGasPrice:    "30000000000",
		colSuccess:     true,
		colTxType:      uint8(2),
		"from_address": "0x4838b106fce9647bdf1e7877bf73ce8b0bad5f97",
		"to_address":   "0x95222290dd7278aa3ddd389cc1e1d165cc4bafe5",
		"input":        "0xabcdef",
		colMetaNetwork: "mainnet",
	})
}
