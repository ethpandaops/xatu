package beacon

import (
	"fmt"
	"time"

	"github.com/ClickHouse/ch-go/proto"
	"github.com/ethpandaops/xatu/pkg/clickhouse/route"
	ethv1 "github.com/ethpandaops/xatu/pkg/proto/eth/v1"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

var beaconSyntheticPayloadStatusResolvedEventNames = []xatu.Event_Name{
	xatu.Event_BEACON_SYNTHETIC_PAYLOAD_STATUS_RESOLVED,
}

func init() {
	r, err := route.NewStaticRoute(
		beaconSyntheticPayloadStatusResolvedTableName,
		beaconSyntheticPayloadStatusResolvedEventNames,
		func() route.ColumnarBatch { return newbeaconSyntheticPayloadStatusResolvedBatch() },
	)
	if err != nil {
		route.RecordError(err)

		return
	}

	if err := route.Register(r); err != nil {
		route.RecordError(err)
	}
}

func (b *beaconSyntheticPayloadStatusResolvedBatch) FlattenTo(
	event *xatu.DecoratedEvent,
) error {
	if event == nil || event.GetEvent() == nil {
		return nil
	}

	if event.GetBeaconSyntheticPayloadStatusResolved() == nil {
		return fmt.Errorf(
			"nil beacon_synthetic_payload_status_resolved payload: %w",
			route.ErrInvalidEvent,
		)
	}

	b.appendRuntime(event)
	b.appendMetadata(event)
	b.appendPayload(event)
	b.appendAdditionalData(event)
	b.rows++

	return nil
}

func (b *beaconSyntheticPayloadStatusResolvedBatch) appendRuntime(
	event *xatu.DecoratedEvent,
) {
	b.UpdatedDateTime.Append(time.Now())

	if ts := event.GetEvent().GetDateTime(); ts != nil {
		b.EventDateTime.Append(ts.AsTime())
	} else {
		b.EventDateTime.Append(time.Time{})
	}
}

func (b *beaconSyntheticPayloadStatusResolvedBatch) appendPayload(
	event *xatu.DecoratedEvent,
) {
	payload := event.GetBeaconSyntheticPayloadStatusResolved()

	if slot := payload.GetSlot(); slot != nil {
		b.Slot.Append(uint32(slot.GetValue())) //nolint:gosec // slot values fit uint32
	} else {
		b.Slot.Append(0)
	}

	b.BlockRoot.Append([]byte(payload.GetBlockRoot()))
	b.BlockHash.Append([]byte(payload.GetBlockHash()))
	b.Status.Append(payloadStatusEnumName(payload.GetStatus()))
	b.PreviousStatus.Append(payloadStatusEnumName(payload.GetPreviousStatus()))

	if v := payload.GetPayloadTimelinessVotesPositive(); v != nil {
		b.PayloadTimelinessVotesPositive.Append(v.GetValue())
	} else {
		b.PayloadTimelinessVotesPositive.Append(0)
	}

	if v := payload.GetPayloadTimelinessVotesNegative(); v != nil {
		b.PayloadTimelinessVotesNegative.Append(proto.NewNullable[uint64](v.GetValue()))
	} else {
		b.PayloadTimelinessVotesNegative.Append(proto.Nullable[uint64]{})
	}

	if v := payload.GetPayloadTimelinessVotesAbsent(); v != nil {
		b.PayloadTimelinessVotesAbsent.Append(proto.NewNullable[uint64](v.GetValue()))
	} else {
		b.PayloadTimelinessVotesAbsent.Append(proto.Nullable[uint64]{})
	}

	if v := payload.GetDataAvailableVotesPositive(); v != nil {
		b.DataAvailableVotesPositive.Append(v.GetValue())
	} else {
		b.DataAvailableVotesPositive.Append(0)
	}

	if v := payload.GetDataAvailableVotesNegative(); v != nil {
		b.DataAvailableVotesNegative.Append(proto.NewNullable[uint64](v.GetValue()))
	} else {
		b.DataAvailableVotesNegative.Append(proto.Nullable[uint64]{})
	}

	if v := payload.GetDataAvailableVotesAbsent(); v != nil {
		b.DataAvailableVotesAbsent.Append(proto.NewNullable[uint64](v.GetValue()))
	} else {
		b.DataAvailableVotesAbsent.Append(proto.Nullable[uint64]{})
	}

	if v := payload.GetPtcSize(); v != nil {
		b.PtcSize.Append(v.GetValue())
	} else {
		b.PtcSize.Append(0)
	}
}

func (b *beaconSyntheticPayloadStatusResolvedBatch) appendAdditionalData(
	event *xatu.DecoratedEvent,
) {
	if event.GetMeta() == nil || event.GetMeta().GetClient() == nil {
		b.SlotStartDateTime.Append(time.Time{})
		b.PropagationSlotStartDiff.Append(0)
		b.Epoch.Append(0)
		b.EpochStartDateTime.Append(time.Time{})

		return
	}

	additional := extractBeaconSlotEpochPropagation(event.GetMeta().GetClient().GetBeaconSyntheticPayloadStatusResolved())

	b.SlotStartDateTime.Append(time.Unix(additional.SlotStartDateTime, 0))
	b.PropagationSlotStartDiff.Append(uint32(additional.PropagationSlotStartDiff)) //nolint:gosec // propagation diff fits uint32
	b.Epoch.Append(uint32(additional.Epoch))                                       //nolint:gosec // epoch fits uint32
	b.EpochStartDateTime.Append(time.Unix(additional.EpochStartDateTime, 0))
}

// payloadStatusEnumName returns the human-readable name for the payload status
// enum used in the ClickHouse table (matches the migration's LowCardinality(String)).
func payloadStatusEnumName(s ethv1.PayloadStatus) string {
	switch s {
	case ethv1.PayloadStatus_PAYLOAD_STATUS_PENDING:
		return "PENDING"
	case ethv1.PayloadStatus_PAYLOAD_STATUS_FULL:
		return "FULL"
	case ethv1.PayloadStatus_PAYLOAD_STATUS_EMPTY:
		return "EMPTY"
	case ethv1.PayloadStatus_PAYLOAD_STATUS_INVALID:
		return "INVALID"
	default:
		return "UNKNOWN"
	}
}
