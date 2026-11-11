package clmimicry

import (
	"context"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

// TestHandleHermesEventGoodbye feeds the REQUEST_GOODBYE event in the shape tysm's patched
// Prysm emits: the remote peer in TraceEvent.PeerID and Code as a uint64 in the payload.
//
//nolint:goconst // payload keys match the tysm emitter's literal map keys
func TestHandleHermesEventGoodbye(t *testing.T) {
	remotePeer, err := peer.Decode(examplePeerID)
	require.NoError(t, err)

	mimicry, mockSink := newPartialTestMimicry(t, EventConfig{GoodbyeEnabled: true})
	mimicry.processor.metaProvider = staticMetaProvider{}
	captured := expectSingleEvent(mockSink)

	require.NoError(t, mimicry.processor.HandleHermesEvent(context.Background(), &TraceEvent{
		Type:      TraceEvent_REQUEST_GOODBYE,
		PeerID:    remotePeer,
		Timestamp: time.Now(),
		Payload: map[string]any{
			"Code":   uint64(129),
			"Reason": "client has too many peers",
			"Agent":  "Lighthouse/v8.0.1-e3ee7fe/x86_64-linux",
		},
	}))

	assert.Equal(t, xatu.Event_LIBP2P_TRACE_GOODBYE, captured.GetEvent().GetName())

	goodbye := captured.GetLibp2PTraceGoodbye()
	assert.Equal(t, remotePeer.String(), goodbye.GetRemotePeer().GetValue())
	assert.Equal(t, uint64(129), goodbye.GetCode().GetValue())
	assert.Equal(t, "client has too many peers", goodbye.GetReason().GetValue())
	assert.Equal(t, "Lighthouse/v8.0.1-e3ee7fe/x86_64-linux", goodbye.GetAgentVersion().GetValue())
}

func TestHandleHermesEventGoodbyeDisabled(t *testing.T) {
	remotePeer, err := peer.Decode(examplePeerID)
	require.NoError(t, err)

	// No sink expectation: a disabled event must not be emitted.
	mimicry, _ := newPartialTestMimicry(t, EventConfig{})
	mimicry.processor.metaProvider = staticMetaProvider{}

	require.NoError(t, mimicry.processor.HandleHermesEvent(context.Background(), &TraceEvent{
		Type:      TraceEvent_REQUEST_GOODBYE,
		PeerID:    remotePeer,
		Timestamp: time.Now(),
		Payload:   map[string]any{"Code": uint64(1)},
	}))
}
