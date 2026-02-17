package flattener

import (
	"fmt"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

// RoutePipeline builds an explicit generic route pipeline for one table/event set.
type RoutePipeline struct {
	table     TableName
	events    []xatu.Event_Name
	stages    []GenericStage
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

// Then appends explicit flattening stages in execution order.
func (rp *RoutePipeline) Then(stages ...GenericStage) *RoutePipeline {
	rp.stages = append(rp.stages, stages...)

	return rp
}

// CommonMetadata copies shared metadata fields onto each row.
func (rp *RoutePipeline) CommonMetadata() *RoutePipeline {
	return rp.Then(StageCommonMetadata)
}

// RuntimeColumns appends writer/runtime columns such as updated_date_time,
// unique_key, and event_date_time.
func (rp *RoutePipeline) RuntimeColumns() *RoutePipeline {
	return rp.Then(StageRuntimeColumns)
}

// EventData flattens the event protobuf payload into row columns.
func (rp *RoutePipeline) EventData() *RoutePipeline {
	return rp.Then(StageEventData)
}

// ClientAdditionalData flattens client additional_data oneof fields.
func (rp *RoutePipeline) ClientAdditionalData() *RoutePipeline {
	return rp.Then(StageClientAdditionalData)
}

// ServerAdditionalData flattens server additional_data oneof fields.
func (rp *RoutePipeline) ServerAdditionalData() *RoutePipeline {
	return rp.Then(StageServerAdditionalData)
}

// TableAliases applies known destination-table aliases.
func (rp *RoutePipeline) TableAliases() *RoutePipeline {
	return rp.Then(StageTableAliases)
}

// RouteAliases applies route-specific alias remapping set via Aliases().
func (rp *RoutePipeline) RouteAliases() *RoutePipeline {
	return rp.Then(StageRouteAliases)
}

// NormalizeDateTimes normalizes date/time fields for ClickHouse output.
func (rp *RoutePipeline) NormalizeDateTimes() *RoutePipeline {
	return rp.Then(StageNormalizeDateTimes)
}

// CommonEnrichment applies shared enrichment logic by event name.
func (rp *RoutePipeline) CommonEnrichment() *RoutePipeline {
	return rp.Then(StageCommonEnrichment)
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
	if len(aliases) == 0 {
		rp.aliases = nil

		return rp
	}

	cloned := make(map[string]string, len(aliases))
	for src, dst := range aliases {
		cloned[src] = dst
	}

	rp.aliases = cloned

	return rp
}

// Build finalizes the route pipeline as a GenericFlattener route.
func (rp *RoutePipeline) Build() Route {
	if err := rp.validate(); err != nil {
		panic(fmt.Sprintf("invalid route pipeline for table %q: %v", rp.table, err))
	}

	return NewGenericFlattenerWithStages(
		rp.table,
		append([]xatu.Event_Name(nil), rp.events...),
		append([]GenericStage(nil), rp.stages...),
		rp.predicate,
		rp.mutator,
		rp.aliases,
	)
}

func (rp *RoutePipeline) validate() error {
	if rp.table == "" {
		return fmt.Errorf("table is required")
	}

	if len(rp.events) == 0 {
		return fmt.Errorf("at least one event is required")
	}

	if len(rp.stages) == 0 {
		return fmt.Errorf("at least one stage is required")
	}

	seen := make(map[GenericStage]struct{}, len(rp.stages))
	hasRouteAliasesStage := false
	lastOrder := 0

	for _, stage := range rp.stages {
		if !stage.valid() {
			return fmt.Errorf("unknown stage %s", stage)
		}

		if _, exists := seen[stage]; exists {
			return fmt.Errorf("duplicate stage %s", stage)
		}

		order := stage.order()
		if order < lastOrder {
			return fmt.Errorf("stage %s is out of order", stage)
		}

		if stage == StageRouteAliases {
			hasRouteAliasesStage = true
		}

		lastOrder = order
		seen[stage] = struct{}{}
	}

	if len(rp.aliases) > 0 && !hasRouteAliasesStage {
		return fmt.Errorf("aliases configured but StageRouteAliases missing")
	}

	return nil
}
