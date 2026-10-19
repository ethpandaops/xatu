package canonical

import (
	"encoding/hex"
	"fmt"
	"strings"
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

	graffiti, err := b.validate(event)
	if err != nil {
		return err
	}

	b.appendRuntime(event)
	b.appendMetadata(event)
	b.appendPayload(event, graffiti)
	b.appendAdditionalData(event)
	b.rows++

	return nil
}

func (b *canonicalBeaconBlockGraffitiBatch) validate(event *xatu.DecoratedEvent) ([]byte, error) {
	payload := event.GetEthV2BeaconBlockGraffiti()

	if payload.GetProposerIndex() == nil {
		return nil, fmt.Errorf("nil ProposerIndex: %w", route.ErrInvalidEvent)
	}

	graffiti, err := hex.DecodeString(strings.TrimPrefix(payload.GetGraffiti(), "0x"))
	if err != nil || len(graffiti) != 32 {
		return nil, fmt.Errorf("invalid Graffiti %q: %w", payload.GetGraffiti(), route.ErrInvalidEvent)
	}

	if event.GetMeta().GetClient().GetEthV2BeaconBlockGraffiti().GetBlock() == nil {
		return nil, fmt.Errorf("nil Block: %w", route.ErrInvalidEvent)
	}

	return graffiti, nil
}

func (b *canonicalBeaconBlockGraffitiBatch) appendRuntime(_ *xatu.DecoratedEvent) {
	b.UpdatedDateTime.Append(time.Now())
}

func (b *canonicalBeaconBlockGraffitiBatch) appendPayload(event *xatu.DecoratedEvent, graffiti []byte) {
	payload := event.GetEthV2BeaconBlockGraffiti()

	b.ProposerIndex.Append(uint32(payload.GetProposerIndex().GetValue())) //nolint:gosec // G115
	b.Graffiti.Append([]byte(payload.GetGraffiti()))
	b.GraffitiText.Append(graffitiText(graffiti))
}

func (b *canonicalBeaconBlockGraffitiBatch) appendAdditionalData(event *xatu.DecoratedEvent) {
	additional := event.GetMeta().GetClient().GetEthV2BeaconBlockGraffiti()
	appendBlockIdentifier(additional.GetBlock(),
		&b.Slot, &b.SlotStartDateTime, &b.Epoch, &b.EpochStartDateTime, &b.BlockVersion, &b.BlockRoot)
}

func graffitiText(graffiti []byte) string {
	return strings.ToValidUTF8(strings.TrimRight(string(graffiti), "\x00"), "�")
}
