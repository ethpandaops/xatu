package canonical

import (
	"time"

	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/flattener"
	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/metadata"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

var canonicalBeaconValidatorsPubkeysEventNames = []xatu.Event_Name{
	xatu.Event_BEACON_API_ETH_V1_BEACON_VALIDATORS,
}

func init() {
	flattener.MustRegister(flattener.NewStaticRoute(
		canonicalBeaconValidatorsPubkeysTableName,
		canonicalBeaconValidatorsPubkeysEventNames,
		func() flattener.ColumnarBatch { return newcanonicalBeaconValidatorsPubkeysBatch() },
	))
}

//nolint:gosec // G115: proto uint64 values are bounded by ClickHouse uint32 column schema
func (b *canonicalBeaconValidatorsPubkeysBatch) FlattenTo(
	event *xatu.DecoratedEvent,
	meta *metadata.CommonMetadata,
) error {
	if event == nil || event.GetEvent() == nil || event.GetEthV1Validators() == nil {
		return nil
	}

	if meta == nil {
		meta = metadata.Extract(event)
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

		b.UpdatedDateTime.Append(now)
		b.Version.Append(uint32(4294967295 - epochStartTime.Unix())) //nolint:gosec // inverse timestamp for ReplacingMergeTree dedup
		b.Epoch.Append(epoch)
		b.EpochStartDateTime.Append(epochStartTime)

		if idx := validator.GetIndex(); idx != nil {
			b.Index.Append(uint32(idx.GetValue()))
		} else {
			b.Index.Append(0)
		}

		if data := validator.GetData(); data != nil && data.GetPubkey() != nil {
			b.Pubkey.Append(data.GetPubkey().GetValue())
		} else {
			b.Pubkey.Append("")
		}

		b.appendMetadata(meta)
		b.rows++
	}

	return nil
}
