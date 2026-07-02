package canonical

import (
	"fmt"
	"time"

	"github.com/ethpandaops/xatu/pkg/clickhouse/route"
	ethv1 "github.com/ethpandaops/xatu/pkg/proto/eth/v1"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

var canonicalBeaconBlockProposerSlashingEventNames = []xatu.Event_Name{
	xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_PROPOSER_SLASHING,
}

func init() {
	r, err := route.NewStaticRoute(
		canonicalBeaconBlockProposerSlashingTableName,
		canonicalBeaconBlockProposerSlashingEventNames,
		func() route.ColumnarBatch { return newcanonicalBeaconBlockProposerSlashingBatch() },
	)
	if err != nil {
		route.RecordError(err)

		return
	}

	if err := route.Register(r); err != nil {
		route.RecordError(err)
	}
}

func (b *canonicalBeaconBlockProposerSlashingBatch) FlattenTo(event *xatu.DecoratedEvent) error {
	if event == nil || event.GetEvent() == nil {
		return nil
	}

	if event.GetEthV2BeaconBlockProposerSlashing() == nil {
		return fmt.Errorf("nil eth_v2_beacon_block_proposer_slashing payload: %w", route.ErrInvalidEvent)
	}

	if err := b.validate(event); err != nil {
		return err
	}

	b.appendRuntime(event)
	b.appendMetadata(event)
	b.appendPayload(event)
	b.appendAdditionalData(event)
	b.rows++

	return nil
}

func (b *canonicalBeaconBlockProposerSlashingBatch) validate(event *xatu.DecoratedEvent) error {
	slashing := event.GetEthV2BeaconBlockProposerSlashing()

	if err := validateProposerSlashingHeader(slashing.GetSignedHeader_1(), "signed_header_1"); err != nil {
		return err
	}

	if err := validateProposerSlashingHeader(slashing.GetSignedHeader_2(), "signed_header_2"); err != nil {
		return err
	}

	additional := event.GetMeta().GetClient().GetEthV2BeaconBlockProposerSlashing()
	if additional == nil {
		return fmt.Errorf("nil additional data: %w", route.ErrInvalidEvent)
	}

	block := additional.GetBlock()
	if block == nil {
		return fmt.Errorf("nil block identifier: %w", route.ErrInvalidEvent)
	}

	if block.GetRoot() == "" {
		return fmt.Errorf("empty block_root: %w", route.ErrInvalidEvent)
	}

	if block.GetVersion() == "" {
		return fmt.Errorf("empty block_version: %w", route.ErrInvalidEvent)
	}

	if block.GetSlot().GetStartDateTime() == nil {
		return fmt.Errorf("nil slot_start_date_time: %w", route.ErrInvalidEvent)
	}

	if block.GetEpoch().GetStartDateTime() == nil {
		return fmt.Errorf("nil epoch_start_date_time: %w", route.ErrInvalidEvent)
	}

	if event.GetMeta().GetClient().GetEthereum().GetNetwork().GetName() == "" {
		return fmt.Errorf("empty meta_network_name: %w", route.ErrInvalidEvent)
	}

	return nil
}

func (b *canonicalBeaconBlockProposerSlashingBatch) appendRuntime(_ *xatu.DecoratedEvent) {
	b.UpdatedDateTime.Append(time.Now())
}

func (b *canonicalBeaconBlockProposerSlashingBatch) appendPayload(event *xatu.DecoratedEvent) {
	slashing := event.GetEthV2BeaconBlockProposerSlashing()
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

func validateProposerSlashingHeader(header *ethv1.SignedBeaconBlockHeaderV2, name string) error {
	if header == nil {
		return fmt.Errorf("nil %s: %w", name, route.ErrInvalidEvent)
	}

	if header.GetSignature() == "" {
		return fmt.Errorf("empty %s_signature: %w", name, route.ErrInvalidEvent)
	}

	msg := header.GetMessage()
	if msg == nil {
		return fmt.Errorf("nil %s_message: %w", name, route.ErrInvalidEvent)
	}

	if msg.GetBodyRoot() == "" {
		return fmt.Errorf("empty %s_message_body_root: %w", name, route.ErrInvalidEvent)
	}

	if msg.GetParentRoot() == "" {
		return fmt.Errorf("empty %s_message_parent_root: %w", name, route.ErrInvalidEvent)
	}

	if msg.GetStateRoot() == "" {
		return fmt.Errorf("empty %s_message_state_root: %w", name, route.ErrInvalidEvent)
	}

	return nil
}
