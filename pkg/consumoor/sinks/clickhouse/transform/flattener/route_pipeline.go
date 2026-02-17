package flattener

import (
	"fmt"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

// RoutePipeline builds a generic route pipeline for one table/event set.
type RoutePipeline struct {
	table     TableName
	events    []xatu.Event_Name
	steps     []GenericStep
	predicate EventPredicate
	mutator   RowMutator
}

// RouteSource holds source event names before selecting a destination table.
type RouteSource struct {
	events []xatu.Event_Name
}

// From starts a route declaration from one or more source event names.
func From(events ...xatu.Event_Name) *RouteSource {
	return &RouteSource{
		events: append([]xatu.Event_Name(nil), events...),
	}
}

// To completes source selection with a destination table and returns a pipeline.
func (rs *RouteSource) To(table TableName) *RoutePipeline {
	return &RoutePipeline{
		table:  table,
		events: append([]xatu.Event_Name(nil), rs.events...),
	}
}

// Apply appends one flattening transform to this route.
func (rp *RoutePipeline) Apply(step GenericStep) *RoutePipeline {
	rp.steps = append(rp.steps, step)

	return rp
}

// If sets conditional routing for this route.
func (rp *RoutePipeline) If(predicate EventPredicate) *RoutePipeline {
	rp.predicate = predicate

	return rp
}

// Mutator sets optional row mutation/fan-out logic for this route.
func (rp *RoutePipeline) Mutator(mutator RowMutator) *RoutePipeline {
	rp.mutator = mutator

	return rp
}

// Build finalizes the route pipeline as a GenericFlattener route.
func (rp *RoutePipeline) Build() Route {
	if err := rp.validate(); err != nil {
		panic(fmt.Sprintf("invalid route pipeline for table %q: %v", rp.table, err))
	}

	return NewGenericFlattenerWithSteps(
		rp.table,
		append([]xatu.Event_Name(nil), rp.events...),
		append([]GenericStep(nil), rp.steps...),
		rp.predicate,
		rp.mutator,
	)
}

func (rp *RoutePipeline) validate() error {
	if rp.table == "" {
		return fmt.Errorf("table is required")
	}

	if len(rp.events) == 0 {
		return fmt.Errorf("at least one event is required")
	}

	if len(rp.steps) == 0 {
		return fmt.Errorf("at least one step is required")
	}

	for i, step := range rp.steps {
		if step == nil {
			return fmt.Errorf("step %d is nil", i)
		}
	}

	return nil
}
