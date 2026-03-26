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

// TODO: Define AdditionalEthV1EventsExecutionPayloadBidData proto message to extract
// slot/epoch/propagation from event.GetMeta().GetClient(). For now, zero-fill these fields.
func (b *beaconApiEthV1EventsExecutionPayloadBidBatch) appendAdditionalData(
	_ *xatu.DecoratedEvent,
) {
	b.Slot.Append(0)
	b.SlotStartDateTime.Append(time.Time{})
	b.PropagationSlotStartDiff.Append(0)
	b.Epoch.Append(0)
	b.EpochStartDateTime.Append(time.Time{})
}
