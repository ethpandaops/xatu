package libp2p

import (
	"testing"

	"github.com/ethpandaops/xatu/pkg/clickhouse/route/testfixture"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

func TestSnapshot_libp2p_gossipsub_payload_attestation_message(t *testing.T) {
	if len(libp2pGossipsubPayloadAttestationMessageEventNames) == 0 {
		t.Skip("no event names registered for libp2p_gossipsub_payload_attestation_message")
	}

	testfixture.AssertSnapshot(t, newlibp2pGossipsubPayloadAttestationMessageBatch(), &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     libp2pGossipsubPayloadAttestationMessageEventNames[0],
			DateTime: testfixture.TS(),
			Id:       testfixture.SnapshotID,
		},
		Meta: testfixture.BaseMeta(),
		// TODO(epbs): Add event-specific Data field and MetaWithAdditional for richer assertions.
	}, 1, map[string]any{
		testfixture.MetaClientNameKey: testfixture.MetaClientName,
		// TODO(epbs): Add payload-specific column assertions.
	})
}
