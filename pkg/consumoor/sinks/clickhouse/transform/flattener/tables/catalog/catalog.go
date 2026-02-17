package catalog

import (
	"fmt"
	"sort"
	"sync"

	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/flattener"
)

// Builder constructs one route for registry registration.
type Builder interface {
	Table() flattener.TableName
	Build() flattener.Route
}

var (
	mu      sync.Mutex
	routes  []flattener.Route
	byTable = make(map[string]struct{}, 96)
)

// MustRegister adds a route built by builder and panics on invalid or duplicate registrations.
func MustRegister(builder Builder) {
	if builder == nil {
		panic("nil route builder")
	}

	table := string(builder.Table())
	if table == "" {
		panic("route has empty table name")
	}

	route := builder.Build()
	if route == nil {
		panic("nil route")
	}

	if route.TableName() != table {
		panic(fmt.Sprintf("route table mismatch: builder=%q built=%q", table, route.TableName()))
	}

	if len(route.EventNames()) == 0 {
		panic(fmt.Sprintf("route %q has no event names", table))
	}

	mu.Lock()
	defer mu.Unlock()

	if _, exists := byTable[table]; exists {
		panic(fmt.Sprintf("duplicate route registration for table %q", table))
	}

	byTable[table] = struct{}{}

	routes = append(routes, route)
}

// All returns all registered routes in deterministic table-name order.
func All() []flattener.Route {
	mu.Lock()
	defer mu.Unlock()

	out := append([]flattener.Route(nil), routes...)
	sort.Slice(out, func(i, j int) bool {
		return out[i].TableName() < out[j].TableName()
	})

	return out
}
