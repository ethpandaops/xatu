package canonical

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/ethpandaops/xatu/pkg/clickhouse/route"
	"github.com/ethpandaops/xatu/pkg/clickhouse/route/testfixture"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

// "GEa579PMfea2" right-padded with null bytes.
const testGraffiti = "0x474561353739504d666561320000000000000000000000000000000000000000"

func blockGraffitiEvent() *xatu.DecoratedEvent {
	return &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_GRAFFITI,
			DateTime: testfixture.TS(),
			Id:       "cbg-1",
		},
		Meta: testfixture.MetaWithAdditional(&xatu.ClientMeta{
			AdditionalData: &xatu.ClientMeta_EthV2BeaconBlockGraffiti{
				EthV2BeaconBlockGraffiti: &xatu.ClientMeta_AdditionalEthV2BeaconBlockGraffitiData{
					Block: &xatu.BlockIdentifier{
						Epoch:   testfixture.EpochAdditional(),
						Slot:    testfixture.SlotEpochAdditional(),
						Root:    "0x00000000000000000000000000000000000000000000000000000000000000bb",
						Version: "fulu",
					},
				},
			},
		}),
		Data: &xatu.DecoratedEvent_EthV2BeaconBlockGraffiti{
			EthV2BeaconBlockGraffiti: &xatu.BeaconBlockGraffitiData{
				ProposerIndex: wrapperspb.UInt64(72898),
				Graffiti:      testGraffiti,
			},
		},
	}
}

func TestSnapshot_canonical_beacon_block_graffiti(t *testing.T) {
	testfixture.AssertSnapshot(t, newcanonicalBeaconBlockGraffitiBatch(), blockGraffitiEvent(), 1, map[string]any{
		"slot":              uint32(100),
		"epoch":             uint32(3),
		"block_root":        "0x00000000000000000000000000000000000000000000000000000000000000bb",
		"block_version":     "fulu",
		"proposer_index":    uint32(72898),
		"graffiti":          testGraffiti,
		"graffiti_text":     "GEa579PMfea2",
		"meta_network_name": "mainnet",
	})
}

func TestGraffitiText(t *testing.T) {
	tests := map[string]struct {
		graffiti []byte
		want     string
	}{
		"empty":            {graffiti: make([]byte, 32), want: ""},
		"full width":       {graffiti: []byte("0123456789abcdef0123456789abcdef"), want: "0123456789abcdef0123456789abcdef"},
		"interior null":    {graffiti: append([]byte("a\x00b"), make([]byte, 29)...), want: "a\x00b"},
		"emoji":            {graffiti: append([]byte("stake \xf0\x9f\x90\xa0"), make([]byte, 22)...), want: "stake \U0001F420"},
		"invalid sequence": {graffiti: append([]byte{'o', 'k', 0xff, 0xfe}, make([]byte, 28)...), want: "ok�"},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tt.want, graffitiText(tt.graffiti))
		})
	}
}

func TestCanonicalBeaconBlockGraffiti_RejectsInvalidEvents(t *testing.T) {
	tests := map[string]func(*xatu.DecoratedEvent){
		"payload": func(e *xatu.DecoratedEvent) {
			e.Data = &xatu.DecoratedEvent_EthV2BeaconBlockGraffiti{}
		},
		"proposer index": func(e *xatu.DecoratedEvent) {
			e.GetEthV2BeaconBlockGraffiti().ProposerIndex = nil
		},
		"empty graffiti": func(e *xatu.DecoratedEvent) {
			e.GetEthV2BeaconBlockGraffiti().Graffiti = ""
		},
		"short graffiti": func(e *xatu.DecoratedEvent) {
			e.GetEthV2BeaconBlockGraffiti().Graffiti = "0x4745"
		},
		"non hex graffiti": func(e *xatu.DecoratedEvent) {
			e.GetEthV2BeaconBlockGraffiti().Graffiti = "0x" + "zz" + testGraffiti[4:]
		},
		"block": func(e *xatu.DecoratedEvent) {
			e.GetMeta().GetClient().GetEthV2BeaconBlockGraffiti().Block = nil
		},
		"additional data": func(e *xatu.DecoratedEvent) {
			e.Meta.Client.AdditionalData = nil
		},
	}

	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			event, ok := proto.Clone(blockGraffitiEvent()).(*xatu.DecoratedEvent)
			require.True(t, ok)

			mutate(event)

			batch := newcanonicalBeaconBlockGraffitiBatch()
			err := batch.FlattenTo(event)

			require.Error(t, err)
			assert.True(t, errors.Is(err, route.ErrInvalidEvent))
			assert.Equal(t, 0, batch.Rows())
		})
	}
}
