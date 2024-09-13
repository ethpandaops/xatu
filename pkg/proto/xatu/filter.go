package xatu

import (
	"fmt"

	"github.com/pkg/errors"
)

type EventFilter interface {
	// EventNames returns the list of event names to filter on.
	EventNames() []string
	// Modules returns the list of modules to filter on.
	Modules() []string
	// ShouldBeDropped returns true if the event should be dropped.
	ShouldBeDropped(event *DecoratedEvent) (bool, error)
}

type EventFilterConfig struct {
	EventNames []string `yaml:"eventNames"`
	Modules    []string `yaml:"modules"`
}

func (f *EventFilterConfig) Validate() error {
	for _, eventName := range f.EventNames {
		if _, ok := Event_Name_value[eventName]; !ok {
			return fmt.Errorf("invalid event name: %s", eventName)
		}
	}

	return nil
}

func NewEventFilter(config *EventFilterConfig) (EventFilter, error) {
	if err := config.Validate(); err != nil {
		return nil, errors.Wrap(err, "invalid event filter config")
	}

	eventNames := make(map[string]struct{}, len(config.EventNames))

	for _, eventName := range config.EventNames {
		eventNames[eventName] = struct{}{}
	}

	modules := make(map[string]struct{}, len(config.Modules))

	for _, module := range config.Modules {
		modules[module] = struct{}{}
	}

	return &eventFilter{
		config:     config,
		eventNames: eventNames,
		modules:    modules,
	}, nil
}

type eventFilter struct {
	config *EventFilterConfig

	eventNames map[string]struct{}
	modules    map[string]struct{}
}

func (f *eventFilter) EventNames() []string {
	return f.config.EventNames
}

func (f *eventFilter) Modules() []string {
	return f.config.Modules
}

func (f *eventFilter) ShouldBeDropped(event *DecoratedEvent) (bool, error) {
	if event == nil {
		return true, errors.New("event is nil")
	}

	if event.Event == nil {
		return true, errors.New("event.event is nil")
	}

	if len(f.eventNames) == 0 && len(f.modules) == 0 {
		return false, nil
	}

	if len(f.eventNames) > 0 {
		shouldDrop, err := f.shouldDropFromEventNames(event)
		if err != nil {
			return true, errors.Wrap(err, "failed to apply event names filter")
		}

		if shouldDrop {
			return true, nil
		}
	}

	if len(f.modules) > 0 {
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
	if len(f.eventNames) == 0 {
		return false, nil
	}

	if event.Event.Name == 0 {
		return true, errors.New("event.event.name is invalid")
	}

	_, ok := f.eventNames[event.Event.Name.String()]

	return !ok, nil
}

func (f *eventFilter) shouldDropFromModules(event *DecoratedEvent) (bool, error) {
	if len(f.modules) == 0 {
		return false, nil
	}

	if event.GetMeta().GetClient().GetModuleName() == 0 {
		return true, errors.New("event.meta.client.module is empty")
	}

	_, ok := f.modules[event.GetMeta().GetClient().GetModuleName().String()]

	return !ok, nil
}
