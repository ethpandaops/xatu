package table

import "github.com/ethpandaops/xatu/pkg/consumoor/flattener"

// All returns all table route definitions used by consumoor.
func All() []flattener.Route {
	routes := make([]flattener.Route, 0, 96)
	routes = append(routes, beaconRoutes()...)
	routes = append(routes, executionRoutes()...)
	routes = append(routes, mevAndNodeRoutes()...)
	routes = append(routes, libp2pRoutes()...)

	return routes
}
