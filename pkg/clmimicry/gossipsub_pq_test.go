package clmimicry

import (
	"context"
	"testing"
	"time"

	"github.com/OffchainLabs/go-bitfield"
	"github.com/OffchainLabs/prysm/v7/consensus-types/primitives"
	ethtypes "github.com/OffchainLabs/prysm/v7/proto/prysm/v1alpha1"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

func pqTestAttestationData(slot uint64) *ethtypes.AttestationData {
	return &ethtypes.AttestationData{
		Slot:            primitives.Slot(slot),
		CommitteeIndex:  0,
		BeaconBlockRoot: append([]byte{0xbb}, make([]byte, 31)...),
		Source:          &ethtypes.Checkpoint{Epoch: 8, Root: append([]byte{0x5c}, make([]byte, 31)...)},
		Target:          &ethtypes.Checkpoint{Epoch: 9, Root: append([]byte{0x7a}, make([]byte, 31)...)},
	}
}

func TestHandleHermesEventPQBlock(t *testing.T) {
	peerID, err := peer.Decode(examplePeerID)
	require.NoError(t, err)

	mimicry, mockSink := newPartialTestMimicry(t, EventConfig{GossipSubBeaconBlockEnabled: true})
	mimicry.processor.metaProvider = staticMetaProvider{}
	captured := expectSingleEvent(mockSink)

	topic := "/eth2/9ab4e0f1/beacon_block/ssz_snappy"
	require.NoError(t, mimicry.processor.HandleHermesEvent(context.Background(), &TraceEvent{
		Type:      TraceEvent_HANDLE_MESSAGE,
		Topic:     topic,
		PeerID:    peerID,
		Timestamp: time.Now(),
		Payload: NewPQBlockPayload([32]byte{0x22}, 1234, 61, &TraceEventPayloadMetaData{
			PeerID:  peerID.String(),
			Topic:   topic,
			MsgID:   "msg-block",
			MsgSize: 9000,
		}),
	}))

	assert.Equal(t, xatu.Event_LIBP2P_TRACE_GOSSIPSUB_BEACON_BLOCK, captured.GetEvent().GetName())

	block := captured.GetLibp2PTraceGossipsubBeaconBlock()
	assert.Equal(t, "0x2200000000000000000000000000000000000000000000000000000000000000", block.GetBlock().GetValue())
	assert.Equal(t, uint64(1234), block.GetSlot().GetValue())
	assert.Equal(t, uint64(61), block.GetProposerIndex().GetValue())

	extra := captured.GetMeta().GetClient().GetLibp2PTraceGossipsubBeaconBlock()
	assert.Equal(t, "msg-block", extra.GetMessageId().GetValue())
	assert.Equal(t, uint32(9000), extra.GetMessageSize().GetValue())
	assert.Equal(t, topic, extra.GetTopic().GetValue())
}

func TestHandleHermesEventSingleAttestationPQ(t *testing.T) {
	peerID, err := peer.Decode(examplePeerID)
	require.NoError(t, err)

	mimicry, mockSink := newPartialTestMimicry(t, EventConfig{GossipSubAttestationEnabled: true})
	mimicry.processor.metaProvider = staticMetaProvider{}
	captured := expectSingleEvent(mockSink)

	topic := "/eth2/9ab4e0f1/beacon_attestation_3/ssz_snappy"
	require.NoError(t, mimicry.processor.HandleHermesEvent(context.Background(), &TraceEvent{
		Type:      TraceEvent_HANDLE_MESSAGE,
		Topic:     topic,
		PeerID:    peerID,
		Timestamp: time.Now(),
		Payload: NewSingleAttestationPQPayload(3, 517, pqTestAttestationData(300), &TraceEventPayloadMetaData{
			PeerID:  peerID.String(),
			Topic:   topic,
			MsgID:   "msg-att",
			MsgSize: 2500,
		}),
	}))

	assert.Equal(t, xatu.Event_LIBP2P_TRACE_GOSSIPSUB_BEACON_ATTESTATION, captured.GetEvent().GetName())

	data := captured.GetLibp2PTraceGossipsubBeaconAttestation().GetData()
	assert.Equal(t, uint64(300), data.GetSlot())
	assert.Equal(t, "0xbb00000000000000000000000000000000000000000000000000000000000000", data.GetBeaconBlockRoot())
	assert.Equal(t, uint64(8), data.GetSource().GetEpoch())
	assert.Equal(t, uint64(9), data.GetTarget().GetEpoch())

	extra := captured.GetMeta().GetClient().GetLibp2PTraceGossipsubBeaconAttestation()
	assert.Equal(t, uint64(3), extra.GetAttestingValidator().GetCommitteeIndex().GetValue())
	assert.Equal(t, uint64(517), extra.GetAttestingValidator().GetIndex().GetValue())
	assert.Equal(t, uint64(300), extra.GetSlot().GetNumber().GetValue())
	assert.Equal(t, "msg-att", extra.GetMessageId().GetValue())
	assert.Equal(t, uint32(2500), extra.GetMessageSize().GetValue())
}

func TestHandleHermesEventAggregateAndProofPQ(t *testing.T) {
	peerID, err := peer.Decode(examplePeerID)
	require.NoError(t, err)

	mimicry, mockSink := newPartialTestMimicry(t, EventConfig{GossipSubAggregateAndProofEnabled: true})
	mimicry.processor.metaProvider = staticMetaProvider{}
	captured := expectSingleEvent(mockSink)

	committeeBits := bitfield.NewBitvector16()
	committeeBits.SetBitAt(11, true)

	aggregationBits := bitfield.NewBitlist(10)
	aggregationBits.SetBitAt(0, true)
	aggregationBits.SetBitAt(9, true)

	topic := "/eth2/9ab4e0f1/beacon_aggregate_and_proof/ssz_snappy"
	require.NoError(t, mimicry.processor.HandleHermesEvent(context.Background(), &TraceEvent{
		Type:      TraceEvent_HANDLE_MESSAGE,
		Topic:     topic,
		PeerID:    peerID,
		Timestamp: time.Now(),
		Payload: NewSignedAggregateAttestationAndProofPQPayload(88, aggregationBits, committeeBits, pqTestAttestationData(301), &TraceEventPayloadMetaData{
			PeerID:  peerID.String(),
			Topic:   topic,
			MsgID:   "msg-agg",
			MsgSize: 40000,
		}),
	}))

	assert.Equal(t, xatu.Event_LIBP2P_TRACE_GOSSIPSUB_AGGREGATE_AND_PROOF, captured.GetEvent().GetName())

	message := captured.GetLibp2PTraceGossipsubAggregateAndProof().GetMessage()
	assert.Equal(t, uint64(88), message.GetAggregatorIndex().GetValue())
	assert.Equal(t, "0x0106", message.GetAggregate().GetAggregationBits())
	assert.Equal(t, uint64(301), message.GetAggregate().GetData().GetSlot().GetValue())
	assert.Equal(t, uint64(11), message.GetAggregate().GetData().GetIndex().GetValue())
	assert.Equal(t, "0xbb00000000000000000000000000000000000000000000000000000000000000", message.GetAggregate().GetData().GetBeaconBlockRoot())

	extra := captured.GetMeta().GetClient().GetLibp2PTraceGossipsubAggregateAndProof()
	assert.Equal(t, uint64(88), extra.GetAggregatorIndex().GetValue())
	assert.Equal(t, uint64(301), extra.GetSlot().GetNumber().GetValue())
	assert.Equal(t, "msg-agg", extra.GetMessageId().GetValue())
}
