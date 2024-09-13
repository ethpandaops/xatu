package xatu_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

func TestNewEventFilter(t *testing.T) {
	testConfig := &xatu.EventFilterConfig{
		EventNames: []string{},
	}

	filter, err := xatu.NewEventFilter(testConfig)
	if err != nil {
		t.Fatal(err)
	}

	assert.NotNil(t, filter)
}

func TestNewEventFilter_InvalidName(t *testing.T) {
	testConfig := &xatu.EventFilterConfig{
		EventNames: []string{"InvalidEventName"},
	}

	_, err := xatu.NewEventFilter(testConfig)

	assert.Error(t, err)
}

func TestEventFilter_Apply(t *testing.T) {
	testEvents := []*xatu.DecoratedEvent{{
		Event: &xatu.Event{
			Name: xatu.Event_BEACON_API_ETH_V1_DEBUG_FORK_CHOICE,
		},
	}, {
		Event: &xatu.Event{
			Name: xatu.Event_BEACON_API_ETH_V1_EVENTS_ATTESTATION,
		},
	}, {
		Event: &xatu.Event{
			Name: xatu.Event_BEACON_API_ETH_V1_EVENTS_BLOCK,
		},
	}}

	testConfig := &xatu.EventFilterConfig{
		EventNames: []string{xatu.Event_BEACON_API_ETH_V1_DEBUG_FORK_CHOICE.String()},
	}

	filter, err := xatu.NewEventFilter(testConfig)
	if err != nil {
		t.Fatal(err)
	}

	filteredEvents := []*xatu.DecoratedEvent{}

	for _, event := range testEvents {
		shouldBeDropped, err := filter.ShouldBeDropped(event)
		if err != nil {
			t.Fatal(err)
		}

		if !shouldBeDropped {
			filteredEvents = append(filteredEvents, event)
		}
	}

	assert.Len(t, filteredEvents, 1)
	assert.Equal(t, testEvents[0], filteredEvents[0])
}

func TestEventFilter_ShouldBeDropped(t *testing.T) {
	testEvent := &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name: xatu.Event_BEACON_API_ETH_V1_DEBUG_FORK_CHOICE,
		},
	}

	testConfig := &xatu.EventFilterConfig{
		EventNames: []string{xatu.Event_BEACON_API_ETH_V1_DEBUG_FORK_CHOICE.String()},
	}
	filter, _ := xatu.NewEventFilter(testConfig)

	shouldBeDropped, err := filter.ShouldBeDropped(testEvent)
	if err != nil {
		t.Fatal(err)
	}

	assert.True(t, shouldBeDropped)
}

func TestEventFilter_AllowEverythingWhenEmpty(t *testing.T) {
	events := []*xatu.DecoratedEvent{}

	for _, eventName := range xatu.Event_Name_value {
		if eventName == int32(xatu.Event_BEACON_API_ETH_V1_EVENTS_UNKNOWN) {
			continue
		}

		events = append(events, &xatu.DecoratedEvent{
			Event: &xatu.Event{
				Name: xatu.Event_Name(eventName),
			},
		})
	}

	emptyConfig := &xatu.EventFilterConfig{}

	filter, err := xatu.NewEventFilter(emptyConfig)
	if err != nil {
		t.Fatal(err)
	}

	filteredEvents := []*xatu.DecoratedEvent{}

	for _, event := range events {
		shouldBeDropped, err := filter.ShouldBeDropped(event)
		if err != nil {
			t.Fatal(err)
		}

		if !shouldBeDropped {
			filteredEvents = append(filteredEvents, event)
		}
	}

	assert.Equal(t, events, filteredEvents)
}

func TestEventFilter_FilterByModules(t *testing.T) {
	testEvent := &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name: xatu.Event_BEACON_API_ETH_V1_EVENTS_FINALIZED_CHECKPOINT,
		},
		Meta: &xatu.Meta{
			Client: &xatu.ClientMeta{
				ModuleName: xatu.ModuleName_RELAY_MONITOR,
			},
		},
	}

	testConfig := &xatu.EventFilterConfig{
		Modules: []string{xatu.ModuleName_RELAY_MONITOR.String()},
	}

	filter, err := xatu.NewEventFilter(testConfig)
	if err != nil {
		t.Fatal(err)
	}

	shouldBeDropped, err := filter.ShouldBeDropped(testEvent)
	if err != nil {
		t.Fatal(err)
	}

	assert.False(t, shouldBeDropped)
}

func TestEventFilter_FilterByEventNamesAndModules(t *testing.T) {
	testEvent := &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name: xatu.Event_BEACON_API_ETH_V1_EVENTS_FINALIZED_CHECKPOINT,
		},
		Meta: &xatu.Meta{
			Client: &xatu.ClientMeta{
				ModuleName: xatu.ModuleName_RELAY_MONITOR,
			},
		},
	}

	testConfig := &xatu.EventFilterConfig{
		EventNames: []string{xatu.Event_BEACON_API_ETH_V1_EVENTS_FINALIZED_CHECKPOINT.String()},
		Modules:    []string{xatu.ModuleName_RELAY_MONITOR.String()},
	}

	filter, err := xatu.NewEventFilter(testConfig)
	if err != nil {
		t.Fatal(err)
	}

	shouldBeDropped, err := filter.ShouldBeDropped(testEvent)
	if err != nil {
		t.Fatal(err)
	}

	assert.False(t, shouldBeDropped)

	// Test with a different source
	testEvent.Meta.Client.ModuleName = xatu.ModuleName_EL_MIMICRY

	shouldBeDropped, err = filter.ShouldBeDropped(testEvent)
	if err != nil {
		t.Fatal(err)
	}

	assert.True(t, shouldBeDropped)

	// Test with a different event name
	testEvent.Meta.Client.ModuleName = xatu.ModuleName_EL_MIMICRY

	testEvent.Event.Name = xatu.Event_BEACON_API_ETH_V1_EVENTS_HEAD

	shouldBeDropped, err = filter.ShouldBeDropped(testEvent)
	if err != nil {
		t.Fatal(err)
	}

	assert.True(t, shouldBeDropped)
}
