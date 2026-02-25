package execution

import (
	"testing"

	"github.com/ethpandaops/xatu/pkg/consumoor/route/testfixture"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func TestSnapshot_consensus_engine_api_new_payload(t *testing.T) {
	testfixture.AssertSnapshot(t, newconsensusEngineApiNewPayloadBatch(), &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_CONSENSUS_ENGINE_API_NEW_PAYLOAD,
			DateTime: testfixture.TS(),
			Id:       "cenp-1",
		},
		Meta: testfixture.MetaWithAdditional(&xatu.ClientMeta{
			AdditionalData: &xatu.ClientMeta_ConsensusEngineApiNewPayload{
				ConsensusEngineApiNewPayload: &xatu.ClientMeta_AdditionalConsensusEngineAPINewPayloadData{},
			},
		}),
		Data: &xatu.DecoratedEvent_ConsensusEngineApiNewPayload{
			ConsensusEngineApiNewPayload: &xatu.ConsensusEngineAPINewPayload{
				DurationMs:    wrapperspb.UInt64(10),
				Slot:          wrapperspb.UInt64(100),
				ProposerIndex: wrapperspb.UInt64(42),
				BlockNumber:   wrapperspb.UInt64(1000),
				GasUsed:       wrapperspb.UInt64(21000),
				GasLimit:      wrapperspb.UInt64(30000000),
				TxCount:       wrapperspb.UInt32(5),
				BlobCount:     wrapperspb.UInt32(2),
			},
		},
	}, 1, map[string]any{
		"meta_client_name": "test-client",
	})
}
