package route_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/ethpandaops/xatu/pkg/clickhouse/route"
	tabledefs "github.com/ethpandaops/xatu/pkg/clickhouse/route/all"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

// TestCompletenessMinimalFlatten verifies that every registered route can
// flatten a minimal event without panicking. This is the cross-cutting
// smoke test — per-table snapshot correctness lives in co-located
// tables/<domain>/<table>_test.go files.
func TestCompletenessMinimalFlatten(t *testing.T) {
	allRoutes, err := tabledefs.All()
	require.NoError(t, err)
	require.NotEmpty(t, allRoutes, "no routes registered")

	for _, r := range allRoutes {
		t.Run(r.TableName(), func(t *testing.T) {
			// Every route must have at least one event name.
			require.NotEmpty(t, r.EventNames(),
				"route %s has no event names", r.TableName())

			batch := r.NewBatch()
			require.NotNil(t, batch)

			eventName := r.EventNames()[0]

			// Flatten a minimal event — must not panic or error.
			event := &xatu.DecoratedEvent{
				Event: &xatu.Event{
					Name:     eventName,
					DateTime: timestamppb.Now(),
					Id:       fmt.Sprintf("completeness-%s", r.TableName()),
				},
			}

			err := batch.FlattenTo(event)
			if errors.Is(err, route.ErrInvalidEvent) {
				// Nil-payload events are expected to be rejected with
				// ErrInvalidEvent — the table writer skips these.
				return
			}

			require.NoError(t, err)

			// If the route produced rows, verify column alignment.
			if batch.Rows() > 0 {
				snapper, ok := batch.(route.Snapshotter)
				if ok {
					snap := snapper.Snapshot()
					assert.Len(t, snap, batch.Rows())
				}

				for _, col := range batch.Input() {
					assert.Equalf(t, batch.Rows(), col.Data.Rows(),
						"column %q misaligned in table %s", col.Name, r.TableName())
				}
			}
		})
	}
}

// supersededByV2 is the shared justification for V1 enum values that are
// kept for proto wire compatibility but never get a ClickHouse route.
const supersededByV2 = "deprecated, superseded by _V2"

// eventNamesWithoutRoute lists Event_Name values that intentionally have no
// ClickHouse route. New entries here must come with a justification —
// typically a deprecated V1 event superseded by a V2 sibling, or an event
// that never lands in ClickHouse.
var eventNamesWithoutRoute = map[xatu.Event_Name]string{
	xatu.Event_BEACON_API_ETH_V1_EVENTS_UNKNOWN:                "sentinel zero value",
	xatu.Event_LIBP2P_TRACE_UNKNOWN:                            "sentinel for unhandled libp2p traces",
	xatu.Event_BEACON_API_ETH_V1_EVENTS_BLOCK:                  supersededByV2,
	xatu.Event_BEACON_API_ETH_V1_EVENTS_CHAIN_REORG:            supersededByV2,
	xatu.Event_BEACON_API_ETH_V1_EVENTS_FINALIZED_CHECKPOINT:   supersededByV2,
	xatu.Event_BEACON_API_ETH_V1_EVENTS_HEAD:                   supersededByV2,
	xatu.Event_BEACON_API_ETH_V1_EVENTS_VOLUNTARY_EXIT:         supersededByV2,
	xatu.Event_BEACON_API_ETH_V1_EVENTS_ATTESTATION:            supersededByV2,
	xatu.Event_BEACON_API_ETH_V1_EVENTS_CONTRIBUTION_AND_PROOF: supersededByV2,
	xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK:                  supersededByV2,
	xatu.Event_BEACON_API_ETH_V1_DEBUG_FORK_CHOICE:             supersededByV2,
	xatu.Event_BEACON_API_ETH_V1_DEBUG_FORK_CHOICE_REORG:       supersededByV2,
	xatu.Event_BEACON_API_ETH_V1_DEBUG_FORK_CHOICE_V2:          "debug fork choice is sentry-tracked but not ClickHouse-bound",
	xatu.Event_BEACON_API_ETH_V1_DEBUG_FORK_CHOICE_REORG_V2:    "debug fork choice is sentry-tracked but not ClickHouse-bound",
	xatu.Event_BEACON_P2P_ATTESTATION:                          "sentry-only legacy event without a ClickHouse table",
}

// TestCompletenessEveryEventNameHasRoute walks every Event_Name enum value and
// asserts each is either registered with a ClickHouse route or explicitly
// listed as not-routed. Adding a new Event_Name without wiring a route (or
// allowlist entry) trips this test, forcing a conscious decision.
func TestCompletenessEveryEventNameHasRoute(t *testing.T) {
	allRoutes, err := tabledefs.All()
	require.NoError(t, err)

	routedEventNames := map[xatu.Event_Name][]string{}

	for _, r := range allRoutes {
		for _, name := range r.EventNames() {
			routedEventNames[name] = append(routedEventNames[name], r.TableName())
		}
	}

	for value, label := range xatu.Event_Name_name {
		eventName := xatu.Event_Name(value)

		t.Run(label, func(t *testing.T) {
			_, routed := routedEventNames[eventName]
			_, allowlisted := eventNamesWithoutRoute[eventName]

			if !routed && !allowlisted {
				t.Fatalf("Event_Name %s has no ClickHouse route and is not in the eventNamesWithoutRoute allowlist — wire a route or add a justification entry", label)
			}

			if routed && allowlisted {
				t.Fatalf("Event_Name %s is both routed (tables %v) and allowlisted as not-routed — remove the allowlist entry", label, routedEventNames[eventName])
			}
		})
	}
}
