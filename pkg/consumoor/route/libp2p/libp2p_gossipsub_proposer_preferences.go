package libp2p

import (
	"fmt"
	"time"

	"github.com/ethpandaops/xatu/pkg/consumoor/route"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

var libp2pGossipsubProposerPreferencesEventNames = []xatu.Event_Name{
	xatu.Event_LIBP2P_TRACE_GOSSIPSUB_PROPOSER_PREFERENCES,
}

func init() {
	r, err := route.NewStaticRoute(
		libp2pGossipsubProposerPreferencesTableName,
		libp2pGossipsubProposerPreferencesEventNames,
		func() route.ColumnarBatch { return newlibp2pGossipsubProposerPreferencesBatch() },
	)
	if err != nil {
		route.RecordError(err)

		return
	}

	if err := route.Register(r); err != nil {
		route.RecordError(err)
	}
}

func (b *libp2pGossipsubProposerPreferencesBatch) FlattenTo(
	event *xatu.DecoratedEvent,
) error {
	if event == nil || event.GetEvent() == nil {
		return nil
	}

	if event.GetLibp2PTraceGossipsubProposerPreferences() == nil {
		return fmt.Errorf(
			"nil libp2p_trace_gossipsub_proposer_preferences payload: %w",
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

func (b *libp2pGossipsubProposerPreferencesBatch) appendRuntime(event *xatu.DecoratedEvent) {
	b.UpdatedDateTime.Append(time.Now())

	if ts := event.GetEvent().GetDateTime(); ts != nil {
		b.EventDateTime.Append(ts.AsTime())
	} else {
		b.EventDateTime.Append(time.Time{})
	}
}

//nolint:gosec // G115: proto uint64 values are bounded by ClickHouse column schema
func (b *libp2pGossipsubProposerPreferencesBatch) appendPayload(event *xatu.DecoratedEvent) {
	payload := event.GetLibp2PTraceGossipsubProposerPreferences()

	if validatorIndex := payload.GetValidatorIndex(); validatorIndex != nil {
		b.ValidatorIndex.Append(uint32(validatorIndex.GetValue()))
	} else {
		b.ValidatorIndex.Append(0)
	}

	b.FeeRecipient.Append([]byte(wrappedStringValue(payload.GetFeeRecipient())))

	if gasLimit := payload.GetGasLimit(); gasLimit != nil {
		b.GasLimit.Append(gasLimit.GetValue())
	} else {
		b.GasLimit.Append(0)
	}
}

// TODO: Define AdditionalLibp2pTraceGossipsubProposerPreferencesData proto message
// to extract slot/epoch/wallclock/propagation/peer/message/topic fields from
// event.GetMeta().GetClient(). For now, zero-fill these fields.
func (b *libp2pGossipsubProposerPreferencesBatch) appendClientAdditionalData(
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
