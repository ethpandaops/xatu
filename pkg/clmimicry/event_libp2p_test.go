package clmimicry

import (
	"context"
	"testing"
	"time"

	"github.com/ethpandaops/ethwallclock"
	"github.com/ethpandaops/xatu/pkg/output"
	"github.com/ethpandaops/xatu/pkg/output/mock"
	"github.com/ethpandaops/xatu/pkg/proto/libp2p"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/google/uuid"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

const examplePeerID = "16Uiu2HAm68jFpjEsRyc1rksPWCorrqwoyR7qdPSvHcinzssnMXJq"

// testMimicry is a test helper struct that holds components needed for testing
type testMimicry struct {
	Config    *Config
	sinks     []output.Sink
	log       *logrus.Entry
	id        uuid.UUID
	metrics   *Metrics
	sharder   *UnifiedSharder
	processor *Processor
}

// GetProcessor returns the processor for testing
func (m *testMimicry) GetProcessor() *Processor {
	return m.processor
}

// testOutputHandler wraps a mock sink to implement OutputHandler interface
type testOutputHandler struct {
	sink output.Sink
}

func (h *testOutputHandler) HandleDecoratedEvent(ctx context.Context, event *xatu.DecoratedEvent) error {
	return h.sink.HandleNewDecoratedEvent(ctx, event)
}

func (h *testOutputHandler) HandleDecoratedEvents(ctx context.Context, events []*xatu.DecoratedEvent) error {
	return h.sink.HandleNewDecoratedEvents(ctx, events)
}

// eventCountAssertion is a helper to validate event counts.
type eventCountAssertion struct {
	eventType xatu.Event_Name
	expected  int
	message   string
}

// Helper to create a mock expectation that validates event counts and additional properties.
type eventValidator func(t *testing.T, events []*xatu.DecoratedEvent)

// Helper to validate a single event
type singleEventValidator func(t *testing.T, event *xatu.DecoratedEvent)

func Test_handleRecvRPCEvent(t *testing.T) {
	// Create a valid peer ID for testing
	peerID, err := peer.Decode(examplePeerID)
	require.NoError(t, err)

	tests := []struct {
		name           string
		config         *Config
		event          *RecvRPCEvent
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
			event: createRPCEvent(peerID, &RpcMeta{
				PeerID: peerID,
				Control: &RpcMetaControl{
					IHave: []RpcControlIHave{
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
			event: &RecvRPCEvent{
				TraceEventBase: TraceEventBase{
					Timestamp: time.Now(),
					PeerID:    peerID,
				},
				Meta: &RpcMeta{
					PeerID: peerID,
					Control: &RpcMetaControl{
						IWant: []RpcControlIWant{
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
			event: &RecvRPCEvent{
				TraceEventBase: TraceEventBase{
					Timestamp: time.Now(),
					PeerID:    peerID,
				},
				Meta: &RpcMeta{
					PeerID: peerID,
					Control: &RpcMetaControl{
						Idontwant: []RpcControlIdontWant{
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
			event: &RecvRPCEvent{
				TraceEventBase: TraceEventBase{
					Timestamp: time.Now(),
					PeerID:    peerID,
				},
				Meta: &RpcMeta{
					PeerID: peerID,
					Control: &RpcMetaControl{
						Graft: []RpcControlGraft{
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
			event: &RecvRPCEvent{
				TraceEventBase: TraceEventBase{
					Timestamp: time.Now(),
					PeerID:    peerID,
				},
				Meta: &RpcMeta{
					PeerID: peerID,
					Control: &RpcMetaControl{
						Prune: []RpcControlPrune{
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
			event: &RecvRPCEvent{
				TraceEventBase: TraceEventBase{
					Timestamp: time.Now(),
					PeerID:    peerID,
				},
				Meta: &RpcMeta{
					PeerID: peerID,
					Control: &RpcMetaControl{
						Prune: []RpcControlPrune{
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
			event: &RecvRPCEvent{
				TraceEventBase: TraceEventBase{
					Timestamp: time.Now(),
					PeerID:    peerID,
				},
				Meta: &RpcMeta{
					PeerID: peerID,
					Control: &RpcMetaControl{
						Prune: []RpcControlPrune{
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
			event: createRPCEvent(peerID, &RpcMeta{
				PeerID: peerID,
				Control: &RpcMetaControl{
					IHave: []RpcControlIHave{
						{
							TopicID: "/eth2/test-topic",
							MsgIDs:  []string{"msg1", "msg2"},
						},
					},
					IWant: []RpcControlIWant{
						{
							MsgIDs: []string{"msg3"},
						},
					},
					Graft: []RpcControlGraft{
						{
							TopicID: "/eth2/test-topic",
						},
					},
					Prune: []RpcControlPrune{
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
			event: &RecvRPCEvent{
				TraceEventBase: TraceEventBase{
					Timestamp: time.Now(),
					PeerID:    peerID,
				},
				Meta: &RpcMeta{
					PeerID: peerID,
					Control: &RpcMetaControl{
						IHave: []RpcControlIHave{
							{
								TopicID: "/eth2/test-topic",
								MsgIDs:  []string{"msg1"},
							},
						},
						IWant: []RpcControlIWant{
							{
								MsgIDs: []string{"msg2"},
							},
						},
						Graft: []RpcControlGraft{
							{
								TopicID: "/eth2/test-topic",
							},
						},
						Prune: []RpcControlPrune{
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
			event: &RecvRPCEvent{
				TraceEventBase: TraceEventBase{
					Timestamp: time.Now(),
					PeerID:    peerID,
				},
				Meta: &RpcMeta{
					PeerID: peerID,
					Control: &RpcMetaControl{
						IHave: []RpcControlIHave{
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
			event: &RecvRPCEvent{
				TraceEventBase: TraceEventBase{
					Timestamp: time.Now(),
					PeerID:    peerID,
				},
				Meta: &RpcMeta{
					PeerID: peerID,
					Control: &RpcMetaControl{
						IHave: []RpcControlIHave{
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
			event: &RecvRPCEvent{
				TraceEventBase: TraceEventBase{
					Timestamp: time.Now(),
					PeerID:    peerID,
				},
				Meta: &RpcMeta{
					PeerID: peerID,
					Subscriptions: []RpcMetaSub{
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
			event: &RecvRPCEvent{
				TraceEventBase: TraceEventBase{
					Timestamp: time.Now(),
					PeerID:    peerID,
				},
				Meta: &RpcMeta{
					PeerID: peerID,
					Messages: []RpcMetaMsg{
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
			event: &RecvRPCEvent{
				TraceEventBase: TraceEventBase{
					Timestamp: time.Now(),
					PeerID:    peerID,
				},
				Meta: &RpcMeta{
					PeerID: peerID,
					Subscriptions: []RpcMetaSub{
						{
							Subscribe: true,
							TopicID:   "/eth2/test-topic",
						},
					},
					Messages: []RpcMetaMsg{
						{
							MsgID: "msg1",
							Topic: "/eth2/test-topic",
						},
					},
					Control: &RpcMetaControl{
						Graft: []RpcControlGraft{
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
			event: &RecvRPCEvent{
				TraceEventBase: TraceEventBase{
					Timestamp: time.Now(),
					PeerID:    peerID,
				},
				Meta: &RpcMeta{
					PeerID:        peerID,
					Subscriptions: []RpcMetaSub{}, // Empty subscriptions
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
			event: &RecvRPCEvent{
				TraceEventBase: TraceEventBase{
					Timestamp: time.Now(),
					PeerID:    peerID,
				},
				Meta: &RpcMeta{
					PeerID:   peerID,
					Messages: []RpcMetaMsg{}, // Empty messages
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
			event: &RecvRPCEvent{
				TraceEventBase: TraceEventBase{
					Timestamp: time.Now(),
					PeerID:    peerID,
				},
				Meta: &RpcMeta{
					PeerID: peerID,
					Subscriptions: []RpcMetaSub{
						{
							Subscribe: true,
							TopicID:   "/eth2/test-topic",
						},
					},
					Messages: []RpcMetaMsg{
						{
							MsgID: "msg1",
							Topic: "/eth2/test-topic",
						},
					},
					Control: &RpcMetaControl{
						IHave: []RpcControlIHave{
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
			event: &RecvRPCEvent{
				TraceEventBase: TraceEventBase{
					Timestamp: time.Now(),
					PeerID:    peerID,
				},
				Meta: &RpcMeta{
					PeerID: peerID,
					Subscriptions: []RpcMetaSub{
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

			err := mimicry.GetProcessor().handleRecvRPCEvent(context.Background(), tt.event, clientMeta, traceMeta)

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
		event          TraceEvent
		expectError    bool
		validateCalls  func(t *testing.T, events []*xatu.DecoratedEvent)
		setupMockCalls func(*mock.MockSink)
	}{
		{
			name: "Basic ADD_PEER event",
			config: &Config{
				Events: EventConfig{
					AddPeerEnabled: true,
				},
			},
			event: &AddPeerEvent{
				TraceEventBase: TraceEventBase{
					PeerID:    peerID,
					Timestamp: time.Now(),
				},
				Protocol: protocol.ID("/meshsub/1.2.0"),
			},
			expectError: false,
			setupMockCalls: func(mockSink *mock.MockSink) {
				expectEventWithValidation(t, mockSink, func(t *testing.T, event *xatu.DecoratedEvent) {
					t.Helper()

					assert.Equal(t, xatu.Event_LIBP2P_TRACE_ADD_PEER, event.GetEvent().GetName())

					addPeerData := event.GetLibp2PTraceAddPeer()
					assert.NotNil(t, addPeerData, "Expected ADD_PEER data to be set")
					assert.Equal(t, peerID.String(), addPeerData.PeerId.GetValue(), "Expected peer ID to match")
					assert.Equal(t, "/meshsub/1.2.0", addPeerData.Protocol.GetValue(), "Expected protocol to match")
				})
			},
		},
		{
			name: "ADD_PEER event with different protocol",
			config: &Config{
				Events: EventConfig{
					AddPeerEnabled: true,
				},
			},
			event: &AddPeerEvent{
				TraceEventBase: TraceEventBase{
					PeerID:    peerID,
					Timestamp: time.Now(),
				},
				Protocol: protocol.ID("/meshsub/1.1.0"),
			},
			expectError: false,
			setupMockCalls: func(mockSink *mock.MockSink) {
				expectEventWithValidation(t, mockSink, func(t *testing.T, event *xatu.DecoratedEvent) {
					t.Helper()

					assert.Equal(t, xatu.Event_LIBP2P_TRACE_ADD_PEER, event.GetEvent().GetName())

					addPeerData := event.GetLibp2PTraceAddPeer()
					assert.NotNil(t, addPeerData, "Expected ADD_PEER data to be set")
					assert.Equal(t, peerID.String(), addPeerData.PeerId.GetValue(), "Expected peer ID to match")
					assert.Equal(t, "/meshsub/1.1.0", addPeerData.Protocol.GetValue(), "Expected protocol to match")
				})
			},
		},
		{
			name: "ADD_PEER event disabled",
			config: &Config{
				Events: EventConfig{
					AddPeerEnabled: false,
				},
			},
			event: &AddPeerEvent{
				TraceEventBase: TraceEventBase{
					PeerID:    peerID,
					Timestamp: time.Now(),
				},
				Protocol: protocol.ID("/meshsub/1.2.0"),
			},
			expectError:    false,
			setupMockCalls: expectNoEvents,
		},
		{
			name: "Multiple ADD_PEER events",
			config: &Config{
				Events: EventConfig{
					AddPeerEnabled: true,
				},
			},
			event: &AddPeerEvent{
				TraceEventBase: TraceEventBase{
					PeerID:    peerID,
					Timestamp: time.Now(),
				},
				Protocol: protocol.ID("/meshsub/1.2.0"),
			},
			expectError: false,
			setupMockCalls: func(mockSink *mock.MockSink) {
				// ADD_PEER sends a single event, not an array
				expectEventWithValidation(t, mockSink, func(t *testing.T, event *xatu.DecoratedEvent) {
					t.Helper()
					assert.Equal(t, xatu.Event_LIBP2P_TRACE_ADD_PEER, event.GetEvent().GetName())
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

			mimicry := createTestMimicry(t, tt.config, mockSink) // Call handleHermesLibp2pEvent which routes to handleAddPeerEvent
			err := mimicry.GetProcessor().HandleHermesEvent(context.Background(), tt.event)

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
		event          TraceEvent
		expectError    bool
		validateCalls  func(t *testing.T, events []*xatu.DecoratedEvent)
		setupMockCalls func(*mock.MockSink)
	}{
		{
			name: "Basic REMOVE_PEER event",
			config: &Config{
				Events: EventConfig{
					RemovePeerEnabled: true,
				},
			},
			event: &RemovePeerEvent{
				TraceEventBase: TraceEventBase{
					PeerID:    peerID,
					Timestamp: time.Now(),
				},
			},
			expectError: false,
			setupMockCalls: func(mockSink *mock.MockSink) {
				expectEventWithValidation(t, mockSink, func(t *testing.T, event *xatu.DecoratedEvent) {
					t.Helper()

					assert.Equal(t, xatu.Event_LIBP2P_TRACE_REMOVE_PEER, event.GetEvent().GetName())

					removePeerData := event.GetLibp2PTraceRemovePeer()
					assert.NotNil(t, removePeerData, "Expected REMOVE_PEER data to be set")
					assert.Equal(t, peerID.String(), removePeerData.PeerId.GetValue(), "Expected peer ID to match")
				})
			},
		},
		{
			name: "REMOVE_PEER event disabled",
			config: &Config{
				Events: EventConfig{
					RemovePeerEnabled: false,
				},
			},
			event: &RemovePeerEvent{
				TraceEventBase: TraceEventBase{
					PeerID:    peerID,
					Timestamp: time.Now(),
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

			mimicry := createTestMimicry(t, tt.config, mockSink) // Call handleHermesLibp2pEvent which routes to handleRemovePeerEvent
			err := mimicry.GetProcessor().HandleHermesEvent(context.Background(), tt.event)

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
		event          TraceEvent
		expectError    bool
		validateCalls  func(t *testing.T, events []*xatu.DecoratedEvent)
		setupMockCalls func(*mock.MockSink)
	}{
		{
			name: "Basic JOIN event",
			config: &Config{
				Events: EventConfig{
					JoinEnabled: true,
				},
			},
			event: &JoinEvent{
				TraceEventBase: TraceEventBase{
					PeerID:    peerID,
					Timestamp: time.Now(),
				},
				Topic: "/eth2/12D3KooWLRPJAA5o6fuqoxn4zqfLVmT6BnfgTdqEGmjPHY1u5KGR/beacon_attestation_1/ssz_snappy",
			},
			expectError: false,
			setupMockCalls: func(mockSink *mock.MockSink) {
				expectEventWithValidation(t, mockSink, func(t *testing.T, event *xatu.DecoratedEvent) {
					t.Helper()

					assert.Equal(t, xatu.Event_LIBP2P_TRACE_JOIN, event.GetEvent().GetName())

					joinData := event.GetLibp2PTraceJoin()
					assert.NotNil(t, joinData, "Expected JOIN data to be set")
					assert.Equal(t, "/eth2/12D3KooWLRPJAA5o6fuqoxn4zqfLVmT6BnfgTdqEGmjPHY1u5KGR/beacon_attestation_1/ssz_snappy", joinData.Topic.GetValue(), "Expected topic to match")
				})
			},
		},
		{
			name: "JOIN event disabled",
			config: &Config{
				Events: EventConfig{
					JoinEnabled: false,
				},
			},
			event: &JoinEvent{
				TraceEventBase: TraceEventBase{
					PeerID:    peerID,
					Timestamp: time.Now(),
				},
				Topic: "/eth2/12D3KooWLRPJAA5o6fuqoxn4zqfLVmT6BnfgTdqEGmjPHY1u5KGR/beacon_attestation_1/ssz_snappy",
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

			mimicry := createTestMimicry(t, tt.config, mockSink) // Call handleHermesLibp2pEvent which routes to handleJoinEvent
			err := mimicry.GetProcessor().HandleHermesEvent(context.Background(), tt.event)

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
		event          TraceEvent
		expectError    bool
		validateCalls  func(t *testing.T, events []*xatu.DecoratedEvent)
		setupMockCalls func(*mock.MockSink)
	}{
		{
			name: "Basic LEAVE event",
			config: &Config{
				Events: EventConfig{
					LeaveEnabled: true,
				},
			},
			event: &LeaveEvent{
				TraceEventBase: TraceEventBase{
					PeerID:    peerID,
					Timestamp: time.Now(),
				},
				Topic: "/eth2/12D3KooWLRPJAA5o6fuqoxn4zqfLVmT6BnfgTdqEGmjPHY1u5KGR/beacon_attestation_1/ssz_snappy",
			},
			expectError: false,
			setupMockCalls: func(mockSink *mock.MockSink) {
				expectEventWithValidation(t, mockSink, func(t *testing.T, event *xatu.DecoratedEvent) {
					t.Helper()

					assert.Equal(t, xatu.Event_LIBP2P_TRACE_LEAVE, event.GetEvent().GetName())

					leaveData := event.GetLibp2PTraceLeave()
					assert.NotNil(t, leaveData, "Expected LEAVE data to be set")
					assert.Equal(t, "/eth2/12D3KooWLRPJAA5o6fuqoxn4zqfLVmT6BnfgTdqEGmjPHY1u5KGR/beacon_attestation_1/ssz_snappy", leaveData.Topic.GetValue(), "Expected topic to match")
				})
			},
		},
		{
			name: "LEAVE event disabled",
			config: &Config{
				Events: EventConfig{
					LeaveEnabled: false,
				},
			},
			event: &LeaveEvent{
				TraceEventBase: TraceEventBase{
					PeerID:    peerID,
					Timestamp: time.Now(),
				},
				Topic: "/eth2/12D3KooWLRPJAA5o6fuqoxn4zqfLVmT6BnfgTdqEGmjPHY1u5KGR/beacon_attestation_1/ssz_snappy",
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

			mimicry := createTestMimicry(t, tt.config, mockSink) // Call handleHermesLibp2pEvent which routes to handleLeaveEvent
			err := mimicry.GetProcessor().HandleHermesEvent(context.Background(), tt.event)

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
		event          TraceEvent
		expectError    bool
		validateCalls  func(t *testing.T, events []*xatu.DecoratedEvent)
		setupMockCalls func(*mock.MockSink)
	}{
		{
			name: "Basic GRAFT event",
			config: &Config{
				Events: EventConfig{
					GraftEnabled: true,
				},
			},
			event: &GraftEvent{
				TraceEventBase: TraceEventBase{
					PeerID:    peerID,
					Timestamp: time.Now(),
				},
				Topic: "/eth2/12D3KooWLRPJAA5o6fuqoxn4zqfLVmT6BnfgTdqEGmjPHY1u5KGR/beacon_block/ssz_snappy",
			},
			expectError: false,
			setupMockCalls: func(mockSink *mock.MockSink) {
				expectEventWithValidation(t, mockSink, func(t *testing.T, event *xatu.DecoratedEvent) {
					t.Helper()

					assert.Equal(t, xatu.Event_LIBP2P_TRACE_GRAFT, event.GetEvent().GetName())

					graftData := event.GetLibp2PTraceGraft()
					assert.NotNil(t, graftData, "Expected GRAFT data to be set")
					assert.Equal(t, peerID.String(), graftData.PeerId.GetValue(), "Expected peer ID to match")
					assert.Equal(t, "/eth2/12D3KooWLRPJAA5o6fuqoxn4zqfLVmT6BnfgTdqEGmjPHY1u5KGR/beacon_block/ssz_snappy", graftData.Topic.GetValue(), "Expected topic to match")
				})
			},
		},
		{
			name: "GRAFT event disabled",
			config: &Config{
				Events: EventConfig{
					GraftEnabled: false,
				},
			},
			event: &GraftEvent{
				TraceEventBase: TraceEventBase{
					PeerID:    peerID,
					Timestamp: time.Now(),
				},
				Topic: "/eth2/12D3KooWLRPJAA5o6fuqoxn4zqfLVmT6BnfgTdqEGmjPHY1u5KGR/beacon_block/ssz_snappy",
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

			mimicry := createTestMimicry(t, tt.config, mockSink) // Call handleHermesLibp2pEvent which routes to handleGraftEvent
			err := mimicry.GetProcessor().HandleHermesEvent(context.Background(), tt.event)

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
		event          TraceEvent
		expectError    bool
		validateCalls  func(t *testing.T, events []*xatu.DecoratedEvent)
		setupMockCalls func(*mock.MockSink)
	}{
		{
			name: "Basic PRUNE event",
			config: &Config{
				Events: EventConfig{
					PruneEnabled: true,
				},
			},
			event: &PruneEvent{
				TraceEventBase: TraceEventBase{
					PeerID:    peerID,
					Timestamp: time.Now(),
				},
				Topic: "/eth2/12D3KooWLRPJAA5o6fuqoxn4zqfLVmT6BnfgTdqEGmjPHY1u5KGR/beacon_block/ssz_snappy",
			},
			expectError: false,
			setupMockCalls: func(mockSink *mock.MockSink) {
				expectEventWithValidation(t, mockSink, func(t *testing.T, event *xatu.DecoratedEvent) {
					t.Helper()

					assert.Equal(t, xatu.Event_LIBP2P_TRACE_PRUNE, event.GetEvent().GetName())

					pruneData := event.GetLibp2PTracePrune()
					assert.NotNil(t, pruneData, "Expected PRUNE data to be set")
					assert.Equal(t, peerID.String(), pruneData.PeerId.GetValue(), "Expected peer ID to match")
					assert.Equal(t, "/eth2/12D3KooWLRPJAA5o6fuqoxn4zqfLVmT6BnfgTdqEGmjPHY1u5KGR/beacon_block/ssz_snappy", pruneData.Topic.GetValue(), "Expected topic to match")
				})
			},
		},
		{
			name: "PRUNE event disabled",
			config: &Config{
				Events: EventConfig{
					PruneEnabled: false,
				},
			},
			event: &PruneEvent{
				TraceEventBase: TraceEventBase{
					PeerID:    peerID,
					Timestamp: time.Now(),
				},
				Topic: "/eth2/12D3KooWLRPJAA5o6fuqoxn4zqfLVmT6BnfgTdqEGmjPHY1u5KGR/beacon_block/ssz_snappy",
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

			mimicry := createTestMimicry(t, tt.config, mockSink) // Call handleHermesLibp2pEvent which routes to handlePruneEvent
			err := mimicry.GetProcessor().HandleHermesEvent(context.Background(), tt.event)

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
		event          TraceEvent
		expectError    bool
		validateCalls  func(t *testing.T, events []*xatu.DecoratedEvent)
		setupMockCalls func(*mock.MockSink)
	}{
		{
			name: "Basic PUBLISH_MESSAGE event",
			config: &Config{
				Events: EventConfig{
					PublishMessageEnabled: true,
				},
			},
			event: &PublishMessageEvent{
				TraceEventBase: TraceEventBase{
					PeerID:    peerID,
					Timestamp: time.Now(),
				},
				MsgID: "msg123",
				Topic: "/eth2/12D3KooWLRPJAA5o6fuqoxn4zqfLVmT6BnfgTdqEGmjPHY1u5KGR/beacon_block/ssz_snappy",
			},
			expectError: false,
			setupMockCalls: func(mockSink *mock.MockSink) {
				expectEventWithValidation(t, mockSink, func(t *testing.T, event *xatu.DecoratedEvent) {
					t.Helper()

					assert.Equal(t, xatu.Event_LIBP2P_TRACE_PUBLISH_MESSAGE, event.GetEvent().GetName())

					publishData := event.GetLibp2PTracePublishMessage()
					assert.NotNil(t, publishData, "Expected PUBLISH_MESSAGE data to be set")
					assert.Equal(t, "msg123", publishData.MsgId.GetValue(), "Expected message ID to match")
					assert.Equal(t, "/eth2/12D3KooWLRPJAA5o6fuqoxn4zqfLVmT6BnfgTdqEGmjPHY1u5KGR/beacon_block/ssz_snappy", publishData.Topic.GetValue(), "Expected topic to match")
				})
			},
		},
		{
			name: "PUBLISH_MESSAGE event disabled",
			config: &Config{
				Events: EventConfig{
					PublishMessageEnabled: false,
				},
			},
			event: &PublishMessageEvent{
				TraceEventBase: TraceEventBase{
					PeerID:    peerID,
					Timestamp: time.Now(),
				},
				MsgID: "msg123",
				Topic: "/eth2/12D3KooWLRPJAA5o6fuqoxn4zqfLVmT6BnfgTdqEGmjPHY1u5KGR/beacon_block/ssz_snappy",
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

			mimicry := createTestMimicry(t, tt.config, mockSink) // Call handleHermesLibp2pEvent which routes to handlePublishMessageEvent
			err := mimicry.GetProcessor().HandleHermesEvent(context.Background(), tt.event)

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
		event          TraceEvent
		expectError    bool
		validateCalls  func(t *testing.T, events []*xatu.DecoratedEvent)
		setupMockCalls func(*mock.MockSink)
	}{
		{
			name: "Basic REJECT_MESSAGE event",
			config: &Config{
				Events: EventConfig{
					RejectMessageEnabled: true,
				},
			},
			event: &RejectMessageEvent{
				TraceEventBase: TraceEventBase{
					PeerID:    peerID,
					Timestamp: time.Now(),
				},
				MsgID:     "msg456",
				Topic:     "/eth2/12D3KooWLRPJAA5o6fuqoxn4zqfLVmT6BnfgTdqEGmjPHY1u5KGR/beacon_block/ssz_snappy",
				Reason:    "validation failed",
				Local:     true,
				MsgSize:   1024,
				SeqNumber: 1,
			},
			expectError: false,
			setupMockCalls: func(mockSink *mock.MockSink) {
				expectEventWithValidation(t, mockSink, func(t *testing.T, event *xatu.DecoratedEvent) {
					t.Helper()

					assert.Equal(t, xatu.Event_LIBP2P_TRACE_REJECT_MESSAGE, event.GetEvent().GetName())

					rejectData := event.GetLibp2PTraceRejectMessage()
					assert.NotNil(t, rejectData, "Expected REJECT_MESSAGE data to be set")
					assert.Equal(t, "msg456", rejectData.MsgId.GetValue(), "Expected message ID to match")
					assert.Equal(t, "/eth2/12D3KooWLRPJAA5o6fuqoxn4zqfLVmT6BnfgTdqEGmjPHY1u5KGR/beacon_block/ssz_snappy", rejectData.Topic.GetValue(), "Expected topic to match")
					assert.Equal(t, peerID.String(), rejectData.PeerId.GetValue(), "Expected peer ID to match")
					assert.Equal(t, "validation failed", rejectData.Reason.GetValue(), "Expected reason to match")
					assert.True(t, rejectData.Local.GetValue(), "Expected local to be true")
					assert.Equal(t, uint32(1024), rejectData.MsgSize.GetValue(), "Expected message size to match")
					assert.Equal(t, uint64(1), rejectData.SeqNumber.GetValue(), "Expected sequence number to match")
				})
			},
		},
		{
			name: "REJECT_MESSAGE event disabled",
			config: &Config{
				Events: EventConfig{
					RejectMessageEnabled: false,
				},
			},
			event: &RejectMessageEvent{
				TraceEventBase: TraceEventBase{
					PeerID:    peerID,
					Timestamp: time.Now(),
				},
				MsgID:     "msg456",
				Topic:     "/eth2/12D3KooWLRPJAA5o6fuqoxn4zqfLVmT6BnfgTdqEGmjPHY1u5KGR/beacon_block/ssz_snappy",
				Reason:    "validation failed",
				Local:     true,
				MsgSize:   1024,
				SeqNumber: 0, // Seq "01" - convert as needed
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

			mimicry := createTestMimicry(t, tt.config, mockSink) // Call handleHermesLibp2pEvent which routes to handleRejectMessageEvent
			err := mimicry.GetProcessor().HandleHermesEvent(context.Background(), tt.event)

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
		event          TraceEvent
		expectError    bool
		validateCalls  func(t *testing.T, events []*xatu.DecoratedEvent)
		setupMockCalls func(*mock.MockSink)
	}{
		{
			name: "Basic DELIVER_MESSAGE event",
			config: &Config{
				Events: EventConfig{
					DeliverMessageEnabled: true,
				},
			},
			event: &DeliverMessageEvent{
				TraceEventBase: TraceEventBase{
					PeerID:    peerID,
					Timestamp: time.Now(),
				},
				MsgID:     "msg789",
				Topic:     "/eth2/12D3KooWLRPJAA5o6fuqoxn4zqfLVmT6BnfgTdqEGmjPHY1u5KGR/beacon_block/ssz_snappy",
				Local:     false,
				MsgSize:   2048,
				SeqNumber: 2,
			},
			expectError: false,
			setupMockCalls: func(mockSink *mock.MockSink) {
				expectEventWithValidation(t, mockSink, func(t *testing.T, event *xatu.DecoratedEvent) {
					t.Helper()

					assert.Equal(t, xatu.Event_LIBP2P_TRACE_DELIVER_MESSAGE, event.GetEvent().GetName())

					deliverData := event.GetLibp2PTraceDeliverMessage()
					assert.NotNil(t, deliverData, "Expected DELIVER_MESSAGE data to be set")
					assert.Equal(t, "msg789", deliverData.MsgId.GetValue(), "Expected message ID to match")
					assert.Equal(t, "/eth2/12D3KooWLRPJAA5o6fuqoxn4zqfLVmT6BnfgTdqEGmjPHY1u5KGR/beacon_block/ssz_snappy", deliverData.Topic.GetValue(), "Expected topic to match")
					assert.Equal(t, peerID.String(), deliverData.PeerId.GetValue(), "Expected peer ID to match")
					assert.False(t, deliverData.Local.GetValue(), "Expected local to be false")
					assert.Equal(t, uint32(2048), deliverData.MsgSize.GetValue(), "Expected message size to match")
					assert.Equal(t, uint64(2), deliverData.SeqNumber.GetValue(), "Expected sequence number to match")
				})
			},
		},
		{
			name: "DELIVER_MESSAGE event disabled",
			config: &Config{
				Events: EventConfig{
					DeliverMessageEnabled: false,
				},
			},
			event: &DeliverMessageEvent{
				TraceEventBase: TraceEventBase{
					PeerID:    peerID,
					Timestamp: time.Now(),
				},
				MsgID:     "msg789",
				Topic:     "/eth2/12D3KooWLRPJAA5o6fuqoxn4zqfLVmT6BnfgTdqEGmjPHY1u5KGR/beacon_block/ssz_snappy",
				Local:     false,
				MsgSize:   2048,
				SeqNumber: 0, // Seq "02" - convert as needed
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

			mimicry := createTestMimicry(t, tt.config, mockSink) // Call handleHermesLibp2pEvent which routes to handleDeliverMessageEvent
			err := mimicry.GetProcessor().HandleHermesEvent(context.Background(), tt.event)

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
		event          TraceEvent
		expectError    bool
		validateCalls  func(t *testing.T, events []*xatu.DecoratedEvent)
		setupMockCalls func(*mock.MockSink)
	}{
		{
			name: "Basic DUPLICATE_MESSAGE event",
			config: &Config{
				Events: EventConfig{
					DuplicateMessageEnabled: true,
				},
			},
			event: &DuplicateMessageEvent{
				TraceEventBase: TraceEventBase{
					PeerID:    peerID,
					Timestamp: time.Now(),
				},
				MsgID:     "msg999",
				Topic:     "/eth2/12D3KooWLRPJAA5o6fuqoxn4zqfLVmT6BnfgTdqEGmjPHY1u5KGR/beacon_attestation_1/ssz_snappy",
				Local:     false,
				MsgSize:   512,
				SeqNumber: 3,
			},
			expectError: false,
			setupMockCalls: func(mockSink *mock.MockSink) {
				expectEventWithValidation(t, mockSink, func(t *testing.T, event *xatu.DecoratedEvent) {
					t.Helper()

					assert.Equal(t, xatu.Event_LIBP2P_TRACE_DUPLICATE_MESSAGE, event.GetEvent().GetName())

					duplicateData := event.GetLibp2PTraceDuplicateMessage()
					assert.NotNil(t, duplicateData, "Expected DUPLICATE_MESSAGE data to be set")
					assert.Equal(t, "msg999", duplicateData.MsgId.GetValue(), "Expected message ID to match")
					assert.Equal(t, "/eth2/12D3KooWLRPJAA5o6fuqoxn4zqfLVmT6BnfgTdqEGmjPHY1u5KGR/beacon_attestation_1/ssz_snappy", duplicateData.Topic.GetValue(), "Expected topic to match")
					assert.Equal(t, peerID.String(), duplicateData.PeerId.GetValue(), "Expected peer ID to match")
					assert.False(t, duplicateData.Local.GetValue(), "Expected local to be false")
					assert.Equal(t, uint32(512), duplicateData.MsgSize.GetValue(), "Expected message size to match")
					assert.Equal(t, uint64(3), duplicateData.SeqNumber.GetValue(), "Expected sequence number to match")
				})
			},
		},
		{
			name: "DUPLICATE_MESSAGE event disabled",
			config: &Config{
				Events: EventConfig{
					DuplicateMessageEnabled: false,
				},
			},
			event: &DuplicateMessageEvent{
				TraceEventBase: TraceEventBase{
					PeerID:    peerID,
					Timestamp: time.Now(),
				},
				MsgID:     "msg999",
				Topic:     "/eth2/12D3KooWLRPJAA5o6fuqoxn4zqfLVmT6BnfgTdqEGmjPHY1u5KGR/beacon_attestation_1/ssz_snappy",
				Local:     false,
				MsgSize:   512,
				SeqNumber: 0, // Seq "03" - convert as needed
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

			mimicry := createTestMimicry(t, tt.config, mockSink) // Call handleHermesLibp2pEvent which routes to handleDuplicateMessageEvent
			err := mimicry.GetProcessor().HandleHermesEvent(context.Background(), tt.event)

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
		event          TraceEvent
		expectError    bool
		validateCalls  func(t *testing.T, events []*xatu.DecoratedEvent)
		setupMockCalls func(*mock.MockSink)
	}{
		{
			name: "SEND_RPC with control messages",
			config: &Config{
				Events: EventConfig{
					SendRPCEnabled:             true,
					RpcMetaControlIHaveEnabled: true,
				},
			},
			event: &SendRPCEvent{
				TraceEventBase: TraceEventBase{
					PeerID:    peerID,
					Timestamp: time.Now(),
				},
				Meta: &RpcMeta{
					PeerID: peerID,
					Control: &RpcMetaControl{
						IHave: []RpcControlIHave{
							{
								TopicID: "/eth2/test-topic",
								MsgIDs:  []string{"msg1", "msg2"},
							},
						},
					},
				},
			},
			expectError: false,
			setupMockCalls: func(mockSink *mock.MockSink) {
				expectEventCounts(t, mockSink,
					eventCountAssertion{xatu.Event_LIBP2P_TRACE_SEND_RPC, 1, "Expected 1 root SEND_RPC event"},
					eventCountAssertion{xatu.Event_LIBP2P_TRACE_RPC_META_CONTROL_IHAVE, 2, "Expected 2 IHAVE events"},
				)
			},
		},
		{
			name: "SEND_RPC with subscriptions",
			config: &Config{
				Events: EventConfig{
					SendRPCEnabled:             true,
					RpcMetaSubscriptionEnabled: true,
				},
			},
			event: &SendRPCEvent{
				TraceEventBase: TraceEventBase{
					PeerID:    peerID,
					Timestamp: time.Now(),
				},
				Meta: &RpcMeta{
					PeerID: peerID,
					Subscriptions: []RpcMetaSub{
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

					rootCount := 0
					subCount := 0
					subscribeCount := 0
					unsubscribeCount := 0

					for _, e := range events {
						switch e.GetEvent().GetName() {
						case xatu.Event_LIBP2P_TRACE_SEND_RPC:
							rootCount++
						case xatu.Event_LIBP2P_TRACE_RPC_META_SUBSCRIPTION:
							subCount++
							subData := e.GetLibp2PTraceRpcMetaSubscription()
							if subData.Subscribe.GetValue() {
								subscribeCount++
							} else {
								unsubscribeCount++
							}
						}
					}

					assert.Equal(t, 1, rootCount, "Expected 1 root SEND_RPC event")
					assert.Equal(t, 2, subCount, "Expected 2 subscription events")
					assert.Equal(t, 1, subscribeCount, "Expected 1 subscribe event")
					assert.Equal(t, 1, unsubscribeCount, "Expected 1 unsubscribe event")
				})
			},
		},
		{
			name: "SEND_RPC with messages",
			config: &Config{
				Events: EventConfig{
					SendRPCEnabled:        true,
					RpcMetaMessageEnabled: true,
				},
			},
			event: &SendRPCEvent{
				TraceEventBase: TraceEventBase{
					PeerID:    peerID,
					Timestamp: time.Now(),
				},
				Meta: &RpcMeta{
					PeerID: peerID,
					Messages: []RpcMetaMsg{
						{
							MsgID: "msg1",
							Topic: "/eth2/beacon_block/ssz_snappy",
						},
						{
							MsgID: "msg2",
							Topic: "/eth2/beacon_attestation/ssz_snappy",
						},
					},
				},
			},
			expectError: false,
			setupMockCalls: func(mockSink *mock.MockSink) {
				expectEventCounts(t, mockSink,
					eventCountAssertion{xatu.Event_LIBP2P_TRACE_SEND_RPC, 1, "Expected 1 root SEND_RPC event"},
					eventCountAssertion{xatu.Event_LIBP2P_TRACE_RPC_META_MESSAGE, 2, "Expected 2 message events"},
				)
			},
		},
		{
			name: "SEND_RPC event disabled",
			config: &Config{
				Events: EventConfig{
					SendRPCEnabled: false,
				},
			},
			event: &SendRPCEvent{
				TraceEventBase: TraceEventBase{
					PeerID:    peerID,
					Timestamp: time.Now(),
				},
				Meta: &RpcMeta{
					PeerID: peerID,
					Control: &RpcMetaControl{
						IHave: []RpcControlIHave{
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
			name: "SEND_RPC with all meta events disabled",
			config: &Config{
				Events: EventConfig{
					SendRPCEnabled:             true,
					RpcMetaSubscriptionEnabled: false,
					RpcMetaMessageEnabled:      false,
					RpcMetaControlIHaveEnabled: false,
				},
			},
			event: &SendRPCEvent{
				TraceEventBase: TraceEventBase{
					PeerID:    peerID,
					Timestamp: time.Now(),
				},
				Meta: &RpcMeta{
					PeerID: peerID,
					Subscriptions: []RpcMetaSub{
						{
							Subscribe: true,
							TopicID:   "/eth2/test-topic",
						},
					},
					Messages: []RpcMetaMsg{
						{
							MsgID: "msg1",
							Topic: "/eth2/test-topic",
						},
					},
					Control: &RpcMetaControl{
						IHave: []RpcControlIHave{
							{
								TopicID: "/eth2/test-topic",
								MsgIDs:  []string{"msg1"},
							},
						},
					},
				},
			},
			expectError:    false,
			setupMockCalls: expectNoEvents, // No events because no child events are enabled
		},
		{
			name: "SEND_RPC with mixed control messages",
			config: &Config{
				Events: EventConfig{
					SendRPCEnabled:             true,
					RpcMetaControlGraftEnabled: true,
					RpcMetaControlPruneEnabled: true,
				},
			},
			event: &SendRPCEvent{
				TraceEventBase: TraceEventBase{
					PeerID:    peerID,
					Timestamp: time.Now(),
				},
				Meta: &RpcMeta{
					PeerID: peerID,
					Control: &RpcMetaControl{
						Graft: []RpcControlGraft{
							{
								TopicID: "/eth2/beacon_block/ssz_snappy",
							},
						},
						Prune: []RpcControlPrune{
							{
								TopicID: "/eth2/beacon_attestation/ssz_snappy",
								PeerIDs: []peer.ID{peer.ID("peer1")},
							},
						},
					},
				},
			},
			expectError: false,
			setupMockCalls: func(mockSink *mock.MockSink) {
				expectEventCounts(t, mockSink,
					eventCountAssertion{xatu.Event_LIBP2P_TRACE_SEND_RPC, 1, "Expected 1 root SEND_RPC event"},
					eventCountAssertion{xatu.Event_LIBP2P_TRACE_RPC_META_CONTROL_GRAFT, 1, "Expected 1 GRAFT event"},
					eventCountAssertion{xatu.Event_LIBP2P_TRACE_RPC_META_CONTROL_PRUNE, 1, "Expected 1 PRUNE event"},
				)
			},
		},
		{
			name: "SEND_RPC with empty control and messages",
			config: &Config{
				Events: EventConfig{
					SendRPCEnabled:             true,
					RpcMetaControlIHaveEnabled: true,
					RpcMetaMessageEnabled:      true,
				},
			},
			event: &SendRPCEvent{
				TraceEventBase: TraceEventBase{
					PeerID:    peerID,
					Timestamp: time.Now(),
				},
				Meta: &RpcMeta{
					PeerID:        peerID,
					Messages:      []RpcMetaMsg{},    // Empty messages
					Subscriptions: []RpcMetaSub{},    // Empty subscriptions
					Control:       &RpcMetaControl{}, // Empty control
				},
			},
			expectError:    false,
			setupMockCalls: expectNoEvents, // No events because no child events to emit
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

			sendRPCEvent, ok := tt.event.(*SendRPCEvent)
			require.True(t, ok, "event must be *SendRPCEvent")

			err := mimicry.GetProcessor().handleSendRPCEvent(context.Background(), sendRPCEvent, clientMeta, traceMeta)

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
		event          TraceEvent
		expectError    bool
		validateCalls  func(t *testing.T, events []*xatu.DecoratedEvent)
		setupMockCalls func(*mock.MockSink)
	}{
		{
			name: "DROP_RPC with control messages",
			config: &Config{
				Events: EventConfig{
					DropRPCEnabled:             true,
					RpcMetaControlIWantEnabled: true,
				},
			},
			event: &DropRPCEvent{
				TraceEventBase: TraceEventBase{
					PeerID:    peerID,
					Timestamp: time.Now(),
				},
				Meta: &RpcMeta{
					PeerID: peerID,
					Control: &RpcMetaControl{
						IWant: []RpcControlIWant{
							{
								MsgIDs: []string{"msg1", "msg2", "msg3"},
							},
						},
					},
				},
			},
			expectError: false,
			setupMockCalls: func(mockSink *mock.MockSink) {
				expectEventCounts(t, mockSink,
					eventCountAssertion{xatu.Event_LIBP2P_TRACE_DROP_RPC, 1, "Expected 1 root DROP_RPC event"},
					eventCountAssertion{xatu.Event_LIBP2P_TRACE_RPC_META_CONTROL_IWANT, 3, "Expected 3 IWANT events"},
				)
			},
		},
		{
			name: "DROP_RPC with subscriptions",
			config: &Config{
				Events: EventConfig{
					DropRPCEnabled:             true,
					RpcMetaSubscriptionEnabled: true,
				},
			},
			event: &DropRPCEvent{
				TraceEventBase: TraceEventBase{
					PeerID:    peerID,
					Timestamp: time.Now(),
				},
				Meta: &RpcMeta{
					PeerID: peerID,
					Subscriptions: []RpcMetaSub{
						{
							Subscribe: true,
							TopicID:   "/eth2/beacon_block/ssz_snappy",
						},
						{
							Subscribe: true,
							TopicID:   "/eth2/beacon_attestation/ssz_snappy",
						},
						{
							Subscribe: true,
							TopicID:   "/eth2/voluntary_exit/ssz_snappy",
						},
					},
				},
			},
			expectError: false,
			setupMockCalls: func(mockSink *mock.MockSink) {
				expectEventCounts(t, mockSink,
					eventCountAssertion{xatu.Event_LIBP2P_TRACE_DROP_RPC, 1, "Expected 1 root DROP_RPC event"},
					eventCountAssertion{xatu.Event_LIBP2P_TRACE_RPC_META_SUBSCRIPTION, 3, "Expected 3 subscription events"},
				)
			},
		},
		{
			name: "DROP_RPC with messages",
			config: &Config{
				Events: EventConfig{
					DropRPCEnabled:        true,
					RpcMetaMessageEnabled: true,
				},
			},
			event: &DropRPCEvent{
				TraceEventBase: TraceEventBase{
					PeerID:    peerID,
					Timestamp: time.Now(),
				},
				Meta: &RpcMeta{
					PeerID: peerID,
					Messages: []RpcMetaMsg{
						{
							MsgID: "msg1",
							Topic: "/eth2/beacon_block/ssz_snappy",
						},
					},
				},
			},
			expectError: false,
			setupMockCalls: func(mockSink *mock.MockSink) {
				expectEventCounts(t, mockSink,
					eventCountAssertion{xatu.Event_LIBP2P_TRACE_DROP_RPC, 1, "Expected 1 root DROP_RPC event"},
					eventCountAssertion{xatu.Event_LIBP2P_TRACE_RPC_META_MESSAGE, 1, "Expected 1 message event"},
				)
			},
		},
		{
			name: "DROP_RPC event disabled",
			config: &Config{
				Events: EventConfig{
					DropRPCEnabled: false,
				},
			},
			event: &DropRPCEvent{
				TraceEventBase: TraceEventBase{
					PeerID:    peerID,
					Timestamp: time.Now(),
				},
				Meta: &RpcMeta{
					PeerID: peerID,
					Messages: []RpcMetaMsg{
						{
							MsgID: "msg1",
							Topic: "/eth2/test-topic",
						},
					},
				},
			},
			expectError:    false,
			setupMockCalls: expectNoEvents,
		},
		{
			name: "DROP_RPC with all meta events disabled",
			config: &Config{
				Events: EventConfig{
					DropRPCEnabled:                 true,
					RpcMetaSubscriptionEnabled:     false,
					RpcMetaMessageEnabled:          false,
					RpcMetaControlIHaveEnabled:     false,
					RpcMetaControlIWantEnabled:     false,
					RpcMetaControlIDontWantEnabled: false,
					RpcMetaControlGraftEnabled:     false,
					RpcMetaControlPruneEnabled:     false,
				},
			},
			event: &DropRPCEvent{
				TraceEventBase: TraceEventBase{
					PeerID:    peerID,
					Timestamp: time.Now(),
				},
				Meta: &RpcMeta{
					PeerID: peerID,
					Subscriptions: []RpcMetaSub{
						{
							Subscribe: true,
							TopicID:   "/eth2/test-topic",
						},
					},
					Messages: []RpcMetaMsg{
						{
							MsgID: "msg1",
							Topic: "/eth2/test-topic",
						},
					},
					Control: &RpcMetaControl{
						IHave: []RpcControlIHave{
							{
								TopicID: "/eth2/test-topic",
								MsgIDs:  []string{"msg1"},
							},
						},
					},
				},
			},
			expectError:    false,
			setupMockCalls: expectNoEvents, // No events because no child events are enabled
		},
		{
			name: "DROP_RPC with PRUNE control messages",
			config: &Config{
				Events: EventConfig{
					DropRPCEnabled:             true,
					RpcMetaControlPruneEnabled: true,
				},
			},
			event: &DropRPCEvent{
				TraceEventBase: TraceEventBase{
					PeerID:    peerID,
					Timestamp: time.Now(),
				},
				Meta: &RpcMeta{
					PeerID: peerID,
					Control: &RpcMetaControl{
						Prune: []RpcControlPrune{
							{
								TopicID: "/eth2/beacon_block/ssz_snappy",
								PeerIDs: []peer.ID{}, // No peer IDs
							},
							{
								TopicID: "/eth2/beacon_attestation/ssz_snappy",
								PeerIDs: []peer.ID{peer.ID("peer1"), peer.ID("peer2")},
							},
						},
					},
				},
			},
			expectError: false,
			setupMockCalls: func(mockSink *mock.MockSink) {
				expectEventCounts(t, mockSink,
					eventCountAssertion{xatu.Event_LIBP2P_TRACE_DROP_RPC, 1, "Expected 1 root DROP_RPC event"},
					eventCountAssertion{xatu.Event_LIBP2P_TRACE_RPC_META_CONTROL_PRUNE, 3, "Expected 3 PRUNE events (1 + 2 peer IDs)"},
				)
			},
		},
		{
			name: "DROP_RPC with IDONTWANT control messages",
			config: &Config{
				Events: EventConfig{
					DropRPCEnabled:                 true,
					RpcMetaControlIDontWantEnabled: true,
				},
			},
			event: &DropRPCEvent{
				TraceEventBase: TraceEventBase{
					PeerID:    peerID,
					Timestamp: time.Now(),
				},
				Meta: &RpcMeta{
					PeerID: peerID,
					Control: &RpcMetaControl{
						Idontwant: []RpcControlIdontWant{
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
					eventCountAssertion{xatu.Event_LIBP2P_TRACE_DROP_RPC, 1, "Expected 1 root DROP_RPC event"},
					eventCountAssertion{xatu.Event_LIBP2P_TRACE_RPC_META_CONTROL_IDONTWANT, 2, "Expected 2 IDONTWANT events"},
				)
			},
		},
		{
			name: "DROP_RPC with mixed meta events",
			config: &Config{
				Events: EventConfig{
					DropRPCEnabled:             true,
					RpcMetaSubscriptionEnabled: true,
					RpcMetaMessageEnabled:      true,
					RpcMetaControlGraftEnabled: true,
				},
			},
			event: &DropRPCEvent{
				TraceEventBase: TraceEventBase{
					PeerID:    peerID,
					Timestamp: time.Now(),
				},
				Meta: &RpcMeta{
					PeerID: peerID,
					Subscriptions: []RpcMetaSub{
						{
							Subscribe: false,
							TopicID:   "/eth2/test-topic",
						},
					},
					Messages: []RpcMetaMsg{
						{
							MsgID: "msg1",
							Topic: "/eth2/test-topic",
						},
						{
							MsgID: "msg2",
							Topic: "/eth2/test-topic",
						},
					},
					Control: &RpcMetaControl{
						Graft: []RpcControlGraft{
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
					eventCountAssertion{xatu.Event_LIBP2P_TRACE_DROP_RPC, 1, "Expected 1 root DROP_RPC event"},
					eventCountAssertion{xatu.Event_LIBP2P_TRACE_RPC_META_SUBSCRIPTION, 1, "Expected 1 subscription event"},
					eventCountAssertion{xatu.Event_LIBP2P_TRACE_RPC_META_MESSAGE, 2, "Expected 2 message events"},
					eventCountAssertion{xatu.Event_LIBP2P_TRACE_RPC_META_CONTROL_GRAFT, 1, "Expected 1 GRAFT event"},
				)
			},
		},
		{
			name: "DROP_RPC with empty meta data",
			config: &Config{
				Events: EventConfig{
					DropRPCEnabled:             true,
					RpcMetaControlIHaveEnabled: true,
				},
			},
			event: &DropRPCEvent{
				TraceEventBase: TraceEventBase{
					PeerID:    peerID,
					Timestamp: time.Now(),
				},
				Meta: &RpcMeta{
					PeerID: peerID,
				},
			},
			expectError:    false,
			setupMockCalls: expectNoEvents, // No events because no child events to emit
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

			dropRPCEvent, ok := tt.event.(*DropRPCEvent)
			require.True(t, ok, "event must be *DropRPCEvent")

			err := mimicry.GetProcessor().handleDropRPCEvent(context.Background(), dropRPCEvent, clientMeta, traceMeta)

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
func createRPCEvent(peerID peer.ID, rpcMeta *RpcMeta) *RecvRPCEvent {
	return &RecvRPCEvent{
		TraceEventBase: TraceEventBase{
			Timestamp: time.Now(),
			PeerID:    peerID,
		},
		Meta: rpcMeta,
	}
}

// testMetaProvider implements MetaProvider for tests.
type testMetaProvider struct{}

// GetClientMeta returns test client metadata.
func (t *testMetaProvider) GetClientMeta(_ context.Context) (*xatu.ClientMeta, error) {
	return createTestClientMeta(), nil
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
func createTestMimicry(t *testing.T, config *Config, sink output.Sink) *testMimicry {
	t.Helper()

	// Create sharder from config if sharding is configured, otherwise disable it
	var sharder *UnifiedSharder
	if config.Sharding.Topics != nil || config.Sharding.NoShardingKeyEvents != nil {
		// Use sharding from config
		var err error
		sharder, err = NewUnifiedSharder(&config.Sharding, true)
		if err != nil {
			t.Fatalf("Failed to create sharder: %v", err)
		}
	} else {
		// Disable sharding for tests that don't configure it
		sharder = &UnifiedSharder{
			config:           &ShardingConfig{},
			eventCategorizer: NewEventCategorizer(),
			enabled:          false,
		}
	}

	mimicry := &testMimicry{
		Config:  config,
		sinks:   []output.Sink{sink},
		log:     logrus.NewEntry(logrus.New()),
		id:      uuid.New(),
		metrics: NewMetrics(t.Name()),
		sharder: sharder,
	}

	wallclock := &ethwallclock.EthereumBeaconChain{}

	mimicry.processor = NewProcessor(
		nil,                            // DutiesProvider - not used in these tests
		&testOutputHandler{sink: sink}, // OutputHandler wrapping the mock sink
		mimicry.metrics,                // MetricsCollector
		&testMetaProvider{},            // MetaProvider for tests
		mimicry.sharder,                // UnifiedSharder
		NewEventCategorizer(),          // EventCategorizer
		wallclock,                      // EthereumBeaconChain
		time.Duration(0),               // clockDrift
		config.Events,                  // EventConfig
		mimicry.log.WithField("component", "processor"),
	)

	return mimicry
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

// Helper to create a mock expectation for a single event
func expectEventWithValidation(t *testing.T, mockSink *mock.MockSink, validator singleEventValidator) {
	t.Helper()

	mockSink.EXPECT().
		HandleNewDecoratedEvent(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, event *xatu.DecoratedEvent) error {
			validator(t, event)

			return nil
		}).
		Times(1)
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
