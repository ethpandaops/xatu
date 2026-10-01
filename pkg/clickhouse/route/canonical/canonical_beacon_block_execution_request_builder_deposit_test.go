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

const testBuilderBlockHash = "0x00000000000000000000000000000000000000000000000000000000000000aa"

func builderDepositEvent() *xatu.DecoratedEvent {
	return &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_EXECUTION_REQUEST_BUILDER_DEPOSIT,
			DateTime: testfixture.TS(),
			Id:       "cberbd-1",
		},
		Meta: testfixture.MetaWithAdditional(&xatu.ClientMeta{
			AdditionalData: &xatu.ClientMeta_EthV2BeaconBlockExecutionRequestBuilderDeposit{
				EthV2BeaconBlockExecutionRequestBuilderDeposit: &xatu.ClientMeta_AdditionalEthV2BeaconBlockExecutionRequestBuilderDepositData{
					Block: &xatu.BlockIdentifier{
						Epoch:   testfixture.EpochAdditional(),
						Slot:    testfixture.SlotEpochAdditional(),
						Root:    "0x00000000000000000000000000000000000000000000000000000000000000bb",
						Version: "gloas",
					},
					PositionInBlock: wrapperspb.UInt64(1),
					BlockNumber:     wrapperspb.UInt64(1234),
					BlockHash:       testBuilderBlockHash,
				},
			},
		}),
		Data: &xatu.DecoratedEvent_EthV2BeaconBlockExecutionRequestBuilderDeposit{
			EthV2BeaconBlockExecutionRequestBuilderDeposit: &ethv1.GloasBuilderDepositRequest{
				Pubkey:                wrapperspb.String("0xabc"),
				WithdrawalCredentials: wrapperspb.String("0x0300000000000000000000000000000000000000000000000000000000000def"),
				Amount:                wrapperspb.UInt64(1000000000),
				Signature:             wrapperspb.String("0x123"),
			},
		},
	}
}

func TestSnapshot_canonical_beacon_block_execution_request_builder_deposit(t *testing.T) {
	testfixture.AssertSnapshot(t, newcanonicalBeaconBlockExecutionRequestBuilderDepositBatch(), builderDepositEvent(), 1, map[string]any{
		"slot":                   uint32(100),
		"epoch":                  uint32(3),
		"block_root":             "0x00000000000000000000000000000000000000000000000000000000000000bb",
		"block_version":          "gloas",
		"block_number":           uint64(1234),
		"block_hash":             testBuilderBlockHash,
		"position_in_block":      uint32(1),
		"pubkey":                 "0xabc",
		"withdrawal_credentials": "0x0300000000000000000000000000000000000000000000000000000000000def",
		"amount":                 "1000000000",
		"signature":              "0x123",
		"meta_network_name":      "mainnet",
	})
}

func TestCanonicalBeaconBlockExecutionRequestBuilderDeposit_RejectsMissingFields(t *testing.T) {
	tests := map[string]func(*xatu.DecoratedEvent){
		"payload": func(e *xatu.DecoratedEvent) {
			e.Data = &xatu.DecoratedEvent_EthV2BeaconBlockExecutionRequestBuilderDeposit{}
		},
		"pubkey": func(e *xatu.DecoratedEvent) {
			e.GetEthV2BeaconBlockExecutionRequestBuilderDeposit().Pubkey = nil
		},
		"withdrawal credentials": func(e *xatu.DecoratedEvent) {
			e.GetEthV2BeaconBlockExecutionRequestBuilderDeposit().WithdrawalCredentials = nil
		},
		"amount": func(e *xatu.DecoratedEvent) {
			e.GetEthV2BeaconBlockExecutionRequestBuilderDeposit().Amount = nil
		},
		"signature": func(e *xatu.DecoratedEvent) {
			e.GetEthV2BeaconBlockExecutionRequestBuilderDeposit().Signature = nil
		},
		"position in block": func(e *xatu.DecoratedEvent) {
			e.GetMeta().GetClient().GetEthV2BeaconBlockExecutionRequestBuilderDeposit().PositionInBlock = nil
		},
		"block number": func(e *xatu.DecoratedEvent) {
			e.GetMeta().GetClient().GetEthV2BeaconBlockExecutionRequestBuilderDeposit().BlockNumber = nil
		},
		"block hash": func(e *xatu.DecoratedEvent) {
			e.GetMeta().GetClient().GetEthV2BeaconBlockExecutionRequestBuilderDeposit().BlockHash = ""
		},
		"additional data": func(e *xatu.DecoratedEvent) {
			e.Meta.Client.AdditionalData = nil
		},
	}

	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			event, ok := proto.Clone(builderDepositEvent()).(*xatu.DecoratedEvent)
			require.True(t, ok)

			mutate(event)

			batch := newcanonicalBeaconBlockExecutionRequestBuilderDepositBatch()
			err := batch.FlattenTo(event)

			require.Error(t, err)
			assert.True(t, errors.Is(err, route.ErrInvalidEvent))
			assert.Equal(t, 0, batch.Rows())
		})
	}
}
