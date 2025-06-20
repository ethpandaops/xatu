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

// eventCountAssertion is a helper to validate event counts.
type eventCountAssertion struct {
	eventType xatu.Event_Name
	expected  int
	message   string
}

// Helper to create a mock expectation that validates event counts and additional properties.
type eventValidator func(t *testing.T, events []*xatu.DecoratedEvent)

func Test_handleRecvRPCEvent(t *testing.T) {
	// Create a valid peer ID for testing
	peerID, err := peer.Decode(examplePeerID)
	require.NoError(t, err)

	tests := []struct {
		name           string
		config         *Config
		event          *host.TraceEvent
		expectError    bool
		validateCalls  func(t *testing.T, events []*xatu.DecoratedEvent)
		setupMockCalls func(*mock.MockSink)
	}{
		{
			name: "IHAVE control messages",
			config: &Config{
				Events: EventConfig{
					RecvRPCEnabled:             true,
					RpcMetaControlIHaveEnabled: true,
				},
			},
			event: createRPCEvent(peerID, &host.RpcMeta{
				PeerID: peerID,
				Control: &host.RpcMetaControl{
					IHave: []host.RpcControlIHave{
						{
							TopicID: "/eth2/test-topic",
							MsgIDs:  []string{"msg1", "msg2", "msg3"},
						},
					},
				},
			}),
			expectError: false,
			setupMockCalls: func(mockSink *mock.MockSink) {
				expectEventCounts(t, mockSink,
					eventCountAssertion{xatu.Event_LIBP2P_TRACE_RPC_META_CONTROL_IHAVE, 3, "Expected 3 IHAVE events (one for each message ID)"},
				)
			},
		},
		{
			name: "IWANT control messages",
			config: &Config{
				Events: EventConfig{
					RecvRPCEnabled:             true,
					RpcMetaControlIWantEnabled: true,
				},
			},
			event: &host.TraceEvent{
				Type:      "RecvRPC",
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
			setupMockCalls: func(mockSink *mock.MockSink) {
				expectEventCounts(t, mockSink,
					eventCountAssertion{xatu.Event_LIBP2P_TRACE_RPC_META_CONTROL_IWANT, 2, "Expected 2 IWANT events (one for each message ID)"},
				)
			},
		},
		{
			name: "IDONTWANT control messages",
			config: &Config{
				Events: EventConfig{
					RecvRPCEnabled:                 true,
					RpcMetaControlIDontWantEnabled: true,
				},
			},
			event: &host.TraceEvent{
				Type:      "RecvRPC",
				PeerID:    peerID,
				Timestamp: time.Now(),
				Payload: &host.RpcMeta{
					PeerID: peerID,
					Control: &host.RpcMetaControl{
						Idontwant: []host.RpcControlIdontWant{
							{
								MsgIDs: []string{"msg1", "msg2", "msg3", "msg4"},
							},
						},
					},
				},
			},
			expectError: false,
			setupMockCalls: func(mockSink *mock.MockSink) {
				expectEventCounts(t, mockSink,
					eventCountAssertion{xatu.Event_LIBP2P_TRACE_RPC_META_CONTROL_IDONTWANT, 4, "Expected 4 IDONTWANT events (one for each message ID)"},
				)
			},
		},
		{
			name: "GRAFT control messages",
			config: &Config{
				Events: EventConfig{
					RecvRPCEnabled:             true,
					RpcMetaControlGraftEnabled: true,
				},
			},
			event: &host.TraceEvent{
				Type:      "RecvRPC",
				PeerID:    peerID,
				Timestamp: time.Now(),
				Payload: &host.RpcMeta{
					PeerID: peerID,
					Control: &host.RpcMetaControl{
						Graft: []host.RpcControlGraft{
							{
								TopicID: "/eth2/test-topic-1",
							},
							{
								TopicID: "/eth2/test-topic-2",
							},
						},
					},
				},
			},
			expectError: false,
			setupMockCalls: func(mockSink *mock.MockSink) {
				expectEventCounts(t, mockSink,
					eventCountAssertion{xatu.Event_LIBP2P_TRACE_RPC_META_CONTROL_GRAFT, 2, "Expected 2 GRAFT events (one for each topic)"},
				)
			},
		},
		{
			name: "PRUNE with peer IDs",
			config: &Config{
				Events: EventConfig{
					RecvRPCEnabled:             true,
					RpcMetaControlPruneEnabled: true,
				},
			},
			event: &host.TraceEvent{
				Type:      "RecvRPC",
				PeerID:    peerID,
				Timestamp: time.Now(),
				Payload: &host.RpcMeta{
					PeerID: peerID,
					Control: &host.RpcMetaControl{
						Prune: []host.RpcControlPrune{
							{
								TopicID: "/eth2/test-topic",
								PeerIDs: []peer.ID{
									peer.ID("peer1"),
									peer.ID("peer2"),
								},
							},
						},
					},
				},
			},
			expectError: false,
			setupMockCalls: func(mockSink *mock.MockSink) {
				expectEventCounts(t, mockSink,
					eventCountAssertion{xatu.Event_LIBP2P_TRACE_RPC_META_CONTROL_PRUNE, 2, "Expected 2 PRUNE events (one for each peer ID)"},
				)
			},
		},
		{
			name: "PRUNE without peer IDs",
			config: &Config{
				Events: EventConfig{
					RecvRPCEnabled:             true,
					RpcMetaControlPruneEnabled: true,
				},
			},
			event: &host.TraceEvent{
				Type:      "RecvRPC",
				PeerID:    peerID,
				Timestamp: time.Now(),
				Payload: &host.RpcMeta{
					PeerID: peerID,
					Control: &host.RpcMetaControl{
						Prune: []host.RpcControlPrune{
							{
								TopicID: "/eth2/test-topic",
								PeerIDs: []peer.ID{}, // Empty peer IDs
							},
						},
					},
				},
			},
			expectError: false,
			setupMockCalls: func(mockSink *mock.MockSink) {
				expectEventCounts(t, mockSink,
					eventCountAssertion{xatu.Event_LIBP2P_TRACE_RPC_META_CONTROL_PRUNE, 1, "Expected 1 PRUNE event with no peer IDs"},
				)
			},
		},
		{
			name: "Multiple PRUNE messages mixed",
			config: &Config{
				Events: EventConfig{
					RecvRPCEnabled:             true,
					RpcMetaControlPruneEnabled: true,
				},
			},
			event: &host.TraceEvent{
				Type:      "RecvRPC",
				PeerID:    peerID,
				Timestamp: time.Now(),
				Payload: &host.RpcMeta{
					PeerID: peerID,
					Control: &host.RpcMetaControl{
						Prune: []host.RpcControlPrune{
							{
								TopicID: "/eth2/test-topic-1",
								PeerIDs: []peer.ID{
									peer.ID("peer1"),
									peer.ID("peer2"),
								}, // 2 peer IDs
							},
							{
								TopicID: "/eth2/test-topic-2",
								PeerIDs: []peer.ID{}, // No peer IDs
							},
						},
					},
				},
			},
			expectError: false,
			setupMockCalls: func(mockSink *mock.MockSink) {
				expectEventCounts(t, mockSink,
					eventCountAssertion{xatu.Event_LIBP2P_TRACE_RPC_META_CONTROL_PRUNE, 3, "Expected 3 PRUNE events total"},
				)
			},
		},
		{
			name: "Mixed control messages",
			config: &Config{
				Events: EventConfig{
					RecvRPCEnabled:             true,
					RpcMetaControlIHaveEnabled: true,
					RpcMetaControlIWantEnabled: true,
					RpcMetaControlGraftEnabled: true,
					RpcMetaControlPruneEnabled: true,
				},
			},
			event: createRPCEvent(peerID, &host.RpcMeta{
				PeerID: peerID,
				Control: &host.RpcMetaControl{
					IHave: []host.RpcControlIHave{
						{
							TopicID: "/eth2/test-topic",
							MsgIDs:  []string{"msg1", "msg2"},
						},
					},
					IWant: []host.RpcControlIWant{
						{
							MsgIDs: []string{"msg3"},
						},
					},
					Graft: []host.RpcControlGraft{
						{
							TopicID: "/eth2/test-topic",
						},
					},
					Prune: []host.RpcControlPrune{
						{
							TopicID: "/eth2/test-topic",
							PeerIDs: []peer.ID{peer.ID("peer1")},
						},
					},
				},
			}),
			expectError: false,
			setupMockCalls: func(mockSink *mock.MockSink) {
				expectEventCounts(t, mockSink,
					eventCountAssertion{xatu.Event_LIBP2P_TRACE_RECV_RPC, 1, "Expected 1 root RPC event"},
					eventCountAssertion{xatu.Event_LIBP2P_TRACE_RPC_META_CONTROL_IHAVE, 2, "Expected 2 IHAVE events"},
					eventCountAssertion{xatu.Event_LIBP2P_TRACE_RPC_META_CONTROL_IWANT, 1, "Expected 1 IWANT event"},
					eventCountAssertion{xatu.Event_LIBP2P_TRACE_RPC_META_CONTROL_GRAFT, 1, "Expected 1 GRAFT event"},
					eventCountAssertion{xatu.Event_LIBP2P_TRACE_RPC_META_CONTROL_PRUNE, 1, "Expected 1 PRUNE event"},
				)
			},
		},
		{
			name: "All control messages disabled",
			config: &Config{
				Events: EventConfig{
					RecvRPCEnabled:                 true,
					RpcMetaControlIHaveEnabled:     false,
					RpcMetaControlIWantEnabled:     false,
					RpcMetaControlIDontWantEnabled: false,
					RpcMetaControlGraftEnabled:     false,
					RpcMetaControlPruneEnabled:     false,
				},
			},
			event: &host.TraceEvent{
				Type:      "RecvRPC",
				PeerID:    peerID,
				Timestamp: time.Now(),
				Payload: &host.RpcMeta{
					PeerID: peerID,
					Control: &host.RpcMetaControl{
						IHave: []host.RpcControlIHave{
							{
								TopicID: "/eth2/test-topic",
								MsgIDs:  []string{"msg1"},
							},
						},
						IWant: []host.RpcControlIWant{
							{
								MsgIDs: []string{"msg2"},
							},
						},
						Graft: []host.RpcControlGraft{
							{
								TopicID: "/eth2/test-topic",
							},
						},
						Prune: []host.RpcControlPrune{
							{
								TopicID: "/eth2/test-topic",
								PeerIDs: []peer.ID{},
							},
						},
					},
				},
			},
			expectError:    false,
			setupMockCalls: expectNoEvents,
		},
		{
			name: "IHAVE with empty message IDs",
			config: &Config{
				Events: EventConfig{
					RecvRPCEnabled:             true,
					RpcMetaControlIHaveEnabled: true,
				},
			},
			event: &host.TraceEvent{
				Type:      "RecvRPC",
				PeerID:    peerID,
				Timestamp: time.Now(),
				Payload: &host.RpcMeta{
					PeerID: peerID,
					Control: &host.RpcMetaControl{
						IHave: []host.RpcControlIHave{
							{
								TopicID: "/eth2/test-topic",
								MsgIDs:  []string{}, // Empty message IDs
							},
						},
					},
				},
			},
			expectError:    false,
			setupMockCalls: expectNoEvents,
		},
		{
			name: "Multiple IHAVE topics",
			config: &Config{
				Events: EventConfig{
					RecvRPCEnabled:             true,
					RpcMetaControlIHaveEnabled: true,
				},
			},
			event: &host.TraceEvent{
				Type:      "RecvRPC",
				PeerID:    peerID,
				Timestamp: time.Now(),
				Payload: &host.RpcMeta{
					PeerID: peerID,
					Control: &host.RpcMetaControl{
						IHave: []host.RpcControlIHave{
							{
								TopicID: "/eth2/topic-1",
								MsgIDs:  []string{"msg1", "msg2"},
							},
							{
								TopicID: "/eth2/topic-2",
								MsgIDs:  []string{"msg3"},
							},
						},
					},
				},
			},
			expectError: false,
			setupMockCalls: func(mockSink *mock.MockSink) {
				expectEventsWithValidation(t, mockSink, func(t *testing.T, events []*xatu.DecoratedEvent) {
					t.Helper()

					ihaveCount := 0
					topics := make(map[string]int)

					for _, e := range events {
						if e.GetEvent().GetName() == xatu.Event_LIBP2P_TRACE_RPC_META_CONTROL_IHAVE {
							ihaveCount++
							ihaveData := e.GetLibp2PTraceRpcMetaControlIhave()
							topics[ihaveData.Topic.GetValue()]++
						}
					}

					assert.Equal(t, 3, ihaveCount, "Expected 3 IHAVE events total")
					assert.Equal(t, 2, topics["/eth2/topic-1"], "Expected 2 events for topic-1")
					assert.Equal(t, 1, topics["/eth2/topic-2"], "Expected 1 event for topic-2")
				})
			},
		},
		{
			name: "Meta subscriptions",
			config: &Config{
				Events: EventConfig{
					RecvRPCEnabled:             true,
					RpcMetaSubscriptionEnabled: true,
				},
			},
			event: &host.TraceEvent{
				Type:      "RecvRPC",
				PeerID:    peerID,
				Timestamp: time.Now(),
				Payload: &host.RpcMeta{
					PeerID: peerID,
					Subscriptions: []host.RpcMetaSub{
						{
							Subscribe: true,
							TopicID:   "/eth2/beacon_block/ssz_snappy",
						},
						{
							Subscribe: false,
							TopicID:   "/eth2/beacon_attestation/ssz_snappy",
						},
					},
				},
			},
			expectError: false,
			setupMockCalls: func(mockSink *mock.MockSink) {
				expectEventsWithValidation(t, mockSink, func(t *testing.T, events []*xatu.DecoratedEvent) {
					t.Helper()

					subCount := 0
					subscribeCount := 0
					unsubscribeCount := 0

					for _, e := range events {
						if e.GetEvent().GetName() == xatu.Event_LIBP2P_TRACE_RPC_META_SUBSCRIPTION {
							subCount++
							subData := e.GetLibp2PTraceRpcMetaSubscription()

							if subData.Subscribe.GetValue() {
								subscribeCount++
							} else {
								unsubscribeCount++
							}
						}
					}

					assert.Equal(t, 2, subCount, "Expected 2 subscription events")
					assert.Equal(t, 1, subscribeCount, "Expected 1 subscribe event")
					assert.Equal(t, 1, unsubscribeCount, "Expected 1 unsubscribe event")
				})
			},
		},
		{
			name: "Meta messages",
			config: &Config{
				Events: EventConfig{
					RecvRPCEnabled:        true,
					RpcMetaMessageEnabled: true,
				},
			},
			event: &host.TraceEvent{
				Type:      "RecvRPC",
				PeerID:    peerID,
				Timestamp: time.Now(),
				Payload: &host.RpcMeta{
					PeerID: peerID,
					Messages: []host.RpcMetaMsg{
						{
							MsgID: "msg1",
							Topic: "/eth2/beacon_block/ssz_snappy",
						},
						{
							MsgID: "msg2",
							Topic: "/eth2/beacon_attestation/ssz_snappy",
						},
						{
							MsgID: "msg3",
							Topic: "/eth2/beacon_block/ssz_snappy",
						},
					},
				},
			},
			expectError: false,
			setupMockCalls: func(mockSink *mock.MockSink) {
				expectEventsWithValidation(t, mockSink, func(t *testing.T, events []*xatu.DecoratedEvent) {
					t.Helper()

					msgCount := 0
					topics := make(map[string]int)

					for _, e := range events {
						if e.GetEvent().GetName() == xatu.Event_LIBP2P_TRACE_RPC_META_MESSAGE {
							msgCount++
							msgData := e.GetLibp2PTraceRpcMetaMessage()
							topics[msgData.TopicId.GetValue()]++

							assert.NotEmpty(t, msgData.MessageId.GetValue(), "Expected MessageId to be set")
						}
					}

					assert.Equal(t, 3, msgCount, "Expected 3 message events")
					assert.Equal(t, 2, topics["/eth2/beacon_block/ssz_snappy"], "Expected 2 beacon block messages")
					assert.Equal(t, 1, topics["/eth2/beacon_attestation/ssz_snappy"], "Expected 1 attestation message")
				})
			},
		},
		{
			name: "Mixed meta events - subscriptions, messages, and control",
			config: &Config{
				Events: EventConfig{
					RecvRPCEnabled:             true,
					RpcMetaSubscriptionEnabled: true,
					RpcMetaMessageEnabled:      true,
					RpcMetaControlGraftEnabled: true,
				},
			},
			event: &host.TraceEvent{
				Type:      "RecvRPC",
				PeerID:    peerID,
				Timestamp: time.Now(),
				Payload: &host.RpcMeta{
					PeerID: peerID,
					Subscriptions: []host.RpcMetaSub{
						{
							Subscribe: true,
							TopicID:   "/eth2/test-topic",
						},
					},
					Messages: []host.RpcMetaMsg{
						{
							MsgID: "msg1",
							Topic: "/eth2/test-topic",
						},
					},
					Control: &host.RpcMetaControl{
						Graft: []host.RpcControlGraft{
							{
								TopicID: "/eth2/test-topic",
							},
						},
					},
				},
			},
			expectError: false,
			setupMockCalls: func(mockSink *mock.MockSink) {
				expectEventCounts(t, mockSink,
					eventCountAssertion{xatu.Event_LIBP2P_TRACE_RECV_RPC, 1, "Expected 1 root RPC event"},
					eventCountAssertion{xatu.Event_LIBP2P_TRACE_RPC_META_SUBSCRIPTION, 1, "Expected 1 subscription event"},
					eventCountAssertion{xatu.Event_LIBP2P_TRACE_RPC_META_MESSAGE, 1, "Expected 1 message event"},
					eventCountAssertion{xatu.Event_LIBP2P_TRACE_RPC_META_CONTROL_GRAFT, 1, "Expected 1 GRAFT event"},
				)
			},
		},
		{
			name: "Meta subscriptions with empty subscriptions array",
			config: &Config{
				Events: EventConfig{
					RecvRPCEnabled:             true,
					RpcMetaSubscriptionEnabled: true,
				},
			},
			event: &host.TraceEvent{
				Type:      "RecvRPC",
				PeerID:    peerID,
				Timestamp: time.Now(),
				Payload: &host.RpcMeta{
					PeerID:        peerID,
					Subscriptions: []host.RpcMetaSub{}, // Empty subscriptions
				},
			},
			expectError:    false,
			setupMockCalls: expectNoEvents,
		},
		{
			name: "Meta messages with empty messages array",
			config: &Config{
				Events: EventConfig{
					RecvRPCEnabled:        true,
					RpcMetaMessageEnabled: true,
				},
			},
			event: &host.TraceEvent{
				Type:      "RecvRPC",
				PeerID:    peerID,
				Timestamp: time.Now(),
				Payload: &host.RpcMeta{
					PeerID:   peerID,
					Messages: []host.RpcMetaMsg{}, // Empty messages
				},
			},
			expectError:    false,
			setupMockCalls: expectNoEvents,
		},
		{
			name: "Meta events all disabled",
			config: &Config{
				Events: EventConfig{
					RecvRPCEnabled:                 true,
					RpcMetaSubscriptionEnabled:     false,
					RpcMetaMessageEnabled:          false,
					RpcMetaControlIHaveEnabled:     false,
					RpcMetaControlIWantEnabled:     false,
					RpcMetaControlIDontWantEnabled: false,
					RpcMetaControlGraftEnabled:     false,
					RpcMetaControlPruneEnabled:     false,
				},
			},
			event: &host.TraceEvent{
				Type:      "RecvRPC",
				PeerID:    peerID,
				Timestamp: time.Now(),
				Payload: &host.RpcMeta{
					PeerID: peerID,
					Subscriptions: []host.RpcMetaSub{
						{
							Subscribe: true,
							TopicID:   "/eth2/test-topic",
						},
					},
					Messages: []host.RpcMetaMsg{
						{
							MsgID: "msg1",
							Topic: "/eth2/test-topic",
						},
					},
					Control: &host.RpcMetaControl{
						IHave: []host.RpcControlIHave{
							{
								TopicID: "/eth2/test-topic",
								MsgIDs:  []string{"msg1"},
							},
						},
					},
				},
			},
			expectError:    false,
			setupMockCalls: expectNoEvents,
		},
		{
			name: "Multiple subscriptions with different topics",
			config: &Config{
				Events: EventConfig{
					RecvRPCEnabled:             true,
					RpcMetaSubscriptionEnabled: true,
				},
			},
			event: &host.TraceEvent{
				Type:      "RecvRPC",
				PeerID:    peerID,
				Timestamp: time.Now(),
				Payload: &host.RpcMeta{
					PeerID: peerID,
					Subscriptions: []host.RpcMetaSub{
						{
							Subscribe: true,
							TopicID:   "/eth2/beacon_block/ssz_snappy",
						},
						{
							Subscribe: true,
							TopicID:   "/eth2/beacon_attestation/ssz_snappy",
						},
						{
							Subscribe: false,
							TopicID:   "/eth2/voluntary_exit/ssz_snappy",
						},
						{
							Subscribe: true,
							TopicID:   "/eth2/sync_committee/ssz_snappy",
						},
					},
				},
			},
			expectError: false,
			setupMockCalls: func(mockSink *mock.MockSink) {
				expectEventsWithValidation(t, mockSink, func(t *testing.T, events []*xatu.DecoratedEvent) {
					t.Helper()

					subCount := 0
					subscribeCount := 0
					unsubscribeCount := 0
					topics := make(map[string]bool)

					for _, e := range events {
						if e.GetEvent().GetName() == xatu.Event_LIBP2P_TRACE_RPC_META_SUBSCRIPTION {
							subCount++
							subData := e.GetLibp2PTraceRpcMetaSubscription()
							topics[subData.TopicId.GetValue()] = true

							if subData.Subscribe.GetValue() {
								subscribeCount++
							} else {
								unsubscribeCount++
							}
						}
					}

					assert.Equal(t, 4, subCount, "Expected 4 subscription events")
					assert.Equal(t, 3, subscribeCount, "Expected 3 subscribe events")
					assert.Equal(t, 1, unsubscribeCount, "Expected 1 unsubscribe event")
					assert.Equal(t, 4, len(topics), "Expected 4 different topics")
				})
			},
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

			err := mimicry.handleRecvRPCEvent(context.Background(), clientMeta, traceMeta, tt.event)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// Helper function to count events by type
func countEventsByType(events []*xatu.DecoratedEvent) map[xatu.Event_Name]int {
	counts := make(map[xatu.Event_Name]int)
	for _, e := range events {
		counts[e.GetEvent().GetName()]++
	}

	return counts
}

// Helper function to create a basic RPC event
func createRPCEvent(peerID peer.ID, rpcMeta *host.RpcMeta) *host.TraceEvent {
	return &host.TraceEvent{
		Type:      "RecvRPC",
		PeerID:    peerID,
		Timestamp: time.Now(),
		Payload:   rpcMeta,
	}
}

// Helper function to create test client metadata
func createTestClientMeta() *xatu.ClientMeta {
	return &xatu.ClientMeta{
		Name: "test-client",
		Id:   uuid.New().String(),
		Ethereum: &xatu.ClientMeta_Ethereum{
			Network: &xatu.ClientMeta_Ethereum_Network{
				Id:   1,
				Name: "testnet",
			},
		},
	}
}

// Helper function to create test trace metadata
func createTestTraceMeta() *libp2p.TraceEventMetadata {
	return &libp2p.TraceEventMetadata{
		PeerId: wrapperspb.String("test-peer"),
	}
}

func validateEventCounts(t *testing.T, events []*xatu.DecoratedEvent, assertions ...eventCountAssertion) {
	t.Helper()

	counts := countEventsByType(events)

	for _, assertion := range assertions {
		assert.Equal(t, assertion.expected, counts[assertion.eventType], assertion.message)
	}
}

// Helper to create a test mimicry instance
func createTestMimicry(t *testing.T, config *Config, sink output.Sink) *Mimicry {
	t.Helper()

	return &Mimicry{
		Config:  config,
		sinks:   []output.Sink{sink},
		log:     logrus.NewEntry(logrus.New()),
		id:      uuid.New(),
		metrics: NewMetrics(t.Name()),
		sharder: &UnifiedSharder{
			config:           &ShardingConfig{},
			eventCategorizer: NewEventCategorizer(),
			enabled:          false, // Disable sharding for tests
		},
	}
}

func expectEventsWithValidation(t *testing.T, mockSink *mock.MockSink, validator eventValidator) {
	t.Helper()

	mockSink.EXPECT().
		HandleNewDecoratedEvents(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, events []*xatu.DecoratedEvent) error {
			validator(t, events)

			return nil
		}).
		Times(1)
}

// Helper to create a mock expectation that validates event counts
func expectEventCounts(t *testing.T, mockSink *mock.MockSink, assertions ...eventCountAssertion) {
	t.Helper()

	mockSink.EXPECT().
		HandleNewDecoratedEvents(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, events []*xatu.DecoratedEvent) error {
			validateEventCounts(t, events, assertions...)

			return nil
		}).
		Times(1)
}

// Helper to create a mock expectation for no events
func expectNoEvents(_ *mock.MockSink) {
	// No calls expected - the parameter is unused but kept for consistency
}
