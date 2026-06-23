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

	extra := event.GetMeta().GetClient().GetEthV1BeaconStateFinalityCheckpoint()
	if extra == nil || extra.GetEpoch() == nil || extra.GetEpoch().GetNumber() == nil {
		return fmt.Errorf("nil Epoch: %w", route.ErrInvalidEvent)
	}

	epoch := uint32(extra.GetEpoch().GetNumber().GetValue()) //nolint:gosec // bounded by uint32 column

	var epochStartTime time.Time
	if start := extra.GetEpoch().GetStartDateTime(); start != nil {
		epochStartTime = start.AsTime()
	}

	b.UpdatedDateTime.Append(time.Now())
	b.Epoch.Append(epoch)
	b.EpochStartDateTime.Append(epochStartTime)
	b.StateID.Append(extra.GetStateId())

	if cp := payload.GetPreviousJustified(); cp != nil {
		b.PreviousJustifiedEpoch.Append(uint32(cp.GetEpoch())) //nolint:gosec // bounded by uint32 column
		b.PreviousJustifiedRoot.Append([]byte(cp.GetRoot()))
	} else {
		b.PreviousJustifiedEpoch.Append(0)
		b.PreviousJustifiedRoot.Append([]byte(""))
	}

	if cp := payload.GetCurrentJustified(); cp != nil {
		b.CurrentJustifiedEpoch.Append(uint32(cp.GetEpoch())) //nolint:gosec // bounded by uint32 column
		b.CurrentJustifiedRoot.Append([]byte(cp.GetRoot()))
	} else {
		b.CurrentJustifiedEpoch.Append(0)
		b.CurrentJustifiedRoot.Append([]byte(""))
	}

	if cp := payload.GetFinalized(); cp != nil {
		b.FinalizedEpoch.Append(uint32(cp.GetEpoch())) //nolint:gosec // bounded by uint32 column
		b.FinalizedRoot.Append([]byte(cp.GetRoot()))
	} else {
		b.FinalizedEpoch.Append(0)
		b.FinalizedRoot.Append([]byte(""))
	}

	b.appendMetadata(event)
	b.rows++

	return nil
}
