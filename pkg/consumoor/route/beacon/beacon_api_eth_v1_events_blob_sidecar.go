package beacon

import (
	"fmt"
	"time"

	"github.com/ethpandaops/xatu/pkg/consumoor/route"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

var beaconApiEthV1EventsBlobSidecarEventNames = []xatu.Event_Name{
	xatu.Event_BEACON_API_ETH_V1_EVENTS_BLOB_SIDECAR,
}

func init() {
	r, err := route.NewStaticRoute(
		beaconApiEthV1EventsBlobSidecarTableName,
		beaconApiEthV1EventsBlobSidecarEventNames,
		func() route.ColumnarBatch { return newbeaconApiEthV1EventsBlobSidecarBatch() },
	)
	if err != nil {
		route.RecordError(err)

		return
	}

	if err := route.Register(r); err != nil {
		route.RecordError(err)
	}
}

func (b *beaconApiEthV1EventsBlobSidecarBatch) FlattenTo(
	event *xatu.DecoratedEvent,
) error {
	if event == nil || event.GetEvent() == nil {
		return nil
	}

	if event.GetEthV1EventsBlobSidecar() == nil {
		return fmt.Errorf("nil eth_v1_events_blob_sidecar payload: %w", route.ErrInvalidEvent)
	}

	b.appendRuntime(event)
	b.appendMetadata(event)
	b.appendPayload(event)
	b.appendAdditionalData(event)
	b.rows++

	return nil
}

func (b *beaconApiEthV1EventsBlobSidecarBatch) appendRuntime(event *xatu.DecoratedEvent) {
	b.UpdatedDateTime.Append(time.Now())

	if ts := event.GetEvent().GetDateTime(); ts != nil {
		b.EventDateTime.Append(ts.AsTime())
	} else {
		b.EventDateTime.Append(time.Time{})
	}
}

func (b *beaconApiEthV1EventsBlobSidecarBatch) appendPayload(event *xatu.DecoratedEvent) {
	sidecar := event.GetEthV1EventsBlobSidecar()
	if slot := sidecar.GetSlot(); slot != nil {
		b.Slot.Append(uint32(slot.GetValue())) //nolint:gosec // slot fits uint32
	} else {
		b.Slot.Append(0)
	}

	b.BlockRoot.Append([]byte(sidecar.GetBlockRoot()))

	if index := sidecar.GetIndex(); index != nil {
		b.BlobIndex.Append(index.GetValue())
	} else {
		b.BlobIndex.Append(0)
	}

	b.KzgCommitment.Append([]byte(sidecar.GetKzgCommitment()))
	b.VersionedHash.Append([]byte(sidecar.GetVersionedHash()))
}

func (b *beaconApiEthV1EventsBlobSidecarBatch) appendAdditionalData(event *xatu.DecoratedEvent) {
	if event.GetMeta() == nil || event.GetMeta().GetClient() == nil {
		b.SlotStartDateTime.Append(time.Time{})
		b.PropagationSlotStartDiff.Append(0)
		b.Epoch.Append(0)
		b.EpochStartDateTime.Append(time.Time{})

		return
	}

	client := event.GetMeta().GetClient()
	additional := extractBeaconSlotEpochPropagation(client.GetEthV1EventsBlobSidecar())

	b.SlotStartDateTime.Append(time.Unix(additional.SlotStartDateTime, 0))
	b.PropagationSlotStartDiff.Append(uint32(additional.PropagationSlotStartDiff)) //nolint:gosec // propagation diff fits uint32
	b.Epoch.Append(uint32(additional.Epoch))                                       //nolint:gosec // epoch fits uint32
	b.EpochStartDateTime.Append(time.Unix(additional.EpochStartDateTime, 0))
}
