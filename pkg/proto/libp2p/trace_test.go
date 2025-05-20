package libp2p

import (
	"testing"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/probe-lab/hermes/host"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	wrapperspb "google.golang.org/protobuf/types/known/wrapperspb"
)

const (
	peerIDStr = "16Uiu2HAm68jFpjEsRyc1rksPWCorrqwoyR7qdPSvHcinzssnMXJq"
)

// TestTraceEventToRPC tests both TraceEventToRecvRPC and TraceEventToSendRPC methods.
// They both have the same structure and behavior.
func TestTraceEventToRPC(t *testing.T) {
	peerID, err := peer.Decode(peerIDStr)
	require.NoError(t, err)

	type traceEventFunc func(*host.TraceEvent) (interface{}, error)
	traceEventFuncs := map[string]traceEventFunc{
		"RecvRPC": func(event *host.TraceEvent) (interface{}, error) {
			return TraceEventToRecvRPC(event)
		},
		"SendRPC": func(event *host.TraceEvent) (interface{}, error) {
			return TraceEventToSendRPC(event)
		},
	}

	// Define test cases
	tests := []struct {
		name        string
		input       *host.TraceEvent
		expectError bool
	}{
		{
			name:  "valid with messages and subscriptions",
			input: createTraceEventWithMessages(peerID),
		},
		{
			name:  "valid with ihave control messages",
			input: createTraceEventWithIHave(peerID),
		},
		{
			name:  "valid with iwant control messages",
			input: createTraceEventWithIWant(peerID),
		},
		{
			name:  "valid with idontwant control messages",
			input: createTraceEventWithIDontWant(peerID),
		},
		{
			name:  "valid with graft control messages",
			input: createTraceEventWithGraft(peerID),
		},
		{
			name:  "valid with prune control messages",
			input: createTraceEventWithPrune(peerID),
		},
		{
			name:  "valid with all control messages",
			input: createTraceEventWithAllControls(peerID),
		},
		{
			name: "invalid payload type",
			input: &host.TraceEvent{
				PeerID:  peerID,
				Payload: "not an RpcMeta",
			},
			expectError: true,
		},
		{
			name:  "nil control",
			input: createTraceEventWithNilControl(peerID),
		},
	}

	// Run tests for each conversion function
	for name, fn := range traceEventFuncs {
		t.Run(name, func(t *testing.T) {
			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					result, err := fn(tt.input)

					if tt.expectError {
						assert.Error(t, err)

						return
					}

					require.NoError(t, err)

					// Create expected output based on the function type.
					var expected interface{}

					payload, ok := tt.input.Payload.(*host.RpcMeta)
					require.True(t, ok)

					meta := createExpectedRPCMeta(peerIDStr, payload)

					if name == "RecvRPC" {
						expected = &RecvRPC{
							PeerId: wrapperspb.String(peerIDStr),
							Meta:   meta,
						}

						actual, ok := result.(*RecvRPC)
						require.True(t, ok)

						// Safe type assertion since we created expected as *RecvRPC above
						expectedRecv, ok := expected.(*RecvRPC)
						require.True(t, ok)

						assertRPCEquals(t, expectedRecv, actual)
					} else {
						expected = &SendRPC{
							PeerId: wrapperspb.String(peerIDStr),
							Meta:   meta,
						}

						actual, ok := result.(*SendRPC)
						require.True(t, ok)

						// Safe type assertion since we created expected as *SendRPC above
						expectedSend, ok := expected.(*SendRPC)
						require.True(t, ok)

						assertRPCEquals(t, expectedSend, actual)
					}
				})
			}
		})
	}
}

// Helper function to create expected RPCMeta from a host.RpcMeta
func createExpectedRPCMeta(peerIDStr string, payload *host.RpcMeta) *RPCMeta {
	meta := &RPCMeta{
		PeerId: wrapperspb.String(peerIDStr),
	}

	// Add messages if any
	if len(payload.Messages) > 0 {
		meta.Messages = createExpectedMessages()
	} else {
		meta.Messages = []*MessageMeta{}
	}

	// Add subscriptions if any
	if len(payload.Subscriptions) > 0 {
		meta.Subscriptions = createExpectedSubscriptions()
	}

	// Add control if any
	if payload.Control != nil {
		meta.Control = &ControlMeta{}

		// Add IHave if any
		if len(payload.Control.IHave) > 0 {
			meta.Control.Ihave = []*ControlIHaveMeta{
				{
					TopicId:    wrapperspb.String("topic1"),
					MessageIds: convertStringValues([]string{"msg1", "msg2"}),
				},
			}
		}

		// Add IWant if any
		if len(payload.Control.IWant) > 0 {
			meta.Control.Iwant = []*ControlIWantMeta{
				{
					MessageIds: convertStringValues([]string{"msg1", "msg2"}),
				},
			}

			// Special case for all controls test
			if len(payload.Control.IHave) > 0 &&
				len(payload.Control.Graft) > 0 &&
				len(payload.Control.Prune) > 0 {
				meta.Control.Iwant = []*ControlIWantMeta{
					{
						MessageIds: convertStringValues([]string{"msg3", "msg4"}),
					},
				}
			}
		}

		// Add IDontWant if any
		if len(payload.Control.Idontwant) > 0 {
			meta.Control.Idontwant = []*ControlIDontWantMeta{
				{
					MessageIds: convertStringValues([]string{"msg1", "msg2"}),
				},
			}

			// Special case for all controls test
			if len(payload.Control.IHave) > 0 &&
				len(payload.Control.IWant) > 0 &&
				len(payload.Control.Graft) > 0 {
				meta.Control.Idontwant = []*ControlIDontWantMeta{
					{
						MessageIds: convertStringValues([]string{"msg5", "msg6"}),
					},
				}
			}
		}

		// Add Graft if any
		if len(payload.Control.Graft) > 0 {
			topicID := "topic1"
			if len(payload.Control.IHave) > 0 &&
				len(payload.Control.IWant) > 0 &&
				len(payload.Control.Prune) > 0 {
				topicID = "topic2"
			}

			meta.Control.Graft = []*ControlGraftMeta{
				{
					TopicId: wrapperspb.String(topicID),
				},
			}
		}

		// Add Prune if any
		if len(payload.Control.Prune) > 0 {
			topicID := "topic1"
			if len(payload.Control.IHave) > 0 &&
				len(payload.Control.IWant) > 0 &&
				len(payload.Control.Graft) > 0 {
				topicID = "topic3"
			}

			meta.Control.Prune = []*ControlPruneMeta{
				{
					TopicId: wrapperspb.String(topicID),
					PeerIds: convertStringValues([]string{peerIDStr}),
				},
			}
		}
	}

	return meta
}

// Generic assertion function that works for both RecvRPC and SendRPC
func assertRPCEquals(t *testing.T, expected, actual interface{}) {
	t.Helper()

	var expectedPeerID, actualPeerID string
	var expectedMeta, actualMeta *RPCMeta

	switch e := expected.(type) {
	case *RecvRPC:
		expectedPeerID = e.PeerId.GetValue()
		expectedMeta = e.Meta

		a, ok := actual.(*RecvRPC)
		require.True(t, ok)

		actualPeerID = a.PeerId.GetValue()
		actualMeta = a.Meta
	case *SendRPC:
		expectedPeerID = e.PeerId.GetValue()
		expectedMeta = e.Meta

		a, ok := actual.(*SendRPC)
		require.True(t, ok)

		actualPeerID = a.PeerId.GetValue()
		actualMeta = a.Meta
	default:
		t.Fatalf("Unexpected type for expected: %T", expected)
	}

	assert.Equal(t, expectedPeerID, actualPeerID)
	assertRPCMetaEquals(t, expectedMeta, actualMeta)
}

// Helper function to assert equality between expected and actual RPCMeta objects
func assertRPCMetaEquals(t *testing.T, expected, actual *RPCMeta) {
	t.Helper()

	if expected.Control != nil {
		require.NotNil(t, actual.Control)

		// Check ihave
		assertControlIHaveEquals(t, expected.Control.Ihave, actual.Control.Ihave)

		// Check iwant
		assertControlIWantEquals(t, expected.Control.Iwant, actual.Control.Iwant)

		// Check idontwant
		assertControlIDontWantEquals(t, expected.Control.Idontwant, actual.Control.Idontwant)

		// Check graft
		assertControlGraftEquals(t, expected.Control.Graft, actual.Control.Graft)

		// Check prune
		assertControlPruneEquals(t, expected.Control.Prune, actual.Control.Prune)
	} else {
		assert.Nil(t, actual.Control)
	}

	// Check messages
	assertMessagesEquals(t, expected.Messages, actual.Messages)

	// Check subscriptions
	assertSubscriptionsEquals(t, expected.Subscriptions, actual.Subscriptions)
}

func createTraceEventWithMessages(peerID peer.ID) *host.TraceEvent {
	return &host.TraceEvent{
		PeerID: peerID,
		Payload: &host.RpcMeta{
			PeerID: peerID,
			Messages: []host.RpcMetaMsg{
				{
					MsgID: "msg1",
					Topic: "topic1",
				},
			},
			Subscriptions: []host.RpcMetaSub{
				{
					Subscribe: true,
					TopicID:   "topic1",
				},
			},
		},
	}
}

func createTraceEventWithIHave(peerID peer.ID) *host.TraceEvent {
	return &host.TraceEvent{
		PeerID: peerID,
		Payload: &host.RpcMeta{
			PeerID: peerID,
			Control: &host.RpcMetaControl{
				IHave: []host.RpcControlIHave{
					{
						TopicID: "topic1",
						MsgIDs:  []string{"msg1", "msg2"},
					},
				},
			},
		},
	}
}

func createTraceEventWithIWant(peerID peer.ID) *host.TraceEvent {
	return &host.TraceEvent{
		PeerID: peerID,
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
	}
}

func createTraceEventWithIDontWant(peerID peer.ID) *host.TraceEvent {
	return &host.TraceEvent{
		PeerID: peerID,
		Payload: &host.RpcMeta{
			PeerID: peerID,
			Control: &host.RpcMetaControl{
				Idontwant: []host.RpcControlIdontWant{
					{
						MsgIDs: []string{"msg1", "msg2"},
					},
				},
			},
		},
	}
}

func createTraceEventWithGraft(peerID peer.ID) *host.TraceEvent {
	return &host.TraceEvent{
		PeerID: peerID,
		Payload: &host.RpcMeta{
			PeerID: peerID,
			Control: &host.RpcMetaControl{
				Graft: []host.RpcControlGraft{
					{
						TopicID: "topic1",
					},
				},
			},
		},
	}
}

func createTraceEventWithPrune(peerID peer.ID) *host.TraceEvent {
	return &host.TraceEvent{
		PeerID: peerID,
		Payload: &host.RpcMeta{
			PeerID: peerID,
			Control: &host.RpcMetaControl{
				Prune: []host.RpcControlPrune{
					{
						TopicID: "topic1",
						PeerIDs: []peer.ID{peerID},
					},
				},
			},
		},
	}
}

func createTraceEventWithAllControls(peerID peer.ID) *host.TraceEvent {
	return &host.TraceEvent{
		PeerID: peerID,
		Payload: &host.RpcMeta{
			PeerID: peerID,
			Control: &host.RpcMetaControl{
				IHave: []host.RpcControlIHave{
					{
						TopicID: "topic1",
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
						TopicID: "topic2",
					},
				},
				Prune: []host.RpcControlPrune{
					{
						TopicID: "topic3",
						PeerIDs: []peer.ID{peerID},
					},
				},
			},
		},
	}
}

func createTraceEventWithNilControl(peerID peer.ID) *host.TraceEvent {
	return &host.TraceEvent{
		PeerID: peerID,
		Payload: &host.RpcMeta{
			PeerID:   peerID,
			Control:  nil,
			Messages: []host.RpcMetaMsg{},
		},
	}
}

func createExpectedMessages() []*MessageMeta {
	return []*MessageMeta{
		{
			MessageId: wrapperspb.String("msg1"),
			Topic:     wrapperspb.String("topic1"),
		},
	}
}

func createExpectedSubscriptions() []*SubMeta {
	return []*SubMeta{
		{
			Subscribe: wrapperspb.Bool(true),
			TopicId:   wrapperspb.String("topic1"),
		},
	}
}

func assertControlIHaveEquals(t *testing.T, expected, actual []*ControlIHaveMeta) {
	t.Helper()

	if len(expected) > 0 {
		require.Equal(t, len(expected), len(actual))

		for i, exp := range expected {
			assert.Equal(t, exp.TopicId.GetValue(), actual[i].TopicId.GetValue())
			assertStringValuesEqual(t, exp.MessageIds, actual[i].MessageIds)
		}
	}
}

func assertControlIWantEquals(t *testing.T, expected, actual []*ControlIWantMeta) {
	t.Helper()

	if len(expected) > 0 {
		require.Equal(t, len(expected), len(actual))

		for i, exp := range expected {
			assertStringValuesEqual(t, exp.MessageIds, actual[i].MessageIds)
		}
	}
}

func assertControlIDontWantEquals(t *testing.T, expected, actual []*ControlIDontWantMeta) {
	t.Helper()

	if len(expected) > 0 {
		require.Equal(t, len(expected), len(actual))

		for i, exp := range expected {
			assertStringValuesEqual(t, exp.MessageIds, actual[i].MessageIds)
		}
	}
}

func assertControlGraftEquals(t *testing.T, expected, actual []*ControlGraftMeta) {
	t.Helper()

	if len(expected) > 0 {
		require.Equal(t, len(expected), len(actual))

		for i, exp := range expected {
			assert.Equal(t, exp.TopicId.GetValue(), actual[i].TopicId.GetValue())
		}
	}
}

func assertControlPruneEquals(t *testing.T, expected, actual []*ControlPruneMeta) {
	t.Helper()

	if len(expected) > 0 {
		require.Equal(t, len(expected), len(actual))

		for i, exp := range expected {
			assert.Equal(t, exp.TopicId.GetValue(), actual[i].TopicId.GetValue())
			assertStringValuesEqual(t, exp.PeerIds, actual[i].PeerIds)
		}
	}
}

func assertMessagesEquals(t *testing.T, expected, actual []*MessageMeta) {
	t.Helper()

	if len(expected) > 0 {
		require.Equal(t, len(expected), len(actual))

		for i, exp := range expected {
			assert.Equal(t, exp.MessageId.GetValue(), actual[i].MessageId.GetValue())
			assert.Equal(t, exp.Topic.GetValue(), actual[i].Topic.GetValue())
		}
	}
}

func assertSubscriptionsEquals(t *testing.T, expected, actual []*SubMeta) {
	t.Helper()

	if len(expected) > 0 {
		require.Equal(t, len(expected), len(actual))

		for i, exp := range expected {
			assert.Equal(t, exp.Subscribe.GetValue(), actual[i].Subscribe.GetValue())
			assert.Equal(t, exp.TopicId.GetValue(), actual[i].TopicId.GetValue())
		}
	}
}

func assertStringValuesEqual(t *testing.T, expected, actual []*wrapperspb.StringValue) {
	t.Helper()

	assert.Equal(t, len(expected), len(actual))
	for j, val := range expected {
		assert.Equal(t, val.GetValue(), actual[j].GetValue())
	}
}
