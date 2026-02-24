package route_test

import (
	"fmt"
	"testing"

	"github.com/ethpandaops/xatu/pkg/consumoor/route"
	tabledefs "github.com/ethpandaops/xatu/pkg/consumoor/route/all"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"
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
