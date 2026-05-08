package libp2p

import (
	"testing"

	"github.com/ethpandaops/xatu/pkg/clickhouse/route/testfixture"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

func TestSnapshot_libp2p_gossipsub_execution_payload_envelope(t *testing.T) {
	if len(libp2pGossipsubExecutionPayloadEnvelopeEventNames) == 0 {
		t.Skip("no event names registered for libp2p_gossipsub_execution_payload_envelope")
	}

	testfixture.AssertSnapshot(t, newlibp2pGossipsubExecutionPayloadEnvelopeBatch(), &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     libp2pGossipsubExecutionPayloadEnvelopeEventNames[0],
			DateTime: testfixture.TS(),
			Id:       "snapshot-1",
		},
		Meta: testfixture.BaseMeta(),
		// TODO(epbs): Add event-specific Data field and MetaWithAdditional for richer assertions.
	}, 1, map[string]any{
		"meta_client_name": "test-client",
		// TODO(epbs): Add payload-specific column assertions.
	})
}
