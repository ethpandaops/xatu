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

func Test_mapLibp2pCoreEventToXatuEvent(t *testing.T) {
	tests := []struct {
		name        string
		event       string
		want        string
		expectError bool
	}{
		{
			name:        "Map CONNECTED event",
			event:       TraceEvent_CONNECTED,
			want:        xatu.Event_LIBP2P_TRACE_CONNECTED.String(),
			expectError: false,
		},
		{
			name:        "Map DISCONNECTED event",
			event:       TraceEvent_DISCONNECTED,
			want:        xatu.Event_LIBP2P_TRACE_DISCONNECTED.String(),
			expectError: false,
		},
		{
			name:        "Unknown event type",
			event:       "UNKNOWN_EVENT",
			want:        "",
			expectError: true,
		},
		{
			name:        "Empty event type",
			event:       "",
			want:        "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := mapLibp2pCoreEventToXatuEvent(tt.event)
			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "unknown libp2p core event")
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

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
		event          *TraceEvent
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
			event: &TraceEvent{
				Type:      TraceEvent_CONNECTED,
				PeerID:    peerID,
				Timestamp: time.Now(),
				Payload: struct {
					RemotePeer   string
					RemoteMaddrs ma.Multiaddr
					AgentVersion string
					Direction    string
					Opened       time.Time
					Limited      bool
				}{
					RemotePeer:   "16Uiu2HAm7QJpRnfGoEeJwqdFaKeVLJm8GhKVYC6fiVBvJnpaXCnq",
					RemoteMaddrs: maddr,
					AgentVersion: "prysm/v4.0.0",
					Direction:    "inbound",
					Opened:       time.Now(),
					Limited:      false,
				},
			},
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
			event: &TraceEvent{
				Type:      TraceEvent_CONNECTED,
				PeerID:    peerID,
				Timestamp: time.Now(),
				Payload: struct {
					RemotePeer   string
					RemoteMaddrs ma.Multiaddr
					AgentVersion string
					Direction    string
					Opened       time.Time
					Limited      bool
				}{
					RemotePeer:   "16Uiu2HAm7QJpRnfGoEeJwqdFaKeVLJm8GhKVYC6fiVBvJnpaXCnq",
					RemoteMaddrs: maddr,
					AgentVersion: "lighthouse/v4.0.0",
					Direction:    "outbound",
					Opened:       time.Now(),
					Limited:      true,
				},
			},
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
			event: &TraceEvent{
				Type:      TraceEvent_CONNECTED,
				PeerID:    peerID,
				Timestamp: time.Now(),
				Payload: struct {
					RemotePeer   string
					RemoteMaddrs ma.Multiaddr
					AgentVersion string
					Direction    string
					Opened       time.Time
					Limited      bool
				}{
					RemotePeer:   "16Uiu2HAm7QJpRnfGoEeJwqdFaKeVLJm8GhKVYC6fiVBvJnpaXCnq",
					RemoteMaddrs: maddr,
					AgentVersion: "prysm/v4.0.0",
					Direction:    "inbound",
					Opened:       time.Now(),
					Limited:      false,
				},
			},
			expectError:    false,
			setupMockCalls: expectNoEvents,
		},
		{
			name: "CONNECTED with invalid payload",
			config: &Config{
				Events: EventConfig{
					ConnectedEnabled: true,
				},
			},
			event: &TraceEvent{
				Type:      TraceEvent_CONNECTED,
				PeerID:    peerID,
				Timestamp: time.Now(),
				Payload:   "invalid payload type",
			},
			expectError:    true,
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
			clientMeta := createTestClientMeta()
			traceMeta := createTestTraceMeta()

			// Call handleHermesLibp2pCoreEvent which routes to handleConnectedEvent
			err := mimicry.GetProcessor().handleHermesLibp2pCoreEvent(context.Background(), tt.event, clientMeta, traceMeta)

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
		event          *TraceEvent
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
			event: &TraceEvent{
				Type:      TraceEvent_DISCONNECTED,
				PeerID:    peerID,
				Timestamp: time.Now(),
				Payload: struct {
					RemotePeer   string
					RemoteMaddrs ma.Multiaddr
					AgentVersion string
					Direction    string
					Opened       time.Time
					Limited      bool
				}{
					RemotePeer:   "16Uiu2HAm7QJpRnfGoEeJwqdFaKeVLJm8GhKVYC6fiVBvJnpaXCnq",
					RemoteMaddrs: maddr,
					AgentVersion: "teku/v23.0.0",
					Direction:    "inbound",
					Opened:       time.Now().Add(-5 * time.Minute),
					Limited:      false,
				},
			},
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
			event: &TraceEvent{
				Type:      TraceEvent_DISCONNECTED,
				PeerID:    peerID,
				Timestamp: time.Now(),
				Payload: struct {
					RemotePeer   string
					RemoteMaddrs ma.Multiaddr
					AgentVersion string
					Direction    string
					Opened       time.Time
					Limited      bool
				}{
					RemotePeer:   "16Uiu2HAm7QJpRnfGoEeJwqdFaKeVLJm8GhKVYC6fiVBvJnpaXCnq",
					RemoteMaddrs: maddr,
					AgentVersion: "nimbus/v23.0.0",
					Direction:    "outbound",
					Opened:       time.Now().Add(-10 * time.Minute),
					Limited:      true,
				},
			},
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
			event: &TraceEvent{
				Type:      TraceEvent_DISCONNECTED,
				PeerID:    peerID,
				Timestamp: time.Now(),
				Payload: struct {
					RemotePeer   string
					RemoteMaddrs ma.Multiaddr
					AgentVersion string
					Direction    string
					Opened       time.Time
					Limited      bool
				}{
					RemotePeer:   "16Uiu2HAm7QJpRnfGoEeJwqdFaKeVLJm8GhKVYC6fiVBvJnpaXCnq",
					RemoteMaddrs: maddr,
					AgentVersion: "prysm/v4.0.0",
					Direction:    "inbound",
					Opened:       time.Now(),
					Limited:      false,
				},
			},
			expectError:    false,
			setupMockCalls: expectNoEvents,
		},
		{
			name: "DISCONNECTED with invalid payload",
			config: &Config{
				Events: EventConfig{
					DisconnectedEnabled: true,
				},
			},
			event: &TraceEvent{
				Type:      TraceEvent_DISCONNECTED,
				PeerID:    peerID,
				Timestamp: time.Now(),
				Payload:   map[string]any{"invalid": "payload"},
			},
			expectError:    true,
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
			clientMeta := createTestClientMeta()
			traceMeta := createTestTraceMeta()

			// Call handleHermesLibp2pCoreEvent which routes to handleDisconnectedEvent
			err := mimicry.GetProcessor().handleHermesLibp2pCoreEvent(context.Background(), tt.event, clientMeta, traceMeta)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func Test_handleHermesLibp2pCoreEvent(t *testing.T) {
	// Create a valid peer ID for testing
	peerID, err := peer.Decode(examplePeerID)
	require.NoError(t, err)

	// Create a multiaddr
	maddr, err := ma.NewMultiaddr("/ip4/10.0.0.1/tcp/30303")
	require.NoError(t, err)

	tests := []struct {
		name           string
		config         *Config
		event          *TraceEvent
		expectError    bool
		validateCalls  func(t *testing.T, events []*xatu.DecoratedEvent)
		setupMockCalls func(*mock.MockSink)
	}{
		{
			name: "Handle CONNECTED event through main handler",
			config: &Config{
				Events: EventConfig{
					ConnectedEnabled: true,
				},
			},
			event: &TraceEvent{
				Type:      TraceEvent_CONNECTED,
				PeerID:    peerID,
				Timestamp: time.Now(),
				Payload: struct {
					RemotePeer   string
					RemoteMaddrs ma.Multiaddr
					AgentVersion string
					Direction    string
					Opened       time.Time
					Limited      bool
				}{
					RemotePeer:   "16Uiu2HAm7QJpRnfGoEeJwqdFaKeVLJm8GhKVYC6fiVBvJnpaXCnq",
					RemoteMaddrs: maddr,
					AgentVersion: "prysm/v4.0.0",
					Direction:    "inbound",
					Opened:       time.Now(),
					Limited:      false,
				},
			},
			expectError: false,
			setupMockCalls: func(mockSink *mock.MockSink) {
				expectEventWithValidation(t, mockSink, func(t *testing.T, event *xatu.DecoratedEvent) {
					t.Helper()

					assert.Equal(t, xatu.Event_LIBP2P_TRACE_CONNECTED, event.Event.Name)
				})
			},
		},
		{
			name: "Handle DISCONNECTED event through main handler",
			config: &Config{
				Events: EventConfig{
					DisconnectedEnabled: true,
				},
			},
			event: &TraceEvent{
				Type:      TraceEvent_DISCONNECTED,
				PeerID:    peerID,
				Timestamp: time.Now(),
				Payload: struct {
					RemotePeer   string
					RemoteMaddrs ma.Multiaddr
					AgentVersion string
					Direction    string
					Opened       time.Time
					Limited      bool
				}{
					RemotePeer:   "16Uiu2HAm7QJpRnfGoEeJwqdFaKeVLJm8GhKVYC6fiVBvJnpaXCnq",
					RemoteMaddrs: maddr,
					AgentVersion: "lighthouse/v4.0.0",
					Direction:    "outbound",
					Opened:       time.Now().Add(-30 * time.Minute),
					Limited:      false,
				},
			},
			expectError: false,
			setupMockCalls: func(mockSink *mock.MockSink) {
				expectEventWithValidation(t, mockSink, func(t *testing.T, event *xatu.DecoratedEvent) {
					t.Helper()

					assert.Equal(t, xatu.Event_LIBP2P_TRACE_DISCONNECTED, event.Event.Name)
				})
			},
		},
		{
			name: "Handle unknown event type",
			config: &Config{
				Events: EventConfig{
					ConnectedEnabled:    true,
					DisconnectedEnabled: true,
				},
			},
			event: &TraceEvent{
				Type:      "UNKNOWN_CORE_EVENT",
				PeerID:    peerID,
				Timestamp: time.Now(),
				Payload:   struct{}{},
			},
			expectError: false, // Should not error, just log and skip
			setupMockCalls: func(mockSink *mock.MockSink) {
				// No events should be sent
			},
		},
		{
			name: "CONNECTED event disabled through config",
			config: &Config{
				Events: EventConfig{
					ConnectedEnabled: false,
				},
			},
			event: &TraceEvent{
				Type:      TraceEvent_CONNECTED,
				PeerID:    peerID,
				Timestamp: time.Now(),
				Payload: struct {
					RemotePeer   string
					RemoteMaddrs ma.Multiaddr
					AgentVersion string
					Direction    string
					Opened       time.Time
					Limited      bool
				}{
					RemotePeer:   "16Uiu2HAm7QJpRnfGoEeJwqdFaKeVLJm8GhKVYC6fiVBvJnpaXCnq",
					RemoteMaddrs: maddr,
					AgentVersion: "prysm/v4.0.0",
					Direction:    "inbound",
					Opened:       time.Now(),
					Limited:      false,
				},
			},
			expectError:    false,
			setupMockCalls: expectNoEvents,
		},
		{
			name: "DISCONNECTED event disabled through config",
			config: &Config{
				Events: EventConfig{
					DisconnectedEnabled: false,
				},
			},
			event: &TraceEvent{
				Type:      TraceEvent_DISCONNECTED,
				PeerID:    peerID,
				Timestamp: time.Now(),
				Payload: struct {
					RemotePeer   string
					RemoteMaddrs ma.Multiaddr
					AgentVersion string
					Direction    string
					Opened       time.Time
					Limited      bool
				}{
					RemotePeer:   "16Uiu2HAm7QJpRnfGoEeJwqdFaKeVLJm8GhKVYC6fiVBvJnpaXCnq",
					RemoteMaddrs: maddr,
					AgentVersion: "prysm/v4.0.0",
					Direction:    "inbound",
					Opened:       time.Now(),
					Limited:      false,
				},
			},
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
			clientMeta := createTestClientMeta()
			traceMeta := createTestTraceMeta()

			err := mimicry.GetProcessor().handleHermesLibp2pCoreEvent(context.Background(), tt.event, clientMeta, traceMeta)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
