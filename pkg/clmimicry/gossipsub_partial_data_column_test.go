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

const (
	testPartialTopic     = "/eth2/fc90fcde/data_column_sidecar_12/ssz_snappy"
	testPartialBlockRoot = "0xcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcd"
	testPartialGroupID   = "0x00cdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcd"
)

func boolPtr(v bool) *bool {
	return &v
}

func newPartialTestMimicry(t *testing.T, events EventConfig) (*testMimicry, *mock.MockSink) {
	t.Helper()

	ctrl := gomock.NewController(t)
	mockSink := mock.NewMockSink(ctrl)

	return createTestMimicryWithWallclock(t, &Config{Events: events}, mockSink), mockSink
}

type capturedEvent struct {
	*xatu.DecoratedEvent
}

func expectSingleEvent(mockSink *mock.MockSink) *capturedEvent {
	captured := &capturedEvent{}

	mockSink.EXPECT().
		HandleNewDecoratedEvent(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, event *xatu.DecoratedEvent) error {
			captured.DecoratedEvent = event

			return nil
		}).
		Times(1)

	return captured
}

func TestPartialDataColumnEvent(t *testing.T) {
	peerID, err := peer.Decode(examplePeerID)
	require.NoError(t, err)

	mimicry, mockSink := newPartialTestMimicry(t, EventConfig{GossipSubPartialDataColumnEnabled: true})
	captured := expectSingleEvent(mockSink)

	slot := uint64(100)
	blockSlot := mimicry.processor.wallclock.Slots().FromNumber(slot)
	timestamp := blockSlot.TimeWindow().Start().Add(1500 * time.Millisecond)

	err = mimicry.processor.handlePartialDataColumnEvent(context.Background(), &TraceEvent{
		Type:      TraceEvent_GOSSIPSUB_PARTIAL_DATA_COLUMN,
		Topic:     testPartialTopic,
		PeerID:    peerID,
		Timestamp: timestamp,
		Payload: &TraceEventPartialDataColumn{
			TraceEventPayloadMetaData: TraceEventPayloadMetaData{
				PeerID:  peerID.String(),
				Topic:   testPartialTopic,
				MsgSize: 4196,
			},
			Direction:              PartialDirectionRecv,
			CapturePoint:           PartialCapturePointWire,
			GroupID:                testPartialGroupID,
			GroupVersion:           0,
			BlockRoot:              testPartialBlockRoot,
			Slot:                   slot,
			ColumnIndex:            12,
			HasMessage:             true,
			CellsCount:             2,
			CellsBitmap:            "0x4501",
			BitmapLength:           9,
			HasHeader:              true,
			KZGCommitmentsCount:    9,
			ProposerIndex:          1234,
			HasMetadata:            true,
			MetadataSize:           8,
			MetadataAvailableCount: 3,
			MetadataRequestsCount:  9,
			MetadataAvailable:      "0x0b02",
			MetadataRequests:       "0xff03",
		},
	}, createTestClientMeta())
	require.NoError(t, err)

	assert.Equal(t, xatu.Event_LIBP2P_TRACE_GOSSIPSUB_PARTIAL_DATA_COLUMN, captured.GetEvent().GetName())
	assert.Equal(t, timestamp.UnixMilli(), captured.GetEvent().GetDateTime().AsTime().UnixMilli())

	data := captured.GetLibp2PTraceGossipsubPartialDataColumn()
	require.NotNil(t, data)
	assert.Equal(t, PartialDirectionRecv, data.GetDirection().GetValue())
	assert.Equal(t, PartialCapturePointWire, data.GetCapturePoint().GetValue())
	assert.Equal(t, testPartialGroupID, data.GetGroupId().GetValue())
	assert.Equal(t, testPartialBlockRoot, data.GetBlockRoot().GetValue())
	assert.Equal(t, uint64(12), data.GetColumnIndex().GetValue())
	assert.True(t, data.GetHasMessage().GetValue())
	assert.Equal(t, uint32(2), data.GetCellsCount().GetValue())
	assert.Equal(t, "0x4501", data.GetCellsBitmap().GetValue())
	assert.Equal(t, uint32(9), data.GetBitmapLength().GetValue())
	assert.True(t, data.GetHasHeader().GetValue())
	assert.Equal(t, uint32(9), data.GetKzgCommitmentsCount().GetValue())
	assert.Equal(t, uint64(1234), data.GetProposerIndex().GetValue())
	assert.True(t, data.GetHasMetadata().GetValue())
	assert.Equal(t, uint32(8), data.GetMetadataSize().GetValue())
	assert.Equal(t, uint32(3), data.GetMetadataAvailableCount().GetValue())
	assert.Equal(t, uint32(9), data.GetMetadataRequestsCount().GetValue())
	assert.Equal(t, "0x0b02", data.GetMetadataAvailable().GetValue())
	assert.Equal(t, "0xff03", data.GetMetadataRequests().GetValue())

	additional := captured.GetMeta().GetClient().GetLibp2PTraceGossipsubPartialDataColumn()
	require.NotNil(t, additional)
	assert.Equal(t, slot, additional.GetSlot().GetNumber().GetValue())
	assert.Equal(t, uint64(3), additional.GetEpoch().GetNumber().GetValue())
	assert.Equal(t, slot, additional.GetWallclockSlot().GetNumber().GetValue())
	assert.Equal(t, uint64(1500), additional.GetPropagation().GetSlotStartDiff().GetValue())
	assert.Equal(t, peerID.String(), additional.GetMetadata().GetPeerId().GetValue())
	assert.Equal(t, testPartialTopic, additional.GetTopic().GetValue())
	assert.Equal(t, uint32(4196), additional.GetMessageSize().GetValue())
}

func TestPartialDataColumnEventUnknownSlot(t *testing.T) {
	peerID, err := peer.Decode(examplePeerID)
	require.NoError(t, err)

	mimicry, mockSink := newPartialTestMimicry(t, EventConfig{GossipSubPartialDataColumnEnabled: true})
	captured := expectSingleEvent(mockSink)

	err = mimicry.processor.handlePartialDataColumnEvent(context.Background(), &TraceEvent{
		Type:      TraceEvent_GOSSIPSUB_PARTIAL_DATA_COLUMN,
		Topic:     testPartialTopic,
		PeerID:    peerID,
		Timestamp: time.Now(),
		Payload: &TraceEventPartialDataColumn{
			TraceEventPayloadMetaData: TraceEventPayloadMetaData{PeerID: peerID.String()},
			Direction:                 PartialDirectionSend,
			CapturePoint:              PartialCapturePointWire,
			ColumnIndex:               12,
			HasMetadata:               true,
		},
	}, createTestClientMeta())
	require.NoError(t, err)

	additional := captured.GetMeta().GetClient().GetLibp2PTraceGossipsubPartialDataColumn()
	require.NotNil(t, additional)
	assert.Nil(t, additional.GetSlot())
	assert.Nil(t, additional.GetEpoch())
	assert.Nil(t, additional.GetPropagation())
	assert.NotNil(t, additional.GetWallclockSlot())
	assert.NotNil(t, additional.GetWallclockEpoch())
	assert.Equal(t, testPartialTopic, additional.GetTopic().GetValue(), "falls back to the trace event topic")
	assert.Equal(t, uint32(0), additional.GetMessageSize().GetValue())
}

func TestPartialDataColumnOutcomeEvent(t *testing.T) {
	peerID, err := peer.Decode(examplePeerID)
	require.NoError(t, err)

	mimicry, mockSink := newPartialTestMimicry(t, EventConfig{GossipSubPartialDataColumnOutcomeEnabled: true})
	captured := expectSingleEvent(mockSink)

	err = mimicry.processor.handlePartialDataColumnEvent(context.Background(), &TraceEvent{
		Type:      TraceEvent_GOSSIPSUB_PARTIAL_DATA_COLUMN_OUTCOME,
		Topic:     testPartialTopic,
		PeerID:    peerID,
		Timestamp: time.Now(),
		Payload: &TraceEventPartialDataColumnOutcome{
			TraceEventPayloadMetaData: TraceEventPayloadMetaData{
				PeerID: peerID.String(),
				Topic:  testPartialTopic,
			},
			GroupID:       testPartialGroupID,
			BlockRoot:     testPartialBlockRoot,
			Slot:          100,
			ColumnIndex:   12,
			Stage:         PartialOutcomeStageCells,
			Outcome:       PartialOutcomeReject,
			Reason:        "invalid kzg proof",
			CellsCount:    3,
			NewCellsCount: 1,
		},
	}, createTestClientMeta())
	require.NoError(t, err)

	assert.Equal(t, xatu.Event_LIBP2P_TRACE_GOSSIPSUB_PARTIAL_DATA_COLUMN_OUTCOME, captured.GetEvent().GetName())

	data := captured.GetLibp2PTraceGossipsubPartialDataColumnOutcome()
	require.NotNil(t, data)
	assert.Equal(t, testPartialGroupID, data.GetGroupId().GetValue())
	assert.Equal(t, testPartialBlockRoot, data.GetBlockRoot().GetValue())
	assert.Equal(t, uint64(12), data.GetColumnIndex().GetValue())
	assert.Equal(t, PartialOutcomeStageCells, data.GetStage().GetValue())
	assert.Equal(t, PartialOutcomeReject, data.GetOutcome().GetValue())
	assert.Equal(t, "invalid kzg proof", data.GetReason().GetValue())
	assert.Equal(t, uint32(3), data.GetCellsCount().GetValue())
	assert.Equal(t, uint32(1), data.GetNewCellsCount().GetValue())

	additional := captured.GetMeta().GetClient().GetLibp2PTraceGossipsubPartialDataColumnOutcome()
	require.NotNil(t, additional)
	assert.Equal(t, uint64(100), additional.GetSlot().GetNumber().GetValue())
	assert.Equal(t, uint64(3), additional.GetEpoch().GetNumber().GetValue())
	assert.NotNil(t, additional.GetWallclockSlot())
	assert.Equal(t, peerID.String(), additional.GetMetadata().GetPeerId().GetValue())
	assert.Equal(t, testPartialTopic, additional.GetTopic().GetValue())
}

func TestDataColumnCompletedEvent(t *testing.T) {
	peerID, err := peer.Decode(examplePeerID)
	require.NoError(t, err)

	tests := []struct {
		name     string
		peerID   string
		via      string
		wantPeer bool
	}{
		{name: "completed by a peer", peerID: peerID.String(), via: DataColumnCompletedViaPartialGossip, wantPeer: true},
		{name: "completed from the EL", via: DataColumnCompletedViaEL},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mimicry, mockSink := newPartialTestMimicry(t, EventConfig{DataColumnCompletedEnabled: true})
			captured := expectSingleEvent(mockSink)

			err := mimicry.processor.handlePartialDataColumnEvent(context.Background(), &TraceEvent{
				Type:      TraceEvent_DATA_COLUMN_COMPLETED,
				Topic:     testPartialTopic,
				PeerID:    peerID,
				Timestamp: time.Now(),
				Payload: &TraceEventDataColumnCompleted{
					TraceEventPayloadMetaData: TraceEventPayloadMetaData{
						PeerID: tt.peerID,
						Topic:  testPartialTopic,
					},
					BlockRoot:           testPartialBlockRoot,
					Slot:                100,
					ColumnIndex:         12,
					CompletedVia:        tt.via,
					CellsTotal:          9,
					CellsFromGossip:     4,
					CellsFromEL:         5,
					ContributingPeers:   2,
					KZGCommitmentsCount: 9,
				},
			}, createTestClientMeta())
			require.NoError(t, err)

			assert.Equal(t, xatu.Event_LIBP2P_TRACE_DATA_COLUMN_COMPLETED, captured.GetEvent().GetName())

			data := captured.GetLibp2PTraceDataColumnCompleted()
			require.NotNil(t, data)
			assert.Equal(t, testPartialBlockRoot, data.GetBlockRoot().GetValue())
			assert.Equal(t, uint64(12), data.GetColumnIndex().GetValue())
			assert.Equal(t, tt.via, data.GetCompletedVia().GetValue())
			assert.Equal(t, uint32(9), data.GetCellsTotal().GetValue())
			assert.Equal(t, uint32(4), data.GetCellsFromGossip().GetValue())
			assert.Equal(t, uint32(5), data.GetCellsFromEl().GetValue())
			assert.Equal(t, uint32(2), data.GetContributingPeers().GetValue())
			assert.Equal(t, uint32(9), data.GetKzgCommitmentsCount().GetValue())

			additional := captured.GetMeta().GetClient().GetLibp2PTraceDataColumnCompleted()
			require.NotNil(t, additional)
			assert.Equal(t, uint64(100), additional.GetSlot().GetNumber().GetValue())
			assert.NotNil(t, additional.GetPropagation())
			assert.Equal(t, testPartialTopic, additional.GetTopic().GetValue())

			if tt.wantPeer {
				assert.Equal(t, tt.peerID, additional.GetMetadata().GetPeerId().GetValue())
			} else {
				assert.Nil(t, additional.GetMetadata())
			}
		})
	}
}

func TestPartialDataColumnEventsDisabled(t *testing.T) {
	peerID, err := peer.Decode(examplePeerID)
	require.NoError(t, err)

	events := []*TraceEvent{
		{Type: TraceEvent_GOSSIPSUB_PARTIAL_DATA_COLUMN, Payload: &TraceEventPartialDataColumn{}},
		{Type: TraceEvent_GOSSIPSUB_PARTIAL_DATA_COLUMN_OUTCOME, Payload: &TraceEventPartialDataColumnOutcome{}},
		{Type: TraceEvent_DATA_COLUMN_COMPLETED, Payload: &TraceEventDataColumnCompleted{}},
	}

	for _, event := range events {
		t.Run(event.Type, func(t *testing.T) {
			// No sink expectations: a disabled event must not reach the output.
			mimicry, _ := newPartialTestMimicry(t, EventConfig{})

			event.PeerID = peerID
			event.Timestamp = time.Now()

			err := mimicry.processor.handlePartialDataColumnEvent(context.Background(), event, createTestClientMeta())
			assert.NoError(t, err)
		})
	}
}

func TestPartialDataColumnEventsInvalidPayload(t *testing.T) {
	peerID, err := peer.Decode(examplePeerID)
	require.NoError(t, err)

	mimicry, _ := newPartialTestMimicry(t, EventConfig{
		GossipSubPartialDataColumnEnabled:        true,
		GossipSubPartialDataColumnOutcomeEnabled: true,
		DataColumnCompletedEnabled:               true,
	})

	for _, eventType := range []string{
		TraceEvent_GOSSIPSUB_PARTIAL_DATA_COLUMN,
		TraceEvent_GOSSIPSUB_PARTIAL_DATA_COLUMN_OUTCOME,
		TraceEvent_DATA_COLUMN_COMPLETED,
	} {
		t.Run(eventType, func(t *testing.T) {
			err := mimicry.processor.handlePartialDataColumnEvent(context.Background(), &TraceEvent{
				Type:      eventType,
				PeerID:    peerID,
				Timestamp: time.Now(),
				Payload:   &TraceEventMessagePayload{},
			}, createTestClientMeta())
			assert.Error(t, err)
		})
	}
}

func TestPartialDataColumnEventSharding(t *testing.T) {
	peerID, err := peer.Decode(examplePeerID)
	require.NoError(t, err)

	ctrl := gomock.NewController(t)
	mockSink := mock.NewMockSink(ctrl)

	mimicry := createTestMimicryWithWallclock(t, &Config{
		Events: EventConfig{GossipSubPartialDataColumnEnabled: true},
		Sharding: ShardingConfig{
			Topics: map[string]*TopicShardingConfig{
				".*data_column_sidecar.*": {TotalShards: 1, ActiveShards: []uint64{0}},
			},
		},
	}, mockSink)

	// Group B shards on topic only, so the topic-less event is dropped.
	captured := expectSingleEvent(mockSink)

	for _, topic := range []string{testPartialTopic, ""} {
		err = mimicry.processor.handlePartialDataColumnEvent(context.Background(), &TraceEvent{
			Type:      TraceEvent_GOSSIPSUB_PARTIAL_DATA_COLUMN,
			Topic:     topic,
			PeerID:    peerID,
			Timestamp: time.Now(),
			Payload: &TraceEventPartialDataColumn{
				TraceEventPayloadMetaData: TraceEventPayloadMetaData{PeerID: peerID.String()},
				ColumnIndex:               12,
			},
		}, createTestClientMeta())
		require.NoError(t, err)
	}

	assert.Equal(t, testPartialTopic, captured.GetMeta().GetClient().GetLibp2PTraceGossipsubPartialDataColumn().GetTopic().GetValue())
}

type staticMetaProvider struct{}

func (staticMetaProvider) GetClientMeta(context.Context) (*xatu.ClientMeta, error) {
	return createTestClientMeta(), nil
}

func TestHandleHermesEventRoutesPartialEvents(t *testing.T) {
	peerID, err := peer.Decode(examplePeerID)
	require.NoError(t, err)

	tests := []struct {
		event *TraceEvent
		want  xatu.Event_Name
	}{
		{
			event: &TraceEvent{Type: TraceEvent_GOSSIPSUB_PARTIAL_DATA_COLUMN, Payload: &TraceEventPartialDataColumn{}},
			want:  xatu.Event_LIBP2P_TRACE_GOSSIPSUB_PARTIAL_DATA_COLUMN,
		},
		{
			event: &TraceEvent{Type: TraceEvent_GOSSIPSUB_PARTIAL_DATA_COLUMN_OUTCOME, Payload: &TraceEventPartialDataColumnOutcome{}},
			want:  xatu.Event_LIBP2P_TRACE_GOSSIPSUB_PARTIAL_DATA_COLUMN_OUTCOME,
		},
		{
			event: &TraceEvent{Type: TraceEvent_DATA_COLUMN_COMPLETED, Payload: &TraceEventDataColumnCompleted{}},
			want:  xatu.Event_LIBP2P_TRACE_DATA_COLUMN_COMPLETED,
		},
	}

	for _, tt := range tests {
		t.Run(tt.event.Type, func(t *testing.T) {
			mimicry, mockSink := newPartialTestMimicry(t, EventConfig{
				GossipSubPartialDataColumnEnabled:        true,
				GossipSubPartialDataColumnOutcomeEnabled: true,
				DataColumnCompletedEnabled:               true,
			})
			mimicry.processor.metaProvider = staticMetaProvider{}
			captured := expectSingleEvent(mockSink)

			tt.event.Topic = testPartialTopic
			tt.event.PeerID = peerID
			tt.event.Timestamp = time.Now()

			require.NoError(t, mimicry.processor.HandleHermesEvent(context.Background(), tt.event))
			assert.Equal(t, tt.want, captured.GetEvent().GetName())
		})
	}
}
