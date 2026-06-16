package canonical

import (
	"fmt"
	"time"

	"github.com/ClickHouse/ch-go/proto"
	"github.com/ethpandaops/xatu/pkg/clickhouse/route"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

var canonicalBeaconBlockAccessListEventNames = []xatu.Event_Name{
	xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_ACCESS_LIST,
}

func init() {
	r, err := route.NewStaticRoute(
		canonicalBeaconBlockAccessListTableName,
		canonicalBeaconBlockAccessListEventNames,
		func() route.ColumnarBatch { return newcanonicalBeaconBlockAccessListBatch() },
	)
	if err != nil {
		route.RecordError(err)

		return
	}

	if err := route.Register(r); err != nil {
		route.RecordError(err)
	}
}

func (b *canonicalBeaconBlockAccessListBatch) FlattenTo(event *xatu.DecoratedEvent) error {
	if event == nil || event.GetEvent() == nil {
		return nil
	}

	if event.GetEthV2BeaconBlockAccessList() == nil {
		return fmt.Errorf("nil eth_v2_beacon_block_access_list payload: %w", route.ErrInvalidEvent)
	}

	if err := b.validate(event); err != nil {
		return err
	}

	b.appendRuntime(event)
	b.appendMetadata(event)

	if err := b.appendPayload(event); err != nil {
		return err
	}

	b.appendAdditionalData(event)
	b.rows++

	return nil
}

func (b *canonicalBeaconBlockAccessListBatch) validate(event *xatu.DecoratedEvent) error {
	payload := event.GetEthV2BeaconBlockAccessList()

	if payload.GetAddress() == nil {
		return fmt.Errorf("nil Address: %w", route.ErrInvalidEvent)
	}

	return nil
}

func (b *canonicalBeaconBlockAccessListBatch) appendRuntime(_ *xatu.DecoratedEvent) {
	b.UpdatedDateTime.Append(time.Now())
}

func (b *canonicalBeaconBlockAccessListBatch) appendPayload(event *xatu.DecoratedEvent) error {
	change := event.GetEthV2BeaconBlockAccessList()

	b.Address.Append([]byte(change.GetAddress().GetValue()))
	b.ChangeType.Append(change.GetChangeType())

	if blockAccessIndex := change.GetBlockAccessIndex(); blockAccessIndex != nil {
		b.BlockAccessIndex.Append(blockAccessIndex.GetValue())
	} else {
		b.BlockAccessIndex.Append(0)
	}

	if storageKey := change.GetStorageKey(); storageKey != nil {
		b.StorageKey.Append([]byte(storageKey.GetValue()))
	} else {
		b.StorageKey.Append(nil)
	}

	if newValue := change.GetNewValue(); newValue != nil {
		b.NewValue.Append(proto.NewNullable[string](newValue.GetValue()))
	} else {
		b.NewValue.Append(proto.Nullable[string]{})
	}

	return nil
}

func (b *canonicalBeaconBlockAccessListBatch) appendAdditionalData(event *xatu.DecoratedEvent) {
	additional := event.GetMeta().GetClient().GetEthV2BeaconBlockAccessList()
	if additional == nil {
		b.Slot.Append(0)
		b.SlotStartDateTime.Append(time.Time{})
		b.Epoch.Append(0)
		b.EpochStartDateTime.Append(time.Time{})
		b.BlockRoot.Append(nil)
		b.BlockNumber.Append(0)
		b.BlockHash.Append(nil)

		return
	}

	appendBlockIdentifier(additional.GetBlock(),
		&b.Slot, &b.SlotStartDateTime, &b.Epoch, &b.EpochStartDateTime, nil, &b.BlockRoot)

	if blockNumber := additional.GetBlockNumber(); blockNumber != nil {
		b.BlockNumber.Append(blockNumber.GetValue())
	} else {
		b.BlockNumber.Append(0)
	}

	b.BlockHash.Append([]byte(additional.GetBlockHash()))
}
