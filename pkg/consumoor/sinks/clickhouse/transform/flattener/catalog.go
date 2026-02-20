package flattener

import (
	"fmt"
	"sort"
	"sync"
)

var (
	catalogMu      sync.Mutex
	catalogRoutes  []Route
	catalogByTable = make(map[string]struct{}, 96)
)

// MustRegister adds route and panics on invalid or duplicate registrations.
func MustRegister(route Route) {
	if route == nil {
		panic("nil route")
	}

	table := route.TableName()
	if table == "" {
		panic("route has empty table name")
	}

	if len(route.EventNames()) == 0 {
		panic(fmt.Sprintf("route %q has no event names", table))
	}

	catalogMu.Lock()
	defer catalogMu.Unlock()

	if _, exists := catalogByTable[table]; exists {
		panic(fmt.Sprintf("duplicate route registration for table %q", table))
	}

	catalogByTable[table] = struct{}{}

	catalogRoutes = append(catalogRoutes, route)
}

// All returns all registered routes in deterministic table-name order.
func All() []Route {
	catalogMu.Lock()
	defer catalogMu.Unlock()

	out := append([]Route(nil), catalogRoutes...)
	sort.Slice(out, func(i, j int) bool {
		return out[i].TableName() < out[j].TableName()
	})

	return out
}
