package clmimicry

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/OffchainLabs/go-bitfield"
	"github.com/OffchainLabs/prysm/v7/consensus-types/primitives"
	enginev1 "github.com/OffchainLabs/prysm/v7/proto/engine/v1"
	ethtypes "github.com/OffchainLabs/prysm/v7/proto/prysm/v1alpha1"
	"github.com/google/uuid"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/ethpandaops/xatu/pkg/output/mock"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

func TestGloasGossipHandlers(t *testing.T) {
	peerID, err := peer.Decode(examplePeerID)
	require.NoError(t, err)

	meta := TraceEventPayloadMetaData{
		MsgID:   uuid.New().String(),
		PeerID:  peerID.String(),
		MsgSize: 512,
	}

	t.Run("beacon_block", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockSink := mock.NewMockSink(ctrl)
		mimicry := createTestMimicryWithWallclock(t, &Config{
			Events: EventConfig{GossipSubBeaconBlockEnabled: true},
		}, mockSink)

		block := gloasTestBlock(123, 42)

		root, rootErr := block.GetBlock().HashTreeRoot()
		require.NoError(t, rootErr)

		mockSink.EXPECT().
			HandleNewDecoratedEvent(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, event *xatu.DecoratedEvent) error {
				assert.Equal(t, xatu.Event_LIBP2P_TRACE_GOSSIPSUB_BEACON_BLOCK, event.GetEvent().GetName())
				data := event.GetLibp2PTraceGossipsubBeaconBlock()
				require.NotNil(t, data)
				assert.Equal(t, uint64(123), data.GetSlot().GetValue())
				assert.Equal(t, uint64(42), data.GetProposerIndex().GetValue())
				assert.Equal(t, fmt.Sprintf("0x%x", root), data.GetBlock().GetValue())

				return nil
			}).
			Times(1)

		blockMeta := meta
		blockMeta.Topic = "/eth2/c4a5f0e2/beacon_block/ssz_snappy"

		err = mimicry.processor.handleGossipBeaconBlock(
			context.Background(),
			createTestClientMeta(),
			&TraceEvent{Type: TraceEvent_HANDLE_MESSAGE, PeerID: peerID, Timestamp: time.Now()},
			NewGloasBlockPayload(block, &blockMeta),
		)
		assert.NoError(t, err)
	})

	t.Run("aggregate_and_proof", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockSink := mock.NewMockSink(ctrl)
		mimicry := createTestMimicryWithWallclock(t, &Config{
			Events: EventConfig{GossipSubAggregateAndProofEnabled: true},
		}, mockSink)

		committeeBits := bitfield.NewBitvector64()
		committeeBits.SetBitAt(5, true)

		agg := &ethtypes.SignedAggregateAttestationAndProofGloas{
			Message: &ethtypes.AggregateAttestationAndProofGloas{
				AggregatorIndex: primitives.ValidatorIndex(77),
				Aggregate: &ethtypes.AttestationGloas{
					AggregationBits: bitfield.NewBitlist(8),
					Data: &ethtypes.AttestationData{
						Slot:            primitives.Slot(456),
						BeaconBlockRoot: make([]byte, 32),
						Source:          &ethtypes.Checkpoint{Root: make([]byte, 32)},
						Target:          &ethtypes.Checkpoint{Root: make([]byte, 32)},
					},
					Signature:     make([]byte, 96),
					CommitteeBits: committeeBits,
				},
				SelectionProof: make([]byte, 96),
			},
			Signature: make([]byte, 96),
		}

		mockSink.EXPECT().
			HandleNewDecoratedEvent(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, event *xatu.DecoratedEvent) error {
				assert.Equal(t, xatu.Event_LIBP2P_TRACE_GOSSIPSUB_AGGREGATE_AND_PROOF, event.GetEvent().GetName())
				data := event.GetLibp2PTraceGossipsubAggregateAndProof()
				require.NotNil(t, data)
				assert.Equal(t, uint64(77), data.GetMessage().GetAggregatorIndex().GetValue())
				assert.Equal(t, uint64(456), data.GetMessage().GetAggregate().GetData().GetSlot().GetValue())
				assert.Equal(t, uint64(5), data.GetMessage().GetAggregate().GetData().GetIndex().GetValue())

				return nil
			}).
			Times(1)

		aggMeta := meta
		aggMeta.Topic = "/eth2/c4a5f0e2/beacon_aggregate_and_proof/ssz_snappy"

		err = mimicry.processor.handleGossipAggregateAndProof(
			context.Background(),
			createTestClientMeta(),
			&TraceEvent{Type: TraceEvent_HANDLE_MESSAGE, PeerID: peerID, Timestamp: time.Now()},
			NewSignedAggregateAttestationAndProofGloasPayload(agg, &aggMeta),
		)
		assert.NoError(t, err)
	})
}

func gloasTestBlock(slot primitives.Slot, proposer primitives.ValidatorIndex) *ethtypes.SignedBeaconBlockGloas {
	return &ethtypes.SignedBeaconBlockGloas{
		Block: &ethtypes.BeaconBlockGloas{
			Slot:          slot,
			ProposerIndex: proposer,
			ParentRoot:    make([]byte, 32),
			StateRoot:     make([]byte, 32),
			Body: &ethtypes.BeaconBlockBodyGloas{
				RandaoReveal: make([]byte, 96),
				Eth1Data: &ethtypes.Eth1Data{
					DepositRoot: make([]byte, 32),
					BlockHash:   make([]byte, 32),
				},
				Graffiti: make([]byte, 32),
				SyncAggregate: &ethtypes.SyncAggregate{
					SyncCommitteeBits:      make([]byte, 64),
					SyncCommitteeSignature: make([]byte, 96),
				},
				SignedExecutionPayloadBid: &ethtypes.SignedExecutionPayloadBid{
					Message: &ethtypes.ExecutionPayloadBid{
						ParentBlockHash:       make([]byte, 32),
						ParentBlockRoot:       make([]byte, 32),
						BlockHash:             make([]byte, 32),
						PrevRandao:            make([]byte, 32),
						FeeRecipient:          make([]byte, 20),
						ExecutionRequestsRoot: make([]byte, 32),
					},
					Signature: make([]byte, 96),
				},
				ParentExecutionRequests: &enginev1.ExecutionRequestsGloas{},
			},
		},
		Signature: make([]byte, 96),
	}
}
