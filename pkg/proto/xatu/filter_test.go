package xatu_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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
		EventNames: []string{xatu.Event_BEACON_API_ETH_V1_EVENTS_FINALIZED_CHECKPOINT.String()},
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

func TestEventFilterConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *xatu.EventFilterConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid excludeEventNames",
			config: &xatu.EventFilterConfig{
				ExcludeEventNames: []string{
					xatu.Event_BEACON_API_ETH_V1_EVENTS_ATTESTATION.String(),
				},
			},
		},
		{
			name: "invalid excludeEventNames",
			config: &xatu.EventFilterConfig{
				ExcludeEventNames: []string{"INVALID_EVENT"},
			},
			wantErr: true,
			errMsg:  "invalid exclude event name",
		},
		{
			name: "valid excludeModules",
			config: &xatu.EventFilterConfig{
				ExcludeModules: []string{
					xatu.ModuleName_RELAY_MONITOR.String(),
				},
			},
		},
		{
			name: "eventNames and excludeEventNames are mutually exclusive",
			config: &xatu.EventFilterConfig{
				EventNames: []string{
					xatu.Event_BEACON_API_ETH_V1_EVENTS_BLOCK.String(),
				},
				ExcludeEventNames: []string{
					xatu.Event_BEACON_API_ETH_V1_EVENTS_ATTESTATION.String(),
				},
			},
			wantErr: true,
			errMsg:  "mutually exclusive",
		},
		{
			name: "modules and excludeModules are mutually exclusive",
			config: &xatu.EventFilterConfig{
				Modules: []string{
					xatu.ModuleName_RELAY_MONITOR.String(),
				},
				ExcludeModules: []string{
					xatu.ModuleName_EL_MIMICRY.String(),
				},
			},
			wantErr: true,
			errMsg:  "mutually exclusive",
		},
		{
			name: "eventNames with excludeModules is allowed",
			config: &xatu.EventFilterConfig{
				EventNames: []string{
					xatu.Event_BEACON_API_ETH_V1_EVENTS_BLOCK.String(),
				},
				ExcludeModules: []string{
					xatu.ModuleName_EL_MIMICRY.String(),
				},
			},
		},
		{
			name: "excludeEventNames with modules is allowed",
			config: &xatu.EventFilterConfig{
				ExcludeEventNames: []string{
					xatu.Event_BEACON_API_ETH_V1_EVENTS_ATTESTATION.String(),
				},
				Modules: []string{
					xatu.ModuleName_RELAY_MONITOR.String(),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestEventFilter_ExcludeEventNames(t *testing.T) {
	tests := []struct {
		name     string
		config   *xatu.EventFilterConfig
		event    *xatu.DecoratedEvent
		wantDrop bool
		wantErr  bool
	}{
		{
			name: "exclude single event name drops matching event",
			config: &xatu.EventFilterConfig{
				ExcludeEventNames: []string{
					xatu.Event_BEACON_API_ETH_V1_EVENTS_ATTESTATION.String(),
				},
			},
			event: &xatu.DecoratedEvent{
				Event: &xatu.Event{
					Name: xatu.Event_BEACON_API_ETH_V1_EVENTS_ATTESTATION,
				},
			},
			wantDrop: true,
		},
		{
			name: "exclude single event name keeps non-matching event",
			config: &xatu.EventFilterConfig{
				ExcludeEventNames: []string{
					xatu.Event_BEACON_API_ETH_V1_EVENTS_ATTESTATION.String(),
				},
			},
			event: &xatu.DecoratedEvent{
				Event: &xatu.Event{
					Name: xatu.Event_BEACON_API_ETH_V1_EVENTS_BLOCK,
				},
			},
			wantDrop: false,
		},
		{
			name: "exclude multiple event names drops all excluded",
			config: &xatu.EventFilterConfig{
				ExcludeEventNames: []string{
					xatu.Event_BEACON_API_ETH_V1_EVENTS_ATTESTATION.String(),
					xatu.Event_BEACON_API_ETH_V1_EVENTS_BLOCK.String(),
				},
			},
			event: &xatu.DecoratedEvent{
				Event: &xatu.Event{
					Name: xatu.Event_BEACON_API_ETH_V1_EVENTS_BLOCK,
				},
			},
			wantDrop: true,
		},
		{
			name: "exclude multiple event names keeps non-excluded",
			config: &xatu.EventFilterConfig{
				ExcludeEventNames: []string{
					xatu.Event_BEACON_API_ETH_V1_EVENTS_ATTESTATION.String(),
					xatu.Event_BEACON_API_ETH_V1_EVENTS_BLOCK.String(),
				},
			},
			event: &xatu.DecoratedEvent{
				Event: &xatu.Event{
					Name: xatu.Event_BEACON_API_ETH_V1_EVENTS_HEAD,
				},
			},
			wantDrop: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter, err := xatu.NewEventFilter(tt.config)
			require.NoError(t, err)

			dropped, err := filter.ShouldBeDropped(tt.event)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantDrop, dropped)
			}
		})
	}
}

func TestEventFilter_ExcludeModules(t *testing.T) {
	tests := []struct {
		name     string
		config   *xatu.EventFilterConfig
		event    *xatu.DecoratedEvent
		wantDrop bool
	}{
		{
			name: "exclude module drops matching event",
			config: &xatu.EventFilterConfig{
				ExcludeModules: []string{
					xatu.ModuleName_RELAY_MONITOR.String(),
				},
			},
			event: &xatu.DecoratedEvent{
				Event: &xatu.Event{
					Name: xatu.Event_BEACON_API_ETH_V1_EVENTS_BLOCK,
				},
				Meta: &xatu.Meta{
					Client: &xatu.ClientMeta{
						ModuleName: xatu.ModuleName_RELAY_MONITOR,
					},
				},
			},
			wantDrop: true,
		},
		{
			name: "exclude module keeps non-matching event",
			config: &xatu.EventFilterConfig{
				ExcludeModules: []string{
					xatu.ModuleName_RELAY_MONITOR.String(),
				},
			},
			event: &xatu.DecoratedEvent{
				Event: &xatu.Event{
					Name: xatu.Event_BEACON_API_ETH_V1_EVENTS_BLOCK,
				},
				Meta: &xatu.Meta{
					Client: &xatu.ClientMeta{
						ModuleName: xatu.ModuleName_EL_MIMICRY,
					},
				},
			},
			wantDrop: false,
		},
		{
			name: "exclude module keeps event with no module set",
			config: &xatu.EventFilterConfig{
				ExcludeModules: []string{
					xatu.ModuleName_RELAY_MONITOR.String(),
				},
			},
			event: &xatu.DecoratedEvent{
				Event: &xatu.Event{
					Name: xatu.Event_BEACON_API_ETH_V1_EVENTS_BLOCK,
				},
			},
			wantDrop: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter, err := xatu.NewEventFilter(tt.config)
			require.NoError(t, err)

			dropped, err := filter.ShouldBeDropped(tt.event)
			require.NoError(t, err)
			assert.Equal(t, tt.wantDrop, dropped)
		})
	}
}

func TestEventFilter_MixedIncludeExcludeAcrossDimensions(t *testing.T) {
	tests := []struct {
		name     string
		config   *xatu.EventFilterConfig
		event    *xatu.DecoratedEvent
		wantDrop bool
	}{
		{
			name: "include eventNames + exclude modules: both pass",
			config: &xatu.EventFilterConfig{
				EventNames: []string{
					xatu.Event_BEACON_API_ETH_V1_EVENTS_BLOCK.String(),
				},
				ExcludeModules: []string{
					xatu.ModuleName_RELAY_MONITOR.String(),
				},
			},
			event: &xatu.DecoratedEvent{
				Event: &xatu.Event{
					Name: xatu.Event_BEACON_API_ETH_V1_EVENTS_BLOCK,
				},
				Meta: &xatu.Meta{
					Client: &xatu.ClientMeta{
						ModuleName: xatu.ModuleName_EL_MIMICRY,
					},
				},
			},
			wantDrop: false,
		},
		{
			name: "include eventNames + exclude modules: event name excluded",
			config: &xatu.EventFilterConfig{
				EventNames: []string{
					xatu.Event_BEACON_API_ETH_V1_EVENTS_BLOCK.String(),
				},
				ExcludeModules: []string{
					xatu.ModuleName_RELAY_MONITOR.String(),
				},
			},
			event: &xatu.DecoratedEvent{
				Event: &xatu.Event{
					Name: xatu.Event_BEACON_API_ETH_V1_EVENTS_HEAD,
				},
				Meta: &xatu.Meta{
					Client: &xatu.ClientMeta{
						ModuleName: xatu.ModuleName_EL_MIMICRY,
					},
				},
			},
			wantDrop: true,
		},
		{
			name: "include eventNames + exclude modules: module excluded",
			config: &xatu.EventFilterConfig{
				EventNames: []string{
					xatu.Event_BEACON_API_ETH_V1_EVENTS_BLOCK.String(),
				},
				ExcludeModules: []string{
					xatu.ModuleName_RELAY_MONITOR.String(),
				},
			},
			event: &xatu.DecoratedEvent{
				Event: &xatu.Event{
					Name: xatu.Event_BEACON_API_ETH_V1_EVENTS_BLOCK,
				},
				Meta: &xatu.Meta{
					Client: &xatu.ClientMeta{
						ModuleName: xatu.ModuleName_RELAY_MONITOR,
					},
				},
			},
			wantDrop: true,
		},
		{
			name: "exclude eventNames + include modules: both pass",
			config: &xatu.EventFilterConfig{
				ExcludeEventNames: []string{
					xatu.Event_BEACON_API_ETH_V1_EVENTS_ATTESTATION.String(),
				},
				Modules: []string{
					xatu.ModuleName_RELAY_MONITOR.String(),
				},
			},
			event: &xatu.DecoratedEvent{
				Event: &xatu.Event{
					Name: xatu.Event_BEACON_API_ETH_V1_EVENTS_BLOCK,
				},
				Meta: &xatu.Meta{
					Client: &xatu.ClientMeta{
						ModuleName: xatu.ModuleName_RELAY_MONITOR,
					},
				},
			},
			wantDrop: false,
		},
		{
			name: "exclude eventNames + include modules: event excluded",
			config: &xatu.EventFilterConfig{
				ExcludeEventNames: []string{
					xatu.Event_BEACON_API_ETH_V1_EVENTS_ATTESTATION.String(),
				},
				Modules: []string{
					xatu.ModuleName_RELAY_MONITOR.String(),
				},
			},
			event: &xatu.DecoratedEvent{
				Event: &xatu.Event{
					Name: xatu.Event_BEACON_API_ETH_V1_EVENTS_ATTESTATION,
				},
				Meta: &xatu.Meta{
					Client: &xatu.ClientMeta{
						ModuleName: xatu.ModuleName_RELAY_MONITOR,
					},
				},
			},
			wantDrop: true,
		},
		{
			name: "exclude eventNames + include modules: module not included",
			config: &xatu.EventFilterConfig{
				ExcludeEventNames: []string{
					xatu.Event_BEACON_API_ETH_V1_EVENTS_ATTESTATION.String(),
				},
				Modules: []string{
					xatu.ModuleName_RELAY_MONITOR.String(),
				},
			},
			event: &xatu.DecoratedEvent{
				Event: &xatu.Event{
					Name: xatu.Event_BEACON_API_ETH_V1_EVENTS_BLOCK,
				},
				Meta: &xatu.Meta{
					Client: &xatu.ClientMeta{
						ModuleName: xatu.ModuleName_EL_MIMICRY,
					},
				},
			},
			wantDrop: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter, err := xatu.NewEventFilter(tt.config)
			require.NoError(t, err)

			dropped, err := filter.ShouldBeDropped(tt.event)
			require.NoError(t, err)
			assert.Equal(t, tt.wantDrop, dropped)
		})
	}
}

func TestEventFilter_Accessors(t *testing.T) {
	config := &xatu.EventFilterConfig{
		ExcludeEventNames: []string{
			xatu.Event_BEACON_API_ETH_V1_EVENTS_ATTESTATION.String(),
		},
		ExcludeModules: []string{
			xatu.ModuleName_RELAY_MONITOR.String(),
		},
	}

	filter, err := xatu.NewEventFilter(config)
	require.NoError(t, err)

	assert.Equal(t, config.ExcludeEventNames, filter.ExcludeEventNames())
	assert.Equal(t, config.ExcludeModules, filter.ExcludeModules())
	assert.Empty(t, filter.EventNames())
	assert.Empty(t, filter.Modules())
}

func TestEventFilter_EmptyExcludeLists(t *testing.T) {
	config := &xatu.EventFilterConfig{
		ExcludeEventNames: []string{},
		ExcludeModules:    []string{},
	}

	filter, err := xatu.NewEventFilter(config)
	require.NoError(t, err)

	event := &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name: xatu.Event_BEACON_API_ETH_V1_EVENTS_BLOCK,
		},
	}

	dropped, err := filter.ShouldBeDropped(event)
	require.NoError(t, err)
	assert.False(t, dropped)
}
