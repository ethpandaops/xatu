package xatu

import (
	"fmt"

	"github.com/pkg/errors"
)

// EventFilter defines the interface for filtering decorated events.
type EventFilter interface {
	// EventNames returns the list of event names to include.
	EventNames() []string
	// ExcludeEventNames returns the list of event names to exclude.
	ExcludeEventNames() []string
	// Modules returns the list of modules to include.
	Modules() []string
	// ExcludeModules returns the list of modules to exclude.
	ExcludeModules() []string
	// ShouldBeDropped returns true if the event should be dropped.
	ShouldBeDropped(event *DecoratedEvent) (bool, error)
}

// EventFilterConfig holds configuration for event filtering.
type EventFilterConfig struct {
	EventNames        []string `yaml:"eventNames"`
	ExcludeEventNames []string `yaml:"excludeEventNames"`
	Modules           []string `yaml:"modules"`
	ExcludeModules    []string `yaml:"excludeModules"`
}

type eventFilter struct {
	config *EventFilterConfig

	eventNames        map[string]struct{}
	excludeEventNames map[string]struct{}
	modules           map[string]struct{}
	excludeModules    map[string]struct{}
}

// NewEventFilter creates a new EventFilter from the given config.
func NewEventFilter(config *EventFilterConfig) (EventFilter, error) {
	if err := config.Validate(); err != nil {
		return nil, errors.Wrap(err, "invalid event filter config")
	}

	eventNames := make(map[string]struct{}, len(config.EventNames))
	for _, eventName := range config.EventNames {
		eventNames[eventName] = struct{}{}
	}

	excludeEventNames := make(map[string]struct{}, len(config.ExcludeEventNames))
	for _, eventName := range config.ExcludeEventNames {
		excludeEventNames[eventName] = struct{}{}
	}

	modules := make(map[string]struct{}, len(config.Modules))
	for _, module := range config.Modules {
		modules[module] = struct{}{}
	}

	excludeModules := make(map[string]struct{}, len(config.ExcludeModules))
	for _, module := range config.ExcludeModules {
		excludeModules[module] = struct{}{}
	}

	return &eventFilter{
		config:            config,
		eventNames:        eventNames,
		excludeEventNames: excludeEventNames,
		modules:           modules,
		excludeModules:    excludeModules,
	}, nil
}

func (f *EventFilterConfig) Validate() error {
	for _, eventName := range f.EventNames {
		if _, ok := Event_Name_value[eventName]; !ok {
			return fmt.Errorf("invalid event name: %s", eventName)
		}
	}

	for _, eventName := range f.ExcludeEventNames {
		if _, ok := Event_Name_value[eventName]; !ok {
			return fmt.Errorf("invalid exclude event name: %s", eventName)
		}
	}

	if len(f.EventNames) > 0 && len(f.ExcludeEventNames) > 0 {
		return fmt.Errorf("eventNames and excludeEventNames are mutually exclusive")
	}

	if len(f.Modules) > 0 && len(f.ExcludeModules) > 0 {
		return fmt.Errorf("modules and excludeModules are mutually exclusive")
	}

	return nil
}

func (f *eventFilter) EventNames() []string {
	return f.config.EventNames
}

func (f *eventFilter) ExcludeEventNames() []string {
	return f.config.ExcludeEventNames
}

func (f *eventFilter) Modules() []string {
	return f.config.Modules
}

func (f *eventFilter) ExcludeModules() []string {
	return f.config.ExcludeModules
}

func (f *eventFilter) ShouldBeDropped(event *DecoratedEvent) (bool, error) {
	if event == nil {
		return true, errors.New("event is nil")
	}

	if event.Event == nil {
		return true, errors.New("event.event is nil")
	}

	hasAnyFilter := len(f.eventNames) > 0 ||
		len(f.excludeEventNames) > 0 ||
		len(f.modules) > 0 ||
		len(f.excludeModules) > 0

	if !hasAnyFilter {
		return false, nil
	}

	if len(f.eventNames) > 0 || len(f.excludeEventNames) > 0 {
		shouldDrop, err := f.shouldDropFromEventNames(event)
		if err != nil {
			return true, errors.Wrap(err, "failed to apply event names filter")
		}

		if shouldDrop {
			return true, nil
		}
	}

	if len(f.modules) > 0 || len(f.excludeModules) > 0 {
		shouldDrop, err := f.shouldDropFromModules(event)
		if err != nil {
			return true, errors.Wrap(err, "failed to apply modules filter")
		}

		if shouldDrop {
			return true, nil
		}
	}

	return false, nil
}

func (f *eventFilter) shouldDropFromEventNames(event *DecoratedEvent) (bool, error) {
	if event.Event.Name == 0 {
		return true, errors.New("event.event.name is invalid")
	}

	name := event.Event.Name.String()

	// Include list: drop if NOT in list.
	if len(f.eventNames) > 0 {
		_, ok := f.eventNames[name]

		return !ok, nil
	}

	// Exclude list: drop if IN list.
	if len(f.excludeEventNames) > 0 {
		_, ok := f.excludeEventNames[name]

		return ok, nil
	}

	return false, nil
}

func (f *eventFilter) shouldDropFromModules(event *DecoratedEvent) (bool, error) {
	moduleName := event.GetMeta().GetClient().GetModuleName()

	// Include list: drop if NOT in list.
	if len(f.modules) > 0 {
		if moduleName == 0 {
			return true, nil
		}

		_, ok := f.modules[moduleName.String()]

		return !ok, nil
	}

	// Exclude list: drop if IN list.
	if len(f.excludeModules) > 0 {
		// If the event has no module set, it doesn't match any exclude entry, so keep it.
		if moduleName == 0 {
			return false, nil
		}

		_, ok := f.excludeModules[moduleName.String()]

		return ok, nil
	}

	return false, nil
}
