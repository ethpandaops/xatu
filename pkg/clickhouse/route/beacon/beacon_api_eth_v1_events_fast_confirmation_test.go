package beacon

import (
	"testing"

	"github.com/ethpandaops/xatu/pkg/clickhouse/route/testfixture"
	ethv1 "github.com/ethpandaops/xatu/pkg/proto/eth/v1"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func TestSnapshot_beacon_api_eth_v1_events_fast_confirmation(t *testing.T) {
	testfixture.AssertSnapshot(t, newbeaconApiEthV1EventsFastConfirmationBatch(), &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_BEACON_API_ETH_V1_EVENTS_FAST_CONFIRMATION,
			DateTime: testfixture.TS(),
			Id:       "fast-confirmation-1",
		},
		Meta: testfixture.MetaWithAdditional(&xatu.ClientMeta{
			AdditionalData: &xatu.ClientMeta_EthV1EventsFastConfirmation{
				EthV1EventsFastConfirmation: &xatu.ClientMeta_AdditionalEthV1EventsFastConfirmationData{
					Slot:           testfixture.SlotEpochAdditional(),
					Epoch:          testfixture.EpochAdditional(),
					Propagation:    testfixture.PropagationAdditional(),
					WallclockSlot:  testfixture.WallclockSlotAdditional(),
					WallclockEpoch: testfixture.WallclockEpochAdditional(),
				},
			},
		}),
		Data: &xatu.DecoratedEvent_EthV1EventsFastConfirmation{
			EthV1EventsFastConfirmation: &ethv1.EventFastConfirmation{
				Slot:  wrapperspb.UInt64(100),
				Block: "0xfastconfirmblock",
			},
		},
	}, 1, map[string]any{
		"slot":                        uint32(100),          //nolint:goconst // shared with sibling event tests
		"block":                       "0xfastconfirmblock", //nolint:goconst // shared with sibling event tests
		"meta_client_name":            "test-client",        //nolint:goconst // shared with sibling event tests
		"meta_network_name":           "mainnet",            //nolint:goconst // shared with sibling event tests
		"propagation_slot_start_diff": uint32(500),
		"epoch":                       uint32(3),
		"wallclock_slot":              uint32(100),
		"wallclock_epoch":             uint32(3),
	})
}

func TestSnapshot_beacon_api_eth_v1_events_fast_confirmation_no_additional_data(t *testing.T) {
	testfixture.AssertSnapshot(t, newbeaconApiEthV1EventsFastConfirmationBatch(), &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_BEACON_API_ETH_V1_EVENTS_FAST_CONFIRMATION,
			DateTime: testfixture.TS(),
			Id:       "fast-confirmation-2",
		},
		Meta: testfixture.MetaWithAdditional(&xatu.ClientMeta{}),
		Data: &xatu.DecoratedEvent_EthV1EventsFastConfirmation{
			EthV1EventsFastConfirmation: &ethv1.EventFastConfirmation{
				Slot:  wrapperspb.UInt64(100),
				Block: "0xfastconfirmblock",
			},
		},
	}, 1, map[string]any{
		"slot":                            uint32(100),
		"block":                           "0xfastconfirmblock",
		"propagation_slot_start_diff":     nil,
		"epoch":                           nil,
		"epoch_start_date_time":           nil,
		"wallclock_slot":                  nil,
		"wallclock_slot_start_date_time":  nil,
		"wallclock_epoch":                 nil,
		"wallclock_epoch_start_date_time": nil,
	})
}
