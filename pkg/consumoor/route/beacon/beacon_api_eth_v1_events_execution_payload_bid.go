package beacon

import (
	"fmt"
	"time"

	"github.com/ethpandaops/xatu/pkg/consumoor/route"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

var beaconApiEthV1EventsExecutionPayloadBidEventNames = []xatu.Event_Name{
	xatu.Event_BEACON_API_ETH_V1_EVENTS_EXECUTION_PAYLOAD_BID,
}

func init() {
	r, err := route.NewStaticRoute(
		beaconApiEthV1EventsExecutionPayloadBidTableName,
		beaconApiEthV1EventsExecutionPayloadBidEventNames,
		func() route.ColumnarBatch { return newbeaconApiEthV1EventsExecutionPayloadBidBatch() },
	)
	if err != nil {
		route.RecordError(err)

		return
	}

	if err := route.Register(r); err != nil {
		route.RecordError(err)
	}
}

func (b *beaconApiEthV1EventsExecutionPayloadBidBatch) FlattenTo(
	event *xatu.DecoratedEvent,
) error {
	if event == nil || event.GetEvent() == nil {
		return nil
	}

	if event.GetEthV1EventsExecutionPayloadBid() == nil {
		return fmt.Errorf("nil eth_v1_events_execution_payload_bid payload: %w", route.ErrInvalidEvent)
	}

	b.appendRuntime(event)
	b.appendMetadata(event)
	b.appendPayload(event)
	b.appendAdditionalData(event)
	b.rows++

	return nil
}

func (b *beaconApiEthV1EventsExecutionPayloadBidBatch) appendRuntime(event *xatu.DecoratedEvent) {
	b.UpdatedDateTime.Append(time.Now())

	if ts := event.GetEvent().GetDateTime(); ts != nil {
		b.EventDateTime.Append(ts.AsTime())
	} else {
		b.EventDateTime.Append(time.Time{})
	}
}

func (b *beaconApiEthV1EventsExecutionPayloadBidBatch) appendPayload(event *xatu.DecoratedEvent) {
	signed := event.GetEthV1EventsExecutionPayloadBid()
	bid := signed.GetMessage()

	if builderIndex := bid.GetBuilderIndex(); builderIndex != nil {
		b.BuilderIndex.Append(builderIndex.GetValue())
	} else {
		b.BuilderIndex.Append(0)
	}

	b.BlockHash.Append([]byte(bid.GetBlockHash()))
	b.ParentBlockHash.Append([]byte(bid.GetParentBlockHash()))
	b.ParentBlockRoot.Append([]byte(bid.GetParentBlockRoot()))

	if value := bid.GetValue(); value != nil {
		b.Value.Append(value.GetValue())
	} else {
		b.Value.Append(0)
	}

	if executionPayment := bid.GetExecutionPayment(); executionPayment != nil {
		b.ExecutionPayment.Append(executionPayment.GetValue())
	} else {
		b.ExecutionPayment.Append(0)
	}

	b.FeeRecipient.Append([]byte(bid.GetFeeRecipient()))

	if gasLimit := bid.GetGasLimit(); gasLimit != nil {
		b.GasLimit.Append(gasLimit.GetValue())
	} else {
		b.GasLimit.Append(0)
	}

	//nolint:gosec // G115: length of blob kzg commitments fits uint32
	b.BlobKzgCommitmentCount.Append(uint32(len(bid.GetBlobKzgCommitments())))
}

func (b *beaconApiEthV1EventsExecutionPayloadBidBatch) appendAdditionalData(
	event *xatu.DecoratedEvent,
) {
	if event.GetMeta() == nil || event.GetMeta().GetClient() == nil {
		b.Slot.Append(0)
		b.SlotStartDateTime.Append(time.Time{})
		b.PropagationSlotStartDiff.Append(0)
		b.Epoch.Append(0)
		b.EpochStartDateTime.Append(time.Time{})

		return
	}

	client := event.GetMeta().GetClient()
	additionalV2 := client.GetEthV1EventsExecutionPayloadBid()
	additional := extractBeaconSlotEpochPropagation(additionalV2)

	if additionalV2 != nil {
		if slot := additionalV2.GetSlot(); slot != nil {
			if num := slot.GetNumber(); num != nil {
				b.Slot.Append(uint32(num.GetValue())) //nolint:gosec // slot fits uint32
			} else {
				b.Slot.Append(0)
			}
		} else {
			b.Slot.Append(0)
		}
	} else {
		b.Slot.Append(0)
	}

	b.SlotStartDateTime.Append(time.Unix(additional.SlotStartDateTime, 0))
	b.PropagationSlotStartDiff.Append(uint32(additional.PropagationSlotStartDiff)) //nolint:gosec // propagation diff fits uint32
	b.Epoch.Append(uint32(additional.Epoch))                                       //nolint:gosec // epoch fits uint32
	b.EpochStartDateTime.Append(time.Unix(additional.EpochStartDateTime, 0))
}
