package consumoor

import (
	"testing"

	"github.com/ethpandaops/xatu/pkg/clickhouse/route"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubRoute struct {
	table  string
	events []xatu.Event_Name
}

func (r stubRoute) EventNames() []xatu.Event_Name             { return r.events }
func (r stubRoute) TableName() string                         { return r.table }
func (r stubRoute) ShouldProcess(_ *xatu.DecoratedEvent) bool { return true }
func (r stubRoute) NewBatch() route.ColumnarBatch             { return nil }

func TestValidateDisabledTablesAgainstRoutes(t *testing.T) {
	routes := []route.Route{
		stubRoute{table: "libp2p_peer"},
		stubRoute{table: "libp2p_connected"},
	}

	t.Run("nil disabled set is fine", func(t *testing.T) {
		require.NoError(t, validateDisabledTablesAgainstRoutes(nil, routes))
	})

	t.Run("accepts known table", func(t *testing.T) {
		require.NoError(t, validateDisabledTablesAgainstRoutes(
			map[string]struct{}{"libp2p_peer": {}}, routes,
		))
	})

	t.Run("rejects typo", func(t *testing.T) {
		err := validateDisabledTablesAgainstRoutes(
			map[string]struct{}{"lib2p2_peer": {}, "libp2p_peer": {}},
			routes,
		)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "lib2p2_peer")
		assert.NotContains(t, err.Error(), "libp2p_peer,")
	})
}

func TestFilterRoutesByTable(t *testing.T) {
	routes := []route.Route{
		stubRoute{table: "libp2p_peer"},
		stubRoute{table: "libp2p_connected"},
		stubRoute{table: "libp2p_disconnected"},
	}

	got := filterRoutesByTable(routes, map[string]struct{}{"libp2p_peer": {}})

	require.Len(t, got, 2)
	assert.Equal(t, "libp2p_connected", got[0].TableName())
	assert.Equal(t, "libp2p_disconnected", got[1].TableName())
}

func TestFilterRoutesByTableEmptyDisabledReturnsAll(t *testing.T) {
	routes := []route.Route{
		stubRoute{table: "libp2p_peer"},
	}

	got := filterRoutesByTable(routes, nil)
	assert.Equal(t, routes, got)
}
