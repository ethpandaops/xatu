package canonical

import (
	"time"

	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/flattener"
	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/metadata"
	ethv1 "github.com/ethpandaops/xatu/pkg/proto/eth/v1"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

var canonicalBeaconBlockProposerSlashingEventNames = []xatu.Event_Name{
	xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_PROPOSER_SLASHING,
}

func init() {
	flattener.MustRegister(flattener.NewStaticRoute(
		canonicalBeaconBlockProposerSlashingTableName,
		canonicalBeaconBlockProposerSlashingEventNames,
		func() flattener.ColumnarBatch { return newcanonicalBeaconBlockProposerSlashingBatch() },
	))
}

func (b *canonicalBeaconBlockProposerSlashingBatch) FlattenTo(
	event *xatu.DecoratedEvent,
	meta *metadata.CommonMetadata,
) error {
	if event == nil || event.GetEvent() == nil {
		return nil
	}

	if meta == nil {
		meta = metadata.Extract(event)
	}

	b.appendRuntime(event)
	b.appendMetadata(meta)
	b.appendPayload(event)
	b.appendAdditionalData(event)
	b.rows++

	return nil
}

func (b *canonicalBeaconBlockProposerSlashingBatch) appendRuntime(_ *xatu.DecoratedEvent) {
	b.UpdatedDateTime.Append(time.Now())
}

func (b *canonicalBeaconBlockProposerSlashingBatch) appendPayload(event *xatu.DecoratedEvent) {
	slashing := event.GetEthV2BeaconBlockProposerSlashing()
	if slashing == nil {
		b.appendNullSignedHeader1()
		b.appendNullSignedHeader2()

		return
	}

	b.appendSignedHeader1(slashing.GetSignedHeader_1())
	b.appendSignedHeader2(slashing.GetSignedHeader_2())
}

func (b *canonicalBeaconBlockProposerSlashingBatch) appendSignedHeader1(header *ethv1.SignedBeaconBlockHeaderV2) {
	if header == nil {
		b.appendNullSignedHeader1()

		return
	}

	b.SignedHeader1Signature.Append(header.GetSignature())

	if msg := header.GetMessage(); msg != nil {
		if slot := msg.GetSlot(); slot != nil {
			b.SignedHeader1MessageSlot.Append(uint32(slot.GetValue())) //nolint:gosec // G115
		} else {
			b.SignedHeader1MessageSlot.Append(0)
		}

		if proposerIndex := msg.GetProposerIndex(); proposerIndex != nil {
			b.SignedHeader1MessageProposerIndex.Append(uint32(proposerIndex.GetValue())) //nolint:gosec // G115
		} else {
			b.SignedHeader1MessageProposerIndex.Append(0)
		}

		b.SignedHeader1MessageBodyRoot.Append([]byte(msg.GetBodyRoot()))
		b.SignedHeader1MessageParentRoot.Append([]byte(msg.GetParentRoot()))
		b.SignedHeader1MessageStateRoot.Append([]byte(msg.GetStateRoot()))
	} else {
		b.SignedHeader1MessageSlot.Append(0)
		b.SignedHeader1MessageProposerIndex.Append(0)
		b.SignedHeader1MessageBodyRoot.Append(nil)
		b.SignedHeader1MessageParentRoot.Append(nil)
		b.SignedHeader1MessageStateRoot.Append(nil)
	}
}

func (b *canonicalBeaconBlockProposerSlashingBatch) appendNullSignedHeader1() {
	b.SignedHeader1Signature.Append("")
	b.SignedHeader1MessageSlot.Append(0)
	b.SignedHeader1MessageProposerIndex.Append(0)
	b.SignedHeader1MessageBodyRoot.Append(nil)
	b.SignedHeader1MessageParentRoot.Append(nil)
	b.SignedHeader1MessageStateRoot.Append(nil)
}

func (b *canonicalBeaconBlockProposerSlashingBatch) appendSignedHeader2(header *ethv1.SignedBeaconBlockHeaderV2) {
	if header == nil {
		b.appendNullSignedHeader2()

		return
	}

	b.SignedHeader2Signature.Append(header.GetSignature())

	if msg := header.GetMessage(); msg != nil {
		if slot := msg.GetSlot(); slot != nil {
			b.SignedHeader2MessageSlot.Append(uint32(slot.GetValue())) //nolint:gosec // G115
		} else {
			b.SignedHeader2MessageSlot.Append(0)
		}

		if proposerIndex := msg.GetProposerIndex(); proposerIndex != nil {
			b.SignedHeader2MessageProposerIndex.Append(uint32(proposerIndex.GetValue())) //nolint:gosec // G115
		} else {
			b.SignedHeader2MessageProposerIndex.Append(0)
		}

		b.SignedHeader2MessageBodyRoot.Append([]byte(msg.GetBodyRoot()))
		b.SignedHeader2MessageParentRoot.Append([]byte(msg.GetParentRoot()))
		b.SignedHeader2MessageStateRoot.Append([]byte(msg.GetStateRoot()))
	} else {
		b.SignedHeader2MessageSlot.Append(0)
		b.SignedHeader2MessageProposerIndex.Append(0)
		b.SignedHeader2MessageBodyRoot.Append(nil)
		b.SignedHeader2MessageParentRoot.Append(nil)
		b.SignedHeader2MessageStateRoot.Append(nil)
	}
}

func (b *canonicalBeaconBlockProposerSlashingBatch) appendNullSignedHeader2() {
	b.SignedHeader2Signature.Append("")
	b.SignedHeader2MessageSlot.Append(0)
	b.SignedHeader2MessageProposerIndex.Append(0)
	b.SignedHeader2MessageBodyRoot.Append(nil)
	b.SignedHeader2MessageParentRoot.Append(nil)
	b.SignedHeader2MessageStateRoot.Append(nil)
}

func (b *canonicalBeaconBlockProposerSlashingBatch) appendAdditionalData(event *xatu.DecoratedEvent) {
	additional := event.GetMeta().GetClient().GetEthV2BeaconBlockProposerSlashing()
	if additional == nil {
		b.Slot.Append(0)
		b.SlotStartDateTime.Append(time.Time{})
		b.Epoch.Append(0)
		b.EpochStartDateTime.Append(time.Time{})
		b.BlockVersion.Append("")
		b.BlockRoot.Append(nil)

		return
	}

	appendBlockIdentifier(additional.GetBlock(),
		&b.Slot, &b.SlotStartDateTime, &b.Epoch, &b.EpochStartDateTime, &b.BlockVersion, &b.BlockRoot)
}
