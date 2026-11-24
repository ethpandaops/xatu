package ethereum

import (
	"testing"

	"github.com/ethpandaops/go-eth2-client/spec"
	"github.com/ethpandaops/go-eth2-client/spec/heze"
	"github.com/ethpandaops/go-eth2-client/spec/phase0"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func hash32(b byte) phase0.Hash32 {
	var h phase0.Hash32

	h[0] = b

	return h
}

func TestBidPayloadWithheld(t *testing.T) {
	parent := hash32(0x01)
	promised := hash32(0x02)
	unrelated := hash32(0x03)

	bid := &payloadBid{
		ParentBlockHash: parent,
		BlockHash:       promised,
	}

	tests := []struct {
		name         string
		nextParent   phase0.Hash32
		wantWithheld bool
		wantErr      bool
	}{
		{
			name:         "next bid builds on promised hash - payload revealed",
			nextParent:   promised,
			wantWithheld: false,
		},
		{
			name:         "next bid builds on parent hash - payload withheld",
			nextParent:   parent,
			wantWithheld: true,
		},
		{
			name:       "next bid builds on unrelated hash - inconsistent view",
			nextParent: unrelated,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nextBid := &payloadBid{
				ParentBlockHash: tt.nextParent,
				BlockHash:       hash32(0x04),
			}

			withheld, err := bidPayloadWithheld(bid, nextBid)
			if tt.wantErr {
				require.Error(t, err)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantWithheld, withheld)
		})
	}
}

// A block whose payload was withheld leaves the EL head unchanged, so a run of
// consecutive withheld payloads all bid on the same parent hash. Ensure the
// decision still resolves for the earlier blocks in such a run.
func TestBidPayloadWithheldConsecutiveWithheld(t *testing.T) {
	elHead := hash32(0x0a)

	blockN := &payloadBid{
		ParentBlockHash: elHead,
		BlockHash:       hash32(0x0b),
	}
	blockN1 := &payloadBid{
		ParentBlockHash: elHead,
		BlockHash:       hash32(0x0c),
	}

	withheld, err := bidPayloadWithheld(blockN, blockN1)
	require.NoError(t, err)
	assert.True(t, withheld)
}

// The withheld-payload check walks Heze blocks too, whose bid is a different type.
func TestGloasBidHeze(t *testing.T) {
	block := &spec.VersionedSignedBeaconBlock{
		Version: spec.DataVersionHeze,
		Heze: &heze.SignedBeaconBlock{Message: &heze.BeaconBlock{
			Slot: 251427,
			Body: &heze.BeaconBlockBody{SignedExecutionPayloadBid: &heze.SignedExecutionPayloadBid{
				Message: &heze.ExecutionPayloadBid{ParentBlockHash: hash32(0x01), BlockHash: hash32(0x02)},
			}},
		}},
	}

	bid, slot, err := gloasBid(block)
	require.NoError(t, err)
	assert.Equal(t, phase0.Slot(251427), slot)
	assert.Equal(t, &payloadBid{ParentBlockHash: hash32(0x01), BlockHash: hash32(0x02)}, bid)
}
