package beacon

import (
	"fmt"
	"time"

	"github.com/ethpandaops/xatu/pkg/consumoor/route"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

var beaconApiEthV1BeaconBlobEventNames = []xatu.Event_Name{
	xatu.Event_BEACON_API_ETH_V1_BEACON_BLOB,
}

func init() {
	r, err := route.NewStaticRoute(
		beaconApiEthV1BeaconBlobTableName,
		beaconApiEthV1BeaconBlobEventNames,
		func() route.ColumnarBatch { return newbeaconApiEthV1BeaconBlobBatch() },
	)
	if err != nil {
		route.RecordError(err)

		return
	}

	if err := route.Register(r); err != nil {
		route.RecordError(err)
	}
}

func (b *beaconApiEthV1BeaconBlobBatch) FlattenTo(event *xatu.DecoratedEvent) error {
	if event == nil || event.GetEvent() == nil {
		return nil
	}

	if event.GetEthV1BeaconBlob() == nil {
		return fmt.Errorf("nil eth_v1_beacon_blob payload: %w", route.ErrInvalidEvent)
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

func (b *beaconApiEthV1BeaconBlobBatch) validate(event *xatu.DecoratedEvent) error {
	payload := event.GetEthV1BeaconBlob()

	if payload.GetSlot() == nil {
		return fmt.Errorf("nil Slot: %w", route.ErrInvalidEvent)
	}

	if payload.GetProposerIndex() == nil {
		return fmt.Errorf("nil ProposerIndex: %w", route.ErrInvalidEvent)
	}

	if payload.GetIndex() == nil {
		return fmt.Errorf("nil Index: %w", route.ErrInvalidEvent)
	}

	return nil
}

func (b *beaconApiEthV1BeaconBlobBatch) appendRuntime(event *xatu.DecoratedEvent) {
	if ts := event.GetEvent().GetDateTime(); ts != nil {
		b.EventDateTime.Append(ts.AsTime())
	} else {
		b.EventDateTime.Append(time.Now())
	}
}

func (b *beaconApiEthV1BeaconBlobBatch) appendPayload(event *xatu.DecoratedEvent) {
	blob := event.GetEthV1BeaconBlob()
	if slot := blob.GetSlot(); slot != nil {
		b.Slot.Append(uint32(slot.GetValue())) //nolint:gosec // slot fits uint32
	} else {
		b.Slot.Append(0)
	}

	b.BlockRoot.Append([]byte(blob.GetBlockRoot()))
	b.BlockParentRoot.Append([]byte(blob.GetBlockParentRoot()))

	if proposerIndex := blob.GetProposerIndex(); proposerIndex != nil {
		b.ProposerIndex.Append(uint32(proposerIndex.GetValue())) //nolint:gosec // proposer index fits uint32
	} else {
		b.ProposerIndex.Append(0)
	}

	if index := blob.GetIndex(); index != nil {
		b.BlobIndex.Append(index.GetValue())
	} else {
		b.BlobIndex.Append(0)
	}

	b.KzgCommitment.Append([]byte(blob.GetKzgCommitment()))
	b.VersionedHash.Append([]byte(blob.GetVersionedHash()))
}

func (b *beaconApiEthV1BeaconBlobBatch) appendAdditionalData(event *xatu.DecoratedEvent) {
	if event.GetMeta() == nil || event.GetMeta().GetClient() == nil {
		b.Epoch.Append(0)
		b.EpochStartDateTime.Append(time.Time{})
		b.SlotStartDateTime.Append(time.Time{})

		return
	}

	additional := event.GetMeta().GetClient().GetEthV1BeaconBlob()
	if additional == nil {
		b.Epoch.Append(0)
		b.EpochStartDateTime.Append(time.Time{})
		b.SlotStartDateTime.Append(time.Time{})

		return
	}

	if epoch := additional.GetEpoch(); epoch != nil {
		if epochNumber := epoch.GetNumber(); epochNumber != nil {
			b.Epoch.Append(uint32(epochNumber.GetValue())) //nolint:gosec // epoch fits uint32
		} else {
			b.Epoch.Append(0)
		}

		if startDateTime := epoch.GetStartDateTime(); startDateTime != nil {
			b.EpochStartDateTime.Append(startDateTime.AsTime())
		} else {
			b.EpochStartDateTime.Append(time.Time{})
		}
	} else {
		b.Epoch.Append(0)
		b.EpochStartDateTime.Append(time.Time{})
	}

	if slot := additional.GetSlot(); slot != nil {
		if startDateTime := slot.GetStartDateTime(); startDateTime != nil {
			b.SlotStartDateTime.Append(startDateTime.AsTime())
		} else {
			b.SlotStartDateTime.Append(time.Time{})
		}
	} else {
		b.SlotStartDateTime.Append(time.Time{})
	}
}
