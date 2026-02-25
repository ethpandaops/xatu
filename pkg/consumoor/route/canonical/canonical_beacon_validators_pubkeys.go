package canonical

import (
	"fmt"
	"time"

	"github.com/ethpandaops/xatu/pkg/consumoor/route"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

var canonicalBeaconValidatorsPubkeysEventNames = []xatu.Event_Name{
	xatu.Event_BEACON_API_ETH_V1_BEACON_VALIDATORS,
}

func init() {
	r, err := route.NewStaticRoute(
		canonicalBeaconValidatorsPubkeysTableName,
		canonicalBeaconValidatorsPubkeysEventNames,
		func() route.ColumnarBatch { return newcanonicalBeaconValidatorsPubkeysBatch() },
	)
	if err != nil {
		route.RecordError(err)

		return
	}

	if err := route.Register(r); err != nil {
		route.RecordError(err)
	}
}

//nolint:gosec // G115: proto uint64 values are bounded by ClickHouse uint32 column schema
func (b *canonicalBeaconValidatorsPubkeysBatch) FlattenTo(event *xatu.DecoratedEvent) error {
	if event == nil || event.GetEvent() == nil {
		return nil
	}

	if event.GetEthV1Validators() == nil {
		return fmt.Errorf("nil payload: %w", route.ErrInvalidEvent)
	}

	if err := b.validate(event); err != nil {
		return err
	}

	var (
		epoch          uint32
		epochStartTime time.Time
	)

	if client := event.GetMeta().GetClient(); client != nil {
		if extra := client.GetEthV1Validators(); extra != nil {
			if epochData := extra.GetEpoch(); epochData != nil {
				if number := epochData.GetNumber(); number != nil {
					epoch = uint32(number.GetValue())
				}

				if start := epochData.GetStartDateTime(); start != nil {
					epochStartTime = start.AsTime()
				}
			}
		}
	}

	now := time.Now()

	for _, validator := range event.GetEthV1Validators().GetValidators() {
		if validator == nil {
			continue
		}

		if validator.GetIndex() == nil {
			continue
		}

		b.UpdatedDateTime.Append(now)
		b.Version.Append(uint32(4294967295 - epochStartTime.Unix())) //nolint:gosec // inverse timestamp for ReplacingMergeTree dedup
		b.Epoch.Append(epoch)
		b.EpochStartDateTime.Append(epochStartTime)

		b.Index.Append(uint32(validator.GetIndex().GetValue()))

		if data := validator.GetData(); data != nil && data.GetPubkey() != nil {
			b.Pubkey.Append(data.GetPubkey().GetValue())
		} else {
			b.Pubkey.Append("")
		}

		b.appendMetadata(event)
		b.rows++
	}

	return nil
}

func (b *canonicalBeaconValidatorsPubkeysBatch) validate(event *xatu.DecoratedEvent) error {
	extra := event.GetMeta().GetClient().GetEthV1Validators()
	if extra == nil || extra.GetEpoch() == nil || extra.GetEpoch().GetNumber() == nil {
		return fmt.Errorf("nil Epoch: %w", route.ErrInvalidEvent)
	}

	return nil
}
