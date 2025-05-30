package clmimicry

import (
	"context"
	"fmt"
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

func Test_handleRecvRPCEvent_DetailedSubscriptionMessages(t *testing.T) {
	// Create a valid peer ID for testing
	peerID, err := peer.Decode(examplePeerID)
	require.NoError(t, err)

	tests := []struct {
		name               string
		config             *Config
		subscriptions      []host.RpcMetaSub
		expectedEventTypes []string
		expectedEventCount int
		expectMockCall     bool
	}{
		{
			name: "subscription messages enabled",
			config: &Config{
				Events: EventConfig{
					RecvRPCEnabled:             true,
					RpcMetaSubscriptionEnabled: true,
				},
				Traces: TracesConfig{
					AlwaysRecordRootRpcEvents: true,
				},
			},
			subscriptions: []host.RpcMetaSub{
				{
					Subscribe: true,
					TopicID:   "test-topic-1",
				},
				{
					Subscribe: false,
					TopicID:   "test-topic-2",
				},
			},
			expectedEventTypes: []string{
				"LIBP2P_TRACE_RECV_RPC",
				"LIBP2P_TRACE_RPC_META_SUBSCRIPTION",
			},
			expectedEventCount: 3,
			expectMockCall:     true,
		},
		{
			name: "subscription messages disabled",
			config: &Config{
				Events: EventConfig{
					RecvRPCEnabled:             true,
					RpcMetaSubscriptionEnabled: false,
				},
				Traces: TracesConfig{
					AlwaysRecordRootRpcEvents: true,
				},
			},
			subscriptions: []host.RpcMetaSub{
				{
					Subscribe: true,
					TopicID:   "test-topic-1",
				},
			},
			expectedEventTypes: []string{
				"LIBP2P_TRACE_RECV_RPC",
			},
			expectedEventCount: 1,
			expectMockCall:     true,
		},
		{
			name: "subscription messages disabled and no AlwaysRecordRootRpcEvents",
			config: &Config{
				Events: EventConfig{
					RecvRPCEnabled:             true,
					RpcMetaSubscriptionEnabled: false,
				},
				Traces: TracesConfig{
					AlwaysRecordRootRpcEvents: false,
				},
			},
			subscriptions: []host.RpcMetaSub{
				{
					Subscribe: true,
					TopicID:   "test-topic-1",
				},
			},
			expectedEventTypes: []string{},
			expectedEventCount: 0,
			expectMockCall:     false,
		},
		{
			name: "multiple subscription messages",
			config: &Config{
				Events: EventConfig{
					RecvRPCEnabled:             true,
					RpcMetaSubscriptionEnabled: true,
				},
				Traces: TracesConfig{
					AlwaysRecordRootRpcEvents: true,
				},
			},
			subscriptions: []host.RpcMetaSub{
				{
					Subscribe: true,
					TopicID:   "topic-1",
				},
				{
					Subscribe: false,
					TopicID:   "topic-2",
				},
				{
					Subscribe: true,
					TopicID:   "topic-3",
				},
			},
			expectedEventTypes: []string{
				"LIBP2P_TRACE_RECV_RPC",
				"LIBP2P_TRACE_RPC_META_SUBSCRIPTION",
			},
			expectedEventCount: 4,
			expectMockCall:     true,
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
			if tt.expectMockCall {
				mockSink.EXPECT().HandleNewDecoratedEvents(gomock.Any(), gomock.Any()).
					Do(func(ctx context.Context, events []*xatu.DecoratedEvent) {
						for _, event := range events {
							receivedEventTypes = append(receivedEventTypes, event.Event.Name.String())
						}
					}).
					Return(nil).
					Times(1)
			}

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
					PeerID:        peerID,
					Subscriptions: tt.subscriptions,
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

func Test_handleRecvRPCEvent_DetailedMetaMessages(t *testing.T) {
	// Create a valid peer ID for testing
	peerID, err := peer.Decode(examplePeerID)
	require.NoError(t, err)

	tests := []struct {
		name               string
		config             *Config
		messages           []host.RpcMetaMsg
		expectedEventTypes []string
		expectedEventCount int
		expectMockCall     bool
	}{
		{
			name: "meta messages enabled",
			config: &Config{
				Events: EventConfig{
					RecvRPCEnabled:        true,
					RpcMetaMessageEnabled: true,
				},
				Traces: TracesConfig{
					AlwaysRecordRootRpcEvents: true,
				},
			},
			messages: []host.RpcMetaMsg{
				{
					MsgID: "msg1",
					Topic: "topic1",
				},
				{
					MsgID: "msg2",
					Topic: "topic2",
				},
			},
			expectedEventTypes: []string{
				"LIBP2P_TRACE_RECV_RPC",
				"LIBP2P_TRACE_RPC_META_MESSAGE",
			},
			expectedEventCount: 3,
			expectMockCall:     true,
		},
		{
			name: "meta messages disabled",
			config: &Config{
				Events: EventConfig{
					RecvRPCEnabled:        true,
					RpcMetaMessageEnabled: false,
				},
				Traces: TracesConfig{
					AlwaysRecordRootRpcEvents: true,
				},
			},
			messages: []host.RpcMetaMsg{
				{
					MsgID: "msg1",
					Topic: "topic1",
				},
			},
			expectedEventTypes: []string{
				"LIBP2P_TRACE_RECV_RPC",
			},
			expectedEventCount: 1,
			expectMockCall:     true,
		},
		{
			name: "meta messages disabled and no AlwaysRecordRootRpcEvents",
			config: &Config{
				Events: EventConfig{
					RecvRPCEnabled:        true,
					RpcMetaMessageEnabled: false,
				},
				Traces: TracesConfig{
					AlwaysRecordRootRpcEvents: false,
				},
			},
			messages: []host.RpcMetaMsg{
				{
					MsgID: "msg1",
					Topic: "topic1",
				},
			},
			expectedEventTypes: []string{},
			expectedEventCount: 0,
			expectMockCall:     false,
		},
		{
			name: "multiple meta messages",
			config: &Config{
				Events: EventConfig{
					RecvRPCEnabled:        true,
					RpcMetaMessageEnabled: true,
				},
				Traces: TracesConfig{
					AlwaysRecordRootRpcEvents: true,
				},
			},
			messages: []host.RpcMetaMsg{
				{
					MsgID: "msg1",
					Topic: "eth2/beacon_block",
				},
				{
					MsgID: "msg2",
					Topic: "eth2/attestation",
				},
				{
					MsgID: "msg3",
					Topic: "eth2/voluntary_exit",
				},
			},
			expectedEventTypes: []string{
				"LIBP2P_TRACE_RECV_RPC",
				"LIBP2P_TRACE_RPC_META_MESSAGE",
			},
			expectedEventCount: 4,
			expectMockCall:     true,
		},
		{
			name: "empty messages list",
			config: &Config{
				Events: EventConfig{
					RecvRPCEnabled:        true,
					RpcMetaMessageEnabled: true,
				},
				Traces: TracesConfig{
					AlwaysRecordRootRpcEvents: true,
				},
			},
			messages: []host.RpcMetaMsg{},
			expectedEventTypes: []string{
				"LIBP2P_TRACE_RECV_RPC",
			},
			expectedEventCount: 1,
			expectMockCall:     true,
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
			if tt.expectMockCall {
				mockSink.EXPECT().HandleNewDecoratedEvents(gomock.Any(), gomock.Any()).
					Do(func(ctx context.Context, events []*xatu.DecoratedEvent) {
						for _, event := range events {
							receivedEventTypes = append(receivedEventTypes, event.Event.Name.String())
						}
					}).
					Return(nil).
					Times(1)
			}

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
					PeerID:   peerID,
					Messages: tt.messages,
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

// Test_AllRPCEvents_MetaDataStripping verifies that root RPC events (Recv/Send/Drop)
// properly strip meta data to prevent message size issues while preserving all information.
//
// Background:
// - RPC events can contain large amounts of meta data (control messages, subscriptions, messages)
// - These large payloads can exceed message size limits in Kafka/Vector.
// - We already explode all meta data into individual events.
// - Therefore, it's safe to strip the bulk data from root events while keeping minimal context.
//
// This test ensures that:
// 1. Root events contain only essential data (PeerId) to stay small.
// 2. Meta field contains only PeerId, not the full control/subscription/message data.
// 3. All detailed meta data is still captured as separate individual events.
// 4. Behavior is consistent across all three RPC event types (Recv/Send/Drop).
// 5. Different scenarios work correctly (various meta data types, configuration options).
func Test_AllRPCEvents_MetaDataStripping(t *testing.T) {
	// Create a valid peer ID for testing.
	peerID, err := peer.Decode(examplePeerID)
	require.NoError(t, err)

	// Test scenarios with different types of meta data.
	testScenarios := []struct {
		name               string
		config             *Config
		controlData        *host.RpcMetaControl
		subscriptions      []host.RpcMetaSub
		messages           []host.RpcMetaMsg
		expectRootEvent    bool
		expectMetaEvents   bool
		expectedEventCount int
	}{
		{
			name: "control data with IHave messages",
			config: &Config{
				Events: EventConfig{
					RecvRPCEnabled:             true,
					SendRPCEnabled:             true,
					DropRPCEnabled:             true,
					RpcMetaControlIHaveEnabled: true,
				},
				Traces: TracesConfig{
					AlwaysRecordRootRpcEvents: true,
				},
			},
			controlData: &host.RpcMetaControl{
				IHave: []host.RpcControlIHave{
					{
						TopicID: "test-topic",
						MsgIDs:  []string{"msg1", "msg2"},
					},
				},
			},
			expectRootEvent:    true,
			expectMetaEvents:   true,
			expectedEventCount: 3, // 1 root + 2 IHave events.
		},
		{
			name: "control data with IWant messages",
			config: &Config{
				Events: EventConfig{
					RecvRPCEnabled:             true,
					SendRPCEnabled:             true,
					DropRPCEnabled:             true,
					RpcMetaControlIWantEnabled: true,
				},
				Traces: TracesConfig{
					AlwaysRecordRootRpcEvents: true,
				},
			},
			controlData: &host.RpcMetaControl{
				IWant: []host.RpcControlIWant{
					{
						MsgIDs: []string{"want1", "want2", "want3"},
					},
				},
			},
			expectRootEvent:    true,
			expectMetaEvents:   true,
			expectedEventCount: 4, // 1 root + 3 IWant events.
		},
		{
			name: "subscription data",
			config: &Config{
				Events: EventConfig{
					RecvRPCEnabled:             true,
					SendRPCEnabled:             true,
					DropRPCEnabled:             true,
					RpcMetaSubscriptionEnabled: true,
				},
				Traces: TracesConfig{
					AlwaysRecordRootRpcEvents: true,
				},
			},
			subscriptions: []host.RpcMetaSub{
				{
					Subscribe: true,
					TopicID:   "eth2/beacon_block",
				},
				{
					Subscribe: false,
					TopicID:   "eth2/attestation",
				},
			},
			expectRootEvent:    true,
			expectMetaEvents:   true,
			expectedEventCount: 3, // 1 root + 2 subscription events.
		},
		{
			name: "message data",
			config: &Config{
				Events: EventConfig{
					RecvRPCEnabled:        true,
					SendRPCEnabled:        true,
					DropRPCEnabled:        true,
					RpcMetaMessageEnabled: true,
				},
				Traces: TracesConfig{
					AlwaysRecordRootRpcEvents: true,
				},
			},
			messages: []host.RpcMetaMsg{
				{
					MsgID: "msg1",
					Topic: "eth2/beacon_block",
				},
				{
					MsgID: "msg2",
					Topic: "eth2/voluntary_exit",
				},
			},
			expectRootEvent:    true,
			expectMetaEvents:   true,
			expectedEventCount: 3, // 1 root + 2 message events.
		},
		{
			name: "complex meta data with multiple types",
			config: &Config{
				Events: EventConfig{
					RecvRPCEnabled:             true,
					SendRPCEnabled:             true,
					DropRPCEnabled:             true,
					RpcMetaControlIHaveEnabled: true,
					RpcMetaControlGraftEnabled: true,
					RpcMetaSubscriptionEnabled: true,
					RpcMetaMessageEnabled:      true,
				},
				Traces: TracesConfig{
					AlwaysRecordRootRpcEvents: true,
				},
			},
			controlData: &host.RpcMetaControl{
				IHave: []host.RpcControlIHave{
					{
						TopicID: "test-topic",
						MsgIDs:  []string{"ihave1"},
					},
				},
				Graft: []host.RpcControlGraft{
					{
						TopicID: "graft-topic",
					},
				},
			},
			subscriptions: []host.RpcMetaSub{
				{
					Subscribe: true,
					TopicID:   "sub-topic",
				},
			},
			messages: []host.RpcMetaMsg{
				{
					MsgID: "complex-msg",
					Topic: "msg-topic",
				},
			},
			expectRootEvent:    true,
			expectMetaEvents:   true,
			expectedEventCount: 5, // 1 root + 1 IHave + 1 Graft + 1 subscription + 1 message.
		},
		{
			name: "no meta data with AlwaysRecordRootRpcEvents enabled",
			config: &Config{
				Events: EventConfig{
					RecvRPCEnabled: true,
					SendRPCEnabled: true,
					DropRPCEnabled: true,
				},
				Traces: TracesConfig{
					AlwaysRecordRootRpcEvents: true,
				},
			},
			expectRootEvent:    true,
			expectMetaEvents:   false,
			expectedEventCount: 1, // 1 root event only.
		},
		{
			name: "no meta data with AlwaysRecordRootRpcEvents disabled",
			config: &Config{
				Events: EventConfig{
					RecvRPCEnabled: true,
					SendRPCEnabled: true,
					DropRPCEnabled: true,
				},
				Traces: TracesConfig{
					AlwaysRecordRootRpcEvents: false,
				},
			},
			expectRootEvent:    false,
			expectMetaEvents:   false,
			expectedEventCount: 0, // No events.
		},
	}

	// Test all three RPC event types to ensure they all strip meta data consistently.
	eventTypes := []struct {
		name          string
		eventType     string
		handlerFunc   func(*Mimicry, context.Context, *xatu.ClientMeta, *libp2p.TraceEventMetadata, *host.TraceEvent) error
		rootEventName xatu.Event_Name
	}{
		{
			name:          "RecvRPC",
			eventType:     "RECV_RPC",
			handlerFunc:   (*Mimicry).handleRecvRPCEvent,
			rootEventName: xatu.Event_LIBP2P_TRACE_RECV_RPC,
		},
		{
			name:          "SendRPC",
			eventType:     "SEND_RPC",
			handlerFunc:   (*Mimicry).handleSendRPCEvent,
			rootEventName: xatu.Event_LIBP2P_TRACE_SEND_RPC,
		},
		{
			name:          "DropRPC",
			eventType:     "DROP_RPC",
			handlerFunc:   (*Mimicry).handleDropRPCEvent,
			rootEventName: xatu.Event_LIBP2P_TRACE_DROP_RPC,
		},
	}

	for scenarioIdx, scenario := range testScenarios {
		t.Run(scenario.name, func(t *testing.T) {
			for eventIdx, eventType := range eventTypes {
				t.Run(eventType.name, func(t *testing.T) {
					ctrl := gomock.NewController(t)
					defer ctrl.Finish()

					mockSink := mock.NewMockSink(ctrl)

					var receivedEvents []*xatu.DecoratedEvent

					// Setup mock to capture events.
					if scenario.expectedEventCount > 0 {
						mockSink.EXPECT().HandleNewDecoratedEvents(gomock.Any(), gomock.Any()).
							Do(func(ctx context.Context, events []*xatu.DecoratedEvent) {
								receivedEvents = events
							}).
							Return(nil).
							Times(1)
					}

					mimicry := &Mimicry{
						Config:  scenario.config,
						sinks:   []output.Sink{mockSink},
						log:     logrus.NewEntry(logrus.New()),
						id:      uuid.New(),
						metrics: NewMetrics(fmt.Sprintf("all_rpc_strip_%d_%d", scenarioIdx, eventIdx)),
					}

					clientMeta := &xatu.ClientMeta{
						Name: "test-client",
						Id:   uuid.New().String(),
					}

					traceMeta := &libp2p.TraceEventMetadata{
						PeerId: wrapperspb.String("test-peer"),
					}

					event := &host.TraceEvent{
						Type:      eventType.eventType,
						PeerID:    peerID,
						Timestamp: time.Now(),
						Payload: &host.RpcMeta{
							PeerID:        peerID,
							Control:       scenario.controlData,
							Subscriptions: scenario.subscriptions,
							Messages:      scenario.messages,
						},
					}

					ctx := context.Background()
					err := eventType.handlerFunc(mimicry, ctx, clientMeta, traceMeta, event)

					require.NoError(t, err)
					assert.Equal(t, scenario.expectedEventCount, len(receivedEvents), "Event count mismatch for %s", eventType.name)

					if scenario.expectRootEvent {
						// Find the root RPC event.
						var rootEvent *xatu.DecoratedEvent
						for _, e := range receivedEvents {
							if e.Event.Name == eventType.rootEventName {
								rootEvent = e

								break
							}
						}

						require.NotNil(t, rootEvent, "Expected root RPC event but didn't find one for %s", eventType.name)

						// Verify the root event structure based on event type.
						switch eventType.rootEventName {
						case xatu.Event_LIBP2P_TRACE_RECV_RPC:
							recvData := rootEvent.GetLibp2PTraceRecvRpc()
							require.NotNil(t, recvData, "Root event should have RecvRPC data")
							assert.NotNil(t, recvData.PeerId, "Root event should have PeerId")
							assert.Equal(t, peerID.String(), recvData.PeerId.GetValue())
							require.NotNil(t, recvData.Meta, "Root event should contain minimal Meta data")
							assert.NotNil(t, recvData.Meta.PeerId, "Root event Meta should have PeerId")
							assert.Equal(t, peerID.String(), recvData.Meta.PeerId.GetValue())
							assert.Nil(t, recvData.Meta.Control, "Root event Meta should not contain Control data")
							assert.Nil(t, recvData.Meta.Subscriptions, "Root event Meta should not contain Subscriptions data")
							assert.Nil(t, recvData.Meta.Messages, "Root event Meta should not contain Messages data")
						case xatu.Event_LIBP2P_TRACE_SEND_RPC:
							sendData := rootEvent.GetLibp2PTraceSendRpc()
							require.NotNil(t, sendData, "Root event should have SendRPC data")
							assert.NotNil(t, sendData.PeerId, "Root event should have PeerId")
							assert.Equal(t, peerID.String(), sendData.PeerId.GetValue())
							require.NotNil(t, sendData.Meta, "Root event should contain minimal Meta data")
							assert.NotNil(t, sendData.Meta.PeerId, "Root event Meta should have PeerId")
							assert.Equal(t, peerID.String(), sendData.Meta.PeerId.GetValue())
							assert.Nil(t, sendData.Meta.Control, "Root event Meta should not contain Control data")
							assert.Nil(t, sendData.Meta.Subscriptions, "Root event Meta should not contain Subscriptions data")
							assert.Nil(t, sendData.Meta.Messages, "Root event Meta should not contain Messages data")
						case xatu.Event_LIBP2P_TRACE_DROP_RPC:
							dropData := rootEvent.GetLibp2PTraceDropRpc()
							require.NotNil(t, dropData, "Root event should have DropRPC data")
							assert.NotNil(t, dropData.PeerId, "Root event should have PeerId")
							assert.Equal(t, peerID.String(), dropData.PeerId.GetValue())
							require.NotNil(t, dropData.Meta, "Root event should contain minimal Meta data")
							assert.NotNil(t, dropData.Meta.PeerId, "Root event Meta should have PeerId")
							assert.Equal(t, peerID.String(), dropData.Meta.PeerId.GetValue())
							assert.Nil(t, dropData.Meta.Control, "Root event Meta should not contain Control data")
							assert.Nil(t, dropData.Meta.Subscriptions, "Root event Meta should not contain Subscriptions data")
							assert.Nil(t, dropData.Meta.Messages, "Root event Meta should not contain Messages data")
						}
					}

					if scenario.expectMetaEvents {
						// Verify meta events are still generated.
						metaEventCount := 0
						for _, e := range receivedEvents {
							if e.Event.Name != eventType.rootEventName {
								metaEventCount++
							}
						}
						assert.Greater(t, metaEventCount, 0, "Should have generated meta events for %s", eventType.name)
					}
				})
			}
		})
	}
}
