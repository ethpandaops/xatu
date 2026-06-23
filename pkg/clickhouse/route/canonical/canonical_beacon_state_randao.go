package canonical

import (
	"fmt"
	"time"

	"github.com/ethpandaops/xatu/pkg/clickhouse/route"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

var canonicalBeaconStateRandaoEventNames = []xatu.Event_Name{
	xatu.Event_BEACON_API_ETH_V1_BEACON_STATE_RANDAO,
}

func init() {
	r, err := route.NewStaticRoute(
		canonicalBeaconStateRandaoTableName,
		canonicalBeaconStateRandaoEventNames,
		func() route.ColumnarBatch { return newcanonicalBeaconStateRandaoBatch() },
	)
	if err != nil {
		route.RecordError(err)

		return
	}

	if err := route.Register(r); err != nil {
		route.RecordError(err)
	}
}

func (b *canonicalBeaconStateRandaoBatch) FlattenTo(event *xatu.DecoratedEvent) error {
	if event == nil || event.GetEvent() == nil {
		return nil
	}

	payload := event.GetEthV1BeaconStateRandao()
	if payload == nil {
		return fmt.Errorf("nil payload: %w", route.ErrInvalidEvent)
	}

	if err := b.validate(event); err != nil {
		return err
	}

	var (
		epoch          uint32
		epochStartTime time.Time
		stateID        string
	)

	if client := event.GetMeta().GetClient(); client != nil {
		if extra := client.GetEthV1BeaconStateRandao(); extra != nil {
			stateID = extra.GetStateId()

			if epochData := extra.GetEpoch(); epochData != nil {
				if number := epochData.GetNumber(); number != nil {
					epoch = uint32(number.GetValue()) //nolint:gosec // bounded by uint32 column
				}

				if start := epochData.GetStartDateTime(); start != nil {
					epochStartTime = start.AsTime()
				}
			}
		}
	}

	b.UpdatedDateTime.Append(time.Now())
	b.Epoch.Append(epoch)
	b.EpochStartDateTime.Append(epochStartTime)
	b.StateID.Append(stateID)
	b.Randao.Append([]byte(payload.GetRandao()))
	b.appendMetadata(event)
	b.rows++

	return nil
}

func (b *canonicalBeaconStateRandaoBatch) validate(event *xatu.DecoratedEvent) error {
	extra := event.GetMeta().GetClient().GetEthV1BeaconStateRandao()
	if extra == nil || extra.GetEpoch() == nil || extra.GetEpoch().GetNumber() == nil {
		return fmt.Errorf("nil Epoch: %w", route.ErrInvalidEvent)
	}

	return nil
}
