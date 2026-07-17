package clmimicry

import (
	"context"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/ethpandaops/xatu/pkg/output/mock"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

func TestGossipsubMessagePayloadIntegration(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSink := mock.NewMockSink(ctrl)

	config := &Config{
		Events: EventConfig{
			GossipSubMessagePayloadEnabled: true,
		},
	}

	mimicry := createTestMimicryWithWallclock(t, config, mockSink)

	peerID, err := peer.Decode(examplePeerID)
	require.NoError(t, err)

	testData := []byte{0xff, 0x06, 0x00, 0x00, 0x73, 0x4e, 0x61, 0x50, 0x70, 0x59}
	testTopic := "/eth2/fc90fcde/voluntary_exit/ssz_snappy"
	testMsgID := "0xdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef"

	mockSink.EXPECT().
		HandleNewDecoratedEvent(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, event *xatu.DecoratedEvent) error {
			assert.Equal(t, xatu.Event_LIBP2P_TRACE_GOSSIPSUB_MESSAGE_PAYLOAD, event.GetEvent().GetName())

			payload := event.GetLibp2PTraceGossipsubMessagePayload()
			require.NotNil(t, payload)
			assert.Equal(t, testData, payload.GetData())
			assert.Equal(t, MessagePayloadOutcomeReject, payload.GetOutcome().GetValue())
			assert.Equal(t, "invalid signature", payload.GetRejectReason().GetValue())

			additional := event.GetMeta().GetClient().GetLibp2PTraceGossipsubMessagePayload()
			require.NotNil(t, additional)
			assert.Equal(t, testTopic, additional.GetTopic().GetValue())
			assert.Equal(t, testMsgID, additional.GetMessageId().GetValue())
			assert.Equal(t, uint32(len(testData)), additional.GetMessageSize().GetValue())

			return nil
		}).
		Times(1)

	event := &TraceEvent{
		Type:      TraceEvent_GOSSIPSUB_MESSAGE_PAYLOAD,
		PeerID:    peerID,
		Timestamp: time.Now(),
	}

	payload := &TraceEventMessagePayload{
		TraceEventPayloadMetaData: TraceEventPayloadMetaData{
			Topic:   testTopic,
			MsgID:   testMsgID,
			MsgSize: len(testData),
			PeerID:  peerID.String(),
		},
		Data:         testData,
		Outcome:      MessagePayloadOutcomeReject,
		RejectReason: "invalid signature",
	}

	err = mimicry.processor.handleGossipMessagePayload(
		context.Background(),
		createTestClientMeta(),
		event,
		payload,
	)
	assert.NoError(t, err)
}

func TestGossipsubMessagePayloadDisabled(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSink := mock.NewMockSink(ctrl)

	config := &Config{
		Events: EventConfig{
			GossipSubMessagePayloadEnabled: false,
		},
	}

	mimicry := createTestMimicryWithWallclock(t, config, mockSink)

	peerID, err := peer.Decode(examplePeerID)
	require.NoError(t, err)

	// No sink expectations: a disabled event must not reach the output.
	event := &TraceEvent{
		Type:      TraceEvent_GOSSIPSUB_MESSAGE_PAYLOAD,
		PeerID:    peerID,
		Timestamp: time.Now(),
		Payload: &TraceEventMessagePayload{
			Data:    []byte{0x01},
			Outcome: MessagePayloadOutcomeDeliver,
		},
	}

	err = mimicry.processor.handleGossipsubMessagePayloadEvent(
		context.Background(),
		event,
		createTestClientMeta(),
	)
	assert.NoError(t, err)
}

func TestGossipsubMessagePayloadInvalidPayloadType(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSink := mock.NewMockSink(ctrl)

	config := &Config{
		Events: EventConfig{
			GossipSubMessagePayloadEnabled: true,
		},
	}

	mimicry := createTestMimicryWithWallclock(t, config, mockSink)

	peerID, err := peer.Decode(examplePeerID)
	require.NoError(t, err)

	event := &TraceEvent{
		Type:      TraceEvent_GOSSIPSUB_MESSAGE_PAYLOAD,
		PeerID:    peerID,
		Timestamp: time.Now(),
		Payload:   "not a message payload",
	}

	err = mimicry.processor.handleGossipsubMessagePayloadEvent(
		context.Background(),
		event,
		createTestClientMeta(),
	)
	assert.Error(t, err)
}
