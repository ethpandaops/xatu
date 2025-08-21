package clmimicry

import (
	"context"
	"testing"
	"time"

	"github.com/ethpandaops/xatu/pkg/output/mock"
	"github.com/ethpandaops/xatu/pkg/proto/libp2p"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/probe-lab/hermes/host"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

// TestDecoupledSharding tests that child events (like IHAVE) can be captured
// independently of their parent events (SendRPC/RecvRPC/DropRPC)
func TestDecoupledSharding(t *testing.T) {
	tests := []struct {
		name          string
		config        *Config
		parentEnabled bool
		childEnabled  bool
		expectParent  bool
		expectChild   bool
		description   string
	}{
		{
			name: "parent_disabled_child_enabled",
			config: &Config{
				Events: EventConfig{
					RecvRPCEnabled:             false, // Parent disabled
					RpcMetaControlIHaveEnabled: true,  // Child enabled
				},
			},
			parentEnabled: false,
			childEnabled:  true,
			expectParent:  false, // Parent should not be sent
			expectChild:   true,  // Child should be sent
			description:   "Parent disabled but child IHAVE events should still be captured",
		},
		{
			name: "parent_enabled_child_enabled",
			config: &Config{
				Events: EventConfig{
					RecvRPCEnabled:             true, // Parent enabled
					RpcMetaControlIHaveEnabled: true, // Child enabled
				},
			},
			parentEnabled: true,
			childEnabled:  true,
			expectParent:  true, // Parent should be sent
			expectChild:   true, // Child should be sent
			description:   "Both parent and child enabled and should be captured",
		},
		{
			name: "parent_enabled_child_disabled",
			config: &Config{
				Events: EventConfig{
					RecvRPCEnabled:             true,  // Parent enabled
					RpcMetaControlIHaveEnabled: false, // Child disabled
				},
			},
			parentEnabled: true,
			childEnabled:  false,
			expectParent:  false, // Parent should not be sent when no child events are produced
			expectChild:   false, // Child should not be sent
			description:   "Parent enabled but child disabled",
		},
		{
			name: "parent_disabled_child_disabled",
			config: &Config{
				Events: EventConfig{
					RecvRPCEnabled:             false, // Parent disabled
					RpcMetaControlIHaveEnabled: false, // Child disabled
				},
			},
			parentEnabled: false,
			childEnabled:  false,
			expectParent:  false, // Parent disabled
			expectChild:   false, // Child disabled
			description:   "Both parent and child disabled",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockSink := mock.NewMockSink(ctrl)

			// Setup expectations
			expectedEventCount := 0
			if tt.expectParent {
				expectedEventCount++
			}
			if tt.expectChild {
				expectedEventCount++
			}

			if expectedEventCount > 0 {
				mockSink.EXPECT().HandleNewDecoratedEvents(gomock.Any(), gomock.Any()).DoAndReturn(
					func(ctx context.Context, events []*xatu.DecoratedEvent) error {
						// Verify we got the expected events
						hasParent := false
						hasChild := false

						for _, event := range events {
							switch event.Event.Name {
							case xatu.Event_LIBP2P_TRACE_RECV_RPC:
								hasParent = true
							case xatu.Event_LIBP2P_TRACE_RPC_META_CONTROL_IHAVE:
								hasChild = true
							}
						}

						assert.Equal(t, tt.expectParent, hasParent, "Parent event presence mismatch")
						assert.Equal(t, tt.expectChild, hasChild, "Child event presence mismatch")

						return nil
					},
				).Times(1)
			}

			// Create mimicry with test configuration
			mimicry := createTestMimicry(t, tt.config, mockSink)

			// Create test event with IHAVE control message
			peerID, err := peer.Decode(examplePeerID)
			require.NoError(t, err)

			event := &host.TraceEvent{
				Type:      "RecvRPC",
				PeerID:    peerID,
				Timestamp: time.Now(),
				Payload: &host.RpcMeta{
					PeerID: peerID,
					Control: &host.RpcMetaControl{
						IHave: []host.RpcControlIHave{
							{
								TopicID: "/eth2/test/beacon_attestation_0/ssz_snappy",
								MsgIDs: []string{
									"test_msg",
								},
							},
						},
					},
				},
			}

			// Handle the event
			clientMeta := createTestClientMeta()
			traceMeta := &libp2p.TraceEventMetadata{
				PeerId: wrapperspb.String(examplePeerID),
			}

			// Call the new handleRecvRPCEvent with the additional parameters
			err = mimicry.GetProcessor().handleRecvRPCEvent(
				context.Background(),
				clientMeta,
				traceMeta,
				event,
				xatu.Event_LIBP2P_TRACE_RECV_RPC.String(),
				"1", // network ID
			)

			// Should not error
			assert.NoError(t, err, tt.description)
		})
	}
}

// TestDecoupledShardingSendRPC tests decoupled sharding for SendRPC events
func TestDecoupledShardingSendRPC(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSink := mock.NewMockSink(ctrl)

	// Configure to disable parent but enable child
	config := &Config{
		Events: EventConfig{
			SendRPCEnabled:             false, // Parent disabled
			RpcMetaControlIHaveEnabled: true,  // Child enabled
		},
	}

	// Expect only child event, not parent
	mockSink.EXPECT().HandleNewDecoratedEvents(gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, events []*xatu.DecoratedEvent) error {
			// Should only have IHAVE event, not SEND_RPC
			assert.Len(t, events, 1)
			assert.Equal(t, xatu.Event_LIBP2P_TRACE_RPC_META_CONTROL_IHAVE, events[0].Event.Name)

			return nil
		},
	).Times(1)

	mimicry := createTestMimicry(t, config, mockSink)

	// Create test event
	peerID, err := peer.Decode(examplePeerID)
	require.NoError(t, err)

	event := &host.TraceEvent{
		Type:      "SendRPC",
		PeerID:    peerID,
		Timestamp: time.Now(),
		Payload: &host.RpcMeta{
			PeerID: peerID,
			Control: &host.RpcMetaControl{
				IHave: []host.RpcControlIHave{
					{
						TopicID: "/eth2/test/beacon_attestation_0/ssz_snappy",
						MsgIDs:  []string{"test_msg"},
					},
				},
			},
		},
	}

	clientMeta := createTestClientMeta()
	traceMeta := &libp2p.TraceEventMetadata{
		PeerId: wrapperspb.String(examplePeerID),
	}

	err = mimicry.GetProcessor().handleSendRPCEvent(
		context.Background(),
		clientMeta,
		traceMeta,
		event,
		xatu.Event_LIBP2P_TRACE_SEND_RPC.String(),
		"1",
	)

	assert.NoError(t, err)
}

// TestDecoupledShardingDropRPC tests decoupled sharding for DropRPC events
func TestDecoupledShardingDropRPC(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSink := mock.NewMockSink(ctrl)

	// Configure to disable parent but enable child
	config := &Config{
		Events: EventConfig{
			DropRPCEnabled:             false, // Parent disabled
			RpcMetaControlIHaveEnabled: true,  // Child enabled
		},
	}

	// Expect only child event, not parent
	mockSink.EXPECT().HandleNewDecoratedEvents(gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, events []*xatu.DecoratedEvent) error {
			// Should only have IHAVE event, not DROP_RPC
			assert.Len(t, events, 1)
			assert.Equal(t, xatu.Event_LIBP2P_TRACE_RPC_META_CONTROL_IHAVE, events[0].Event.Name)

			return nil
		},
	).Times(1)

	mimicry := createTestMimicry(t, config, mockSink)

	// Create test event
	peerID, err := peer.Decode(examplePeerID)
	require.NoError(t, err)

	event := &host.TraceEvent{
		Type:      "DropRPC",
		PeerID:    peerID,
		Timestamp: time.Now(),
		Payload: &host.RpcMeta{
			PeerID: peerID,
			Control: &host.RpcMetaControl{
				IHave: []host.RpcControlIHave{
					{
						TopicID: "/eth2/test/beacon_attestation_0/ssz_snappy",
						MsgIDs:  []string{"test_msg"},
					},
				},
			},
		},
	}

	clientMeta := createTestClientMeta()
	traceMeta := &libp2p.TraceEventMetadata{
		PeerId: wrapperspb.String(examplePeerID),
	}

	err = mimicry.GetProcessor().handleDropRPCEvent(
		context.Background(),
		clientMeta,
		traceMeta,
		event,
		xatu.Event_LIBP2P_TRACE_DROP_RPC.String(),
		"1",
	)

	assert.NoError(t, err)
}

// TestDecoupledShardingComplexScenario tests a complex scenario with multiple child event types
func TestDecoupledShardingComplexScenario(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSink := mock.NewMockSink(ctrl)

	// Configure with parent disabled but multiple child types enabled
	config := &Config{
		Events: EventConfig{
			RecvRPCEnabled:                 false, // Parent disabled
			RpcMetaControlIHaveEnabled:     true,  // Child enabled
			RpcMetaControlIWantEnabled:     true,  // Child enabled
			RpcMetaControlIDontWantEnabled: true,  // Child enabled
			RpcMetaControlGraftEnabled:     true,  // Child enabled
			RpcMetaControlPruneEnabled:     true,  // Child enabled
		},
	}

	// Expect multiple child events but no parent
	mockSink.EXPECT().HandleNewDecoratedEvents(gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, events []*xatu.DecoratedEvent) error {
			// Should have multiple child events but no RECV_RPC
			hasParent := false
			hasIHave := false
			hasIWant := false
			hasIDontWant := false
			hasGraft := false
			hasPrune := false

			for _, event := range events {
				switch event.Event.Name {
				case xatu.Event_LIBP2P_TRACE_RECV_RPC:
					hasParent = true
				case xatu.Event_LIBP2P_TRACE_RPC_META_CONTROL_IHAVE:
					hasIHave = true
				case xatu.Event_LIBP2P_TRACE_RPC_META_CONTROL_IWANT:
					hasIWant = true
				case xatu.Event_LIBP2P_TRACE_RPC_META_CONTROL_IDONTWANT:
					hasIDontWant = true
				case xatu.Event_LIBP2P_TRACE_RPC_META_CONTROL_GRAFT:
					hasGraft = true
				case xatu.Event_LIBP2P_TRACE_RPC_META_CONTROL_PRUNE:
					hasPrune = true
				}
			}

			assert.False(t, hasParent, "Should not have parent event")
			assert.True(t, hasIHave, "Should have IHAVE event")
			assert.True(t, hasIWant, "Should have IWANT event")
			assert.True(t, hasIDontWant, "Should have IDONTWANT event")
			assert.True(t, hasGraft, "Should have GRAFT event")
			assert.True(t, hasPrune, "Should have PRUNE event")

			return nil
		},
	).Times(1)

	mimicry := createTestMimicry(t, config, mockSink)

	// Create test event with multiple control messages
	peerID, err := peer.Decode(examplePeerID)
	require.NoError(t, err)

	event := &host.TraceEvent{
		Type:      "RecvRPC",
		PeerID:    peerID,
		Timestamp: time.Now(),
		Payload: &host.RpcMeta{
			PeerID: peerID,
			Control: &host.RpcMetaControl{
				IHave: []host.RpcControlIHave{
					{
						TopicID: "/eth2/test/beacon_attestation_0/ssz_snappy",
						MsgIDs:  []string{"test_msg"},
					},
				},
				IWant: []host.RpcControlIWant{
					{
						MsgIDs: []string{"test_msg"},
					},
				},
				Idontwant: []host.RpcControlIdontWant{
					{
						MsgIDs: []string{"test_msg"},
					},
				},
				Graft: []host.RpcControlGraft{
					{
						TopicID: "/eth2/test/beacon_attestation_0/ssz_snappy",
					},
				},
				Prune: []host.RpcControlPrune{
					{
						TopicID: "/eth2/test/beacon_attestation_0/ssz_snappy",
					},
				},
			},
		},
	}

	clientMeta := createTestClientMeta()
	traceMeta := &libp2p.TraceEventMetadata{
		PeerId: wrapperspb.String(examplePeerID),
	}

	err = mimicry.GetProcessor().handleRecvRPCEvent(
		context.Background(),
		clientMeta,
		traceMeta,
		event,
		xatu.Event_LIBP2P_TRACE_RECV_RPC.String(),
		"1",
	)

	assert.NoError(t, err)
}
