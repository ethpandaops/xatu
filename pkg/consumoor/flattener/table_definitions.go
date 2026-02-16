package flattener

import "github.com/ethpandaops/xatu/pkg/proto/xatu"

type TableSourceBuilder func(table TableName, events []xatu.Event_Name) Route

// TableSource defines one event source and handler strategy for a table.
type TableSource struct {
	Events []xatu.Event_Name
	Build  TableSourceBuilder
}

// TableDefinition describes one destination table and how it is populated.
type TableDefinition struct {
	Name    TableName
	Sources []TableSource
}

func Table(name TableName, sources ...TableSource) TableDefinition {
	return TableDefinition{
		Name:    name,
		Sources: sources,
	}
}

func GenericSource(events []xatu.Event_Name, opts ...GenericRouteOption) TableSource {
	return TableSource{
		Events: events,
		Build: func(table TableName, sourceEvents []xatu.Event_Name) Route {
			return NewGenericRoute(table, sourceEvents, opts...)
		},
	}
}

func CustomSource(events []xatu.Event_Name, build TableSourceBuilder) TableSource {
	return TableSource{
		Events: events,
		Build:  build,
	}
}

func GenericTable(table TableName, events []xatu.Event_Name, opts ...GenericRouteOption) TableDefinition {
	return Table(table, GenericSource(events, opts...))
}

func ValidatorsFanoutTable(table TableName, kind ValidatorsFanoutKind) TableDefinition {
	return Table(
		table,
		CustomSource(
			[]xatu.Event_Name{xatu.Event_BEACON_API_ETH_V1_BEACON_VALIDATORS},
			func(name TableName, _ []xatu.Event_Name) Route {
				return NewValidatorsFanoutFlattener(name, kind)
			},
		),
	)
}

func routesFromTables(tables []TableDefinition) []Route {
	routes := make([]Route, 0, len(tables))

	for _, table := range tables {
		for _, source := range table.Sources {
			if source.Build == nil {
				continue
			}

			route := source.Build(table.Name, source.Events)
			if route == nil {
				continue
			}

			routes = append(routes, route)
		}
	}

	return routes
}
