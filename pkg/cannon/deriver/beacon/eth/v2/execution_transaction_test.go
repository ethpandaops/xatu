package v2

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethpandaops/go-eth2-client/spec"
	"github.com/ethpandaops/go-eth2-client/spec/bellatrix"
	"github.com/ethpandaops/go-eth2-client/spec/capella"
	"github.com/ethpandaops/go-eth2-client/spec/deneb"
	"github.com/ethpandaops/go-eth2-client/spec/electra"
	"github.com/ethpandaops/go-eth2-client/spec/gloas"
	"github.com/ethpandaops/go-eth2-client/spec/phase0"
	"github.com/holiman/uint256"
)

func TestGetGasPrice_Legacy(t *testing.T) {
	tx := types.NewTx(&types.LegacyTx{
		GasPrice: big.NewInt(1_000),
		Gas:      21_000,
	})

	got, err := GetGasPrice(&spec.VersionedSignedBeaconBlock{Version: spec.DataVersionGloas}, tx, nil)
	if err != nil {
		t.Fatalf("unexpected error for legacy tx: %v", err)
	}

	if got.Cmp(big.NewInt(1_000)) != 0 {
		t.Errorf("legacy gas price: got %s want 1000", got)
	}
}

func TestGetGasPrice_AccessList(t *testing.T) {
	tx := types.NewTx(&types.AccessListTx{
		GasPrice: big.NewInt(2_000),
		Gas:      21_000,
	})

	// Type-1 is also fork-agnostic — the block argument is unused for this branch.
	got, err := GetGasPrice(&spec.VersionedSignedBeaconBlock{Version: spec.DataVersionGloas}, tx, nil)
	if err != nil {
		t.Fatalf("unexpected error for access-list tx: %v", err)
	}

	if got.Cmp(big.NewInt(2_000)) != 0 {
		t.Errorf("access-list gas price: got %s want 2000", got)
	}
}

func TestGetGasPrice_DynamicFee_PreGloas(t *testing.T) {
	dynamicFeeTx := func() *types.Transaction {
		return types.NewTx(&types.DynamicFeeTx{
			Gas:       21_000,
			GasTipCap: big.NewInt(2),
			GasFeeCap: big.NewInt(100),
		})
	}

	tests := []struct {
		name  string
		block *spec.VersionedSignedBeaconBlock
		want  *big.Int
	}{
		{
			// Bellatrix's BaseFeePerGas is a fixed [32]byte — the existing
			// production code reads it via big.Int.SetBytes (big-endian), so
			// put the value in the least-significant byte to encode 10.
			name: "Bellatrix base_fee=10",
			block: &spec.VersionedSignedBeaconBlock{
				Version: spec.DataVersionBellatrix,
				Bellatrix: &bellatrix.SignedBeaconBlock{
					Message: &bellatrix.BeaconBlock{
						Body: &bellatrix.BeaconBlockBody{
							ExecutionPayload: &bellatrix.ExecutionPayload{
								BaseFeePerGas: [32]byte{31: 10},
							},
						},
					},
				},
			},
			want: big.NewInt(12), // baseFee 10 + tip 2 = 12, below cap 100
		},
		{
			name: "Deneb base_fee=20",
			block: &spec.VersionedSignedBeaconBlock{
				Version: spec.DataVersionDeneb,
				Deneb: &deneb.SignedBeaconBlock{
					Message: &deneb.BeaconBlock{
						Body: &deneb.BeaconBlockBody{
							ExecutionPayload: &deneb.ExecutionPayload{
								BaseFeePerGas: uint256.NewInt(20),
							},
						},
					},
				},
			},
			want: big.NewInt(22),
		},
		{
			name: "Electra base_fee=30",
			block: &spec.VersionedSignedBeaconBlock{
				Version: spec.DataVersionElectra,
				Electra: &electra.SignedBeaconBlock{
					Message: &electra.BeaconBlock{
						Body: &electra.BeaconBlockBody{
							ExecutionPayload: &deneb.ExecutionPayload{
								BaseFeePerGas: uint256.NewInt(30),
							},
						},
					},
				},
			},
			want: big.NewInt(32),
		},
		{
			name: "Fulu base_fee=40",
			block: &spec.VersionedSignedBeaconBlock{
				Version: spec.DataVersionFulu,
				// Fulu reuses electra.SignedBeaconBlock in the SDK — no
				// dedicated fulu.* types exist for the block.
				Fulu: &electra.SignedBeaconBlock{
					Message: &electra.BeaconBlock{
						Body: &electra.BeaconBlockBody{
							ExecutionPayload: &deneb.ExecutionPayload{
								BaseFeePerGas: uint256.NewInt(40),
							},
						},
					},
				},
			},
			want: big.NewInt(42),
		},
		{
			name: "FeeCap clamps very high baseFee",
			block: &spec.VersionedSignedBeaconBlock{
				Version: spec.DataVersionDeneb,
				Deneb: &deneb.SignedBeaconBlock{
					Message: &deneb.BeaconBlock{
						Body: &deneb.BeaconBlockBody{
							ExecutionPayload: &deneb.ExecutionPayload{
								BaseFeePerGas: uint256.NewInt(1_000),
							},
						},
					},
				},
			},
			want: big.NewInt(100), // baseFee 1000 + tip 2 > cap 100, clamped
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetGasPrice(tt.block, dynamicFeeTx(), nil)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if got.Cmp(tt.want) != 0 {
				t.Errorf("got %s want %s", got, tt.want)
			}
		})
	}
}

func TestGetGasPrice_Gloas_NilEnvelope(t *testing.T) {
	tx := types.NewTx(&types.DynamicFeeTx{
		Gas:       21_000,
		GasTipCap: big.NewInt(2),
		GasFeeCap: big.NewInt(100),
	})

	block := &spec.VersionedSignedBeaconBlock{Version: spec.DataVersionGloas}

	if _, err := GetGasPrice(block, tx, nil); err == nil {
		t.Fatal("expected error for Gloas EIP-1559 tx with nil envelope")
	}

	// Also reject a partially-formed envelope.
	if _, err := GetGasPrice(block, tx, &gloas.SignedExecutionPayloadEnvelope{}); err == nil {
		t.Fatal("expected error for Gloas EIP-1559 tx with envelope.Message == nil")
	}

	if _, err := GetGasPrice(block, tx, &gloas.SignedExecutionPayloadEnvelope{
		Message: &gloas.ExecutionPayloadEnvelope{},
	}); err == nil {
		t.Fatal("expected error for Gloas EIP-1559 tx with envelope.Message.Payload == nil")
	}
}

func TestGetGasPrice_Gloas_WithEnvelope(t *testing.T) {
	tx := types.NewTx(&types.DynamicFeeTx{
		Gas:       21_000,
		GasTipCap: big.NewInt(3),
		GasFeeCap: big.NewInt(100),
	})

	envelope := &gloas.SignedExecutionPayloadEnvelope{
		Message: &gloas.ExecutionPayloadEnvelope{
			Payload: &gloas.ExecutionPayload{
				ParentHash:    phase0.Hash32{0x01},
				FeeRecipient:  bellatrix.ExecutionAddress{},
				Withdrawals:   []*capella.Withdrawal{},
				BaseFeePerGas: uint256.NewInt(50),
			},
		},
	}

	block := &spec.VersionedSignedBeaconBlock{Version: spec.DataVersionGloas}

	got, err := GetGasPrice(block, tx, envelope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got.Cmp(big.NewInt(53)) != 0 {
		t.Errorf("Gloas gas price: got %s want 53 (baseFee 50 + tip 3)", got)
	}
}

func TestGetGasPrice_UnknownTxType(t *testing.T) {
	// SetCodeTx (type 4) is supported; any other type beyond {0,1,2,3,4}
	// falls through to the "unknown transaction type" error. Currently
	// go-ethereum's types package only defines up through type 4, so this
	// test just guards the error path against a hypothetical type-5.
	// Use a DynamicFeeTx with an unknown version on the block to trigger
	// the "unknown block version" branch instead, which is reachable today.
	tx := types.NewTx(&types.DynamicFeeTx{
		Gas:       21_000,
		GasTipCap: big.NewInt(1),
		GasFeeCap: big.NewInt(2),
	})

	block := &spec.VersionedSignedBeaconBlock{Version: spec.DataVersionUnknown}

	if _, err := GetGasPrice(block, tx, nil); err == nil {
		t.Fatal("expected error for unknown block version with EIP-1559 tx")
	}
}
