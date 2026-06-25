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

	if err := validateSignedHeader("SignedHeader_1", slashing.GetSignedHeader_1()); err != nil {
		return err
	}

	return validateSignedHeader("SignedHeader_2", slashing.GetSignedHeader_2())
}

func (b *canonicalBeaconBlockProposerSlashingBatch) appendRuntime(_ *xatu.DecoratedEvent) {
	b.UpdatedDateTime.Append(time.Now())
}

func (b *canonicalBeaconBlockProposerSlashingBatch) appendPayload(event *xatu.DecoratedEvent) {
	slashing := event.GetEthV2BeaconBlockProposerSlashing()
	b.appendSignedHeader1(slashing.GetSignedHeader_1())
	b.appendSignedHeader2(slashing.GetSignedHeader_2())
}

//nolint:gosec // G115: proto uint64 values are bounded by ClickHouse uint32 column schema
func (b *canonicalBeaconBlockProposerSlashingBatch) appendSignedHeader1(header *ethv1.SignedBeaconBlockHeaderV2) {
	b.SignedHeader1Signature.Append(header.GetSignature())

	msg := header.GetMessage()
	b.SignedHeader1MessageSlot.Append(uint32(msg.GetSlot().GetValue()))
	b.SignedHeader1MessageProposerIndex.Append(uint32(msg.GetProposerIndex().GetValue()))
	b.SignedHeader1MessageBodyRoot.Append([]byte(msg.GetBodyRoot()))
	b.SignedHeader1MessageParentRoot.Append([]byte(msg.GetParentRoot()))
	b.SignedHeader1MessageStateRoot.Append([]byte(msg.GetStateRoot()))
}

//nolint:gosec // G115: proto uint64 values are bounded by ClickHouse uint32 column schema
func (b *canonicalBeaconBlockProposerSlashingBatch) appendSignedHeader2(header *ethv1.SignedBeaconBlockHeaderV2) {
	b.SignedHeader2Signature.Append(header.GetSignature())

	msg := header.GetMessage()
	b.SignedHeader2MessageSlot.Append(uint32(msg.GetSlot().GetValue()))
	b.SignedHeader2MessageProposerIndex.Append(uint32(msg.GetProposerIndex().GetValue()))
	b.SignedHeader2MessageBodyRoot.Append([]byte(msg.GetBodyRoot()))
	b.SignedHeader2MessageParentRoot.Append([]byte(msg.GetParentRoot()))
	b.SignedHeader2MessageStateRoot.Append([]byte(msg.GetStateRoot()))
}

// validateSignedHeader returns route.ErrInvalidEvent when a required field of a
// signed block header is absent, so the slashing is dropped rather than written
// with fabricated zero slot/proposer_index values.
func validateSignedHeader(name string, header *ethv1.SignedBeaconBlockHeaderV2) error {
	if header == nil {
		return fmt.Errorf("nil %s: %w", name, route.ErrInvalidEvent)
	}

	msg := header.GetMessage()
	if msg == nil {
		return fmt.Errorf("nil %s.Message: %w", name, route.ErrInvalidEvent)
	}

	if msg.GetSlot() == nil {
		return fmt.Errorf("nil %s.Message.Slot: %w", name, route.ErrInvalidEvent)
	}

	if msg.GetProposerIndex() == nil {
		return fmt.Errorf("nil %s.Message.ProposerIndex: %w", name, route.ErrInvalidEvent)
	}

	return nil
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
