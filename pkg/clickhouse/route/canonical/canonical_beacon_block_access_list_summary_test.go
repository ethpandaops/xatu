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

const testBalHash = "0x77d73a8c93424f86fec3b88a664fe5a13134e1a017a2d5d96eac421773f4baa0"

func blockAccessListSummaryEvent() *xatu.DecoratedEvent {
	return &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_ACCESS_LIST_SUMMARY,
			DateTime: testfixture.TS(),
			Id:       "cbals-1",
		},
		Meta: testfixture.MetaWithAdditional(&xatu.ClientMeta{
			AdditionalData: &xatu.ClientMeta_EthV2BeaconBlockAccessListSummary{
				EthV2BeaconBlockAccessListSummary: &xatu.ClientMeta_AdditionalEthV2BeaconBlockAccessListSummaryData{
					Block: &xatu.BlockIdentifier{
						Epoch:   testfixture.EpochAdditional(),
						Slot:    testfixture.SlotEpochAdditional(),
						Root:    "0x00000000000000000000000000000000000000000000000000000000000000bb",
						Version: "gloas",
					},
					BlockNumber: wrapperspb.UInt64(1234),
					BlockHash:   testBuilderBlockHash,
				},
			},
		}),
		Data: &xatu.DecoratedEvent_EthV2BeaconBlockAccessListSummary{
			EthV2BeaconBlockAccessListSummary: &ethv1.BlockAccessListSummary{
				AccountsTouched:     wrapperspb.UInt32(4),
				StorageSlotsChanged: wrapperspb.UInt32(3),
				StorageChanges:      wrapperspb.UInt32(5),
				StorageReads:        wrapperspb.UInt32(8),
				BalanceChanges:      wrapperspb.UInt32(2),
				NonceChanges:        wrapperspb.UInt32(1),
				CodeChanges:         wrapperspb.UInt32(1),
				TotalChanges:        wrapperspb.UInt32(9),
				BalSizeBytes:        wrapperspb.UInt32(211),
				BalHash:             wrapperspb.String(testBalHash),
			},
		},
	}
}

func TestSnapshot_canonical_beacon_block_access_list_summary(t *testing.T) {
	testfixture.AssertSnapshot(t, newcanonicalBeaconBlockAccessListSummaryBatch(), blockAccessListSummaryEvent(), 1, map[string]any{
		"slot":                  uint32(100),
		"epoch":                 uint32(3),
		"block_root":            "0x00000000000000000000000000000000000000000000000000000000000000bb",
		"block_version":         "gloas",
		"block_number":          uint64(1234),
		"block_hash":            testBuilderBlockHash,
		"accounts_touched":      uint32(4),
		"storage_slots_changed": uint32(3),
		"storage_changes":       uint32(5),
		"storage_reads":         uint32(8),
		"balance_changes":       uint32(2),
		"nonce_changes":         uint32(1),
		"code_changes":          uint32(1),
		"total_changes":         uint32(9),
		"bal_size_bytes":        uint32(211),
		"bal_hash":              testBalHash,
		"meta_network_name":     "mainnet",
	})
}

func TestCanonicalBeaconBlockAccessListSummary_RejectsMissingFields(t *testing.T) {
	tests := map[string]func(*xatu.DecoratedEvent){
		"payload": func(e *xatu.DecoratedEvent) {
			e.Data = &xatu.DecoratedEvent_EthV2BeaconBlockAccessListSummary{}
		},
		"accounts touched": func(e *xatu.DecoratedEvent) {
			e.GetEthV2BeaconBlockAccessListSummary().AccountsTouched = nil
		},
		"storage slots changed": func(e *xatu.DecoratedEvent) {
			e.GetEthV2BeaconBlockAccessListSummary().StorageSlotsChanged = nil
		},
		"total changes": func(e *xatu.DecoratedEvent) {
			e.GetEthV2BeaconBlockAccessListSummary().TotalChanges = nil
		},
		"bal size": func(e *xatu.DecoratedEvent) {
			e.GetEthV2BeaconBlockAccessListSummary().BalSizeBytes = nil
		},
		"bal hash": func(e *xatu.DecoratedEvent) {
			e.GetEthV2BeaconBlockAccessListSummary().BalHash = nil
		},
		"block number": func(e *xatu.DecoratedEvent) {
			e.GetMeta().GetClient().GetEthV2BeaconBlockAccessListSummary().BlockNumber = nil
		},
		"block hash": func(e *xatu.DecoratedEvent) {
			e.GetMeta().GetClient().GetEthV2BeaconBlockAccessListSummary().BlockHash = ""
		},
		"additional data": func(e *xatu.DecoratedEvent) {
			e.Meta.Client.AdditionalData = nil
		},
	}

	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			event, ok := proto.Clone(blockAccessListSummaryEvent()).(*xatu.DecoratedEvent)
			require.True(t, ok)

			mutate(event)

			batch := newcanonicalBeaconBlockAccessListSummaryBatch()
			err := batch.FlattenTo(event)

			require.Error(t, err)
			assert.True(t, errors.Is(err, route.ErrInvalidEvent))
			assert.Equal(t, 0, batch.Rows())
		})
	}
}
