package tables

import (
	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/flattener"
	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/flattener/tables/catalog"
)

// All returns all registered table routes.
// Domain packages (beacon, canonical, execution, libp2p, mev, node)
// register their routes via init() and are imported in later PRs.
func All() []flattener.Route {
	return catalog.All()
}
