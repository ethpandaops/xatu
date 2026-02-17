package tables

import (
	"github.com/ethpandaops/xatu/pkg/consumoor/flattener"
	_ "github.com/ethpandaops/xatu/pkg/consumoor/flattener/tables/beacon"
	_ "github.com/ethpandaops/xatu/pkg/consumoor/flattener/tables/canonical"
	"github.com/ethpandaops/xatu/pkg/consumoor/flattener/tables/catalog"
	_ "github.com/ethpandaops/xatu/pkg/consumoor/flattener/tables/execution"
	_ "github.com/ethpandaops/xatu/pkg/consumoor/flattener/tables/libp2p"
	_ "github.com/ethpandaops/xatu/pkg/consumoor/flattener/tables/mev"
	_ "github.com/ethpandaops/xatu/pkg/consumoor/flattener/tables/node"
)

// All returns all registered table routes.
func All() []flattener.Route {
	return catalog.All()
}
