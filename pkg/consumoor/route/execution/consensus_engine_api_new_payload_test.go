package execution

import (
	"testing"

	"github.com/ethpandaops/xatu/pkg/consumoor/route/testfixture"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
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
			ConsensusEngineApiNewPayload: &xatu.ConsensusEngineAPINewPayload{},
		},
	}, 1, map[string]any{
		"meta_client_name": "test-client",
	})
}
