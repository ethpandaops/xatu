package beacon

import (
	"strings"
	"testing"

	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/ethpandaops/xatu/pkg/clickhouse/route/testfixture"
	ethv1 "github.com/ethpandaops/xatu/pkg/proto/eth/v1"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

const (
	colBuilderIndex  = "builder_index"
	colExecBlockHash = "block_hash"
	colStateRoot     = "state_root"
	colSlotNumber    = "slot_number"
)

// TestSnapshot_beacon_api_eth_v1_events_execution_payload verifies the
// envelope SSE event flattens to ClickHouse columns. Includes coverage
// for EIP-7843 SLOTNUM — the execution payload's `slot_number` field is
// projected onto the `slot_number` ClickHouse column.
//
// The `slot_number_absent` case exercises the path where the emitting CL
// hasn't populated `slot_number` (e.g., a pre-EIP-7843 CL release on a
// Gloas-aware client): the column must be NULL, not zero.
func TestSnapshot_beacon_api_eth_v1_events_execution_payload(t *testing.T) {
	if len(beaconApiEthV1EventsExecutionPayloadEventNames) == 0 {
		t.Skip("no event names registered for beacon_api_eth_v1_events_execution_payload")
	}

	cases := []struct {
		name       string
		slotNumber *uint64 // nil → CL sent the envelope without slot_number
	}{
		{
			name:       "slot_number_present",
			slotNumber: uint64Ptr(8_675_309),
		},
		{
			name:       "slot_number_absent",
			slotNumber: nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			payload := &ethv1.ExecutionPayloadGloas{
				ParentHash:    "0x3333333333333333333333333333333333333333333333333333333333333333",
				FeeRecipient:  "0x1111111111111111111111111111111111111111",
				StateRoot:     "0xffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
				ReceiptsRoot:  "0x2222222222222222222222222222222222222222222222222222222222222222",
				LogsBloom:     "0x" + strings.Repeat("00", 256),
				PrevRandao:    "0x0f10000000000000000000000000000000000000000000000000000000000000",
				BlockNumber:   wrapperspb.UInt64(123456),
				GasLimit:      wrapperspb.UInt64(30_000_000),
				GasUsed:       wrapperspb.UInt64(21_000),
				Timestamp:     wrapperspb.UInt64(1_700_000_000),
				BaseFeePerGas: "7",
				BlockHash:     "0xeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee",
				BlobGasUsed:   wrapperspb.UInt64(131072),
				ExcessBlobGas: wrapperspb.UInt64(0),
			}
			if tc.slotNumber != nil {
				payload.SlotNumber = wrapperspb.UInt64(*tc.slotNumber)
			}

			envelope := &ethv1.SignedExecutionPayloadEnvelope{
				Message: &ethv1.ExecutionPayloadEnvelope{
					BuilderIndex:    wrapperspb.UInt64(42),
					BeaconBlockRoot: "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
					Payload:         payload,
				},
				Signature: "0xdead",
			}

			expected := map[string]any{
				testfixture.MetaClientNameKey: testfixture.MetaClientName,
				colBuilderIndex:               uint64(42),
				colExecBlockHash:              "0xeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee",
				colStateRoot:                  "0xffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
				colSlotNumber:                 nullableUint64Expected(tc.slotNumber),
			}

			testfixture.AssertSnapshot(t, newbeaconApiEthV1EventsExecutionPayloadBatch(), &xatu.DecoratedEvent{
				Event: &xatu.Event{
					Name:     beaconApiEthV1EventsExecutionPayloadEventNames[0],
					DateTime: testfixture.TS(),
					Id:       testfixture.SnapshotID,
				},
				Meta: testfixture.BaseMeta(),
				Data: &xatu.DecoratedEvent_EthV1EventsExecutionPayload{
					EthV1EventsExecutionPayload: envelope,
				},
			}, 1, expected)
		})
	}
}
