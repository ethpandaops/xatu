package all

import (
	"github.com/ethpandaops/xatu/pkg/clickhouse/route"
	_ "github.com/ethpandaops/xatu/pkg/clickhouse/route/beacon"
	_ "github.com/ethpandaops/xatu/pkg/clickhouse/route/canonical"
	_ "github.com/ethpandaops/xatu/pkg/clickhouse/route/execution"
	_ "github.com/ethpandaops/xatu/pkg/clickhouse/route/libp2p"
	_ "github.com/ethpandaops/xatu/pkg/clickhouse/route/mev"
	_ "github.com/ethpandaops/xatu/pkg/clickhouse/route/node"
)

// All returns all registered table routes.
func All() ([]route.Route, error) {
	return route.All()
}
