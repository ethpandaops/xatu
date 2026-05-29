package clmimicry

import (
	"context"
	"testing"
	"time"

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

func TestEPBSHandlerIntegration(t *testing.T) {
	peerID, err := peer.Decode(examplePeerID)
	require.NoError(t, err)

	t.Run("execution_payload (envelope)", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockSink := mock.NewMockSink(ctrl)
		mimicry := createTestMimicryWithWallclock(t, &Config{
			Events: EventConfig{GossipSubExecutionPayloadEnvelopeEnabled: true},
		}, mockSink)

		mockSink.EXPECT().
			HandleNewDecoratedEvent(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, event *xatu.DecoratedEvent) error {
				assert.Equal(t, xatu.Event_LIBP2P_TRACE_GOSSIPSUB_EXECUTION_PAYLOAD_ENVELOPE, event.GetEvent().GetName())
				data := event.GetLibp2PTraceGossipsubExecutionPayloadEnvelope()
				require.NotNil(t, data)
				assert.Equal(t, uint64(123), data.GetSlot().GetValue())
				assert.Equal(t, uint64(7), data.GetBuilderIndex().GetValue())

				return nil
			}).
			Times(1)

		payload := &TraceEventExecutionPayloadEnvelope{
			TraceEventPayloadMetaData: TraceEventPayloadMetaData{
				Topic:  "/eth2/test/execution_payload/ssz_snappy",
				MsgID:  uuid.New().String(),
				PeerID: peerID.String(),
			},
			ExecutionPayloadEnvelope: &ethtypes.SignedExecutionPayloadEnvelope{
				Message: &ethtypes.ExecutionPayloadEnvelope{
					BuilderIndex:    primitives.BuilderIndex(7),
					BeaconBlockRoot: make([]byte, 32),
					Payload: &enginev1.ExecutionPayloadGloas{
						BlockHash:  make([]byte, 32),
						StateRoot:  make([]byte, 32),
						SlotNumber: primitives.Slot(123),
					},
				},
			},
		}

		err = mimicry.processor.handleGossipExecutionPayloadEnvelope(
			context.Background(),
			createTestClientMeta(),
			&TraceEvent{Type: TraceEvent_HANDLE_MESSAGE, PeerID: peerID, Timestamp: time.Now()},
			payload,
		)
		assert.NoError(t, err)
	})

	t.Run("execution_payload_bid", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockSink := mock.NewMockSink(ctrl)
		mimicry := createTestMimicryWithWallclock(t, &Config{
			Events: EventConfig{GossipSubExecutionPayloadBidEnabled: true},
		}, mockSink)

		mockSink.EXPECT().
			HandleNewDecoratedEvent(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, event *xatu.DecoratedEvent) error {
				assert.Equal(t, xatu.Event_LIBP2P_TRACE_GOSSIPSUB_EXECUTION_PAYLOAD_BID, event.GetEvent().GetName())
				data := event.GetLibp2PTraceGossipsubExecutionPayloadBid()
				require.NotNil(t, data)
				assert.Equal(t, uint64(456), data.GetSlot().GetValue())
				assert.Equal(t, uint64(9), data.GetBuilderIndex().GetValue())
				assert.Equal(t, uint64(1_000_000_000), data.GetValue().GetValue())
				assert.Equal(t, uint32(2), data.GetBlobKzgCommitmentCount().GetValue())

				return nil
			}).
			Times(1)

		payload := &TraceEventExecutionPayloadBid{
			TraceEventPayloadMetaData: TraceEventPayloadMetaData{
				Topic:  "/eth2/test/execution_payload_bid/ssz_snappy",
				MsgID:  uuid.New().String(),
				PeerID: peerID.String(),
			},
			ExecutionPayloadBid: &ethtypes.SignedExecutionPayloadBid{
				Message: &ethtypes.ExecutionPayloadBid{
					Slot:             primitives.Slot(456),
					BuilderIndex:     primitives.BuilderIndex(9),
					BlockHash:        make([]byte, 32),
					ParentBlockHash:  make([]byte, 32),
					Value:            primitives.Gwei(1_000_000_000),
					ExecutionPayment: primitives.Gwei(500_000_000),
					FeeRecipient:     make([]byte, 20),
					GasLimit:         30_000_000,
					BlobKzgCommitments: [][]byte{
						make([]byte, 48),
						make([]byte, 48),
					},
				},
			},
		}

		err = mimicry.processor.handleGossipExecutionPayloadBid(
			context.Background(),
			createTestClientMeta(),
			&TraceEvent{Type: TraceEvent_HANDLE_MESSAGE, PeerID: peerID, Timestamp: time.Now()},
			payload,
		)
		assert.NoError(t, err)
	})

	t.Run("payload_attestation_message", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockSink := mock.NewMockSink(ctrl)
		mimicry := createTestMimicryWithWallclock(t, &Config{
			Events: EventConfig{GossipSubPayloadAttestationMessageEnabled: true},
		}, mockSink)

		mockSink.EXPECT().
			HandleNewDecoratedEvent(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, event *xatu.DecoratedEvent) error {
				assert.Equal(t, xatu.Event_LIBP2P_TRACE_GOSSIPSUB_PAYLOAD_ATTESTATION_MESSAGE, event.GetEvent().GetName())
				data := event.GetLibp2PTraceGossipsubPayloadAttestationMessage()
				require.NotNil(t, data)
				assert.Equal(t, uint64(789), data.GetSlot().GetValue())
				assert.Equal(t, uint64(42), data.GetValidatorIndex().GetValue())
				assert.True(t, data.GetPayloadPresent().GetValue())
				assert.False(t, data.GetBlobDataAvailable().GetValue())

				return nil
			}).
			Times(1)

		payload := &TraceEventPayloadAttestationMessage{
			TraceEventPayloadMetaData: TraceEventPayloadMetaData{
				Topic:  "/eth2/test/payload_attestation_message/ssz_snappy",
				MsgID:  uuid.New().String(),
				PeerID: peerID.String(),
			},
			PayloadAttestationMessage: &ethtypes.PayloadAttestationMessage{
				ValidatorIndex: primitives.ValidatorIndex(42),
				Data: &ethtypes.PayloadAttestationData{
					Slot:              primitives.Slot(789),
					BeaconBlockRoot:   make([]byte, 32),
					PayloadPresent:    true,
					BlobDataAvailable: false,
				},
			},
		}

		err = mimicry.processor.handleGossipPayloadAttestationMessage(
			context.Background(),
			createTestClientMeta(),
			&TraceEvent{Type: TraceEvent_HANDLE_MESSAGE, PeerID: peerID, Timestamp: time.Now()},
			payload,
		)
		assert.NoError(t, err)
	})

	t.Run("proposer_preferences", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockSink := mock.NewMockSink(ctrl)
		mimicry := createTestMimicryWithWallclock(t, &Config{
			Events: EventConfig{GossipSubProposerPreferencesEnabled: true},
		}, mockSink)

		mockSink.EXPECT().
			HandleNewDecoratedEvent(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, event *xatu.DecoratedEvent) error {
				assert.Equal(t, xatu.Event_LIBP2P_TRACE_GOSSIPSUB_PROPOSER_PREFERENCES, event.GetEvent().GetName())
				data := event.GetLibp2PTraceGossipsubProposerPreferences()
				require.NotNil(t, data)
				assert.Equal(t, uint64(321), data.GetSlot().GetValue())
				assert.Equal(t, uint64(99), data.GetValidatorIndex().GetValue())
				assert.Equal(t, uint64(45_000_000), data.GetGasLimit().GetValue())

				return nil
			}).
			Times(1)

		payload := &TraceEventProposerPreferences{
			TraceEventPayloadMetaData: TraceEventPayloadMetaData{
				Topic:  "/eth2/test/proposer_preferences/ssz_snappy",
				MsgID:  uuid.New().String(),
				PeerID: peerID.String(),
			},
			ProposerPreferences: &ethtypes.SignedProposerPreferences{
				Message: &ethtypes.ProposerPreferences{
					ProposalSlot:   primitives.Slot(321),
					ValidatorIndex: primitives.ValidatorIndex(99),
					FeeRecipient:   make([]byte, 20),
					TargetGasLimit: 45_000_000,
				},
			},
		}

		err = mimicry.processor.handleGossipProposerPreferences(
			context.Background(),
			createTestClientMeta(),
			&TraceEvent{Type: TraceEvent_HANDLE_MESSAGE, PeerID: peerID, Timestamp: time.Now()},
			payload,
		)
		assert.NoError(t, err)
	})
}
