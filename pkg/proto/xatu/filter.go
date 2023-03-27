package xatu

import (
	"fmt"

	"github.com/pkg/errors"
)

type EventFilter interface {
	// EventNames returns the list of event names to filter on.
	EventNames() []string

	// ShouldBeDropped returns true if the event should be dropped.
	ShouldBeDropped(event *DecoratedEvent) (bool, error)
}

type EventFilterConfig struct {
	EventNames []string `yaml:"eventNames"`
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

	return &eventFilter{
		eventNames: eventNames,
	}, nil
}

type eventFilter struct {
	config *EventFilterConfig

	eventNames map[string]struct{}
}

func (f *eventFilter) EventNames() []string {
	return f.config.EventNames
}

func (f *eventFilter) ShouldBeDropped(event *DecoratedEvent) (bool, error) {
	if event == nil {
		return true, errors.New("event is nil")
	}

	if event.Event == nil {
		return true, errors.New("event.event is nil")
	}

	if len(f.eventNames) == 0 {
		return false, nil
	}

	return f.applyEventNamesFilter(event)
}

func (f *eventFilter) applyEventNamesFilter(event *DecoratedEvent) (bool, error) {
	if len(f.eventNames) == 0 {
		return false, nil
	}

	if event.Event.Name == 0 {
		return true, errors.New("event.event.name is invalid")
	}

	_, ok := f.eventNames[event.Event.Name.String()]

	return !ok, nil
}
