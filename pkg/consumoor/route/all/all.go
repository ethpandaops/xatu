package all

import (
	"github.com/ethpandaops/xatu/pkg/consumoor/route"
	_ "github.com/ethpandaops/xatu/pkg/consumoor/route/beacon"
	_ "github.com/ethpandaops/xatu/pkg/consumoor/route/canonical"
	_ "github.com/ethpandaops/xatu/pkg/consumoor/route/execution"
	_ "github.com/ethpandaops/xatu/pkg/consumoor/route/libp2p"
	_ "github.com/ethpandaops/xatu/pkg/consumoor/route/mev"
	_ "github.com/ethpandaops/xatu/pkg/consumoor/route/node"
)

// All returns all registered table routes.
func All() ([]route.Route, error) {
	return route.All()
}
