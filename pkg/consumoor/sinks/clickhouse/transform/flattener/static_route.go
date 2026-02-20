package flattener

import (
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

// EventPredicate determines whether a route should process a specific event.
type EventPredicate func(event *xatu.DecoratedEvent) bool

// StaticRouteOption configures optional route behavior.
type StaticRouteOption func(*staticRoute)

// WithStaticRoutePredicate sets conditional route processing behavior.
func WithStaticRoutePredicate(predicate EventPredicate) StaticRouteOption {
	return func(route *staticRoute) {
		route.should = predicate
	}
}

// NewStaticRoute creates a route from explicit event names, table name,
// and a ColumnarBatch factory.
func NewStaticRoute(
	table TableName,
	events []xatu.Event_Name,
	batchFactory func() ColumnarBatch,
	opts ...StaticRouteOption,
) Route {
	if table == "" {
		panic("static route has empty table")
	}

	if len(events) == 0 {
		panic("static route has no events")
	}

	if batchFactory == nil {
		panic("static route has nil batch factory")
	}

	route := &staticRoute{
		table:        table,
		events:       append([]xatu.Event_Name(nil), events...),
		batchFactory: batchFactory,
	}

	for _, opt := range opts {
		if opt != nil {
			opt(route)
		}
	}

	return route
}

type staticRoute struct {
	table        TableName
	events       []xatu.Event_Name
	should       EventPredicate
	batchFactory func() ColumnarBatch
}

func (r *staticRoute) EventNames() []xatu.Event_Name {
	return append([]xatu.Event_Name(nil), r.events...)
}

func (r *staticRoute) TableName() string {
	return string(r.table)
}

func (r *staticRoute) ShouldProcess(event *xatu.DecoratedEvent) bool {
	if r.should == nil {
		return true
	}

	return r.should(event)
}

func (r *staticRoute) NewBatch() ColumnarBatch {
	return r.batchFactory()
}
