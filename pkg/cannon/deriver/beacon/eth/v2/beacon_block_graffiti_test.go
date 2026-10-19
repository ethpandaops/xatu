package v2

import (
	"testing"

	"github.com/ethpandaops/go-eth2-client/spec"
	"github.com/ethpandaops/go-eth2-client/spec/gloas"
	"github.com/ethpandaops/go-eth2-client/spec/phase0"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

func graffitiBytes(s string) [32]byte {
	var g [32]byte

	copy(g[:], s)

	return g
}

func TestBeaconBlockGraffitiDeriver_DeriveEvents(t *testing.T) {
	deriver := &BeaconBlockGraffitiDeriver{clientMeta: &xatu.ClientMeta{Name: "cannon-test"}}
	identifier := testBlockIdentifier()

	tests := []struct {
		name          string
		block         *spec.VersionedSignedBeaconBlock
		proposerIndex uint64
		graffiti      string
	}{
		{
			name: "phase0",
			block: &spec.VersionedSignedBeaconBlock{
				Version: spec.DataVersionPhase0,
				Phase0: &phase0.SignedBeaconBlock{Message: &phase0.BeaconBlock{
					Slot:          1,
					ProposerIndex: 7,
					Body:          &phase0.BeaconBlockBody{Graffiti: graffitiBytes("Lighthouse/v1.0.0")},
				}},
			},
			proposerIndex: 7,
			graffiti:      "0x4c69676874686f7573652f76312e302e30000000000000000000000000000000",
		},
		{
			name: "gloas with client version codes",
			block: &spec.VersionedSignedBeaconBlock{
				Version: spec.DataVersionGloas,
				Gloas: &gloas.SignedBeaconBlock{Message: &gloas.BeaconBlock{
					Slot:          123,
					ProposerIndex: 1536979,
					Body:          &gloas.BeaconBlockBody{Graffiti: graffitiBytes("GEa579PMfea2")},
				}},
			},
			proposerIndex: 1536979,
			graffiti:      "0x474561353739504d666561320000000000000000000000000000000000000000",
		},
		{
			name:          "empty graffiti",
			block:         gloasBlock(5),
			proposerIndex: 0,
			graffiti:      "0x0000000000000000000000000000000000000000000000000000000000000000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			events, err := deriver.deriveEvents(tt.block, identifier)
			require.NoError(t, err)
			require.Len(t, events, 1)

			event := events[0]
			assert.Equal(t, xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_GRAFFITI, event.GetEvent().GetName())
			assert.Equal(t, "cannon-test", event.GetMeta().GetClient().GetName())

			data := event.GetEthV2BeaconBlockGraffiti()
			require.NotNil(t, data)
			assert.Equal(t, tt.proposerIndex, data.GetProposerIndex().GetValue())
			assert.Equal(t, tt.graffiti, data.GetGraffiti())

			additional := event.GetMeta().GetClient().GetEthV2BeaconBlockGraffiti()
			require.NotNil(t, additional)
			assert.Same(t, identifier, additional.GetBlock())
		})
	}

	t.Run("block without a body is an error", func(t *testing.T) {
		block := &spec.VersionedSignedBeaconBlock{
			Version: spec.DataVersionHeze,
		}

		events, err := deriver.deriveEvents(block, identifier)
		require.Error(t, err)
		assert.Nil(t, events)
	})

	t.Run("does not mutate the shared client metadata", func(t *testing.T) {
		_, err := deriver.deriveEvents(gloasBlock(5), identifier)
		require.NoError(t, err)
		assert.Nil(t, deriver.clientMeta.GetAdditionalData())
	})
}
