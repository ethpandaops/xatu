package clmimicry

import (
	"context"
	"encoding/hex"
	"fmt"
	"testing"
	"time"

	"github.com/OffchainLabs/prysm/v6/consensus-types/primitives"
	ethtypes "github.com/OffchainLabs/prysm/v6/proto/prysm/v1alpha1"
	"github.com/ethpandaops/ethwallclock"
	"github.com/ethpandaops/xatu/pkg/output"
	"github.com/ethpandaops/xatu/pkg/output/mock"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/google/uuid"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/probe-lab/hermes/eth/events"
	"github.com/probe-lab/hermes/host"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func TestDataColumnSidecarIntegration(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create a mock sink
	mockSink := mock.NewMockSink(ctrl)

	// Create config with data column sidecar enabled
	config := &Config{
		Events: EventConfig{
			GossipSubDataColumnSidecarEnabled: true,
		},
	}

	// Create mimicry instance with a proper wallclock
	mimicry := createTestMimicryWithWallclock(t, config, mockSink)

	// Create test data
	peerID, err := peer.Decode(examplePeerID)
	require.NoError(t, err)

	stateRoot := [32]byte{10, 20, 30, 40}
	parentRoot := [32]byte{50, 60, 70, 80}

	// Create multiple data column sidecars with different indices
	sidecars := []struct {
		index uint64
		slot  uint64
	}{
		{index: 0, slot: 100},
		{index: 5, slot: 100},
		{index: 127, slot: 200},
	}

	// Set up expectations for all sidecars
	mockSink.EXPECT().
		HandleNewDecoratedEvent(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, event *xatu.DecoratedEvent) error {
			assert.Equal(t, xatu.Event_LIBP2P_TRACE_GOSSIPSUB_DATA_COLUMN_SIDECAR, event.GetEvent().GetName())
			assert.NotNil(t, event.GetLibp2PTraceGossipsubDataColumnSidecar())

			return nil
		}).
		Times(len(sidecars))

	// Process each sidecar
	for _, sidecar := range sidecars {
		event := &host.TraceEvent{
			Type:      "HANDLE_MESSAGE",
			PeerID:    peerID,
			Timestamp: time.Now(),
		}

		payload := &events.TraceEventDataColumnSidecar{
			TraceEventPayloadMetaData: host.TraceEventPayloadMetaData{
				Topic:   "/eth2/fc90fcde/data_column_sidecar_0/ssz_snappy",
				MsgID:   "msg-" + fmt.Sprintf("%d", sidecar.index),
				MsgSize: int(1000 + sidecar.index),
				PeerID:  peerID.String(),
			},
			DataColumnSidecar: &ethtypes.DataColumnSidecar{
				Index: sidecar.index,
				KzgCommitments: [][]byte{
					make([]byte, 48),
					make([]byte, 48),
				},
				SignedBlockHeader: &ethtypes.SignedBeaconBlockHeader{
					Header: &ethtypes.BeaconBlockHeader{
						Slot:          primitives.Slot(sidecar.slot),
						ProposerIndex: primitives.ValidatorIndex(42),
						StateRoot:     stateRoot[:],
						ParentRoot:    parentRoot[:],
						BodyRoot:      bodyRoot[:],
					},
				},
			},
		}

		err := mimicry.processor.handleGossipDataColumnSidecar(
			context.Background(),
			createTestClientMeta(),
			event,
			payload,
		)
		assert.NoError(t, err)
	}
}

func TestDataColumnSidecarPropagationTiming(t *testing.T) {
	// Create test wallclock with specific genesis time
	genesisTime := time.Date(2020, time.December, 1, 12, 0, 0, 0, time.UTC)
	wallclock := ethwallclock.NewEthereumBeaconChain(genesisTime, 12*time.Second, 32)

	// Create processor with known clock drift
	clockDrift := 500 * time.Millisecond
	processor := &Processor{
		wallclock:  wallclock,
		clockDrift: clockDrift,
	}

	// Calculate slot start time for slot 100
	slot := wallclock.Slots().FromNumber(100)
	slotStartTime := slot.TimeWindow().Start()

	tests := []struct {
		name                  string
		eventTimestamp        time.Time
		expectedSlotStartDiff uint64
	}{
		{
			name:                  "Event exactly at slot start",
			eventTimestamp:        slotStartTime,
			expectedSlotStartDiff: uint64(clockDrift.Milliseconds()),
		},
		{
			name:                  "Event 1 second after slot start",
			eventTimestamp:        slotStartTime.Add(1 * time.Second),
			expectedSlotStartDiff: uint64((1*time.Second + clockDrift).Milliseconds()),
		},
		{
			name:                  "Event 500ms before slot start",
			eventTimestamp:        slotStartTime.Add(-500 * time.Millisecond),
			expectedSlotStartDiff: uint64((clockDrift - 500*time.Millisecond).Milliseconds()),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload := &events.TraceEventDataColumnSidecar{
				TraceEventPayloadMetaData: host.TraceEventPayloadMetaData{
					Topic:   "/eth2/test/data_column_sidecar_0/ssz_snappy",
					MsgID:   "test-msg",
					MsgSize: 1000,
					PeerID:  "test-peer",
				},
				DataColumnSidecar: &ethtypes.DataColumnSidecar{
					Index: 0,
					SignedBlockHeader: &ethtypes.SignedBeaconBlockHeader{
						Header: &ethtypes.BeaconBlockHeader{
							Slot: primitives.Slot(100),
						},
					},
				},
			}

			data, err := processor.createAdditionalGossipSubDataColumnSidecarData(
				payload,
				tt.eventTimestamp,
			)

			assert.NoError(t, err)
			assert.NotNil(t, data)
			assert.NotNil(t, data.GetPropagation())
			assert.NotNil(t, data.GetPropagation().GetSlotStartDiff())
			assert.Equal(t, tt.expectedSlotStartDiff, data.GetPropagation().GetSlotStartDiff().GetValue())
		})
	}
}

func TestDataColumnSidecarEdgeCases(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSink := mock.NewMockSink(ctrl)
	config := &Config{
		Events: EventConfig{
			GossipSubDataColumnSidecarEnabled: true,
		},
	}
	mimicry := createTestMimicryWithWallclock(t, config, mockSink)

	peerID, err := peer.Decode(examplePeerID)
	require.NoError(t, err)

	t.Run("Empty state and parent roots", func(t *testing.T) {
		mockSink.EXPECT().
			HandleNewDecoratedEvent(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, event *xatu.DecoratedEvent) error {
				data := event.GetLibp2PTraceGossipsubDataColumnSidecar()
				assert.Equal(t, wrapperspb.String("").GetValue(), data.StateRoot.GetValue())
				assert.Equal(t, wrapperspb.String("").GetValue(), data.ParentRoot.GetValue())
				assert.Equal(t, wrapperspb.String("").GetValue(), data.BlockRoot.GetValue())

				return nil
			}).
			Times(1)

		payload := &events.TraceEventDataColumnSidecar{
			TraceEventPayloadMetaData: host.TraceEventPayloadMetaData{
				Topic:  "/eth2/test/data_column_sidecar_0/ssz_snappy",
				MsgID:  "test",
				PeerID: peerID.String(),
			},
			DataColumnSidecar: &ethtypes.DataColumnSidecar{
				Index: 0,
				SignedBlockHeader: &ethtypes.SignedBeaconBlockHeader{
					Header: &ethtypes.BeaconBlockHeader{
						Slot:          primitives.Slot(100),
						ProposerIndex: primitives.ValidatorIndex(1),
						StateRoot:     []byte{},
						ParentRoot:    []byte{},
						BodyRoot:      []byte{},
					},
				},
			},
		}

		err := mimicry.processor.handleGossipDataColumnSidecar(
			context.Background(),
			createTestClientMeta(),
			&host.TraceEvent{
				Type:      "HANDLE_MESSAGE",
				PeerID:    peerID,
				Timestamp: time.Now(),
			},
			payload,
		)
		assert.NoError(t, err)
	})

	t.Run("Maximum index value", func(t *testing.T) {
		mockSink.EXPECT().
			HandleNewDecoratedEvent(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, event *xatu.DecoratedEvent) error {
				data := event.GetLibp2PTraceGossipsubDataColumnSidecar()
				assert.Equal(t, uint64(127), data.Index.GetValue())

				return nil
			}).
			Times(1)

		payload := &events.TraceEventDataColumnSidecar{
			TraceEventPayloadMetaData: host.TraceEventPayloadMetaData{
				Topic:  "/eth2/test/data_column_sidecar_127/ssz_snappy",
				MsgID:  "max-index",
				PeerID: peerID.String(),
			},
			DataColumnSidecar: &ethtypes.DataColumnSidecar{
				Index: 127, // Maximum valid index for data columns
				KzgCommitments: [][]byte{
					make([]byte, 48),
					make([]byte, 48),
					make([]byte, 48),
					make([]byte, 48),
				},
				SignedBlockHeader: &ethtypes.SignedBeaconBlockHeader{
					Header: &ethtypes.BeaconBlockHeader{
						Slot:          primitives.Slot(1000),
						ProposerIndex: primitives.ValidatorIndex(100),
						StateRoot:     make([]byte, 32),
						ParentRoot:    make([]byte, 32),
						BodyRoot:      make([]byte, 32),
					},
				},
			},
		}

		err := mimicry.processor.handleGossipDataColumnSidecar(
			context.Background(),
			createTestClientMeta(),
			&host.TraceEvent{
				Type:      "HANDLE_MESSAGE",
				PeerID:    peerID,
				Timestamp: time.Now(),
			},
			payload,
		)
		assert.NoError(t, err)
	})
}

// Helper function to create a test mimicry with a properly initialized wallclock
func createTestMimicryWithWallclock(t *testing.T, config *Config, sink output.Sink) *Mimicry {
	t.Helper()

	// Initialize wallclock with proper genesis time
	genesisTime := time.Date(2020, time.December, 1, 12, 0, 0, 0, time.UTC)
	wallclock := ethwallclock.NewEthereumBeaconChain(genesisTime, 12*time.Second, 32)

	// Create sharder from config if sharding is configured, otherwise disable it
	var sharder *UnifiedSharder
	if config.Sharding.Topics != nil || config.Sharding.NoShardingKeyEvents != nil {
		var err error
		sharder, err = NewUnifiedSharder(&config.Sharding, true)
		if err != nil {
			t.Fatalf("Failed to create sharder: %v", err)
		}
	} else {
		sharder = &UnifiedSharder{
			config:           &ShardingConfig{},
			eventCategorizer: NewEventCategorizer(),
			enabled:          false,
		}
	}

	mimicry := &Mimicry{
		Config:  config,
		sinks:   []output.Sink{sink},
		log:     logrus.NewEntry(logrus.New()),
		id:      uuid.New(),
		metrics: NewMetrics(t.Name()),
		sharder: sharder,
	}

	mimicry.processor = NewProcessor(
		mimicry,               // DutiesProvider
		mimicry,               // OutputHandler
		mimicry.metrics,       // MetricsCollector
		mimicry,               // MetaProvider
		mimicry.sharder,       // UnifiedSharder
		NewEventCategorizer(), // EventCategorizer
		wallclock,             // EthereumBeaconChain
		time.Duration(0),      // clockDrift
		config.Events,         // EventConfig
		mimicry.log.WithField("component", "processor"),
	)

	return mimicry
}

func Test_handleGossipDataColumnSidecar(t *testing.T) {
	// Create a valid peer ID for testing
	peerID, err := peer.Decode(examplePeerID)
	require.NoError(t, err)

	// Create sample state and parent roots
	stateRoot := [32]byte{1, 2, 3, 4}
	parentRoot := [32]byte{5, 6, 7, 8}

	tests := []struct {
		name           string
		config         *Config
		event          *host.TraceEvent
		payload        *events.TraceEventDataColumnSidecar
		expectError    bool
		errorMessage   string
		setupMockCalls func(*mock.MockSink)
		validateEvent  singleEventValidator
	}{
		{
			name: "Valid data column sidecar event",
			config: &Config{
				Events: EventConfig{
					GossipSubDataColumnSidecarEnabled: true,
				},
			},
			event: &host.TraceEvent{
				Type:      "HANDLE_MESSAGE",
				PeerID:    peerID,
				Timestamp: time.Now(),
				Topic:     "/eth2/fc90fcde/data_column_sidecar_0/ssz_snappy",
			},
			payload: &events.TraceEventDataColumnSidecar{
				TraceEventPayloadMetaData: host.TraceEventPayloadMetaData{
					Topic:   "/eth2/fc90fcde/data_column_sidecar_0/ssz_snappy",
					MsgID:   "msg123",
					MsgSize: 1000,
					PeerID:  peerID.String(),
				},
				DataColumnSidecar: &ethtypes.DataColumnSidecar{
					Index: 5,
					KzgCommitments: [][]byte{
						make([]byte, 48),
						make([]byte, 48),
						make([]byte, 48),
					},
					SignedBlockHeader: &ethtypes.SignedBeaconBlockHeader{
						Header: &ethtypes.BeaconBlockHeader{
							Slot:          primitives.Slot(100),
							ProposerIndex: primitives.ValidatorIndex(42),
							StateRoot:     stateRoot[:],
							ParentRoot:    parentRoot[:],
						},
					},
				},
			},
			expectError: false,
			setupMockCalls: func(mockSink *mock.MockSink) {
				expectEventWithValidation(t, mockSink, func(t *testing.T, event *xatu.DecoratedEvent) {
					t.Helper()

					assert.Equal(t, xatu.Event_LIBP2P_TRACE_GOSSIPSUB_DATA_COLUMN_SIDECAR, event.GetEvent().GetName())

					data := event.GetLibp2PTraceGossipsubDataColumnSidecar()
					assert.NotNil(t, data)
					assert.Equal(t, uint64(5), data.Index.GetValue())
					assert.Equal(t, uint64(100), data.Slot.GetValue())
					assert.Equal(t, uint64(42), data.ProposerIndex.GetValue())
					assert.Equal(t, hex.EncodeToString(stateRoot[:]), data.StateRoot.GetValue())
					assert.Equal(t, hex.EncodeToString(parentRoot[:]), data.ParentRoot.GetValue())
					assert.Equal(t, uint32(3), data.KzgCommitmentsCount.GetValue())
					assert.Equal(t, hex.EncodeToString(bodyRoot[:]), data.BlockRoot.GetValue())

					// Check metadata
					meta := event.GetMeta().GetClient()
					assert.NotNil(t, meta.GetLibp2PTraceGossipsubDataColumnSidecar())

					additionalData := meta.GetLibp2PTraceGossipsubDataColumnSidecar()
					assert.NotNil(t, additionalData.GetSlot())
					assert.Equal(t, uint64(100), additionalData.GetSlot().GetNumber().GetValue())
					assert.NotNil(t, additionalData.GetEpoch())
					assert.NotNil(t, additionalData.GetWallclockSlot())
					assert.NotNil(t, additionalData.GetWallclockEpoch())
					assert.NotNil(t, additionalData.GetPropagation())
					assert.Equal(t, peerID.String(), additionalData.GetMetadata().GetPeerId().GetValue())
					assert.Equal(t, "/eth2/fc90fcde/data_column_sidecar_0/ssz_snappy", additionalData.GetTopic().GetValue())
					assert.Equal(t, uint32(1000), additionalData.GetMessageSize().GetValue())
					assert.Equal(t, "msg123", additionalData.GetMessageId().GetValue())
				})
			},
		},
		{
			name: "Event with nil data column sidecar",
			config: &Config{
				Events: EventConfig{
					GossipSubDataColumnSidecarEnabled: true,
				},
			},
			event: &host.TraceEvent{
				Type:      "HANDLE_MESSAGE",
				PeerID:    peerID,
				Timestamp: time.Now(),
				Topic:     "/eth2/fc90fcde/data_column_sidecar_0/ssz_snappy",
			},
			payload: &events.TraceEventDataColumnSidecar{
				TraceEventPayloadMetaData: host.TraceEventPayloadMetaData{
					Topic:  "/eth2/fc90fcde/data_column_sidecar_0/ssz_snappy",
					MsgID:  "msg123",
					PeerID: peerID.String(),
				},
				DataColumnSidecar: nil,
			},
			expectError:    true,
			errorMessage:   "handleGossipDataColumnSidecar() called with nil data column sidecar",
			setupMockCalls: expectNoEvents,
		},
		{
			name: "Event disabled in config",
			config: &Config{
				Events: EventConfig{
					GossipSubDataColumnSidecarEnabled: false,
				},
			},
			event: &host.TraceEvent{
				Type:      "HANDLE_MESSAGE",
				PeerID:    peerID,
				Timestamp: time.Now(),
				Topic:     "/eth2/fc90fcde/data_column_sidecar_0/ssz_snappy",
			},
			payload: &events.TraceEventDataColumnSidecar{
				TraceEventPayloadMetaData: host.TraceEventPayloadMetaData{
					Topic:  "/eth2/fc90fcde/data_column_sidecar_0/ssz_snappy",
					MsgID:  "msg123",
					PeerID: peerID.String(),
				},
				DataColumnSidecar: &ethtypes.DataColumnSidecar{
					Index: 5,
					SignedBlockHeader: &ethtypes.SignedBeaconBlockHeader{
						Header: &ethtypes.BeaconBlockHeader{
							Slot:          primitives.Slot(100),
							ProposerIndex: primitives.ValidatorIndex(42),
							StateRoot:     stateRoot[:],
							ParentRoot:    parentRoot[:],
							BodyRoot:      bodyRoot[:],
						},
					},
				},
			},
			expectError:    false,
			setupMockCalls: expectNoEvents,
		},
		{
			name: "Event with zero index",
			config: &Config{
				Events: EventConfig{
					GossipSubDataColumnSidecarEnabled: true,
				},
			},
			event: &host.TraceEvent{
				Type:      "HANDLE_MESSAGE",
				PeerID:    peerID,
				Timestamp: time.Now(),
				Topic:     "/eth2/fc90fcde/data_column_sidecar_0/ssz_snappy",
			},
			payload: &events.TraceEventDataColumnSidecar{
				TraceEventPayloadMetaData: host.TraceEventPayloadMetaData{
					Topic:   "/eth2/fc90fcde/data_column_sidecar_0/ssz_snappy",
					MsgID:   "msg456",
					MsgSize: 2000,
					PeerID:  peerID.String(),
				},
				DataColumnSidecar: &ethtypes.DataColumnSidecar{
					Index:          0,
					KzgCommitments: [][]byte{}, // No KZG commitments
					SignedBlockHeader: &ethtypes.SignedBeaconBlockHeader{
						Header: &ethtypes.BeaconBlockHeader{
							Slot:          primitives.Slot(200),
							ProposerIndex: primitives.ValidatorIndex(1),
							StateRoot:     stateRoot[:],
							ParentRoot:    parentRoot[:],
							BodyRoot:      bodyRoot[:],
						},
					},
				},
			},
			expectError: false,
			setupMockCalls: func(mockSink *mock.MockSink) {
				expectEventWithValidation(t, mockSink, func(t *testing.T, event *xatu.DecoratedEvent) {
					t.Helper()

					assert.Equal(t, xatu.Event_LIBP2P_TRACE_GOSSIPSUB_DATA_COLUMN_SIDECAR, event.GetEvent().GetName())

					data := event.GetLibp2PTraceGossipsubDataColumnSidecar()
					assert.NotNil(t, data)
					assert.Equal(t, uint64(0), data.Index.GetValue())
					assert.Equal(t, uint64(200), data.Slot.GetValue())
					assert.Equal(t, uint64(1), data.ProposerIndex.GetValue())
					assert.Equal(t, uint32(0), data.KzgCommitmentsCount.GetValue())
					assert.Equal(t, hex.EncodeToString(bodyRoot[:]), data.BlockRoot.GetValue())
				})
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockSink := mock.NewMockSink(ctrl)
			mimicry := createTestMimicryWithWallclock(t, tt.config, mockSink)

			// Set up mock expectations
			if tt.setupMockCalls != nil {
				tt.setupMockCalls(mockSink)
			}

			// Create client metadata
			clientMeta := createTestClientMeta()

			// Set the payload in the event
			tt.event.Payload = tt.payload

			// Call the gossipsub event handler which routes to the data column sidecar handler
			// This ensures the event enabled check is properly applied
			err := mimicry.processor.handleHermesGossipSubEvent(
				context.Background(),
				tt.event,
				clientMeta,
				createTestTraceMeta(),
			)

			// Validate error expectations
			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMessage != "" {
					assert.Contains(t, err.Error(), tt.errorMessage)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func Test_createAdditionalGossipSubDataColumnSidecarData(t *testing.T) {
	// Create test wallclock
	wallclock := ethwallclock.NewEthereumBeaconChain(
		time.Date(2020, time.December, 1, 12, 0, 0, 0, time.UTC),
		12*time.Second,
		32,
	)

	// Create a processor with the wallclock
	processor := &Processor{
		wallclock:  wallclock,
		clockDrift: 100 * time.Millisecond,
	}

	tests := []struct {
		name         string
		payload      *events.TraceEventDataColumnSidecar
		timestamp    time.Time
		expectError  bool
		validateData func(t *testing.T, data *xatu.ClientMeta_AdditionalLibP2PTraceGossipSubDataColumnSidecarData)
	}{
		{
			name: "Valid payload with typical values",
			payload: &events.TraceEventDataColumnSidecar{
				TraceEventPayloadMetaData: host.TraceEventPayloadMetaData{
					Topic:   "/eth2/fc90fcde/data_column_sidecar_5/ssz_snappy",
					MsgID:   "msg123",
					MsgSize: 1500,
					PeerID:  "test-peer",
				},
				DataColumnSidecar: &ethtypes.DataColumnSidecar{
					Index: 5,
					SignedBlockHeader: &ethtypes.SignedBeaconBlockHeader{
						Header: &ethtypes.BeaconBlockHeader{
							Slot:          primitives.Slot(1000),
							ProposerIndex: primitives.ValidatorIndex(42),
						},
					},
				},
			},
			timestamp:   time.Now(),
			expectError: false,
			validateData: func(t *testing.T, data *xatu.ClientMeta_AdditionalLibP2PTraceGossipSubDataColumnSidecarData) {
				t.Helper()

				assert.NotNil(t, data.GetWallclockSlot())
				assert.NotNil(t, data.GetWallclockEpoch())
				assert.NotNil(t, data.GetSlot())
				assert.Equal(t, uint64(1000), data.GetSlot().GetNumber().GetValue())
				assert.NotNil(t, data.GetEpoch())
				assert.Equal(t, uint64(31), data.GetEpoch().GetNumber().GetValue()) // slot 1000 / 32 = epoch 31
				assert.NotNil(t, data.GetPropagation())
				assert.NotNil(t, data.GetPropagation().GetSlotStartDiff())
				assert.Equal(t, "test-peer", data.GetMetadata().GetPeerId().GetValue())
				assert.Equal(t, "/eth2/fc90fcde/data_column_sidecar_5/ssz_snappy", data.GetTopic().GetValue())
				assert.Equal(t, uint32(1500), data.GetMessageSize().GetValue())
				assert.Equal(t, "msg123", data.GetMessageId().GetValue())
			},
		},
		{
			name: "Payload with slot 0",
			payload: &events.TraceEventDataColumnSidecar{
				TraceEventPayloadMetaData: host.TraceEventPayloadMetaData{
					Topic:   "/eth2/fc90fcde/data_column_sidecar_0/ssz_snappy",
					MsgID:   "genesis-msg",
					MsgSize: 500,
					PeerID:  "genesis-peer",
				},
				DataColumnSidecar: &ethtypes.DataColumnSidecar{
					Index: 0,
					SignedBlockHeader: &ethtypes.SignedBeaconBlockHeader{
						Header: &ethtypes.BeaconBlockHeader{
							Slot:          primitives.Slot(0),
							ProposerIndex: primitives.ValidatorIndex(0),
						},
					},
				},
			},
			timestamp:   time.Now(),
			expectError: false,
			validateData: func(t *testing.T, data *xatu.ClientMeta_AdditionalLibP2PTraceGossipSubDataColumnSidecarData) {
				t.Helper()

				assert.NotNil(t, data.GetSlot())
				assert.Equal(t, uint64(0), data.GetSlot().GetNumber().GetValue())
				assert.NotNil(t, data.GetEpoch())
				assert.Equal(t, uint64(0), data.GetEpoch().GetNumber().GetValue())
				assert.Equal(t, "genesis-peer", data.GetMetadata().GetPeerId().GetValue())
				assert.Equal(t, uint32(500), data.GetMessageSize().GetValue())
				assert.Equal(t, "genesis-msg", data.GetMessageId().GetValue())
			},
		},
		{
			name: "Payload with large slot number",
			payload: &events.TraceEventDataColumnSidecar{
				TraceEventPayloadMetaData: host.TraceEventPayloadMetaData{
					Topic:   "/eth2/fc90fcde/data_column_sidecar_127/ssz_snappy",
					MsgID:   "large-msg",
					MsgSize: 10000,
					PeerID:  "large-peer",
				},
				DataColumnSidecar: &ethtypes.DataColumnSidecar{
					Index: 127,
					SignedBlockHeader: &ethtypes.SignedBeaconBlockHeader{
						Header: &ethtypes.BeaconBlockHeader{
							Slot:          primitives.Slot(1000000),
							ProposerIndex: primitives.ValidatorIndex(999),
						},
					},
				},
			},
			timestamp:   time.Now(),
			expectError: false,
			validateData: func(t *testing.T, data *xatu.ClientMeta_AdditionalLibP2PTraceGossipSubDataColumnSidecarData) {
				t.Helper()

				assert.NotNil(t, data.GetSlot())
				assert.Equal(t, uint64(1000000), data.GetSlot().GetNumber().GetValue())
				assert.NotNil(t, data.GetEpoch())
				assert.Equal(t, uint64(31250), data.GetEpoch().GetNumber().GetValue()) // slot 1000000 / 32 = epoch 31250
				assert.Equal(t, "/eth2/fc90fcde/data_column_sidecar_127/ssz_snappy", data.GetTopic().GetValue())
				assert.Equal(t, uint32(10000), data.GetMessageSize().GetValue())
			},
		},
		{
			name: "Payload with nil signed block header",
			payload: &events.TraceEventDataColumnSidecar{
				TraceEventPayloadMetaData: host.TraceEventPayloadMetaData{
					Topic:   "/eth2/fc90fcde/data_column_sidecar_0/ssz_snappy",
					MsgID:   "nil-header",
					MsgSize: 100,
					PeerID:  "nil-peer",
				},
				DataColumnSidecar: &ethtypes.DataColumnSidecar{
					Index:             0,
					SignedBlockHeader: nil,
				},
			},
			timestamp:   time.Now(),
			expectError: false, // Should handle gracefully
			validateData: func(t *testing.T, data *xatu.ClientMeta_AdditionalLibP2PTraceGossipSubDataColumnSidecarData) {
				t.Helper()

				// The function should handle nil header gracefully
				assert.NotNil(t, data)
				assert.Equal(t, "nil-peer", data.GetMetadata().GetPeerId().GetValue())
				assert.Equal(t, uint32(100), data.GetMessageSize().GetValue())
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := processor.createAdditionalGossipSubDataColumnSidecarData(
				tt.payload,
				tt.timestamp,
			)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, data)
				if tt.validateData != nil {
					tt.validateData(t, data)
				}
			}
		})
	}
}
