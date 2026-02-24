package execution

import (
	"testing"

	"github.com/ethpandaops/xatu/pkg/consumoor/route/testfixture"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func TestSnapshot_mempool_transaction(t *testing.T) {
	testfixture.AssertSnapshot(t, newmempoolTransactionBatch(), &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_MEMPOOL_TRANSACTION_V2,
			DateTime: testfixture.TS(),
			Id:       "mtx-1",
		},
		Meta: testfixture.MetaWithAdditional(&xatu.ClientMeta{
			AdditionalData: &xatu.ClientMeta_MempoolTransactionV2{
				MempoolTransactionV2: &xatu.ClientMeta_AdditionalMempoolTransactionV2Data{
					Hash:  "0xhash",
					From:  "0xfrom",
					Nonce: wrapperspb.UInt64(42),
				},
			},
		}),
	}, 1, map[string]any{
		"hash": "0xhash",
		"from": "0xfrom",
	})
}
