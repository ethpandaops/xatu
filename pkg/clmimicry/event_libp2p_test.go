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
