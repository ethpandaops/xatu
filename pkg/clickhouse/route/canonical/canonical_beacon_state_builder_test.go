package canonical

import (
	"testing"

	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/ethpandaops/xatu/pkg/clickhouse/route/testfixture"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

func TestSnapshot_canonical_beacon_state_builder(t *testing.T) {
	pubkey := "0x" + repeatHex("ab", 48)

	testfixture.AssertSnapshot(t, newcanonicalBeaconStateBuilderBatch(), &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_BEACON_API_ETH_V1_BEACON_STATE_BUILDER,
			DateTime: testfixture.TS(),
			Id:       "builder-1",
		},
		Meta: testfixture.MetaWithAdditional(&xatu.ClientMeta{
			AdditionalData: &xatu.ClientMeta_EthV1BeaconStateBuilder{
				EthV1BeaconStateBuilder: &xatu.ClientMeta_AdditionalEthV1BeaconStateBuilderData{
					Epoch:   testfixture.EpochAdditional(),
					StateId: "96",
				},
			},
		}),
		Data: &xatu.DecoratedEvent_EthV1BeaconStateBuilder{
			EthV1BeaconStateBuilder: &xatu.BuilderData{
				Index:             wrapperspb.UInt64(12),
				Pubkey:            pubkey,
				Version:           wrapperspb.UInt32(0),
				ExecutionAddress:  "0x67565aa29bcf60c3fdcd0b79a36cd85cc13c7a91",
				Balance:           wrapperspb.UInt64(39_550_000_000),
				DepositEpoch:      wrapperspb.UInt64(1434),
				WithdrawableEpoch: wrapperspb.UInt64(18446744073709551615),
				Status:            "active",
			},
		},
	}, 1, map[string]any{
		"epoch":              uint32(3),
		"state_id":           "96",
		"builder_index":      uint64(12),
		"pubkey":             pubkey,
		"version":            uint8(0),
		"execution_address":  "0x67565aa29bcf60c3fdcd0b79a36cd85cc13c7a91",
		"balance":            uint64(39_550_000_000),
		"deposit_epoch":      uint64(1434),
		"withdrawable_epoch": uint64(18446744073709551615),
		"status":             "active",
	})
}

func repeatHex(b string, n int) string {
	out := ""
	for i := 0; i < n; i++ {
		out += b
	}

	return out
}
