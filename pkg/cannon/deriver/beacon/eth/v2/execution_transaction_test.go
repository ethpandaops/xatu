package v2

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethpandaops/go-eth2-client/spec"
	"github.com/ethpandaops/go-eth2-client/spec/capella"
)

// capellaBlockWithBaseFee builds a minimal Capella block whose execution payload
// carries the given base fee (stored little-endian, as on the wire).
func capellaBlockWithBaseFee(baseFee uint64) *spec.VersionedSignedBeaconBlock {
	var le [32]byte

	v := baseFee
	for i := 0; v > 0; i++ {
		le[i] = byte(v & 0xff)
		v >>= 8
	}

	return &spec.VersionedSignedBeaconBlock{
		Version: spec.DataVersionCapella,
		Capella: &capella.SignedBeaconBlock{
			Message: &capella.BeaconBlock{
				Body: &capella.BeaconBlockBody{
					ExecutionPayload: &capella.ExecutionPayload{
						BaseFeePerGasLE: le,
					},
				},
			},
		},
	}
}

// TestGetGasPriceCapella verifies that gas price derivation works for Capella blocks.
// Capella exposes the same little-endian BaseFeePerGasLE layout as Bellatrix; before the
// fix the switch had no Capella arm and every EIP-1559 transaction in a Capella block
// returned "unknown block version", permanently halting execution-transaction backfill
// across the Capella range.
func TestGetGasPriceCapella(t *testing.T) {
	const baseFee = 100

	t.Run("eip1559 base fee plus tip", func(t *testing.T) {
		tx := types.NewTx(&types.DynamicFeeTx{
			ChainID:   big.NewInt(1),
			Nonce:     0,
			GasTipCap: big.NewInt(10),
			GasFeeCap: big.NewInt(1000),
			Gas:       21000,
			Value:     big.NewInt(0),
		})

		got, err := GetGasPrice(capellaBlockWithBaseFee(baseFee), tx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// effective gas price = min(maxFeePerGas, baseFee + maxPriorityFeePerGas)
		if want := big.NewInt(baseFee + 10); got.Cmp(want) != 0 {
			t.Fatalf("gas price = %s, want %s", got, want)
		}
	})

	t.Run("capped by max fee per gas", func(t *testing.T) {
		tx := types.NewTx(&types.DynamicFeeTx{
			ChainID:   big.NewInt(1),
			Nonce:     0,
			GasTipCap: big.NewInt(50),
			GasFeeCap: big.NewInt(120),
			Gas:       21000,
			Value:     big.NewInt(0),
		})

		got, err := GetGasPrice(capellaBlockWithBaseFee(baseFee), tx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// baseFee + tip = 150, but capped at GasFeeCap = 120.
		if want := big.NewInt(120); got.Cmp(want) != 0 {
			t.Fatalf("gas price = %s, want %s", got, want)
		}
	})

	t.Run("legacy transaction uses gas price directly", func(t *testing.T) {
		tx := types.NewTx(&types.LegacyTx{
			Nonce:    0,
			GasPrice: big.NewInt(500),
			Gas:      21000,
			Value:    big.NewInt(0),
		})

		got, err := GetGasPrice(capellaBlockWithBaseFee(baseFee), tx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if want := big.NewInt(500); got.Cmp(want) != 0 {
			t.Fatalf("gas price = %s, want %s", got, want)
		}
	})
}
