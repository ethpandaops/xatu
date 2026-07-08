package xatu_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

const (
	testMutatedNetworkName = "my-network"
	testNetworkNameSuffix  = "--abc123"
)

func newNetworkEvent(name string, id uint64) *xatu.DecoratedEvent {
	return &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name: xatu.Event_BEACON_API_ETH_V1_EVENTS_BLOCK,
		},
		Meta: &xatu.Meta{
			Client: &xatu.ClientMeta{
				Ethereum: &xatu.ClientMeta_Ethereum{
					Network: &xatu.ClientMeta_Ethereum_Network{
						Name: name,
						Id:   id,
					},
				},
			},
		},
	}
}

func TestEventMutatorConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  xatu.EventMutatorConfig
		wantErr bool
	}{
		{
			name:   "empty config is valid",
			config: xatu.EventMutatorConfig{},
		},
		{
			name: "value only",
			config: xatu.EventMutatorConfig{
				MetaNetworkName: xatu.MetaNetworkNameMutationConfig{Value: testMutatedNetworkName},
			},
		},
		{
			name: "prefix and suffix",
			config: xatu.EventMutatorConfig{
				MetaNetworkName: xatu.MetaNetworkNameMutationConfig{Prefix: "pre-", Suffix: "-post"},
			},
		},
		{
			name: "value with suffix is invalid",
			config: xatu.EventMutatorConfig{
				MetaNetworkName: xatu.MetaNetworkNameMutationConfig{Value: testMutatedNetworkName, Suffix: "-post"},
			},
			wantErr: true,
		},
		{
			name: "value with prefix is invalid",
			config: xatu.EventMutatorConfig{
				MetaNetworkName: xatu.MetaNetworkNameMutationConfig{Value: testMutatedNetworkName, Prefix: "pre-"},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestEventMutator_IsEnabled(t *testing.T) {
	disabled, err := xatu.NewEventMutator(&xatu.EventMutatorConfig{})
	require.NoError(t, err)
	assert.False(t, disabled.IsEnabled())

	enabled, err := xatu.NewEventMutator(&xatu.EventMutatorConfig{
		MetaNetworkName: xatu.MetaNetworkNameMutationConfig{Suffix: "-a"},
	})
	require.NoError(t, err)
	assert.True(t, enabled.IsEnabled())
}

func TestEventMutator_Mutate_MetaNetworkName(t *testing.T) {
	tests := []struct {
		name     string
		config   xatu.MetaNetworkNameMutationConfig
		event    *xatu.DecoratedEvent
		wantName string
	}{
		{
			name:     "value replaces name",
			config:   xatu.MetaNetworkNameMutationConfig{Value: testMutatedNetworkName},
			event:    newNetworkEvent("devnet-1", 12345),
			wantName: testMutatedNetworkName,
		},
		{
			name:     "suffix appends to name",
			config:   xatu.MetaNetworkNameMutationConfig{Suffix: testNetworkNameSuffix},
			event:    newNetworkEvent("devnet-1", 12345),
			wantName: "devnet-1--abc123",
		},
		{
			name:     "prefix prepends to name",
			config:   xatu.MetaNetworkNameMutationConfig{Prefix: "relay/"},
			event:    newNetworkEvent("devnet-1", 12345),
			wantName: "relay/devnet-1",
		},
		{
			name:     "prefix and suffix combine",
			config:   xatu.MetaNetworkNameMutationConfig{Prefix: "relay/", Suffix: testNetworkNameSuffix},
			event:    newNetworkEvent("devnet-1", 12345),
			wantName: "relay/devnet-1--abc123",
		},
		{
			name:     "empty name is left untouched",
			config:   xatu.MetaNetworkNameMutationConfig{Suffix: testNetworkNameSuffix},
			event:    newNetworkEvent("", 12345),
			wantName: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mutator, err := xatu.NewEventMutator(&xatu.EventMutatorConfig{MetaNetworkName: tt.config})
			require.NoError(t, err)

			mutator.Mutate(tt.event)

			assert.Equal(t, tt.wantName, tt.event.GetMeta().GetClient().GetEthereum().GetNetwork().GetName())
			assert.Equal(t, uint64(12345), tt.event.GetMeta().GetClient().GetEthereum().GetNetwork().GetId())
		})
	}
}

func TestEventMutator_Mutate_MissingMetadata(t *testing.T) {
	mutator, err := xatu.NewEventMutator(&xatu.EventMutatorConfig{
		MetaNetworkName: xatu.MetaNetworkNameMutationConfig{Value: testMutatedNetworkName},
	})
	require.NoError(t, err)

	tests := []struct {
		name  string
		event *xatu.DecoratedEvent
	}{
		{name: "nil event", event: nil},
		{name: "no meta", event: &xatu.DecoratedEvent{Event: &xatu.Event{}}},
		{
			name: "no ethereum meta",
			event: &xatu.DecoratedEvent{
				Event: &xatu.Event{},
				Meta:  &xatu.Meta{Client: &xatu.ClientMeta{}},
			},
		},
		{
			name: "no network meta",
			event: &xatu.DecoratedEvent{
				Event: &xatu.Event{},
				Meta: &xatu.Meta{
					Client: &xatu.ClientMeta{Ethereum: &xatu.ClientMeta_Ethereum{}},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Must not panic, and must not fabricate network metadata.
			mutator.Mutate(tt.event)
			assert.Nil(t, tt.event.GetMeta().GetClient().GetEthereum().GetNetwork())
		})
	}
}

func TestEventMutator_Mutate_Disabled(t *testing.T) {
	mutator, err := xatu.NewEventMutator(&xatu.EventMutatorConfig{})
	require.NoError(t, err)

	event := newNetworkEvent("devnet-1", 12345)
	mutator.Mutate(event)

	assert.Equal(t, "devnet-1", event.GetMeta().GetClient().GetEthereum().GetNetwork().GetName())
}
