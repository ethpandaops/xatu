package canonical

import (
	"fmt"
	"time"

	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/flattener"
	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/metadata"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

func init() {
	flattener.MustRegister(flattener.NewStaticRoute(
		canonicalBeaconBlockProposerSlashingTableName,
		[]xatu.Event_Name{xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_PROPOSER_SLASHING},
		func() flattener.ColumnarBatch {
			return newcanonicalBeaconBlockProposerSlashingBatch()
		},
	))
}

//nolint:gosec // G115: header field values fit uint32.
func (b *canonicalBeaconBlockProposerSlashingBatch) FlattenTo(
	event *xatu.DecoratedEvent,
	meta *metadata.CommonMetadata,
) error {
	if event == nil || event.GetEvent() == nil {
		return nil
	}

	payload := event.GetEthV2BeaconBlockProposerSlashing()
	if payload == nil {
		return fmt.Errorf("nil ProposerSlashing payload: %w", flattener.ErrInvalidEvent)
	}

	if err := b.validate(event); err != nil {
		return err
	}

	addl := event.GetMeta().GetClient().GetEthV2BeaconBlockProposerSlashing()
	blk := addl.GetBlock()

	b.UpdatedDateTime.Append(time.Now())
	b.Slot.Append(uint32(blk.GetSlot().GetNumber().GetValue()))
	b.SlotStartDateTime.Append(blk.GetSlot().GetStartDateTime().AsTime())
	b.Epoch.Append(uint32(blk.GetEpoch().GetNumber().GetValue()))
	b.EpochStartDateTime.Append(blk.GetEpoch().GetStartDateTime().AsTime())
	b.BlockRoot.Append([]byte(blk.GetRoot()))
	b.BlockVersion.Append(blk.GetVersion())

	h1 := payload.GetSignedHeader_1()
	h1m := h1.GetMessage()
	b.SignedHeader1MessageSlot.Append(uint32(h1m.GetSlot().GetValue()))
	b.SignedHeader1MessageProposerIndex.Append(uint32(h1m.GetProposerIndex().GetValue()))
	b.SignedHeader1MessageBodyRoot.Append([]byte(h1m.GetBodyRoot()))
	b.SignedHeader1MessageParentRoot.Append([]byte(h1m.GetParentRoot()))
	b.SignedHeader1MessageStateRoot.Append([]byte(h1m.GetStateRoot()))
	b.SignedHeader1Signature.Append(h1.GetSignature())

	h2 := payload.GetSignedHeader_2()
	h2m := h2.GetMessage()
	b.SignedHeader2MessageSlot.Append(uint32(h2m.GetSlot().GetValue()))
	b.SignedHeader2MessageProposerIndex.Append(uint32(h2m.GetProposerIndex().GetValue()))
	b.SignedHeader2MessageBodyRoot.Append([]byte(h2m.GetBodyRoot()))
	b.SignedHeader2MessageParentRoot.Append([]byte(h2m.GetParentRoot()))
	b.SignedHeader2MessageStateRoot.Append([]byte(h2m.GetStateRoot()))
	b.SignedHeader2Signature.Append(h2.GetSignature())

	b.appendMetadata(meta)
	b.rows++

	return nil
}

func (b *canonicalBeaconBlockProposerSlashingBatch) validate(event *xatu.DecoratedEvent) error {
	addl := event.GetMeta().GetClient().GetEthV2BeaconBlockProposerSlashing()
	if addl == nil {
		return fmt.Errorf("nil ProposerSlashing additional data: %w", flattener.ErrInvalidEvent)
	}

	blk := addl.GetBlock()
	if blk == nil {
		return fmt.Errorf("nil Block identifier: %w", flattener.ErrInvalidEvent)
	}

	if blk.GetEpoch() == nil || blk.GetEpoch().GetNumber() == nil {
		return fmt.Errorf("nil Epoch: %w", flattener.ErrInvalidEvent)
	}

	if blk.GetSlot() == nil || blk.GetSlot().GetNumber() == nil {
		return fmt.Errorf("nil Slot: %w", flattener.ErrInvalidEvent)
	}

	return nil
}
