package node

import (
	"testing"

	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/ethpandaops/xatu/pkg/clickhouse/route/testfixture"
	noderecord "github.com/ethpandaops/xatu/pkg/proto/noderecord"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

func TestSnapshot_node_record_consensus(t *testing.T) {
	testfixture.AssertSnapshot(t, newnodeRecordConsensusBatch(), &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_NODE_RECORD_CONSENSUS,
			DateTime: testfixture.TS(),
			Id:       "nrc-1",
		},
		Meta: testfixture.BaseMeta(),
		Data: &xatu.DecoratedEvent_NodeRecordConsensus{
			NodeRecordConsensus: &noderecord.Consensus{
				PeerId:         wrapperspb.String(""),
				Timestamp:      wrapperspb.Int64(1705312800),
				FinalizedEpoch: wrapperspb.UInt64(100),
				HeadSlot:       wrapperspb.UInt64(3200),
			},
		},
	}, 1, map[string]any{
		"meta_client_name": "test-client",
	})
}
