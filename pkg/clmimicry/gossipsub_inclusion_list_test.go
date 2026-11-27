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

func TestHandleHermesEventInclusionList(t *testing.T) {
	peerID, err := peer.Decode(examplePeerID)
	require.NoError(t, err)

	mimicry, mockSink := newPartialTestMimicry(t, EventConfig{GossipSubInclusionListEnabled: true})
	mimicry.processor.metaProvider = staticMetaProvider{}
	captured := expectSingleEvent(mockSink)

	slot := uint64(300)
	slotWindow := mimicry.processor.wallclock.Slots().FromNumber(slot)
	timestamp := slotWindow.TimeWindow().Start().Add(2500 * time.Millisecond)
	dependentRoot := make([]byte, 32)
	dependentRoot[31] = 0xab

	topic := "/eth2/4a26c58b/inclusion_list/ssz_snappy"
	require.NoError(t, mimicry.processor.HandleHermesEvent(context.Background(), &TraceEvent{
		Type:      TraceEvent_HANDLE_MESSAGE,
		Topic:     topic,
		PeerID:    peerID,
		Timestamp: timestamp,
		Payload: NewInclusionListPayload(slot, 42, dependentRoot, [][]byte{[]byte("abc"), {}}, &TraceEventPayloadMetaData{
			PeerID:  peerID.String(),
			Topic:   topic,
			MsgID:   "msg-1",
			MsgSize: 120,
		}),
	}))

	assert.Equal(t, xatu.Event_LIBP2P_TRACE_GOSSIPSUB_INCLUSION_LIST, captured.GetEvent().GetName())

	il := captured.GetLibp2PTraceGossipsubInclusionList()
	assert.Equal(t, slot, il.GetSlot().GetValue())
	assert.Equal(t, uint64(42), il.GetValidatorIndex().GetValue())
	assert.Equal(t, "0x00000000000000000000000000000000000000000000000000000000000000ab", il.GetDependentRoot().GetValue())
	assert.Equal(t, uint32(2), il.GetTransactionsCount().GetValue())
	assert.Equal(t, uint64(3), il.GetTransactionsTotalBytes().GetValue())
	assert.Equal(t, []string{
		"0x4e03657aea45a94fc7d47ba826c8d667c0d1e6e33a64a036ec44f58fa12d6c45",
		"0xc5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470",
	}, il.GetTransactionHashes())

	extra := captured.GetMeta().GetClient().GetLibp2PTraceGossipsubInclusionList()
	assert.Equal(t, slot, extra.GetSlot().GetNumber().GetValue())
	assert.Equal(t, uint64(2500), extra.GetPropagation().GetSlotStartDiff().GetValue())
	assert.Equal(t, "msg-1", extra.GetMessageId().GetValue())
	assert.Equal(t, uint32(120), extra.GetMessageSize().GetValue())
	assert.Equal(t, peerID.String(), extra.GetMetadata().GetPeerId().GetValue())
}

func TestHandleHermesEventHezeBlock(t *testing.T) {
	peerID, err := peer.Decode(examplePeerID)
	require.NoError(t, err)

	mimicry, mockSink := newPartialTestMimicry(t, EventConfig{GossipSubBeaconBlockEnabled: true})
	mimicry.processor.metaProvider = staticMetaProvider{}
	captured := expectSingleEvent(mockSink)

	root := [32]byte{0x11}

	topic := "/eth2/4a26c58b/beacon_block/ssz_snappy"
	require.NoError(t, mimicry.processor.HandleHermesEvent(context.Background(), &TraceEvent{
		Type:      TraceEvent_HANDLE_MESSAGE,
		Topic:     topic,
		PeerID:    peerID,
		Timestamp: time.Now(),
		Payload: NewHezeBlockPayload(root, 263000, 77, &TraceEventPayloadMetaData{
			PeerID: peerID.String(),
			Topic:  topic,
			MsgID:  "msg-2",
		}),
	}))

	assert.Equal(t, xatu.Event_LIBP2P_TRACE_GOSSIPSUB_BEACON_BLOCK, captured.GetEvent().GetName())

	block := captured.GetLibp2PTraceGossipsubBeaconBlock()
	assert.Equal(t, "0x1100000000000000000000000000000000000000000000000000000000000000", block.GetBlock().GetValue())
	assert.Equal(t, uint64(263000), block.GetSlot().GetValue())
	assert.Equal(t, uint64(77), block.GetProposerIndex().GetValue())
	assert.Equal(t, "msg-2", captured.GetMeta().GetClient().GetLibp2PTraceGossipsubBeaconBlock().GetMessageId().GetValue())
}
