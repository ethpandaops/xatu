package execution

import (
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/ethpandaops/xatu/pkg/clickhouse/route"
	"github.com/ethpandaops/xatu/pkg/clickhouse/route/testfixture"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

func TestSnapshot_mempool_transaction_removed(t *testing.T) {
	testfixture.AssertSnapshot(t, newmempoolTransactionRemovedBatch(), &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_MEMPOOL_TRANSACTION_REMOVED,
			DateTime: testfixture.TS(),
			Id:       "mtr-1",
		},
		Meta: testfixture.BaseMeta(),
		Data: &xatu.DecoratedEvent_MempoolTransactionRemoved{
			MempoolTransactionRemoved: &xatu.MempoolTransactionRemoved{
				Hash:                 "0xf04aa8bf1ba4989dddbf1976fce0e2d9eb6a0b405500477c423d1cd853331873",
				From:                 "0x1111111111111111111111111111111111111111",
				Nonce:                wrapperspb.UInt64(6),
				Type:                 wrapperspb.UInt32(2),
				Gas:                  wrapperspb.UInt64(21000),
				GasTipCap:            "1000000000",
				GasFeeCap:            "30000000000",
				Reason:               "replaced",
				Pool:                 "pending",
				ReplacementHash:      "0x88a4c68fdb528cdbb40bb10ee0c640c8026da927564a6640098fd30613e70433",
				ReplacementGasTipCap: "1100000000",
				ReplacementGasFeeCap: "33000000000",
				FirstSeen:            testfixture.TS(),
				LastSeen:             testfixture.TS(),
			},
		},
	}, 1, map[string]any{
		"hash":                    "0xf04aa8bf1ba4989dddbf1976fce0e2d9eb6a0b405500477c423d1cd853331873",
		"nonce":                   uint64(6),
		"reason":                  "replaced",
		"pool":                    "pending",
		"replacement_hash":        "0x88a4c68fdb528cdbb40bb10ee0c640c8026da927564a6640098fd30613e70433",
		"to":                      nil,
		"blob_gas_fee_cap":        nil,
		"account_nonce":           nil,
		"meta_client_name":        "test-client",
		"replacement_gas_fee_cap": "33000000000",
	})
}

func TestMempoolTransactionRemovedRejectsBadFeeWithoutAppending(t *testing.T) {
	batch := newmempoolTransactionRemovedBatch()

	err := batch.FlattenTo(&xatu.DecoratedEvent{
		Event: &xatu.Event{Name: xatu.Event_MEMPOOL_TRANSACTION_REMOVED, DateTime: testfixture.TS()},
		Meta:  testfixture.BaseMeta(),
		Data: &xatu.DecoratedEvent_MempoolTransactionRemoved{
			MempoolTransactionRemoved: &xatu.MempoolTransactionRemoved{
				Hash:      "0xf04aa8bf1ba4989dddbf1976fce0e2d9eb6a0b405500477c423d1cd853331873",
				From:      "0x1111111111111111111111111111111111111111",
				Reason:    "dropped",
				GasTipCap: "1",
				GasFeeCap: "not-a-number",
			},
		},
	})

	require.ErrorIs(t, err, route.ErrInvalidEvent)
	require.Equal(t, 0, batch.Rows())
	require.Equal(t, 0, batch.UpdatedDateTime.Rows(), "no column was appended")
}
