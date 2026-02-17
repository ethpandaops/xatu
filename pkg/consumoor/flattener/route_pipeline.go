package flattener

import (
	"fmt"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

type routeStepSpec struct {
	name  string
	build func(rp *RoutePipeline) GenericStep
}

var (
	routeStepCommonMetadata = routeStepSpec{
		name: "common_metadata",
		build: func(_ *RoutePipeline) GenericStep {
			return commonMetadataStep()
		},
	}
	routeStepRuntimeColumns = routeStepSpec{
		name: "runtime_columns",
		build: func(_ *RoutePipeline) GenericStep {
			return runtimeColumnsStep()
		},
	}
	routeStepEventData = routeStepSpec{
		name: "event_data",
		build: func(_ *RoutePipeline) GenericStep {
			return eventDataStep()
		},
	}
	routeStepClientAdditionalData = routeStepSpec{
		name: "client_additional_data",
		build: func(_ *RoutePipeline) GenericStep {
			return clientAdditionalDataStep()
		},
	}
	routeStepServerAdditionalData = routeStepSpec{
		name: "server_additional_data",
		build: func(_ *RoutePipeline) GenericStep {
			return serverAdditionalDataStep()
		},
	}
	routeStepTableAliases = routeStepSpec{
		name: "table_aliases",
		build: func(rp *RoutePipeline) GenericStep {
			return tableAliasesStep(rp.table)
		},
	}
	routeStepRouteAliases = routeStepSpec{
		name: "route_aliases",
		build: func(rp *RoutePipeline) GenericStep {
			return routeAliasesStep(rp.aliases)
		},
	}
	routeStepNormalizeDateTimes = routeStepSpec{
		name: "normalize_date_times",
		build: func(_ *RoutePipeline) GenericStep {
			return normalizeDateTimesStep()
		},
	}
	routeStepCommonEnrichment = routeStepSpec{
		name: "common_enrichment",
		build: func(_ *RoutePipeline) GenericStep {
			return commonEnrichmentStep()
		},
	}
)

// RoutePipeline builds an explicit generic route pipeline for one table/event set.
type RoutePipeline struct {
	table     TableName
	events    []xatu.Event_Name
	steps     []routeStepSpec
	predicate EventPredicate
	mutator   RowMutator
	aliases   map[string]string
}

// RouteTo starts a route pipeline for a destination table and event names.
func RouteTo(table TableName, events ...xatu.Event_Name) *RoutePipeline {
	return &RoutePipeline{
		table:  table,
		events: append([]xatu.Event_Name(nil), events...),
	}
}

func (rp *RoutePipeline) addStep(step routeStepSpec) *RoutePipeline {
	rp.steps = append(rp.steps, step)

	return rp
}

// CommonMetadata copies shared metadata fields onto each row.
func (rp *RoutePipeline) CommonMetadata() *RoutePipeline {
	return rp.addStep(routeStepCommonMetadata)
}

// RuntimeColumns appends writer/runtime columns such as updated_date_time,
// unique_key, and event_date_time.
func (rp *RoutePipeline) RuntimeColumns() *RoutePipeline {
	return rp.addStep(routeStepRuntimeColumns)
}

// EventData flattens the event protobuf payload into row columns.
func (rp *RoutePipeline) EventData() *RoutePipeline {
	return rp.addStep(routeStepEventData)
}

// ClientAdditionalData flattens client additional_data oneof fields.
func (rp *RoutePipeline) ClientAdditionalData() *RoutePipeline {
	return rp.addStep(routeStepClientAdditionalData)
}

// ServerAdditionalData flattens server additional_data oneof fields.
func (rp *RoutePipeline) ServerAdditionalData() *RoutePipeline {
	return rp.addStep(routeStepServerAdditionalData)
}

// TableAliases applies known destination-table aliases.
func (rp *RoutePipeline) TableAliases() *RoutePipeline {
	return rp.addStep(routeStepTableAliases)
}

// RouteAliases applies route-specific alias remapping set via Aliases().
func (rp *RoutePipeline) RouteAliases() *RoutePipeline {
	return rp.addStep(routeStepRouteAliases)
}

// NormalizeDateTimes normalizes date/time fields for ClickHouse output.
func (rp *RoutePipeline) NormalizeDateTimes() *RoutePipeline {
	return rp.addStep(routeStepNormalizeDateTimes)
}

// CommonEnrichment applies shared enrichment logic by event name.
func (rp *RoutePipeline) CommonEnrichment() *RoutePipeline {
	return rp.addStep(routeStepCommonEnrichment)
}

// Predicate sets conditional routing for this route.
func (rp *RoutePipeline) Predicate(predicate EventPredicate) *RoutePipeline {
	rp.predicate = predicate

	return rp
}

// Mutator sets optional row mutation/fan-out logic for this route.
func (rp *RoutePipeline) Mutator(mutator RowMutator) *RoutePipeline {
	rp.mutator = mutator

	return rp
}

// Aliases sets route-specific alias remapping.
func (rp *RoutePipeline) Aliases(aliases map[string]string) *RoutePipeline {
	rp.aliases = cloneAliases(aliases)

	return rp
}

// Build finalizes the route pipeline as a GenericFlattener route.
func (rp *RoutePipeline) Build() Route {
	if err := rp.validate(); err != nil {
		panic(fmt.Sprintf("invalid route pipeline for table %q: %v", rp.table, err))
	}

	steps := make([]compiledGenericStep, 0, len(rp.steps))
	for _, step := range rp.steps {
		steps = append(steps, compiledGenericStep{
			name: step.name,
			run:  step.build(rp),
		})
	}

	return NewGenericFlattenerWithSteps(
		rp.table,
		append([]xatu.Event_Name(nil), rp.events...),
		steps,
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

	seen := make(map[string]struct{}, len(rp.steps))
	hasRouteAliasesStep := false

	for _, step := range rp.steps {
		if step.name == "" {
			return fmt.Errorf("step name is required")
		}

		if step.build == nil {
			return fmt.Errorf("step %q has nil builder", step.name)
		}

		if _, exists := seen[step.name]; exists {
			return fmt.Errorf("duplicate step %q", step.name)
		}

		if step.name == routeStepRouteAliases.name {
			hasRouteAliasesStep = true
		}

		seen[step.name] = struct{}{}
	}

	if len(rp.aliases) > 0 && !hasRouteAliasesStep {
		return fmt.Errorf("aliases configured but RouteAliases() step missing")
	}

	return nil
}
