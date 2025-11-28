package clmimicry

import (
	"context"
	"testing"
	"time"

	"github.com/ethpandaops/xatu/pkg/output/mock"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/libp2p/go-libp2p/core/peer"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// Test removed: mapLibp2pCoreEventToXatuEvent no longer exists.
// Event mapping is now handled via type switches on typed events.

func Test_handleConnectedEvent(t *testing.T) {
	// Create a valid peer ID for testing
	peerID, err := peer.Decode(examplePeerID)
	require.NoError(t, err)

	// Create a multiaddr
	maddr, err := ma.NewMultiaddr("/ip4/127.0.0.1/tcp/9000")
	require.NoError(t, err)

	tests := []struct {
		name           string
		config         *Config
		event          *ConnectedEvent
		expectError    bool
		validateCalls  func(t *testing.T, events []*xatu.DecoratedEvent)
		setupMockCalls func(*mock.MockSink)
	}{
		{
			name: "Basic CONNECTED event",
			config: &Config{
				Events: EventConfig{
					ConnectedEnabled: true,
				},
			},
			event: NewConnectedEvent(
				time.Now(),
				peerID,
				"16Uiu2HAm7QJpRnfGoEeJwqdFaKeVLJm8GhKVYC6fiVBvJnpaXCnq",
				maddr,
				"prysm/v4.0.0",
				"inbound",
				time.Now(),
				false,
			),
			expectError: false,
			setupMockCalls: func(mockSink *mock.MockSink) {
				expectEventWithValidation(t, mockSink, func(t *testing.T, event *xatu.DecoratedEvent) {
					t.Helper()

					assert.Equal(t, xatu.Event_LIBP2P_TRACE_CONNECTED, event.Event.Name)
				})
			},
			validateCalls: func(t *testing.T, events []*xatu.DecoratedEvent) {
				t.Helper()

				require.Len(t, events, 1)
				assert.Equal(t, xatu.Event_LIBP2P_TRACE_CONNECTED, events[0].Event.Name)

				// Validate the connected data
				connectedData := events[0].GetLibp2PTraceConnected()
				require.NotNil(t, connectedData)
				assert.Equal(t, "16Uiu2HAm7QJpRnfGoEeJwqdFaKeVLJm8GhKVYC6fiVBvJnpaXCnq", connectedData.RemotePeer.GetValue())
				assert.Equal(t, "/ip4/127.0.0.1/tcp/9000", connectedData.RemoteMaddrs.GetValue())
				assert.Equal(t, "prysm/v4.0.0", connectedData.AgentVersion.GetValue())
				assert.Equal(t, "inbound", connectedData.Direction.GetValue())
				assert.False(t, connectedData.Limited.GetValue())
			},
		},
		{
			name: "CONNECTED event with limited connection",
			config: &Config{
				Events: EventConfig{
					ConnectedEnabled: true,
				},
			},
			event: NewConnectedEvent(
				time.Now(),
				peerID,
				"16Uiu2HAm7QJpRnfGoEeJwqdFaKeVLJm8GhKVYC6fiVBvJnpaXCnq",
				maddr,
				"lighthouse/v4.0.0",
				"outbound",
				time.Now(),
				true,
			),
			expectError: false,
			setupMockCalls: func(mockSink *mock.MockSink) {
				expectEventWithValidation(t, mockSink, func(t *testing.T, event *xatu.DecoratedEvent) {
					t.Helper()

					assert.Equal(t, xatu.Event_LIBP2P_TRACE_CONNECTED, event.Event.Name)
				})
			},
			validateCalls: func(t *testing.T, events []*xatu.DecoratedEvent) {
				t.Helper()

				require.Len(t, events, 1)
				connectedData := events[0].GetLibp2PTraceConnected()
				require.NotNil(t, connectedData)
				assert.Equal(t, "outbound", connectedData.Direction.GetValue())
				assert.True(t, connectedData.Limited.GetValue())
			},
		},
		{
			name: "CONNECTED event disabled",
			config: &Config{
				Events: EventConfig{
					ConnectedEnabled: false,
				},
			},
			event: NewConnectedEvent(
				time.Now(),
				peerID,
				"16Uiu2HAm7QJpRnfGoEeJwqdFaKeVLJm8GhKVYC6fiVBvJnpaXCnq",
				maddr,
				"prysm/v4.0.0",
				"inbound",
				time.Now(),
				false,
			),
			expectError:    false,
			setupMockCalls: expectNoEvents,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockSink := mock.NewMockSink(ctrl)
			tt.setupMockCalls(mockSink)

			mimicry := createTestMimicry(t, tt.config, mockSink)

			// Call HandleHermesEvent which routes to handleConnectedEvent
			err := mimicry.GetProcessor().HandleHermesEvent(context.Background(), tt.event)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func Test_handleDisconnectedEvent(t *testing.T) {
	// Create a valid peer ID for testing
	peerID, err := peer.Decode(examplePeerID)
	require.NoError(t, err)

	// Create a multiaddr
	maddr, err := ma.NewMultiaddr("/ip4/192.168.1.1/tcp/13000")
	require.NoError(t, err)

	tests := []struct {
		name           string
		config         *Config
		event          *DisconnectedEvent
		expectError    bool
		validateCalls  func(t *testing.T, events []*xatu.DecoratedEvent)
		setupMockCalls func(*mock.MockSink)
	}{
		{
			name: "Basic DISCONNECTED event",
			config: &Config{
				Events: EventConfig{
					DisconnectedEnabled: true,
				},
			},
			event: NewDisconnectedEvent(
				time.Now(),
				peerID,
				"16Uiu2HAm7QJpRnfGoEeJwqdFaKeVLJm8GhKVYC6fiVBvJnpaXCnq",
				maddr,
				"teku/v23.0.0",
				"inbound",
				time.Now().Add(-5*time.Minute),
				false,
			),
			expectError: false,
			setupMockCalls: func(mockSink *mock.MockSink) {
				expectEventWithValidation(t, mockSink, func(t *testing.T, event *xatu.DecoratedEvent) {
					t.Helper()

					assert.Equal(t, xatu.Event_LIBP2P_TRACE_DISCONNECTED, event.Event.Name)
				})
			},
			validateCalls: func(t *testing.T, events []*xatu.DecoratedEvent) {
				t.Helper()

				require.Len(t, events, 1)
				assert.Equal(t, xatu.Event_LIBP2P_TRACE_DISCONNECTED, events[0].Event.Name)

				// Validate the disconnected data
				disconnectedData := events[0].GetLibp2PTraceDisconnected()
				require.NotNil(t, disconnectedData)
				assert.Equal(t, "16Uiu2HAm7QJpRnfGoEeJwqdFaKeVLJm8GhKVYC6fiVBvJnpaXCnq", disconnectedData.RemotePeer.GetValue())
				assert.Equal(t, "/ip4/192.168.1.1/tcp/13000", disconnectedData.RemoteMaddrs.GetValue())
				assert.Equal(t, "teku/v23.0.0", disconnectedData.AgentVersion.GetValue())
				assert.Equal(t, "inbound", disconnectedData.Direction.GetValue())
				assert.False(t, disconnectedData.Limited.GetValue())
			},
		},
		{
			name: "DISCONNECTED event with limited connection",
			config: &Config{
				Events: EventConfig{
					DisconnectedEnabled: true,
				},
			},
			event: NewDisconnectedEvent(
				time.Now(),
				peerID,
				"16Uiu2HAm7QJpRnfGoEeJwqdFaKeVLJm8GhKVYC6fiVBvJnpaXCnq",
				maddr,
				"nimbus/v23.0.0",
				"outbound",
				time.Now().Add(-10*time.Minute),
				true,
			),
			expectError: false,
			setupMockCalls: func(mockSink *mock.MockSink) {
				expectEventWithValidation(t, mockSink, func(t *testing.T, event *xatu.DecoratedEvent) {
					t.Helper()

					assert.Equal(t, xatu.Event_LIBP2P_TRACE_DISCONNECTED, event.Event.Name)
				})
			},
			validateCalls: func(t *testing.T, events []*xatu.DecoratedEvent) {
				t.Helper()

				require.Len(t, events, 1)
				disconnectedData := events[0].GetLibp2PTraceDisconnected()
				require.NotNil(t, disconnectedData)
				assert.Equal(t, "outbound", disconnectedData.Direction.GetValue())
				assert.True(t, disconnectedData.Limited.GetValue())
			},
		},
		{
			name: "DISCONNECTED event disabled",
			config: &Config{
				Events: EventConfig{
					DisconnectedEnabled: false,
				},
			},
			event: NewDisconnectedEvent(
				time.Now(),
				peerID,
				"16Uiu2HAm7QJpRnfGoEeJwqdFaKeVLJm8GhKVYC6fiVBvJnpaXCnq",
				maddr,
				"prysm/v4.0.0",
				"inbound",
				time.Now(),
				false,
			),
			expectError:    false,
			setupMockCalls: expectNoEvents,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockSink := mock.NewMockSink(ctrl)
			tt.setupMockCalls(mockSink)

			mimicry := createTestMimicry(t, tt.config, mockSink)

			// Call HandleHermesEvent which routes to handleDisconnectedEvent
			err := mimicry.GetProcessor().HandleHermesEvent(context.Background(), tt.event)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// Test removed: handleHermesLibp2pCoreEvent no longer exists.
// Event handling is now done through HandleHermesEvent with typed events.
// Individual event type tests (Test_handleConnectedEvent, Test_handleDisconnectedEvent)
// provide sufficient coverage.
