package libp2p

import (
	"fmt"
	"time"

	"github.com/ethpandaops/xatu/pkg/consumoor/route"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

var libp2pGossipsubPayloadAttestationMessageEventNames = []xatu.Event_Name{
	xatu.Event_LIBP2P_TRACE_GOSSIPSUB_PAYLOAD_ATTESTATION_MESSAGE,
}

func init() {
	r, err := route.NewStaticRoute(
		libp2pGossipsubPayloadAttestationMessageTableName,
		libp2pGossipsubPayloadAttestationMessageEventNames,
		func() route.ColumnarBatch { return newlibp2pGossipsubPayloadAttestationMessageBatch() },
	)
	if err != nil {
		route.RecordError(err)

		return
	}

	if err := route.Register(r); err != nil {
		route.RecordError(err)
	}
}

func (b *libp2pGossipsubPayloadAttestationMessageBatch) FlattenTo(
	event *xatu.DecoratedEvent,
) error {
	if event == nil || event.GetEvent() == nil {
		return nil
	}

	if event.GetLibp2PTraceGossipsubPayloadAttestationMessage() == nil {
		return fmt.Errorf(
			"nil libp2p_trace_gossipsub_payload_attestation_message payload: %w",
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

func (b *libp2pGossipsubPayloadAttestationMessageBatch) appendRuntime(
	event *xatu.DecoratedEvent,
) {
	b.UpdatedDateTime.Append(time.Now())

	if ts := event.GetEvent().GetDateTime(); ts != nil {
		b.EventDateTime.Append(ts.AsTime())
	} else {
		b.EventDateTime.Append(time.Time{})
	}
}

//nolint:gosec // G115: proto uint64 values are bounded by ClickHouse column schema
func (b *libp2pGossipsubPayloadAttestationMessageBatch) appendPayload(
	event *xatu.DecoratedEvent,
) {
	payload := event.GetLibp2PTraceGossipsubPayloadAttestationMessage()

	if validatorIndex := payload.GetValidatorIndex(); validatorIndex != nil {
		b.ValidatorIndex.Append(uint32(validatorIndex.GetValue()))
	} else {
		b.ValidatorIndex.Append(0)
	}

	b.BeaconBlockRoot.Append([]byte(wrappedStringValue(payload.GetBeaconBlockRoot())))

	if pp := payload.GetPayloadPresent(); pp != nil {
		b.PayloadPresent.Append(pp.GetValue())
	} else {
		b.PayloadPresent.Append(false)
	}

	if bda := payload.GetBlobDataAvailable(); bda != nil {
		b.BlobDataAvailable.Append(bda.GetValue())
	} else {
		b.BlobDataAvailable.Append(false)
	}
}

// TODO: Define AdditionalLibp2pTraceGossipsubPayloadAttestationMessageData proto message
// to extract slot/epoch/wallclock/propagation/peer/message/topic fields from
// event.GetMeta().GetClient(). For now, zero-fill these fields.
func (b *libp2pGossipsubPayloadAttestationMessageBatch) appendClientAdditionalData(
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
