package consumoor

import (
	"testing"

	"github.com/ethpandaops/xatu/pkg/clickhouse/route"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testTableLibp2pPeer    = "libp2p_peer"
	testTableLibp2pConnect = "libp2p_connected"
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
		stubRoute{table: testTableLibp2pPeer},
		stubRoute{table: testTableLibp2pConnect},
	}

	t.Run("nil disabled set is fine", func(t *testing.T) {
		require.NoError(t, validateDisabledTablesAgainstRoutes(nil, routes))
	})

	t.Run("accepts known table", func(t *testing.T) {
		require.NoError(t, validateDisabledTablesAgainstRoutes(
			map[string]struct{}{testTableLibp2pPeer: {}}, routes,
		))
	})

	t.Run("rejects typo", func(t *testing.T) {
		err := validateDisabledTablesAgainstRoutes(
			map[string]struct{}{"lib2p2_peer": {}, testTableLibp2pPeer: {}},
			routes,
		)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "lib2p2_peer")
		assert.NotContains(t, err.Error(), testTableLibp2pPeer+",")
	})
}

func TestFilterRoutesByTable(t *testing.T) {
	routes := []route.Route{
		stubRoute{table: testTableLibp2pPeer},
		stubRoute{table: testTableLibp2pConnect},
		stubRoute{table: "libp2p_disconnected"},
	}

	got := filterRoutesByTable(routes, map[string]struct{}{testTableLibp2pPeer: {}})

	require.Len(t, got, 2)
	assert.Equal(t, testTableLibp2pConnect, got[0].TableName())
	assert.Equal(t, "libp2p_disconnected", got[1].TableName())
}

func TestFilterRoutesByTableEmptyDisabledReturnsAll(t *testing.T) {
	routes := []route.Route{
		stubRoute{table: testTableLibp2pPeer},
	}

	got := filterRoutesByTable(routes, nil)
	assert.Equal(t, routes, got)
}
