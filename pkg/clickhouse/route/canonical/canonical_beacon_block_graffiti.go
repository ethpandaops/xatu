package canonical

import (
	"fmt"
	"time"

	"github.com/ethpandaops/xatu/pkg/clickhouse/route"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

var canonicalBeaconBlockGraffitiEventNames = []xatu.Event_Name{
	xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_GRAFFITI,
}

func init() {
	r, err := route.NewStaticRoute(
		canonicalBeaconBlockGraffitiTableName,
		canonicalBeaconBlockGraffitiEventNames,
		func() route.ColumnarBatch { return newcanonicalBeaconBlockGraffitiBatch() },
	)
	if err != nil {
		route.RecordError(err)

		return
	}

	if err := route.Register(r); err != nil {
		route.RecordError(err)
	}
}

func (b *canonicalBeaconBlockGraffitiBatch) FlattenTo(event *xatu.DecoratedEvent) error {
	if event == nil || event.GetEvent() == nil {
		return nil
	}

	if event.GetEthV2BeaconBlockGraffiti() == nil {
		return fmt.Errorf("nil eth_v2_beacon_block_graffiti payload: %w", route.ErrInvalidEvent)
	}

	graffitiText, err := b.validate(event)
	if err != nil {
		return err
	}

	b.appendRuntime(event)
	b.appendMetadata(event)
	b.appendPayload(event, graffitiText)
	b.appendAdditionalData(event)
	b.rows++

	return nil
}

func (b *canonicalBeaconBlockGraffitiBatch) validate(event *xatu.DecoratedEvent) (string, error) {
	payload := event.GetEthV2BeaconBlockGraffiti()

	if payload.GetProposerIndex() == nil {
		return "", fmt.Errorf("nil ProposerIndex: %w", route.ErrInvalidEvent)
	}

	_, text, ok := route.DecodeGraffiti(payload.GetGraffiti())
	if !ok {
		return "", fmt.Errorf("invalid Graffiti %q: %w", payload.GetGraffiti(), route.ErrInvalidEvent)
	}

	if event.GetMeta().GetClient().GetEthV2BeaconBlockGraffiti().GetBlock() == nil {
		return "", fmt.Errorf("nil Block: %w", route.ErrInvalidEvent)
	}

	return text, nil
}

func (b *canonicalBeaconBlockGraffitiBatch) appendRuntime(_ *xatu.DecoratedEvent) {
	b.UpdatedDateTime.Append(time.Now())
}

func (b *canonicalBeaconBlockGraffitiBatch) appendPayload(event *xatu.DecoratedEvent, graffitiText string) {
	payload := event.GetEthV2BeaconBlockGraffiti()

	b.ProposerIndex.Append(uint32(payload.GetProposerIndex().GetValue())) //nolint:gosec // G115
	b.Graffiti.Append([]byte(payload.GetGraffiti()))
	b.GraffitiText.Append(graffitiText)
}

func (b *canonicalBeaconBlockGraffitiBatch) appendAdditionalData(event *xatu.DecoratedEvent) {
	additional := event.GetMeta().GetClient().GetEthV2BeaconBlockGraffiti()
	appendBlockIdentifier(additional.GetBlock(),
		&b.Slot, &b.SlotStartDateTime, &b.Epoch, &b.EpochStartDateTime, &b.BlockVersion, &b.BlockRoot)
}
