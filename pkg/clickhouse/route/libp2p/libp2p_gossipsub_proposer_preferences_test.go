package libp2p

import (
	"testing"

	"github.com/ethpandaops/xatu/pkg/clickhouse/route/testfixture"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

func TestSnapshot_libp2p_gossipsub_proposer_preferences(t *testing.T) {
	if len(libp2pGossipsubProposerPreferencesEventNames) == 0 {
		t.Skip("no event names registered for libp2p_gossipsub_proposer_preferences")
	}

	testfixture.AssertSnapshot(t, newlibp2pGossipsubProposerPreferencesBatch(), &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     libp2pGossipsubProposerPreferencesEventNames[0],
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
