package libp2p

import (
	"fmt"
	"time"

	"github.com/ethpandaops/xatu/pkg/consumoor/route"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

var libp2pGossipsubExecutionPayloadBidEventNames = []xatu.Event_Name{
	xatu.Event_LIBP2P_TRACE_GOSSIPSUB_EXECUTION_PAYLOAD_BID,
}

func init() {
	r, err := route.NewStaticRoute(
		libp2pGossipsubExecutionPayloadBidTableName,
		libp2pGossipsubExecutionPayloadBidEventNames,
		func() route.ColumnarBatch { return newlibp2pGossipsubExecutionPayloadBidBatch() },
	)
	if err != nil {
		route.RecordError(err)

		return
	}

	if err := route.Register(r); err != nil {
		route.RecordError(err)
	}
}

func (b *libp2pGossipsubExecutionPayloadBidBatch) FlattenTo(
	event *xatu.DecoratedEvent,
) error {
	if event == nil || event.GetEvent() == nil {
		return nil
	}

	if event.GetLibp2PTraceGossipsubExecutionPayloadBid() == nil {
		return fmt.Errorf(
			"nil libp2p_trace_gossipsub_execution_payload_bid payload: %w",
			route.ErrInvalidEvent,
		)
	}

	b.appendRuntime(event)
	b.appendMetadata(event)
	b.appendPayload(event)
	b.appendClientAdditionalData(event)
	b.rows++

	return nil
}

func (b *libp2pGossipsubExecutionPayloadBidBatch) appendRuntime(event *xatu.DecoratedEvent) {
	b.UpdatedDateTime.Append(time.Now())

	if ts := event.GetEvent().GetDateTime(); ts != nil {
		b.EventDateTime.Append(ts.AsTime())
	} else {
		b.EventDateTime.Append(time.Time{})
	}
}

func (b *libp2pGossipsubExecutionPayloadBidBatch) appendPayload(event *xatu.DecoratedEvent) {
	payload := event.GetLibp2PTraceGossipsubExecutionPayloadBid()

	if builderIndex := payload.GetBuilderIndex(); builderIndex != nil {
		b.BuilderIndex.Append(builderIndex.GetValue())
	} else {
		b.BuilderIndex.Append(0)
	}

	b.BlockHash.Append([]byte(wrappedStringValue(payload.GetBlockHash())))
	b.ParentBlockHash.Append([]byte(wrappedStringValue(payload.GetParentBlockHash())))

	if value := payload.GetValue(); value != nil {
		b.Value.Append(value.GetValue())
	} else {
		b.Value.Append(0)
	}

	if executionPayment := payload.GetExecutionPayment(); executionPayment != nil {
		b.ExecutionPayment.Append(executionPayment.GetValue())
	} else {
		b.ExecutionPayment.Append(0)
	}

	b.FeeRecipient.Append([]byte(wrappedStringValue(payload.GetFeeRecipient())))

	if gasLimit := payload.GetGasLimit(); gasLimit != nil {
		b.GasLimit.Append(gasLimit.GetValue())
	} else {
		b.GasLimit.Append(0)
	}

	if blobKzgCount := payload.GetBlobKzgCommitmentCount(); blobKzgCount != nil {
		b.BlobKzgCommitmentCount.Append(blobKzgCount.GetValue())
	} else {
		b.BlobKzgCommitmentCount.Append(0)
	}
}

// TODO: Define AdditionalLibp2pTraceGossipsubExecutionPayloadBidData proto message
// to extract slot/epoch/wallclock/propagation/peer/message/topic fields from
// event.GetMeta().GetClient(). For now, zero-fill these fields.
func (b *libp2pGossipsubExecutionPayloadBidBatch) appendClientAdditionalData(
	_ *xatu.DecoratedEvent,
) {
	b.Slot.Append(0)
	b.SlotStartDateTime.Append(time.Time{})
	b.Epoch.Append(0)
	b.EpochStartDateTime.Append(time.Time{})
	b.WallclockSlot.Append(0)
	b.WallclockSlotStartDateTime.Append(time.Time{})
	b.WallclockEpoch.Append(0)
	b.WallclockEpochStartDateTime.Append(time.Time{})
	b.PropagationSlotStartDiff.Append(0)
	b.Version.Append(4294967295)
	b.MessageID.Append("")
	b.MessageSize.Append(0)
	b.TopicLayer.Append("")
	b.TopicForkDigestValue.Append("")
	b.TopicName.Append("")
	b.TopicEncoding.Append("")
	b.PeerIDUniqueKey.Append(0)
}
