package canonical

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/ethpandaops/xatu/pkg/clickhouse/route"
	"github.com/ethpandaops/xatu/pkg/clickhouse/route/testfixture"
	ethv1 "github.com/ethpandaops/xatu/pkg/proto/eth/v1"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

func builderExitEvent() *xatu.DecoratedEvent {
	return &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_EXECUTION_REQUEST_BUILDER_EXIT,
			DateTime: testfixture.TS(),
			Id:       "cberbe-1",
		},
		Meta: testfixture.MetaWithAdditional(&xatu.ClientMeta{
			AdditionalData: &xatu.ClientMeta_EthV2BeaconBlockExecutionRequestBuilderExit{
				EthV2BeaconBlockExecutionRequestBuilderExit: &xatu.ClientMeta_AdditionalEthV2BeaconBlockExecutionRequestBuilderExitData{
					Block: &xatu.BlockIdentifier{
						Epoch:   testfixture.EpochAdditional(),
						Slot:    testfixture.SlotEpochAdditional(),
						Root:    "0x00000000000000000000000000000000000000000000000000000000000000bb",
						Version: "gloas",
					},
					PositionInBlock: wrapperspb.UInt64(2),
					BlockNumber:     wrapperspb.UInt64(1234),
					BlockHash:       testBuilderBlockHash,
				},
			},
		}),
		Data: &xatu.DecoratedEvent_EthV2BeaconBlockExecutionRequestBuilderExit{
			EthV2BeaconBlockExecutionRequestBuilderExit: &ethv1.GloasBuilderExitRequest{
				SourceAddress: wrapperspb.String("0x000000000000000000000000000000000000dead"),
				Pubkey:        wrapperspb.String("0xabcdef"),
			},
		},
	}
}

func TestSnapshot_canonical_beacon_block_execution_request_builder_exit(t *testing.T) {
	testfixture.AssertSnapshot(t, newcanonicalBeaconBlockExecutionRequestBuilderExitBatch(), builderExitEvent(), 1, map[string]any{
		"slot":              uint32(100),
		"epoch":             uint32(3),
		"block_root":        "0x00000000000000000000000000000000000000000000000000000000000000bb",
		"block_version":     "gloas",
		"block_number":      uint64(1234),
		"block_hash":        testBuilderBlockHash,
		"position_in_block": uint32(2),
		"source_address":    "0x000000000000000000000000000000000000dead",
		"pubkey":            "0xabcdef",
		"meta_network_name": "mainnet",
	})
}

func TestCanonicalBeaconBlockExecutionRequestBuilderExit_RejectsMissingFields(t *testing.T) {
	tests := map[string]func(*xatu.DecoratedEvent){
		"payload": func(e *xatu.DecoratedEvent) {
			e.Data = &xatu.DecoratedEvent_EthV2BeaconBlockExecutionRequestBuilderExit{}
		},
		"source address": func(e *xatu.DecoratedEvent) {
			e.GetEthV2BeaconBlockExecutionRequestBuilderExit().SourceAddress = nil
		},
		"pubkey": func(e *xatu.DecoratedEvent) {
			e.GetEthV2BeaconBlockExecutionRequestBuilderExit().Pubkey = nil
		},
		"position in block": func(e *xatu.DecoratedEvent) {
			e.GetMeta().GetClient().GetEthV2BeaconBlockExecutionRequestBuilderExit().PositionInBlock = nil
		},
		"block number": func(e *xatu.DecoratedEvent) {
			e.GetMeta().GetClient().GetEthV2BeaconBlockExecutionRequestBuilderExit().BlockNumber = nil
		},
		"block hash": func(e *xatu.DecoratedEvent) {
			e.GetMeta().GetClient().GetEthV2BeaconBlockExecutionRequestBuilderExit().BlockHash = ""
		},
		"additional data": func(e *xatu.DecoratedEvent) {
			e.Meta.Client.AdditionalData = nil
		},
	}

	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			event, ok := proto.Clone(builderExitEvent()).(*xatu.DecoratedEvent)
			require.True(t, ok)

			mutate(event)

			batch := newcanonicalBeaconBlockExecutionRequestBuilderExitBatch()
			err := batch.FlattenTo(event)

			require.Error(t, err)
			assert.True(t, errors.Is(err, route.ErrInvalidEvent))
			assert.Equal(t, 0, batch.Rows())
		})
	}
}
