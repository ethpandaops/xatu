package route

import (
	"errors"
	"fmt"
	"sort"
	"sync"
)

var (
	catalogMu          sync.Mutex
	catalogRoutes      []Route
	catalogByTable     = make(map[string]struct{}, 96)
	registrationErrors []error
)

// Register adds a route to the global catalog, returning an error on
// invalid or duplicate registrations.
func Register(route Route) error {
	if route == nil {
		return fmt.Errorf("nil route")
	}

	table := route.TableName()
	if table == "" {
		return fmt.Errorf("route has empty table name")
	}

	if len(route.EventNames()) == 0 {
		return fmt.Errorf("route %q has no event names", table)
	}

	catalogMu.Lock()
	defer catalogMu.Unlock()

	if _, exists := catalogByTable[table]; exists {
		return fmt.Errorf("duplicate route registration for table %q", table)
	}

	catalogByTable[table] = struct{}{}

	catalogRoutes = append(catalogRoutes, route)

	return nil
}

// RecordError stores an error encountered during init-time route
// registration. Accumulated errors are surfaced by All().
func RecordError(err error) {
	catalogMu.Lock()
	defer catalogMu.Unlock()

	registrationErrors = append(registrationErrors, err)
}

// All returns all registered routes in deterministic table-name order.
// It returns an error if any route registrations failed during init.
func All() ([]Route, error) {
	catalogMu.Lock()
	defer catalogMu.Unlock()

	if len(registrationErrors) > 0 {
		return nil, errors.Join(registrationErrors...)
	}

	out := append([]Route(nil), catalogRoutes...)
	sort.Slice(out, func(i, j int) bool {
		return out[i].TableName() < out[j].TableName()
	})

	return out, nil
}
