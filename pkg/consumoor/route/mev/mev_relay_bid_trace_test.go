package mev

import (
	"testing"

	"github.com/ethpandaops/xatu/pkg/consumoor/route/testfixture"
	mevrelay "github.com/ethpandaops/xatu/pkg/proto/mevrelay"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func TestSnapshot_mev_relay_bid_trace(t *testing.T) {
	testfixture.AssertSnapshot(t, newmevRelayBidTraceBatch(), &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_MEV_RELAY_BID_TRACE_BUILDER_BLOCK_SUBMISSION,
			DateTime: testfixture.TS(),
			Id:       "mev-1",
		},
		Meta: testfixture.MetaWithAdditional(&xatu.ClientMeta{
			AdditionalData: &xatu.ClientMeta_MevRelayBidTraceBuilderBlockSubmission{
				MevRelayBidTraceBuilderBlockSubmission: &xatu.ClientMeta_AdditionalMevRelayBidTraceBuilderBlockSubmissionData{
					Slot:  testfixture.SlotEpochAdditional(),
					Epoch: testfixture.EpochAdditional(),
					Relay: &mevrelay.Relay{Name: wrapperspb.String("flashbots")},
				},
			},
		}),
		Data: &xatu.DecoratedEvent_MevRelayBidTraceBuilderBlockSubmission{
			MevRelayBidTraceBuilderBlockSubmission: &mevrelay.BidTrace{
				Slot:                 wrapperspb.UInt64(100),
				BlockNumber:          wrapperspb.UInt64(1000),
				BuilderPubkey:        wrapperspb.String("0xbuilder"),
				ProposerPubkey:       wrapperspb.String("0xproposer"),
				ProposerFeeRecipient: wrapperspb.String("0xfee"),
				GasLimit:             wrapperspb.UInt64(30000000),
				GasUsed:              wrapperspb.UInt64(15000000),
				Value:                wrapperspb.String("1000000000000000000"),
				NumTx:                wrapperspb.UInt64(50),
				Timestamp:            wrapperspb.Int64(1705312800),
				TimestampMs:          wrapperspb.Int64(1705312800000),
				OptimisticSubmission: wrapperspb.Bool(false),
			},
		},
	}, 1, map[string]any{
		"slot":           uint32(100),
		"builder_pubkey": "0xbuilder",
		"relay_name":     "flashbots",
	})
}
