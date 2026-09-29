package beacon

import (
	"testing"

	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/ethpandaops/xatu/pkg/clickhouse/route/testfixture"
	ethv1 "github.com/ethpandaops/xatu/pkg/proto/eth/v1"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

const (
	colBlockRoot  = "block_root"
	colBlockHash  = "block_hash"
	statusPENDING = "PENDING"

	colSlot                            = "slot"
	colStatus                          = "status"
	colPreviousStatus                  = "previous_status"
	colPayloadTimelinessVotesPositive  = "payload_timeliness_votes_positive"
	colPayloadTimelinessVotesNegative_ = "payload_timeliness_votes_negative"
	colPayloadTimelinessVotesAbsent_   = "payload_timeliness_votes_absent"
	colDataAvailableVotesPositive_     = "data_available_votes_positive"
	colDataAvailableVotesNegative_     = "data_available_votes_negative"
	colDataAvailableVotesAbsent_       = "data_available_votes_absent"
	colPtcSize_                        = "ptc_size"

	blockRoot64A = "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	blockHash64B = "0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	blockRoot64C = "0xcccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"
	blockHash64D = "0xdddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd"
)

// uint64Ptr returns a pointer to v. Used to populate the optional
// three-state PTC vote counts when the test scenario simulates a CL that
// surfaces the breakdown (PR #5180).
func uint64Ptr(v uint64) *uint64 { return &v }

// TestSnapshot_beacon_synthetic_payload_status_resolved verifies that the
// fork-choice payload-status-resolved synthetic event plumbs through to
// ClickHouse columns correctly, including the three-state PTC vote
// breakdown (positive / negative / absent) introduced by consensus-specs
// PR #5180.
//
// Two scenarios are exercised:
//
//   - known: the emitting CL exposes the full Optional[boolean] breakdown
//     and all six counts are non-nil. `positive + negative + absent ==
//     ptc_size` for both metrics.
//
//   - unknown: the CL doesn't surface the breakdown — only positive
//     counts are populated; the negative and absent counts are NULL.
func TestSnapshot_beacon_synthetic_payload_status_resolved(t *testing.T) {
	if len(beaconSyntheticPayloadStatusResolvedEventNames) == 0 {
		t.Skip("no event names registered for beacon_synthetic_payload_status_resolved")
	}

	cases := []struct {
		name           string
		slot           uint64
		blockRoot      string
		blockHash      string
		status         ethv1.PayloadStatus
		statusStr      string
		previousStatus ethv1.PayloadStatus

		payloadPositive uint64
		payloadNegative *uint64
		payloadAbsent   *uint64

		dataPositive uint64
		dataNegative *uint64
		dataAbsent   *uint64

		ptcSize uint64
	}{
		{
			name:            "three_state_breakdown_known",
			slot:            12345,
			blockRoot:       blockRoot64A,
			blockHash:       blockHash64B,
			status:          ethv1.PayloadStatus_PAYLOAD_STATUS_FULL,
			statusStr:       "FULL",
			previousStatus:  ethv1.PayloadStatus_PAYLOAD_STATUS_PENDING,
			payloadPositive: 400,
			payloadNegative: uint64Ptr(50),
			payloadAbsent:   uint64Ptr(62),
			dataPositive:    380,
			dataNegative:    uint64Ptr(70),
			dataAbsent:      uint64Ptr(62),
			ptcSize:         512,
		},
		{
			name:            "three_state_breakdown_unknown",
			slot:            12346,
			blockRoot:       blockRoot64C,
			blockHash:       blockHash64D,
			status:          ethv1.PayloadStatus_PAYLOAD_STATUS_EMPTY,
			statusStr:       "EMPTY",
			previousStatus:  ethv1.PayloadStatus_PAYLOAD_STATUS_PENDING,
			payloadPositive: 100,
			payloadNegative: nil,
			payloadAbsent:   nil,
			dataPositive:    90,
			dataNegative:    nil,
			dataAbsent:      nil,
			ptcSize:         512,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			payload := &ethv1.PayloadStatusResolved{
				Slot:                           wrapperspb.UInt64(tc.slot),
				BlockRoot:                      tc.blockRoot,
				BlockHash:                      tc.blockHash,
				Status:                         tc.status,
				PreviousStatus:                 tc.previousStatus,
				PayloadTimelinessVotesPositive: wrapperspb.UInt64(tc.payloadPositive),
				DataAvailableVotesPositive:     wrapperspb.UInt64(tc.dataPositive),
				PtcSize:                        wrapperspb.UInt64(tc.ptcSize),
			}
			if tc.payloadNegative != nil {
				payload.PayloadTimelinessVotesNegative = wrapperspb.UInt64(*tc.payloadNegative)
			}

			if tc.payloadAbsent != nil {
				payload.PayloadTimelinessVotesAbsent = wrapperspb.UInt64(*tc.payloadAbsent)
			}

			if tc.dataNegative != nil {
				payload.DataAvailableVotesNegative = wrapperspb.UInt64(*tc.dataNegative)
			}

			if tc.dataAbsent != nil {
				payload.DataAvailableVotesAbsent = wrapperspb.UInt64(*tc.dataAbsent)
			}

			expected := map[string]any{
				testfixture.MetaClientNameKey:      testfixture.MetaClientName,
				colSlot:                            uint32(tc.slot),
				colBlockRoot:                       tc.blockRoot,
				colBlockHash:                       tc.blockHash,
				colStatus:                          tc.statusStr,
				colPreviousStatus:                  statusPENDING,
				colPayloadTimelinessVotesPositive:  tc.payloadPositive,
				colPayloadTimelinessVotesNegative_: nullableUint64Expected(tc.payloadNegative),
				colPayloadTimelinessVotesAbsent_:   nullableUint64Expected(tc.payloadAbsent),
				colDataAvailableVotesPositive_:     tc.dataPositive,
				colDataAvailableVotesNegative_:     nullableUint64Expected(tc.dataNegative),
				colDataAvailableVotesAbsent_:       nullableUint64Expected(tc.dataAbsent),
				colPtcSize_:                        tc.ptcSize,
			}

			testfixture.AssertSnapshot(t, newbeaconSyntheticPayloadStatusResolvedBatch(), &xatu.DecoratedEvent{
				Event: &xatu.Event{
					Name:     beaconSyntheticPayloadStatusResolvedEventNames[0],
					DateTime: testfixture.TS(),
					Id:       testfixture.SnapshotID,
				},
				Meta: testfixture.BaseMeta(),
				Data: &xatu.DecoratedEvent_BeaconSyntheticPayloadStatusResolved{
					BeaconSyntheticPayloadStatusResolved: payload,
				},
			}, 1, expected)
		})
	}
}

// nullableUint64Expected returns the expected snapshot value for an
// Optional[uint64] column — the dereferenced value when non-nil, otherwise
// nil (which the snapshot helper compares to a SQL NULL projection).
func nullableUint64Expected(v *uint64) any {
	if v == nil {
		return nil
	}

	return *v
}
