package route

import "github.com/ethpandaops/xatu/pkg/proto/xatu"

// intentionallyUnsupportedEvents documents event types consumoor deliberately
// does not flatten into ClickHouse tables.
var intentionallyUnsupportedEvents = map[xatu.Event_Name]string{
	xatu.Event_BEACON_API_ETH_V1_DEBUG_FORK_CHOICE:             "debug stream not modeled as table output",
	xatu.Event_BEACON_API_ETH_V1_DEBUG_FORK_CHOICE_V2:          "debug stream not modeled as table output",
	xatu.Event_BEACON_API_ETH_V1_DEBUG_FORK_CHOICE_REORG:       "debug stream not modeled as table output",
	xatu.Event_BEACON_API_ETH_V1_DEBUG_FORK_CHOICE_REORG_V2:    "debug stream not modeled as table output",
	xatu.Event_BEACON_API_ETH_V1_EVENTS_ATTESTATION:            "deprecated in favor of V2 event",
	xatu.Event_BEACON_API_ETH_V1_EVENTS_BLOCK:                  "deprecated in favor of V2 event",
	xatu.Event_BEACON_API_ETH_V1_EVENTS_CHAIN_REORG:            "deprecated in favor of V2 event",
	xatu.Event_BEACON_API_ETH_V1_EVENTS_CONTRIBUTION_AND_PROOF: "deprecated in favor of V2 event",
	xatu.Event_BEACON_API_ETH_V1_EVENTS_FINALIZED_CHECKPOINT:   "deprecated in favor of V2 event",
	xatu.Event_BEACON_API_ETH_V1_EVENTS_HEAD:                   "deprecated in favor of V2 event",
	xatu.Event_BEACON_API_ETH_V1_EVENTS_VOLUNTARY_EXIT:         "deprecated in favor of V2 event",
	xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK:                  "deprecated in favor of V2 event",
	xatu.Event_BEACON_P2P_ATTESTATION:                          "legacy event path not consumed by consumoor",
}

// UnsupportedReason returns the documented reason when an event is
// intentionally unsupported by consumoor flatteners.
func UnsupportedReason(event xatu.Event_Name) (string, bool) {
	reason, ok := intentionallyUnsupportedEvents[event]

	return reason, ok
}

// UnsupportedEvents returns a copy of intentionally unsupported event reasons.
func UnsupportedEvents() map[xatu.Event_Name]string {
	out := make(map[xatu.Event_Name]string, len(intentionallyUnsupportedEvents))
	for event, reason := range intentionallyUnsupportedEvents {
		out[event] = reason
	}

	return out
}
