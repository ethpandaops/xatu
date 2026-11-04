package cannon

import (
	"errors"
	"testing"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

// A request with Type set but the Data oneof left empty is a legal wire
// message. Marshal used to accept it and silently write an empty JSON
// object as the value, which reset whatever checkpoint that location
// represented. It should now reject the request instead.
func TestMarshal_TypeSetDataAbsent_ReturnsError(t *testing.T) {
	msg := &xatu.CannonLocation{
		NetworkId: "mainnet",
		Type:      xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_VOLUNTARY_EXIT,
	}

	l := &Location{}

	err := l.Marshal(msg)
	if err == nil {
		t.Fatalf("expected an error, got nil (l.Value = %q)", l.Value)
	}

	if !errors.Is(err, ErrLocationDataRequired) {
		t.Fatalf("expected ErrLocationDataRequired, got: %v", err)
	}

	if l.Value != "" {
		t.Fatalf("l.Value should be untouched on rejection, got %q", l.Value)
	}
}

// Every Type-set/Data-absent case is expected to go through the same
// marshalLocationData check, so a few representative types are covered
// explicitly here too.
func TestMarshal_TypeSetDataAbsent_ReturnsError_AcrossTypes(t *testing.T) {
	cases := []struct {
		name string
		typ  xatu.CannonType
	}{
		{"EXECUTION_CANONICAL_BLOCK", xatu.CannonType_EXECUTION_CANONICAL_BLOCK},
		{"EXECUTION_CANONICAL_TRANSACTION", xatu.CannonType_EXECUTION_CANONICAL_TRANSACTION},
		{"BEACON_API_ETH_V1_BEACON_STATE_FINALITY_CHECKPOINT", xatu.CannonType_BEACON_API_ETH_V1_BEACON_STATE_FINALITY_CHECKPOINT},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			l := &Location{}

			err := l.Marshal(&xatu.CannonLocation{NetworkId: "mainnet", Type: tc.typ})
			if !errors.Is(err, ErrLocationDataRequired) {
				t.Fatalf("expected ErrLocationDataRequired for %s, got: %v", tc.name, err)
			}
		})
	}
}

// Rows written before this validation existed may still contain an empty
// marker. Unmarshal should keep reading them back without error, since
// there is no migration to clean up historical rows.
func TestUnmarshalOfLegacyEmptyMarker_StillReadsBackSafely(t *testing.T) {
	l := &Location{
		NetworkID: "mainnet",
		Type:      "BEACON_API_ETH_V2_BEACON_BLOCK_VOLUNTARY_EXIT",
		Value:     "{}",
	}

	msg, err := l.Unmarshal()
	if err != nil {
		t.Fatalf("Unmarshal returned an error reading legacy data: %v", err)
	}

	data := msg.GetEthV2BeaconBlockVoluntaryExit()
	if data == nil {
		t.Fatal("expected a non-nil, zero-valued VoluntaryExit data message")
	}
}
