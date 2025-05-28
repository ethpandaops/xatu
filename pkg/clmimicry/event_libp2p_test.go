package clmimicry

import (
	"context"
	"testing"
	"time"

	"github.com/ethpandaops/xatu/pkg/output"
	"github.com/ethpandaops/xatu/pkg/output/mock"
	"github.com/ethpandaops/xatu/pkg/proto/libp2p"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/google/uuid"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/probe-lab/hermes/host"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

const examplePeerID = "16Uiu2HAm68jFpjEsRyc1rksPWCorrqwoyR7qdPSvHcinzssnMXJq"

func Test_handleRecvRPCEvent(t *testing.T) {
	// Create a valid peer ID for testing
	peerID, err := peer.Decode(examplePeerID)
	require.NoError(t, err)

	tests := []struct {
		name           string
		config         *Config
		event          *host.TraceEvent
		expectError    bool
		expectCalls    int
		setupMockCalls func(*mock.MockSink)
	}{
		{
			name: "successful recv rpc event with control data",
			config: &Config{
				Events: EventConfig{
					RecvRPCEnabled: true,
				},
				Traces: TracesConfig{
					AlwaysRecordRootRpcEvents: true,
				},
			},
			event: &host.TraceEvent{
				Type:      "RECV_RPC",
				PeerID:    peerID,
				Timestamp: time.Now(),
				Payload: &host.RpcMeta{
					PeerID: peerID,
					Control: &host.RpcMetaControl{
						IHave: []host.RpcControlIHave{
							{
								TopicID: "test-topic",
								MsgIDs:  []string{"msg1", "msg2"},
							},
						},
					},
				},
			},
			expectError: false,
			expectCalls: 1,
			setupMockCalls: func(mockSink *mock.MockSink) {
				mockSink.EXPECT().
					HandleNewDecoratedEvents(gomock.Any(), gomock.Any()).
					Return(nil).
					Times(1)
			},
		},
		{
			name: "recv rpc event disabled",
			config: &Config{
				Events: EventConfig{
					RecvRPCEnabled: false,
				},
			},
			event: &host.TraceEvent{
				Type:      "RECV_RPC",
				PeerID:    peerID,
				Timestamp: time.Now(),
				Payload: &host.RpcMeta{
					PeerID: peerID,
				},
			},
			expectError: false,
			expectCalls: 0,
			setupMockCalls: func(mockSink *mock.MockSink) {
				// No calls expected
			},
		},
		{
			name: "recv rpc event with no control data and AlwaysRecordRootRpcEvents disabled",
			config: &Config{
				Events: EventConfig{
					RecvRPCEnabled: true,
				},
				Traces: TracesConfig{
					AlwaysRecordRootRpcEvents: false,
				},
			},
			event: &host.TraceEvent{
				Type:      "RECV_RPC",
				PeerID:    peerID,
				Timestamp: time.Now(),
				Payload: &host.RpcMeta{
					PeerID:  peerID,
					Control: nil,
				},
			},
			expectError: false,
			expectCalls: 0,
			setupMockCalls: func(mockSink *mock.MockSink) {
				// No calls expected since no meta and AlwaysRecordRootRpcEvents is false
			},
		},
		{
			name: "recv rpc event with AlwaysRecordRootRpcEvents enabled but no control data",
			config: &Config{
				Events: EventConfig{
					RecvRPCEnabled: true,
				},
				Traces: TracesConfig{
					AlwaysRecordRootRpcEvents: true,
				},
			},
			event: &host.TraceEvent{
				Type:      "RECV_RPC",
				PeerID:    peerID,
				Timestamp: time.Now(),
				Payload: &host.RpcMeta{
					PeerID:  peerID,
					Control: nil,
				},
			},
			expectError: false,
			expectCalls: 1,
			setupMockCalls: func(mockSink *mock.MockSink) {
				mockSink.EXPECT().
					HandleNewDecoratedEvents(gomock.Any(), gomock.Any()).
					Return(nil).
					Times(1)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockSink := mock.NewMockSink(ctrl)
			tt.setupMockCalls(mockSink)

			mimicry := &Mimicry{
				Config:  tt.config,
				sinks:   []output.Sink{mockSink},
				log:     logrus.NewEntry(logrus.New()),
				id:      uuid.New(),
				metrics: NewMetrics(tt.name),
			}

			clientMeta := &xatu.ClientMeta{
				Name: "test-client",
				Id:   uuid.New().String(),
			}

			traceMeta := &libp2p.TraceEventMetadata{
				PeerId: wrapperspb.String("test-peer"),
			}

			ctx := context.Background()
			err := mimicry.handleRecvRPCEvent(ctx, clientMeta, traceMeta, tt.event)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func Test_handleSendRPCEvent(t *testing.T) {
	// Create a valid peer ID for testing
	peerID, err := peer.Decode(examplePeerID)
	require.NoError(t, err)

	tests := []struct {
		name           string
		config         *Config
		event          *host.TraceEvent
		expectError    bool
		expectCalls    int
		setupMockCalls func(*mock.MockSink)
	}{
		{
			name: "successful send rpc event with control data",
			config: &Config{
				Events: EventConfig{
					SendRPCEnabled: true,
				},
				Traces: TracesConfig{
					AlwaysRecordRootRpcEvents: true,
				},
			},
			event: &host.TraceEvent{
				Type:      "SEND_RPC",
				PeerID:    peerID,
				Timestamp: time.Now(),
				Payload: &host.RpcMeta{
					PeerID: peerID,
					Control: &host.RpcMetaControl{
						IHave: []host.RpcControlIHave{
							{
								TopicID: "test-topic",
								MsgIDs:  []string{"msg1", "msg2"},
							},
						},
					},
				},
			},
			expectError: false,
			expectCalls: 1,
			setupMockCalls: func(mockSink *mock.MockSink) {
				mockSink.EXPECT().
					HandleNewDecoratedEvents(gomock.Any(), gomock.Any()).
					Return(nil).
					Times(1)
			},
		},
		{
			name: "send rpc event disabled",
			config: &Config{
				Events: EventConfig{
					SendRPCEnabled: false,
				},
			},
			event: &host.TraceEvent{
				Type:      "SEND_RPC",
				PeerID:    peerID,
				Timestamp: time.Now(),
				Payload: &host.RpcMeta{
					PeerID: peerID,
				},
			},
			expectError: false,
			expectCalls: 0,
			setupMockCalls: func(mockSink *mock.MockSink) {
				// No calls expected
			},
		},
		{
			name: "send rpc event with no control data and AlwaysRecordRootRpcEvents disabled",
			config: &Config{
				Events: EventConfig{
					SendRPCEnabled: true,
				},
				Traces: TracesConfig{
					AlwaysRecordRootRpcEvents: false,
				},
			},
			event: &host.TraceEvent{
				Type:      "SEND_RPC",
				PeerID:    peerID,
				Timestamp: time.Now(),
				Payload: &host.RpcMeta{
					PeerID:  peerID,
					Control: nil,
				},
			},
			expectError: false,
			expectCalls: 0,
			setupMockCalls: func(mockSink *mock.MockSink) {
				// No calls expected since no meta and AlwaysRecordRootRpcEvents is false
			},
		},
		{
			name: "send rpc event with AlwaysRecordRootRpcEvents enabled but no control data",
			config: &Config{
				Events: EventConfig{
					SendRPCEnabled: true,
				},
				Traces: TracesConfig{
					AlwaysRecordRootRpcEvents: true,
				},
			},
			event: &host.TraceEvent{
				Type:      "SEND_RPC",
				PeerID:    peerID,
				Timestamp: time.Now(),
				Payload: &host.RpcMeta{
					PeerID:  peerID,
					Control: nil,
				},
			},
			expectError: false,
			expectCalls: 1,
			setupMockCalls: func(mockSink *mock.MockSink) {
				mockSink.EXPECT().
					HandleNewDecoratedEvents(gomock.Any(), gomock.Any()).
					Return(nil).
					Times(1)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockSink := mock.NewMockSink(ctrl)
			tt.setupMockCalls(mockSink)

			mimicry := &Mimicry{
				Config:  tt.config,
				sinks:   []output.Sink{mockSink},
				log:     logrus.NewEntry(logrus.New()),
				id:      uuid.New(),
				metrics: NewMetrics(tt.name),
			}

			clientMeta := &xatu.ClientMeta{
				Name: "test-client",
				Id:   uuid.New().String(),
			}

			traceMeta := &libp2p.TraceEventMetadata{
				PeerId: wrapperspb.String("test-peer"),
			}

			ctx := context.Background()
			err := mimicry.handleSendRPCEvent(ctx, clientMeta, traceMeta, tt.event)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func Test_handleDropRPCEvent(t *testing.T) {
	// Create a valid peer ID for testing
	peerID, err := peer.Decode(examplePeerID)
	require.NoError(t, err)

	tests := []struct {
		name           string
		config         *Config
		event          *host.TraceEvent
		expectError    bool
		expectCalls    int
		setupMockCalls func(*mock.MockSink)
	}{
		{
			name: "successful drop rpc event with control data",
			config: &Config{
				Events: EventConfig{
					DropRPCEnabled: true,
				},
				Traces: TracesConfig{
					AlwaysRecordRootRpcEvents: true,
				},
			},
			event: &host.TraceEvent{
				Type:      "DROP_RPC",
				PeerID:    peerID,
				Timestamp: time.Now(),
				Payload: &host.RpcMeta{
					PeerID: peerID,
					Control: &host.RpcMetaControl{
						IWant: []host.RpcControlIWant{
							{
								MsgIDs: []string{"msg1", "msg2"},
							},
						},
					},
				},
			},
			expectError: false,
			expectCalls: 1,
			setupMockCalls: func(mockSink *mock.MockSink) {
				mockSink.EXPECT().
					HandleNewDecoratedEvents(gomock.Any(), gomock.Any()).
					Return(nil).
					Times(1)
			},
		},
		{
			name: "drop rpc event disabled",
			config: &Config{
				Events: EventConfig{
					DropRPCEnabled: false,
				},
			},
			event: &host.TraceEvent{
				Type:      "DROP_RPC",
				PeerID:    peerID,
				Timestamp: time.Now(),
				Payload: &host.RpcMeta{
					PeerID: peerID,
				},
			},
			expectError: false,
			expectCalls: 0,
			setupMockCalls: func(mockSink *mock.MockSink) {
				// No calls expected
			},
		},
		{
			name: "drop rpc event with no control data and AlwaysRecordRootRpcEvents disabled",
			config: &Config{
				Events: EventConfig{
					DropRPCEnabled: true,
				},
				Traces: TracesConfig{
					AlwaysRecordRootRpcEvents: false,
				},
			},
			event: &host.TraceEvent{
				Type:      "DROP_RPC",
				PeerID:    peerID,
				Timestamp: time.Now(),
				Payload: &host.RpcMeta{
					PeerID:  peerID,
					Control: nil,
				},
			},
			expectError: false,
			expectCalls: 0,
			setupMockCalls: func(mockSink *mock.MockSink) {
				// No calls expected since no meta and AlwaysRecordRootRpcEvents is false
			},
		},
		{
			name: "drop rpc event with AlwaysRecordRootRpcEvents enabled but no control data",
			config: &Config{
				Events: EventConfig{
					DropRPCEnabled: true,
				},
				Traces: TracesConfig{
					AlwaysRecordRootRpcEvents: true,
				},
			},
			event: &host.TraceEvent{
				Type:      "DROP_RPC",
				PeerID:    peerID,
				Timestamp: time.Now(),
				Payload: &host.RpcMeta{
					PeerID:  peerID,
					Control: nil,
				},
			},
			expectError: false,
			expectCalls: 1,
			setupMockCalls: func(mockSink *mock.MockSink) {
				mockSink.EXPECT().
					HandleNewDecoratedEvents(gomock.Any(), gomock.Any()).
					Return(nil).
					Times(1)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockSink := mock.NewMockSink(ctrl)
			tt.setupMockCalls(mockSink)

			mimicry := &Mimicry{
				Config:  tt.config,
				sinks:   []output.Sink{mockSink},
				log:     logrus.NewEntry(logrus.New()),
				id:      uuid.New(),
				metrics: NewMetrics(tt.name),
			}

			clientMeta := &xatu.ClientMeta{
				Name: "test-client",
				Id:   uuid.New().String(),
			}

			traceMeta := &libp2p.TraceEventMetadata{
				PeerId: wrapperspb.String("test-peer"),
			}

			ctx := context.Background()
			err := mimicry.handleDropRPCEvent(ctx, clientMeta, traceMeta, tt.event)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func Test_handleAddPeerEvent(t *testing.T) {
	// Create a valid peer ID for testing
	peerID, err := peer.Decode(examplePeerID)
	require.NoError(t, err)

	tests := []struct {
		name           string
		config         *Config
		event          *host.TraceEvent
		expectError    bool
		expectCalls    int
		setupMockCalls func(*mock.MockSink)
	}{
		{
			name: "successful add peer event",
			config: &Config{
				Events: EventConfig{
					AddPeerEnabled: true,
				},
			},
			event: &host.TraceEvent{
				Type:      "ADD_PEER",
				PeerID:    peerID,
				Timestamp: time.Now(),
				Payload: map[string]any{
					"PeerID":   peerID,
					"Protocol": protocol.ID("test-protocol"),
				},
			},
			expectError: false,
			expectCalls: 1,
			setupMockCalls: func(mockSink *mock.MockSink) {
				mockSink.EXPECT().
					HandleNewDecoratedEvent(gomock.Any(), gomock.Any()).
					Return(nil).
					Times(1)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockSink := mock.NewMockSink(ctrl)
			tt.setupMockCalls(mockSink)

			mimicry := &Mimicry{
				Config:  tt.config,
				sinks:   []output.Sink{mockSink},
				log:     logrus.NewEntry(logrus.New()),
				id:      uuid.New(),
				metrics: NewMetrics(tt.name),
			}

			clientMeta := &xatu.ClientMeta{
				Name: "test-client",
				Id:   uuid.New().String(),
			}

			traceMeta := &libp2p.TraceEventMetadata{
				PeerId: wrapperspb.String("test-peer"),
			}

			ctx := context.Background()
			err := mimicry.handleAddPeerEvent(ctx, clientMeta, traceMeta, tt.event)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func Test_handleRemovePeerEvent(t *testing.T) {
	// Create a valid peer ID for testing
	peerID, err := peer.Decode(examplePeerID)
	require.NoError(t, err)

	tests := []struct {
		name           string
		config         *Config
		event          *host.TraceEvent
		expectError    bool
		expectCalls    int
		setupMockCalls func(*mock.MockSink)
	}{
		{
			name: "successful remove peer event",
			config: &Config{
				Events: EventConfig{
					RemovePeerEnabled: true,
				},
			},
			event: &host.TraceEvent{
				Type:      "REMOVE_PEER",
				PeerID:    peerID,
				Timestamp: time.Now(),
				Payload: map[string]any{
					"PeerID":   peerID,
					"Protocol": protocol.ID("test-protocol"),
				},
			},
			expectError: false,
			expectCalls: 1,
			setupMockCalls: func(mockSink *mock.MockSink) {
				mockSink.EXPECT().
					HandleNewDecoratedEvent(gomock.Any(), gomock.Any()).
					Return(nil).
					Times(1)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockSink := mock.NewMockSink(ctrl)
			tt.setupMockCalls(mockSink)

			mimicry := &Mimicry{
				Config:  tt.config,
				sinks:   []output.Sink{mockSink},
				log:     logrus.NewEntry(logrus.New()),
				id:      uuid.New(),
				metrics: NewMetrics(tt.name),
			}

			clientMeta := &xatu.ClientMeta{
				Name: "test-client",
				Id:   uuid.New().String(),
			}

			traceMeta := &libp2p.TraceEventMetadata{
				PeerId: wrapperspb.String("test-peer"),
			}

			ctx := context.Background()
			err := mimicry.handleRemovePeerEvent(ctx, clientMeta, traceMeta, tt.event)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func Test_handleJoinEvent(t *testing.T) {
	// Create a valid peer ID for testing
	peerID, err := peer.Decode(examplePeerID)
	require.NoError(t, err)

	tests := []struct {
		name           string
		config         *Config
		event          *host.TraceEvent
		expectError    bool
		expectCalls    int
		setupMockCalls func(*mock.MockSink)
	}{
		{
			name: "successful join event",
			config: &Config{
				Events: EventConfig{
					JoinEnabled: true,
				},
			},
			event: &host.TraceEvent{
				Type:      "JOIN",
				PeerID:    peerID,
				Timestamp: time.Now(),
				Payload: map[string]any{
					"Topic": "test-topic",
				},
			},
			expectError: false,
			expectCalls: 1,
			setupMockCalls: func(mockSink *mock.MockSink) {
				mockSink.EXPECT().
					HandleNewDecoratedEvent(gomock.Any(), gomock.Any()).
					Return(nil).
					Times(1)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockSink := mock.NewMockSink(ctrl)
			tt.setupMockCalls(mockSink)

			mimicry := &Mimicry{
				Config:  tt.config,
				sinks:   []output.Sink{mockSink},
				log:     logrus.NewEntry(logrus.New()),
				id:      uuid.New(),
				metrics: NewMetrics(tt.name),
			}

			clientMeta := &xatu.ClientMeta{
				Name: "test-client",
				Id:   uuid.New().String(),
			}

			traceMeta := &libp2p.TraceEventMetadata{
				PeerId: wrapperspb.String("test-peer"),
			}

			ctx := context.Background()
			err := mimicry.handleJoinEvent(ctx, clientMeta, traceMeta, tt.event)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func Test_handleLeaveEvent(t *testing.T) {
	// Create a valid peer ID for testing
	peerID, err := peer.Decode(examplePeerID)
	require.NoError(t, err)

	tests := []struct {
		name           string
		config         *Config
		event          *host.TraceEvent
		expectError    bool
		expectCalls    int
		setupMockCalls func(*mock.MockSink)
	}{
		{
			name: "successful leave event",
			config: &Config{
				Events: EventConfig{
					LeaveEnabled: true,
				},
			},
			event: &host.TraceEvent{
				Type:      "LEAVE",
				PeerID:    peerID,
				Timestamp: time.Now(),
				Payload: map[string]any{
					"Topic": "test-topic",
				},
			},
			expectError: false,
			expectCalls: 1,
			setupMockCalls: func(mockSink *mock.MockSink) {
				mockSink.EXPECT().
					HandleNewDecoratedEvent(gomock.Any(), gomock.Any()).
					Return(nil).
					Times(1)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockSink := mock.NewMockSink(ctrl)
			tt.setupMockCalls(mockSink)

			mimicry := &Mimicry{
				Config:  tt.config,
				sinks:   []output.Sink{mockSink},
				log:     logrus.NewEntry(logrus.New()),
				id:      uuid.New(),
				metrics: NewMetrics(tt.name),
			}

			clientMeta := &xatu.ClientMeta{
				Name: "test-client",
				Id:   uuid.New().String(),
			}

			traceMeta := &libp2p.TraceEventMetadata{
				PeerId: wrapperspb.String("test-peer"),
			}

			ctx := context.Background()
			err := mimicry.handleLeaveEvent(ctx, clientMeta, traceMeta, tt.event)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func Test_handleGraftEvent(t *testing.T) {
	// Create a valid peer ID for testing
	peerID, err := peer.Decode(examplePeerID)
	require.NoError(t, err)

	tests := []struct {
		name           string
		config         *Config
		event          *host.TraceEvent
		expectError    bool
		expectCalls    int
		setupMockCalls func(*mock.MockSink)
	}{
		{
			name: "successful graft event",
			config: &Config{
				Events: EventConfig{
					GraftEnabled: true,
				},
			},
			event: &host.TraceEvent{
				Type:      "GRAFT",
				PeerID:    peerID,
				Timestamp: time.Now(),
				Payload: map[string]any{
					"PeerID": peerID,
					"Topic":  "test-topic",
				},
			},
			expectError: false,
			expectCalls: 1,
			setupMockCalls: func(mockSink *mock.MockSink) {
				mockSink.EXPECT().
					HandleNewDecoratedEvent(gomock.Any(), gomock.Any()).
					Return(nil).
					Times(1)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockSink := mock.NewMockSink(ctrl)
			tt.setupMockCalls(mockSink)

			mimicry := &Mimicry{
				Config:  tt.config,
				sinks:   []output.Sink{mockSink},
				log:     logrus.NewEntry(logrus.New()),
				id:      uuid.New(),
				metrics: NewMetrics(tt.name),
			}

			clientMeta := &xatu.ClientMeta{
				Name: "test-client",
				Id:   uuid.New().String(),
			}

			traceMeta := &libp2p.TraceEventMetadata{
				PeerId: wrapperspb.String("test-peer"),
			}

			ctx := context.Background()
			err := mimicry.handleGraftEvent(ctx, clientMeta, traceMeta, tt.event)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func Test_handlePruneEvent(t *testing.T) {
	// Create a valid peer ID for testing
	peerID, err := peer.Decode(examplePeerID)
	require.NoError(t, err)

	tests := []struct {
		name           string
		config         *Config
		event          *host.TraceEvent
		expectError    bool
		expectCalls    int
		setupMockCalls func(*mock.MockSink)
	}{
		{
			name: "successful prune event",
			config: &Config{
				Events: EventConfig{
					PruneEnabled: true,
				},
			},
			event: &host.TraceEvent{
				Type:      "PRUNE",
				PeerID:    peerID,
				Timestamp: time.Now(),
				Payload: map[string]any{
					"PeerID": peerID,
					"Topic":  "test-topic",
				},
			},
			expectError: false,
			expectCalls: 1,
			setupMockCalls: func(mockSink *mock.MockSink) {
				mockSink.EXPECT().
					HandleNewDecoratedEvent(gomock.Any(), gomock.Any()).
					Return(nil).
					Times(1)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockSink := mock.NewMockSink(ctrl)
			tt.setupMockCalls(mockSink)

			mimicry := &Mimicry{
				Config:  tt.config,
				sinks:   []output.Sink{mockSink},
				log:     logrus.NewEntry(logrus.New()),
				id:      uuid.New(),
				metrics: NewMetrics(tt.name),
			}

			clientMeta := &xatu.ClientMeta{
				Name: "test-client",
				Id:   uuid.New().String(),
			}

			traceMeta := &libp2p.TraceEventMetadata{
				PeerId: wrapperspb.String("test-peer"),
			}

			ctx := context.Background()
			err := mimicry.handlePruneEvent(ctx, clientMeta, traceMeta, tt.event)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func Test_handlePublishMessageEvent(t *testing.T) {
	// Create a valid peer ID for testing
	peerID, err := peer.Decode(examplePeerID)
	require.NoError(t, err)

	tests := []struct {
		name           string
		config         *Config
		event          *host.TraceEvent
		expectError    bool
		expectCalls    int
		setupMockCalls func(*mock.MockSink)
	}{
		{
			name: "successful publish message event",
			config: &Config{
				Events: EventConfig{
					PublishMessageEnabled: true,
				},
			},
			event: &host.TraceEvent{
				Type:      "PUBLISH_MESSAGE",
				PeerID:    peerID,
				Timestamp: time.Now(),
				Payload: map[string]any{
					"MsgID": "test-message-id",
					"Topic": "test-topic",
				},
			},
			expectError: false,
			expectCalls: 1,
			setupMockCalls: func(mockSink *mock.MockSink) {
				mockSink.EXPECT().
					HandleNewDecoratedEvent(gomock.Any(), gomock.Any()).
					Return(nil).
					Times(1)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockSink := mock.NewMockSink(ctrl)
			tt.setupMockCalls(mockSink)

			mimicry := &Mimicry{
				Config:  tt.config,
				sinks:   []output.Sink{mockSink},
				log:     logrus.NewEntry(logrus.New()),
				id:      uuid.New(),
				metrics: NewMetrics(tt.name),
			}

			clientMeta := &xatu.ClientMeta{
				Name: "test-client",
				Id:   uuid.New().String(),
			}

			traceMeta := &libp2p.TraceEventMetadata{
				PeerId: wrapperspb.String("test-peer"),
			}

			ctx := context.Background()
			err := mimicry.handlePublishMessageEvent(ctx, clientMeta, traceMeta, tt.event)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func Test_handleRejectMessageEvent(t *testing.T) {
	// Create a valid peer ID for testing
	peerID, err := peer.Decode(examplePeerID)
	require.NoError(t, err)

	tests := []struct {
		name           string
		config         *Config
		event          *host.TraceEvent
		expectError    bool
		expectCalls    int
		setupMockCalls func(*mock.MockSink)
	}{
		{
			name: "successful reject message event",
			config: &Config{
				Events: EventConfig{
					RejectMessageEnabled: true,
				},
			},
			event: &host.TraceEvent{
				Type:      "REJECT_MESSAGE",
				PeerID:    peerID,
				Timestamp: time.Now(),
				Payload: map[string]any{
					"MsgID":   "test-message-id",
					"Topic":   "test-topic",
					"PeerID":  peerID,
					"Reason":  "test-reason",
					"Local":   false,
					"MsgSize": 100,
					"Seq":     "01",
				},
			},
			expectError: false,
			expectCalls: 1,
			setupMockCalls: func(mockSink *mock.MockSink) {
				mockSink.EXPECT().
					HandleNewDecoratedEvent(gomock.Any(), gomock.Any()).
					Return(nil).
					Times(1)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockSink := mock.NewMockSink(ctrl)
			tt.setupMockCalls(mockSink)

			mimicry := &Mimicry{
				Config:  tt.config,
				sinks:   []output.Sink{mockSink},
				log:     logrus.NewEntry(logrus.New()),
				id:      uuid.New(),
				metrics: NewMetrics(tt.name),
			}

			clientMeta := &xatu.ClientMeta{
				Name: "test-client",
				Id:   uuid.New().String(),
			}

			traceMeta := &libp2p.TraceEventMetadata{
				PeerId: wrapperspb.String("test-peer"),
			}

			ctx := context.Background()
			err := mimicry.handleRejectMessageEvent(ctx, clientMeta, traceMeta, tt.event)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func Test_handleDuplicateMessageEvent(t *testing.T) {
	// Create a valid peer ID for testing
	peerID, err := peer.Decode(examplePeerID)
	require.NoError(t, err)

	tests := []struct {
		name           string
		config         *Config
		event          *host.TraceEvent
		expectError    bool
		expectCalls    int
		setupMockCalls func(*mock.MockSink)
	}{
		{
			name: "successful duplicate message event",
			config: &Config{
				Events: EventConfig{
					DuplicateMessageEnabled: true,
				},
			},
			event: &host.TraceEvent{
				Type:      "DUPLICATE_MESSAGE",
				PeerID:    peerID,
				Timestamp: time.Now(),
				Payload: map[string]any{
					"MsgID":   "test-message-id",
					"Topic":   "test-topic",
					"PeerID":  peerID,
					"Local":   false,
					"MsgSize": 100,
					"Seq":     "01",
				},
			},
			expectError: false,
			expectCalls: 1,
			setupMockCalls: func(mockSink *mock.MockSink) {
				mockSink.EXPECT().
					HandleNewDecoratedEvent(gomock.Any(), gomock.Any()).
					Return(nil).
					Times(1)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockSink := mock.NewMockSink(ctrl)
			tt.setupMockCalls(mockSink)

			mimicry := &Mimicry{
				Config:  tt.config,
				sinks:   []output.Sink{mockSink},
				log:     logrus.NewEntry(logrus.New()),
				id:      uuid.New(),
				metrics: NewMetrics(tt.name),
			}

			clientMeta := &xatu.ClientMeta{
				Name: "test-client",
				Id:   uuid.New().String(),
			}

			traceMeta := &libp2p.TraceEventMetadata{
				PeerId: wrapperspb.String("test-peer"),
			}

			ctx := context.Background()
			err := mimicry.handleDuplicateMessageEvent(ctx, clientMeta, traceMeta, tt.event)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func Test_handleDeliverMessageEvent(t *testing.T) {
	// Create a valid peer ID for testing
	peerID, err := peer.Decode(examplePeerID)
	require.NoError(t, err)

	tests := []struct {
		name           string
		config         *Config
		event          *host.TraceEvent
		expectError    bool
		expectCalls    int
		setupMockCalls func(*mock.MockSink)
	}{
		{
			name: "successful deliver message event",
			config: &Config{
				Events: EventConfig{
					DeliverMessageEnabled: true,
				},
			},
			event: &host.TraceEvent{
				Type:      "DELIVER_MESSAGE",
				PeerID:    peerID,
				Timestamp: time.Now(),
				Payload: map[string]any{
					"MsgID":   "test-message-id",
					"Topic":   "test-topic",
					"PeerID":  peerID,
					"Local":   false,
					"MsgSize": 100,
					"Seq":     "01",
				},
			},
			expectError: false,
			expectCalls: 1,
			setupMockCalls: func(mockSink *mock.MockSink) {
				mockSink.EXPECT().
					HandleNewDecoratedEvent(gomock.Any(), gomock.Any()).
					Return(nil).
					Times(1)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockSink := mock.NewMockSink(ctrl)
			tt.setupMockCalls(mockSink)

			mimicry := &Mimicry{
				Config:  tt.config,
				sinks:   []output.Sink{mockSink},
				log:     logrus.NewEntry(logrus.New()),
				id:      uuid.New(),
				metrics: NewMetrics(tt.name),
			}

			clientMeta := &xatu.ClientMeta{
				Name: "test-client",
				Id:   uuid.New().String(),
			}

			traceMeta := &libp2p.TraceEventMetadata{
				PeerId: wrapperspb.String("test-peer"),
			}

			ctx := context.Background()
			err := mimicry.handleDeliverMessageEvent(ctx, clientMeta, traceMeta, tt.event)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func Test_handleRecvRPCEvent_DetailedControlMessages(t *testing.T) {
	// Create a valid peer ID for testing
	peerID, err := peer.Decode(examplePeerID)
	require.NoError(t, err)

	tests := []struct {
		name               string
		config             *Config
		controlData        *host.RpcMetaControl
		expectedEventTypes []string
		expectedEventCount int
	}{
		{
			name: "all control message types enabled",
			config: &Config{
				Events: EventConfig{
					RecvRPCEnabled:                 true,
					RpcMetaControlIHaveEnabled:     true,
					RpcMetaControlIWantEnabled:     true,
					RpcMetaControlIDontWantEnabled: true,
					RpcMetaControlGraftEnabled:     true,
					RpcMetaControlPruneEnabled:     true,
				},
			},
			controlData: &host.RpcMetaControl{
				IHave: []host.RpcControlIHave{
					{
						TopicID: "test-topic-1",
						MsgIDs:  []string{"msg1", "msg2"},
					},
				},
				IWant: []host.RpcControlIWant{
					{
						MsgIDs: []string{"msg3", "msg4"},
					},
				},
				Idontwant: []host.RpcControlIdontWant{
					{
						MsgIDs: []string{"msg5", "msg6"},
					},
				},
				Graft: []host.RpcControlGraft{
					{
						TopicID: "graft-topic",
					},
				},
				Prune: []host.RpcControlPrune{
					{
						TopicID: "prune-topic",
						PeerIDs: []peer.ID{peerID},
					},
				},
			},
			expectedEventTypes: []string{
				"LIBP2P_TRACE_RECV_RPC",
				"LIBP2P_TRACE_RPC_META_CONTROL_IHAVE",
				"LIBP2P_TRACE_RPC_META_CONTROL_IWANT",
				"LIBP2P_TRACE_RPC_META_CONTROL_IDONTWANT",
				"LIBP2P_TRACE_RPC_META_CONTROL_GRAFT",
				"LIBP2P_TRACE_RPC_META_CONTROL_PRUNE",
			},
			expectedEventCount: 9,
		},
		{
			name: "only IHave enabled",
			config: &Config{
				Events: EventConfig{
					RecvRPCEnabled:                 true,
					RpcMetaControlIHaveEnabled:     true,
					RpcMetaControlIWantEnabled:     false,
					RpcMetaControlIDontWantEnabled: false,
					RpcMetaControlGraftEnabled:     false,
					RpcMetaControlPruneEnabled:     false,
				},
			},
			controlData: &host.RpcMetaControl{
				IHave: []host.RpcControlIHave{
					{
						TopicID: "test-topic-1",
						MsgIDs:  []string{"msg1", "msg2"},
					},
				},
				IWant: []host.RpcControlIWant{
					{
						MsgIDs: []string{"msg3", "msg4"},
					},
				},
			},
			expectedEventTypes: []string{
				"LIBP2P_TRACE_RECV_RPC",
				"LIBP2P_TRACE_RPC_META_CONTROL_IHAVE",
			},
			expectedEventCount: 3,
		},
		{
			name: "only Graft and Prune enabled",
			config: &Config{
				Events: EventConfig{
					RecvRPCEnabled:                 true,
					RpcMetaControlIHaveEnabled:     false,
					RpcMetaControlIWantEnabled:     false,
					RpcMetaControlIDontWantEnabled: false,
					RpcMetaControlGraftEnabled:     true,
					RpcMetaControlPruneEnabled:     true,
				},
			},
			controlData: &host.RpcMetaControl{
				IHave: []host.RpcControlIHave{
					{
						TopicID: "test-topic-1",
						MsgIDs:  []string{"msg1", "msg2"},
					},
				},
				Graft: []host.RpcControlGraft{
					{
						TopicID: "graft-topic",
					},
				},
				Prune: []host.RpcControlPrune{
					{
						TopicID: "prune-topic",
						PeerIDs: []peer.ID{peerID},
					},
				},
			},
			expectedEventTypes: []string{
				"LIBP2P_TRACE_RECV_RPC",
				"LIBP2P_TRACE_RPC_META_CONTROL_GRAFT",
				"LIBP2P_TRACE_RPC_META_CONTROL_PRUNE",
			},
			expectedEventCount: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockSink := mock.NewMockSink(ctrl)

			// Track actual event types received
			receivedEventTypes := []string{}

			// Setup mock to capture and verify events
			mockSink.EXPECT().HandleNewDecoratedEvents(gomock.Any(), gomock.Any()).
				Do(func(ctx context.Context, events []*xatu.DecoratedEvent) {
					for _, event := range events {
						receivedEventTypes = append(receivedEventTypes, event.Event.Name.String())
					}
				}).
				Return(nil).
				Times(1)

			mimicry := &Mimicry{
				Config:  tt.config,
				sinks:   []output.Sink{mockSink},
				log:     logrus.NewEntry(logrus.New()),
				id:      uuid.New(),
				metrics: NewMetrics(tt.name),
			}

			clientMeta := &xatu.ClientMeta{
				Name: "test-client",
				Id:   uuid.New().String(),
			}

			traceMeta := &libp2p.TraceEventMetadata{
				PeerId: wrapperspb.String("test-peer"),
			}

			event := &host.TraceEvent{
				Type:      "RECV_RPC",
				PeerID:    peerID,
				Timestamp: time.Now(),
				Payload: &host.RpcMeta{
					PeerID:  peerID,
					Control: tt.controlData,
				},
			}

			ctx := context.Background()
			err := mimicry.handleRecvRPCEvent(ctx, clientMeta, traceMeta, event)

			require.NoError(t, err)
			assert.Equal(t, tt.expectedEventCount, len(receivedEventTypes))

			// Verify that all expected event types are present
			for _, expectedType := range tt.expectedEventTypes {
				assert.Contains(t, receivedEventTypes, expectedType)
			}
		})
	}
}
