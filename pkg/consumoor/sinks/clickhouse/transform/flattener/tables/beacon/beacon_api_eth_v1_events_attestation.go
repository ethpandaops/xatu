package beacon

import (
	"fmt"
	"strconv"

	"github.com/ClickHouse/ch-go/proto"
	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/flattener"
	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/metadata"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

func init() {
	flattener.MustRegister(flattener.NewStaticRoute(
		beaconApiEthV1EventsAttestationTableName,
		[]xatu.Event_Name{xatu.Event_BEACON_API_ETH_V1_EVENTS_ATTESTATION_V2},
		func() flattener.ColumnarBatch {
			return newbeaconApiEthV1EventsAttestationBatch()
		},
	))
}

func (b *beaconApiEthV1EventsAttestationBatch) FlattenTo(
	event *xatu.DecoratedEvent,
	meta *metadata.CommonMetadata,
) error {
	if event == nil || event.GetEvent() == nil {
		return nil
	}

	payload := event.GetEthV1EventsAttestationV2()
	if payload == nil {
		return fmt.Errorf("nil EthV1EventsAttestationV2 payload: %w", flattener.ErrInvalidEvent)
	}

	if err := b.validate(event); err != nil {
		return err
	}

	addl := event.GetMeta().GetClient().GetEthV1EventsAttestationV2()
	data := payload.GetData()

	b.EventDateTime.Append(event.GetEvent().GetDateTime().AsTime())
	b.Slot.Append(uint32(data.GetSlot().GetValue())) //nolint:gosec // G115: slot fits uint32.
	b.SlotStartDateTime.Append(addl.GetSlot().GetStartDateTime().AsTime())
	b.PropagationSlotStartDiff.Append(uint32(addl.GetPropagation().GetSlotStartDiff().GetValue())) //nolint:gosec // G115: propagation diff fits uint32.
	b.CommitteeIndex.Append(strconv.FormatUint(data.GetIndex().GetValue(), 10))

	// AttestingValidatorIndex is nullable.
	if addl.GetAttestingValidator() != nil && addl.GetAttestingValidator().GetIndex() != nil {
		b.AttestingValidatorIndex.Append(proto.NewNullable[uint32](uint32(addl.GetAttestingValidator().GetIndex().GetValue()))) //nolint:gosec // G115: validator index fits uint32.
	} else {
		b.AttestingValidatorIndex.Append(proto.Nullable[uint32]{})
	}

	if addl.GetAttestingValidator() != nil && addl.GetAttestingValidator().GetCommitteeIndex() != nil {
		b.AttestingValidatorCommitteeIndex.Append(strconv.FormatUint(addl.GetAttestingValidator().GetCommitteeIndex().GetValue(), 10))
	} else {
		b.AttestingValidatorCommitteeIndex.Append("")
	}

	b.AggregationBits.Append(payload.GetAggregationBits())
	b.BeaconBlockRoot.Append([]byte(data.GetBeaconBlockRoot()))
	b.Epoch.Append(uint32(addl.GetEpoch().GetNumber().GetValue())) //nolint:gosec // G115: epoch fits uint32.
	b.EpochStartDateTime.Append(addl.GetEpoch().GetStartDateTime().AsTime())
	b.SourceEpoch.Append(uint32(data.GetSource().GetEpoch().GetValue())) //nolint:gosec // G115: source epoch fits uint32.
	b.SourceEpochStartDateTime.Append(addl.GetSource().GetEpoch().GetStartDateTime().AsTime())
	b.SourceRoot.Append([]byte(data.GetSource().GetRoot()))
	b.TargetEpoch.Append(uint32(data.GetTarget().GetEpoch().GetValue())) //nolint:gosec // G115: target epoch fits uint32.
	b.TargetEpochStartDateTime.Append(addl.GetTarget().GetEpoch().GetStartDateTime().AsTime())
	b.TargetRoot.Append([]byte(data.GetTarget().GetRoot()))

	b.appendMetadata(meta)
	b.rows++

	return nil
}

func (b *beaconApiEthV1EventsAttestationBatch) validate(event *xatu.DecoratedEvent) error {
	payload := event.GetEthV1EventsAttestationV2()
	data := payload.GetData()

	if data == nil {
		return fmt.Errorf("nil attestation Data: %w", flattener.ErrInvalidEvent)
	}

	if data.GetSlot() == nil {
		return fmt.Errorf("nil Slot: %w", flattener.ErrInvalidEvent)
	}

	addl := event.GetMeta().GetClient().GetEthV1EventsAttestationV2()
	if addl == nil {
		return fmt.Errorf("nil EthV1EventsAttestationV2 additional data: %w", flattener.ErrInvalidEvent)
	}

	if addl.GetEpoch() == nil || addl.GetEpoch().GetNumber() == nil {
		return fmt.Errorf("nil Epoch: %w", flattener.ErrInvalidEvent)
	}

	if addl.GetSlot() == nil {
		return fmt.Errorf("nil additional Slot: %w", flattener.ErrInvalidEvent)
	}

	if addl.GetPropagation() == nil || addl.GetPropagation().GetSlotStartDiff() == nil {
		return fmt.Errorf("nil PropagationSlotStartDiff: %w", flattener.ErrInvalidEvent)
	}

	return nil
}
