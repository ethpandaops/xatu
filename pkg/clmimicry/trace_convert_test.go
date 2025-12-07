package clmimicry

import (
	"testing"

	"github.com/ethpandaops/xatu/pkg/proto/libp2p"
	"github.com/libp2p/go-libp2p/core/peer"
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

	type traceEventFunc func(*TraceEvent) (interface{}, error)

	traceEventFuncs := map[string]traceEventFunc{
		"RecvRPC": func(event *TraceEvent) (interface{}, error) {
			return TraceEventToRecvRPC(event)
		},
		"SendRPC": func(event *TraceEvent) (interface{}, error) {
			return TraceEventToSendRPC(event)
		},
	}

	// Define test cases
	tests := []struct {
		name        string
		input       *TraceEvent
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
			input: &TraceEvent{
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

					payload, ok := tt.input.Payload.(*RpcMeta)
					require.True(t, ok)

					meta := createExpectedRPCMeta(peerIDStr, payload)

					if name == "RecvRPC" {
						expected = &libp2p.RecvRPC{
							PeerId: wrapperspb.String(peerIDStr),
							Meta:   meta,
						}

						actual, ok := result.(*libp2p.RecvRPC)
						require.True(t, ok)

						// Safe type assertion since we created expected as *RecvRPC above
						expectedRecv, ok := expected.(*libp2p.RecvRPC)
						require.True(t, ok)

						assertRPCEquals(t, expectedRecv, actual)
					} else {
						expected = &libp2p.SendRPC{
							PeerId: wrapperspb.String(peerIDStr),
							Meta:   meta,
						}

						actual, ok := result.(*libp2p.SendRPC)
						require.True(t, ok)

						// Safe type assertion since we created expected as *SendRPC above
						expectedSend, ok := expected.(*libp2p.SendRPC)
						require.True(t, ok)

						assertRPCEquals(t, expectedSend, actual)
					}
				})
			}
		})
	}
}

// Helper function to create expected RPCMeta from a RpcMeta
func createExpectedRPCMeta(peerIDStr string, payload *RpcMeta) *libp2p.RPCMeta {
	meta := &libp2p.RPCMeta{
		PeerId: wrapperspb.String(peerIDStr),
	}

	// Add messages if any
	if len(payload.Messages) > 0 {
		meta.Messages = createExpectedMessages()
	} else {
		meta.Messages = []*libp2p.MessageMeta{}
	}

	// Add subscriptions if any
	if len(payload.Subscriptions) > 0 {
		meta.Subscriptions = createExpectedSubscriptions()
	}

	// Add control if any
	if payload.Control != nil {
		meta.Control = &libp2p.ControlMeta{}

		// Add IHave if any
		if len(payload.Control.IHave) > 0 {
			meta.Control.Ihave = []*libp2p.ControlIHaveMeta{
				{
					TopicId:    wrapperspb.String("topic1"),
					MessageIds: convertStringValues([]string{"msg1", "msg2"}),
				},
			}
		}

		// Add IWant if any
		if len(payload.Control.IWant) > 0 {
			meta.Control.Iwant = []*libp2p.ControlIWantMeta{
				{
					MessageIds: convertStringValues([]string{"msg1", "msg2"}),
				},
			}

			// Special case for all controls test
			if len(payload.Control.IHave) > 0 &&
				len(payload.Control.Graft) > 0 &&
				len(payload.Control.Prune) > 0 {
				meta.Control.Iwant = []*libp2p.ControlIWantMeta{
					{
						MessageIds: convertStringValues([]string{"msg3", "msg4"}),
					},
				}
			}
		}

		// Add IDontWant if any
		if len(payload.Control.Idontwant) > 0 {
			meta.Control.Idontwant = []*libp2p.ControlIDontWantMeta{
				{
					MessageIds: convertStringValues([]string{"msg1", "msg2"}),
				},
			}

			// Special case for all controls test
			if len(payload.Control.IHave) > 0 &&
				len(payload.Control.IWant) > 0 &&
				len(payload.Control.Graft) > 0 {
				meta.Control.Idontwant = []*libp2p.ControlIDontWantMeta{
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

			meta.Control.Graft = []*libp2p.ControlGraftMeta{
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

			meta.Control.Prune = []*libp2p.ControlPruneMeta{
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

	var expectedMeta, actualMeta *libp2p.RPCMeta

	switch e := expected.(type) {
	case *libp2p.RecvRPC:
		expectedPeerID = e.PeerId.GetValue()
		expectedMeta = e.Meta

		a, ok := actual.(*libp2p.RecvRPC)
		require.True(t, ok)

		actualPeerID = a.PeerId.GetValue()
		actualMeta = a.Meta
	case *libp2p.SendRPC:
		expectedPeerID = e.PeerId.GetValue()
		expectedMeta = e.Meta

		a, ok := actual.(*libp2p.SendRPC)
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
func assertRPCMetaEquals(t *testing.T, expected, actual *libp2p.RPCMeta) {
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

func createTraceEventWithMessages(peerID peer.ID) *TraceEvent {
	return &TraceEvent{
		PeerID: peerID,
		Payload: &RpcMeta{
			PeerID: peerID,
			Messages: []RpcMetaMsg{
				{
					MsgID: "msg1",
					Topic: "topic1",
				},
			},
			Subscriptions: []RpcMetaSub{
				{
					Subscribe: true,
					TopicID:   "topic1",
				},
			},
		},
	}
}

func createTraceEventWithIHave(peerID peer.ID) *TraceEvent {
	return &TraceEvent{
		PeerID: peerID,
		Payload: &RpcMeta{
			PeerID: peerID,
			Control: &RpcMetaControl{
				IHave: []RpcControlIHave{
					{
						TopicID: "topic1",
						MsgIDs:  []string{"msg1", "msg2"},
					},
				},
			},
		},
	}
}

func createTraceEventWithIWant(peerID peer.ID) *TraceEvent {
	return &TraceEvent{
		PeerID: peerID,
		Payload: &RpcMeta{
			PeerID: peerID,
			Control: &RpcMetaControl{
				IWant: []RpcControlIWant{
					{
						MsgIDs: []string{"msg1", "msg2"},
					},
				},
			},
		},
	}
}

func createTraceEventWithIDontWant(peerID peer.ID) *TraceEvent {
	return &TraceEvent{
		PeerID: peerID,
		Payload: &RpcMeta{
			PeerID: peerID,
			Control: &RpcMetaControl{
				Idontwant: []RpcControlIdontWant{
					{
						MsgIDs: []string{"msg1", "msg2"},
					},
				},
			},
		},
	}
}

func createTraceEventWithGraft(peerID peer.ID) *TraceEvent {
	return &TraceEvent{
		PeerID: peerID,
		Payload: &RpcMeta{
			PeerID: peerID,
			Control: &RpcMetaControl{
				Graft: []RpcControlGraft{
					{
						TopicID: "topic1",
					},
				},
			},
		},
	}
}

func createTraceEventWithPrune(peerID peer.ID) *TraceEvent {
	return &TraceEvent{
		PeerID: peerID,
		Payload: &RpcMeta{
			PeerID: peerID,
			Control: &RpcMetaControl{
				Prune: []RpcControlPrune{
					{
						TopicID: "topic1",
						PeerIDs: []peer.ID{peerID},
					},
				},
			},
		},
	}
}

func createTraceEventWithAllControls(peerID peer.ID) *TraceEvent {
	return &TraceEvent{
		PeerID: peerID,
		Payload: &RpcMeta{
			PeerID: peerID,
			Control: &RpcMetaControl{
				IHave: []RpcControlIHave{
					{
						TopicID: "topic1",
						MsgIDs:  []string{"msg1", "msg2"},
					},
				},
				IWant: []RpcControlIWant{
					{
						MsgIDs: []string{"msg3", "msg4"},
					},
				},
				Idontwant: []RpcControlIdontWant{
					{
						MsgIDs: []string{"msg5", "msg6"},
					},
				},
				Graft: []RpcControlGraft{
					{
						TopicID: "topic2",
					},
				},
				Prune: []RpcControlPrune{
					{
						TopicID: "topic3",
						PeerIDs: []peer.ID{peerID},
					},
				},
			},
		},
	}
}

func createTraceEventWithNilControl(peerID peer.ID) *TraceEvent {
	return &TraceEvent{
		PeerID: peerID,
		Payload: &RpcMeta{
			PeerID:   peerID,
			Control:  nil,
			Messages: []RpcMetaMsg{},
		},
	}
}

func createExpectedMessages() []*libp2p.MessageMeta {
	return []*libp2p.MessageMeta{
		{
			MessageId: wrapperspb.String("msg1"),
			TopicId:   wrapperspb.String("topic1"),
		},
	}
}

func createExpectedSubscriptions() []*libp2p.SubMeta {
	return []*libp2p.SubMeta{
		{
			Subscribe: wrapperspb.Bool(true),
			TopicId:   wrapperspb.String("topic1"),
		},
	}
}

func assertControlIHaveEquals(t *testing.T, expected, actual []*libp2p.ControlIHaveMeta) {
	t.Helper()

	if len(expected) > 0 {
		require.Equal(t, len(expected), len(actual))

		for i, exp := range expected {
			assert.Equal(t, exp.TopicId.GetValue(), actual[i].TopicId.GetValue())
			assertStringValuesEqual(t, exp.MessageIds, actual[i].MessageIds)
		}
	}
}

func assertControlIWantEquals(t *testing.T, expected, actual []*libp2p.ControlIWantMeta) {
	t.Helper()

	if len(expected) > 0 {
		require.Equal(t, len(expected), len(actual))

		for i, exp := range expected {
			assertStringValuesEqual(t, exp.MessageIds, actual[i].MessageIds)
		}
	}
}

func assertControlIDontWantEquals(t *testing.T, expected, actual []*libp2p.ControlIDontWantMeta) {
	t.Helper()

	if len(expected) > 0 {
		require.Equal(t, len(expected), len(actual))

		for i, exp := range expected {
			assertStringValuesEqual(t, exp.MessageIds, actual[i].MessageIds)
		}
	}
}

func assertControlGraftEquals(t *testing.T, expected, actual []*libp2p.ControlGraftMeta) {
	t.Helper()

	if len(expected) > 0 {
		require.Equal(t, len(expected), len(actual))

		for i, exp := range expected {
			assert.Equal(t, exp.TopicId.GetValue(), actual[i].TopicId.GetValue())
		}
	}
}

func assertControlPruneEquals(t *testing.T, expected, actual []*libp2p.ControlPruneMeta) {
	t.Helper()

	if len(expected) > 0 {
		require.Equal(t, len(expected), len(actual))

		for i, exp := range expected {
			assert.Equal(t, exp.TopicId.GetValue(), actual[i].TopicId.GetValue())
			assertStringValuesEqual(t, exp.PeerIds, actual[i].PeerIds)
		}
	}
}

func assertMessagesEquals(t *testing.T, expected, actual []*libp2p.MessageMeta) {
	t.Helper()

	if len(expected) > 0 {
		require.Equal(t, len(expected), len(actual))

		for i, exp := range expected {
			assert.Equal(t, exp.MessageId.GetValue(), actual[i].MessageId.GetValue())
			assert.Equal(t, exp.TopicId.GetValue(), actual[i].TopicId.GetValue())
		}
	}
}

func assertSubscriptionsEquals(t *testing.T, expected, actual []*libp2p.SubMeta) {
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

func TestTraceEventToCustodyProbe(t *testing.T) {
	peerID, err := peer.Decode(peerIDStr)
	require.NoError(t, err)

	tests := []struct {
		name        string
		input       *TraceEvent
		expectError bool
		validate    func(t *testing.T, result *libp2p.DataColumnCustodyProbe)
	}{
		{
			name: "typed TraceEventCustodyProbe payload",
			input: &TraceEvent{
				PeerID: peerID,
				Payload: &TraceEventCustodyProbe{
					PeerID:     &peerID,
					Epoch:      100,
					Slot:       3200,
					Column:     42,
					BlockHash:  "0xabcdef1234567890",
					Result:     "success",
					Duration:   1500000000, // 1.5 seconds in nanoseconds
					ColumnSize: 128,
					Error:      "",
				},
			},
			validate: func(t *testing.T, result *libp2p.DataColumnCustodyProbe) {
				assert.Equal(t, peerIDStr, result.PeerId.GetValue())
				assert.Equal(t, uint32(100), result.Epoch.GetValue())
				assert.Equal(t, uint32(3200), result.Slot.GetValue())
				assert.Equal(t, uint32(42), result.ColumnIndex.GetValue())
				assert.Equal(t, "0xabcdef1234567890", result.BeaconBlockRoot.GetValue())
				assert.Equal(t, "success", result.Result.GetValue())
				assert.Equal(t, int64(1500), result.ResponseTimeMs.GetValue())
				assert.Equal(t, uint32(128), result.ColumnRowsCount.GetValue())
				assert.Nil(t, result.Error)
			},
		},
		{
			name: "typed TraceEventCustodyProbe payload with error",
			input: &TraceEvent{
				PeerID: peerID,
				Payload: &TraceEventCustodyProbe{
					PeerID:     &peerID,
					Epoch:      100,
					Slot:       3200,
					Column:     42,
					BlockHash:  "0xabcdef1234567890",
					Result:     "failure",
					Duration:   500000000,
					ColumnSize: 0,
					Error:      "timeout",
				},
			},
			validate: func(t *testing.T, result *libp2p.DataColumnCustodyProbe) {
				assert.Equal(t, peerIDStr, result.PeerId.GetValue())
				assert.Equal(t, "failure", result.Result.GetValue())
				assert.Equal(t, "timeout", result.Error.GetValue())
			},
		},
		{
			name: "map payload (backwards compatibility)",
			input: &TraceEvent{
				PeerID: peerID,
				Payload: map[string]any{
					"PeerID":      peerIDStr,
					"Epoch":       uint64(100),
					"Slot":        uint64(3200),
					"ColumnIndex": uint64(42),
					"BlockHash":   "0xabcdef1234567890",
					"Result":      "success",
					"DurationMs":  int64(1500),
					"ColumnSize":  128,
				},
			},
			validate: func(t *testing.T, result *libp2p.DataColumnCustodyProbe) {
				assert.Equal(t, peerIDStr, result.PeerId.GetValue())
				assert.Equal(t, uint32(100), result.Epoch.GetValue())
				assert.Equal(t, uint32(3200), result.Slot.GetValue())
				assert.Equal(t, uint32(42), result.ColumnIndex.GetValue())
				assert.Equal(t, "0xabcdef1234567890", result.BeaconBlockRoot.GetValue())
				assert.Equal(t, "success", result.Result.GetValue())
				assert.Equal(t, int64(1500), result.ResponseTimeMs.GetValue())
				assert.Equal(t, uint32(128), result.ColumnRowsCount.GetValue())
			},
		},
		{
			name: "invalid payload type",
			input: &TraceEvent{
				PeerID:  peerID,
				Payload: "not a valid payload",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := TraceEventToCustodyProbe(tt.input)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "invalid payload type for CustodyProbe")

				return
			}

			require.NoError(t, err)
			require.NotNil(t, result)

			if tt.validate != nil {
				tt.validate(t, result)
			}
		})
	}
}
