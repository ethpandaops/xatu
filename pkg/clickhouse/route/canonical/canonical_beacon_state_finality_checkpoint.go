package canonical

import (
	"fmt"
	"time"

	"github.com/ethpandaops/xatu/pkg/clickhouse/route"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

var canonicalBeaconStateFinalityCheckpointEventNames = []xatu.Event_Name{
	xatu.Event_BEACON_API_ETH_V1_BEACON_STATE_FINALITY_CHECKPOINT,
}

func init() {
	r, err := route.NewStaticRoute(
		canonicalBeaconStateFinalityCheckpointTableName,
		canonicalBeaconStateFinalityCheckpointEventNames,
		func() route.ColumnarBatch { return newcanonicalBeaconStateFinalityCheckpointBatch() },
	)
	if err != nil {
		route.RecordError(err)

		return
	}

	if err := route.Register(r); err != nil {
		route.RecordError(err)
	}
}

func (b *canonicalBeaconStateFinalityCheckpointBatch) FlattenTo(event *xatu.DecoratedEvent) error {
	if event == nil || event.GetEvent() == nil {
		return nil
	}

	payload := event.GetEthV1BeaconStateFinalityCheckpoint()
	if payload == nil {
		return fmt.Errorf("nil payload: %w", route.ErrInvalidEvent)
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

func (b *canonicalBeaconStateFinalityCheckpointBatch) validate(event *xatu.DecoratedEvent) error {
	payload := event.GetEthV1BeaconStateFinalityCheckpoint()

	extra := event.GetMeta().GetClient().GetEthV1BeaconStateFinalityCheckpoint()
	if extra == nil || extra.GetEpoch() == nil || extra.GetEpoch().GetNumber() == nil {
		return fmt.Errorf("nil Epoch: %w", route.ErrInvalidEvent)
	}

	if extra.GetEpoch().GetStartDateTime() == nil {
		return fmt.Errorf("nil EpochStartDateTime: %w", route.ErrInvalidEvent)
	}

	if extra.GetStateId() == "" {
		return fmt.Errorf("nil StateID: %w", route.ErrInvalidEvent)
	}

	if event.GetMeta().GetClient().GetEthereum().GetNetwork().GetName() == "" {
		return fmt.Errorf("nil MetaNetworkName: %w", route.ErrInvalidEvent)
	}

	if payload.GetPreviousJustified() == nil {
		return fmt.Errorf("nil PreviousJustified: %w", route.ErrInvalidEvent)
	}

	if payload.GetPreviousJustified().GetRoot() == "" {
		return fmt.Errorf("nil PreviousJustifiedRoot: %w", route.ErrInvalidEvent)
	}

	if payload.GetCurrentJustified() == nil {
		return fmt.Errorf("nil CurrentJustified: %w", route.ErrInvalidEvent)
	}

	if payload.GetCurrentJustified().GetRoot() == "" {
		return fmt.Errorf("nil CurrentJustifiedRoot: %w", route.ErrInvalidEvent)
	}

	if payload.GetFinalized() == nil {
		return fmt.Errorf("nil Finalized: %w", route.ErrInvalidEvent)
	}

	if payload.GetFinalized().GetRoot() == "" {
		return fmt.Errorf("nil FinalizedRoot: %w", route.ErrInvalidEvent)
	}

	return nil
}

func (b *canonicalBeaconStateFinalityCheckpointBatch) appendRuntime(_ *xatu.DecoratedEvent) {
	b.UpdatedDateTime.Append(time.Now())
}

func (b *canonicalBeaconStateFinalityCheckpointBatch) appendPayload(event *xatu.DecoratedEvent) {
	payload := event.GetEthV1BeaconStateFinalityCheckpoint()
	previous := payload.GetPreviousJustified()
	current := payload.GetCurrentJustified()
	finalized := payload.GetFinalized()

	b.PreviousJustifiedEpoch.Append(uint32(previous.GetEpoch())) //nolint:gosec // bounded by uint32 column
	b.PreviousJustifiedRoot.Append([]byte(previous.GetRoot()))
	b.CurrentJustifiedEpoch.Append(uint32(current.GetEpoch())) //nolint:gosec // bounded by uint32 column
	b.CurrentJustifiedRoot.Append([]byte(current.GetRoot()))
	b.FinalizedEpoch.Append(uint32(finalized.GetEpoch())) //nolint:gosec // bounded by uint32 column
	b.FinalizedRoot.Append([]byte(finalized.GetRoot()))
}

func (b *canonicalBeaconStateFinalityCheckpointBatch) appendAdditionalData(event *xatu.DecoratedEvent) {
	extra := event.GetMeta().GetClient().GetEthV1BeaconStateFinalityCheckpoint()
	epoch := extra.GetEpoch()

	b.Epoch.Append(uint32(epoch.GetNumber().GetValue())) //nolint:gosec // bounded by uint32 column

	if start := epoch.GetStartDateTime(); start != nil {
		b.EpochStartDateTime.Append(start.AsTime())
	} else {
		b.EpochStartDateTime.Append(time.Time{})
	}

	b.StateID.Append(extra.GetStateId())
}
