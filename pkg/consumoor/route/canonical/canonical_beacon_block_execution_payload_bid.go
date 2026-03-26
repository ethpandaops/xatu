package canonical

import (
	"fmt"
	"time"

	"github.com/ethpandaops/xatu/pkg/consumoor/route"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

var canonicalBeaconBlockExecutionPayloadBidEventNames = []xatu.Event_Name{
	xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_EXECUTION_PAYLOAD_BID,
}

func init() {
	r, err := route.NewStaticRoute(
		canonicalBeaconBlockExecutionPayloadBidTableName,
		canonicalBeaconBlockExecutionPayloadBidEventNames,
		func() route.ColumnarBatch { return newcanonicalBeaconBlockExecutionPayloadBidBatch() },
	)
	if err != nil {
		route.RecordError(err)

		return
	}

	if err := route.Register(r); err != nil {
		route.RecordError(err)
	}
}

func (b *canonicalBeaconBlockExecutionPayloadBidBatch) FlattenTo(
	event *xatu.DecoratedEvent,
) error {
	if event == nil || event.GetEvent() == nil {
		return nil
	}

	if event.GetEthV2BeaconBlockExecutionPayloadBid() == nil {
		return fmt.Errorf("nil eth_v2_beacon_block_execution_payload_bid payload: %w", route.ErrInvalidEvent)
	}

	b.appendRuntime(event)
	b.appendMetadata(event)
	b.appendPayload(event)
	b.appendAdditionalData(event)
	b.rows++

	return nil
}

func (b *canonicalBeaconBlockExecutionPayloadBidBatch) appendRuntime(_ *xatu.DecoratedEvent) {
	b.UpdatedDateTime.Append(time.Now())
}

func (b *canonicalBeaconBlockExecutionPayloadBidBatch) appendPayload(event *xatu.DecoratedEvent) {
	signed := event.GetEthV2BeaconBlockExecutionPayloadBid()
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

	b.PrevRandao.Append([]byte(bid.GetPrevRandao()))

	//nolint:gosec // G115: length of blob kzg commitments fits uint32
	b.BlobKzgCommitmentCount.Append(uint32(len(bid.GetBlobKzgCommitments())))
}

// TODO: Define AdditionalEthV2BeaconBlockExecutionPayloadBidData proto message to extract
// block identifier (slot/epoch/block_root/block_version) from event.GetMeta().GetClient().
// For now, zero-fill these fields.
func (b *canonicalBeaconBlockExecutionPayloadBidBatch) appendAdditionalData(
	_ *xatu.DecoratedEvent,
) {
	b.Slot.Append(0)
	b.SlotStartDateTime.Append(time.Time{})
	b.Epoch.Append(0)
	b.EpochStartDateTime.Append(time.Time{})
	b.BlockRoot.Append(nil)
	b.BlockVersion.Append("")
}
