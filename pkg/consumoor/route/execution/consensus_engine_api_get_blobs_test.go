package execution

import (
	"testing"

	"github.com/ethpandaops/xatu/pkg/consumoor/route/testfixture"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func TestSnapshot_consensus_engine_api_get_blobs(t *testing.T) {
	testfixture.AssertSnapshot(t, newconsensusEngineApiGetBlobsBatch(), &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_CONSENSUS_ENGINE_API_GET_BLOBS,
			DateTime: testfixture.TS(),
			Id:       "cegb-1",
		},
		Meta: testfixture.MetaWithAdditional(&xatu.ClientMeta{
			AdditionalData: &xatu.ClientMeta_ConsensusEngineApiGetBlobs{
				ConsensusEngineApiGetBlobs: &xatu.ClientMeta_AdditionalConsensusEngineAPIGetBlobsData{},
			},
		}),
		Data: &xatu.DecoratedEvent_ConsensusEngineApiGetBlobs{
			ConsensusEngineApiGetBlobs: &xatu.ConsensusEngineAPIGetBlobs{
				DurationMs:     wrapperspb.UInt64(10),
				Slot:           wrapperspb.UInt64(100),
				RequestedCount: wrapperspb.UInt32(3),
				ReturnedCount:  wrapperspb.UInt32(2),
			},
		},
	}, 1, map[string]any{
		"meta_client_name": "test-client",
	})
}
