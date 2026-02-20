package transform

import (
	"fmt"
	"sync/atomic"
	"testing"

	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/flattener"
	"github.com/ethpandaops/xatu/pkg/consumoor/telemetry"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

type filterTestRoute struct {
	table  flattener.TableName
	events []xatu.Event_Name
}

var testMetricNSCounter uint64

func (r filterTestRoute) EventNames() []xatu.Event_Name {
	return r.events
}

func (r filterTestRoute) TableName() string {
	return string(r.table)
}

func (r filterTestRoute) ShouldProcess(_ *xatu.DecoratedEvent) bool {
	return true
}

func (r filterTestRoute) NewBatch() flattener.ColumnarBatch {
	return nil
}

func TestNewRouterSkipsDisabledEvents(t *testing.T) {
	routes := []flattener.Route{
		filterTestRoute{
			table:  flattener.TableName("table_a"),
			events: []xatu.Event_Name{xatu.Event_LIBP2P_TRACE_CONNECTED},
		},
		filterTestRoute{
			table:  flattener.TableName("table_b"),
			events: []xatu.Event_Name{xatu.Event_LIBP2P_TRACE_CONNECTED, xatu.Event_LIBP2P_TRACE_DISCONNECTED},
		},
		filterTestRoute{
			table:  flattener.TableName("table_c"),
			events: []xatu.Event_Name{xatu.Event_LIBP2P_TRACE_JOIN},
		},
	}

	router := New(
		logrus.New(),
		routes,
		[]xatu.Event_Name{xatu.Event_LIBP2P_TRACE_CONNECTED},
		newTestMetrics(),
	)

	require.NotContains(t, router.routesByEvent, xatu.Event_LIBP2P_TRACE_CONNECTED)
	require.Contains(t, router.routesByEvent, xatu.Event_LIBP2P_TRACE_DISCONNECTED)
	require.Contains(t, router.routesByEvent, xatu.Event_LIBP2P_TRACE_JOIN)

	disconnectedRoutes := router.routesByEvent[xatu.Event_LIBP2P_TRACE_DISCONNECTED]
	require.Len(t, disconnectedRoutes, 1)
	require.Equal(t, "table_b", disconnectedRoutes[0].TableName())
}

func newTestMetrics() *telemetry.Metrics {
	ns := fmt.Sprintf("xatu_consumoor_router_test_%d", atomic.AddUint64(&testMetricNSCounter, 1))

	return telemetry.NewMetrics(ns)
}
