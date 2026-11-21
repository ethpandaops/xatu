package xatu

import (
	"fmt"
)

// EventMutator mutates decorated events in place before they are forwarded
// or persisted. Implementations must be safe for concurrent use.
type EventMutator interface {
	// IsEnabled returns true when at least one mutation is configured.
	IsEnabled() bool
	// Mutate applies the configured mutations to the event in place.
	Mutate(event *DecoratedEvent)
}

// EventMutatorConfig holds configuration for mutating decorated events.
type EventMutatorConfig struct {
	// MetaNetworkName rewrites the client-asserted network name
	// (meta.client.ethereum.network.name).
	MetaNetworkName MetaNetworkNameMutationConfig `yaml:"metaNetworkName"`
}

// MetaNetworkNameMutationConfig rewrites the client-asserted network name.
// Useful for pipelines that need to re-namespace events, e.g. relays in
// front of networks that share a genesis (and therefore derive identical
// network names), or consumers that enforce a canonical name for everything
// they persist. Value is mutually exclusive with Prefix/Suffix.
type MetaNetworkNameMutationConfig struct {
	// Value replaces the network name outright.
	Value string `yaml:"value"`
	// Prefix is prepended to the existing network name.
	Prefix string `yaml:"prefix"`
	// Suffix is appended to the existing network name.
	Suffix string `yaml:"suffix"`
}

type eventMutator struct {
	config *EventMutatorConfig
}

// NewEventMutator creates a new EventMutator from the given config.
func NewEventMutator(config *EventMutatorConfig) (EventMutator, error) {
	if config == nil {
		config = &EventMutatorConfig{}
	}

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid event mutator config: %w", err)
	}

	return &eventMutator{config: config}, nil
}

// Validate checks the configuration for errors.
func (c *EventMutatorConfig) Validate() error {
	return c.MetaNetworkName.Validate()
}

// Enabled returns true when at least one mutation is configured.
func (c *EventMutatorConfig) Enabled() bool {
	return c.MetaNetworkName.enabled()
}

// Validate checks the configuration for errors.
func (c *MetaNetworkNameMutationConfig) Validate() error {
	if c.Value != "" && (c.Prefix != "" || c.Suffix != "") {
		return fmt.Errorf("metaNetworkName.value is mutually exclusive with prefix/suffix")
	}

	return nil
}

func (c *MetaNetworkNameMutationConfig) enabled() bool {
	return c.Value != "" || c.Prefix != "" || c.Suffix != ""
}

func (m *eventMutator) IsEnabled() bool {
	return m.config.Enabled()
}

func (m *eventMutator) Mutate(event *DecoratedEvent) {
	if !m.IsEnabled() || event == nil {
		return
	}

	m.mutateMetaNetworkName(event)
}

// mutateMetaNetworkName rewrites meta.client.ethereum.network.name. Events
// that carry no network metadata are left untouched: fabricating the
// structure would assert a network the client never claimed, and
// prefixing/suffixing an empty name would produce a bare affix.
func (m *eventMutator) mutateMetaNetworkName(event *DecoratedEvent) {
	cfg := m.config.MetaNetworkName
	if !cfg.enabled() {
		return
	}

	network := event.GetMeta().GetClient().GetEthereum().GetNetwork()
	if network == nil || network.GetName() == "" {
		return
	}

	if cfg.Value != "" {
		network.Name = cfg.Value

		return
	}

	network.Name = cfg.Prefix + network.GetName() + cfg.Suffix
}
