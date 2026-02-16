package flattener

import "github.com/ethpandaops/xatu/pkg/proto/xatu"

type genericRouteConfig struct {
	predicate EventPredicate
	mutator   RowMutator
	aliases   map[string]string
}

type GenericRouteOption func(*genericRouteConfig)

func WithPredicate(predicate EventPredicate) GenericRouteOption {
	return func(cfg *genericRouteConfig) {
		cfg.predicate = predicate
	}
}

func WithMutator(mutator RowMutator) GenericRouteOption {
	return func(cfg *genericRouteConfig) {
		cfg.mutator = mutator
	}
}

func WithAliases(aliases map[string]string) GenericRouteOption {
	return func(cfg *genericRouteConfig) {
		cfg.aliases = aliases
	}
}

func NewGenericRoute(table TableName, events []xatu.Event_Name, opts ...GenericRouteOption) Flattener {
	cfg := &genericRouteConfig{}

	for _, opt := range opts {
		if opt != nil {
			opt(cfg)
		}
	}

	return NewGenericFlattener(table, events, cfg.predicate, cfg.mutator, cfg.aliases)
}
