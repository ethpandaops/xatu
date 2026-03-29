package canonical

import (
	"testing"

	"github.com/ethpandaops/xatu/pkg/consumoor/route/testfixture"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

func TestSnapshot_canonical_beacon_block_payload_attestation(t *testing.T) {
	if len(canonicalBeaconBlockPayloadAttestationEventNames) == 0 {
		t.Skip("no event names registered for canonical_beacon_block_payload_attestation")
	}

	testfixture.AssertSnapshot(t, newcanonicalBeaconBlockPayloadAttestationBatch(), &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     canonicalBeaconBlockPayloadAttestationEventNames[0],
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
