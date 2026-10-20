package v2

import (
	"context"
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
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ethpandaops/xatu/pkg/cannon/ethereum"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
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
			// Bellatrix's BaseFeePerGasLE is a fixed little-endian [32]byte —
			// the production code reverses it to big-endian then SetBytes,
			// so put the value in the least-significant byte (index 0) to encode 10.
			name: "Bellatrix base_fee=10",
			block: &spec.VersionedSignedBeaconBlock{
				Version: spec.DataVersionBellatrix,
				Bellatrix: &bellatrix.SignedBeaconBlock{
					Message: &bellatrix.BeaconBlock{
						Body: &bellatrix.BeaconBlockBody{
							ExecutionPayload: &bellatrix.ExecutionPayload{
								BaseFeePerGasLE: [32]byte{0: 10},
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

// Mainnet slot 6316823 is a Capella block; expected prices are the EL receipts' effectiveGasPrice.
func TestGetGasPrice_CapellaBlock(t *testing.T) {
	block := loadEraBlock(t, "mainnet-6316823", spec.DataVersionCapella)

	want := []int64{
		41000000000, 40051643706, 38051643706, 38051643706, 38000000000, 37833130593,
		37430618558, 37401643706, 37351643706, 37151643706, 37151643706, 37151643706,
	}

	transactions, err := (&ExecutionTransactionDeriver{}).getExecutionTransactions(context.Background(), block, nil)
	require.NoError(t, err)
	require.Len(t, transactions, len(want))

	for i, tx := range transactions {
		got, err := GetGasPrice(block, tx, nil)
		require.NoError(t, err)
		assert.Equal(t, big.NewInt(want[i]), got, "transaction %d", i)
	}
}

func TestWithinBlobRetention(t *testing.T) {
	const current, window = 483421, 4096

	assert.True(t, withinBlobRetention(current, current, window))
	assert.True(t, withinBlobRetention(current-window, current, window))
	assert.False(t, withinBlobRetention(current-window-1, current, window))
	// Mainnet execution transaction backfill stalled here on 2026-03-19.
	assert.False(t, withinBlobRetention(431117, current, window))
}

// Mainnet slot 13795744 (Fulu): its blobs are long pruned, so the beacon node answers
// 400 "Insufficient data columns to reconstruct blobs" and the deriver used to stall.
func TestExecutionTransactionDeriver_PrunedBlobsFuluBlock(t *testing.T) {
	block := loadEraBlock(t, "mainnet-13795744", spec.DataVersionFulu)

	deriver := &ExecutionTransactionDeriver{
		log:        logrus.New(),
		clientMeta: &xatu.ClientMeta{Name: "cannon-test"},
	}

	const blobTx = 173

	t.Run("blob sidecars unavailable", func(t *testing.T) {
		events, err := deriver.deriveEvents(context.Background(), block, nil, nil, testBlockIdentifier(), big.NewInt(1))
		require.NoError(t, err)
		require.Len(t, events, 175)

		totalGasPrice := new(big.Int)

		for i, event := range events {
			tx := event.GetEthV2BeaconBlockExecutionTransaction()
			additional := event.GetMeta().GetClient().GetEthV2BeaconBlockExecutionTransaction()

			gasPrice, ok := new(big.Int).SetString(tx.GetGasPrice(), 10)
			require.True(t, ok)
			totalGasPrice.Add(totalGasPrice, gasPrice)

			if i == blobTx {
				assert.Equal(t, uint32(3), tx.GetType().GetValue())
				assert.Equal(t, []string{"0x013a686e30a01becc9ef89d91341f52643c0fa9b5b484524943c3a2e169adb1f"}, tx.GetBlobHashes())
				assert.Empty(t, additional.GetBlobSidecarsSize())
				assert.Empty(t, additional.GetBlobSidecarsEmptySize())

				continue
			}

			assert.Equal(t, "0", additional.GetBlobSidecarsSize())
			assert.Equal(t, "0", additional.GetBlobSidecarsEmptySize())
		}

		// Sum of the EL receipts' effectiveGasPrice for execution block 24562996.
		assert.Equal(t, "166805221010", totalGasPrice.String())

		first := events[0].GetEthV2BeaconBlockExecutionTransaction()
		assert.Equal(t, "0xd2ee541adcd995e0d639dc02591a27a2266b06a6e9c40268cf895e3f29fb36fd", first.GetHash())
		assert.Equal(t, "0x5B43453FCE04b92E190f391a83136bfBeCEDEFd1", first.GetFrom())
		assert.Equal(t, "0xfBd4cdB413E45a52E2C8312f670e9cE67E794C37", first.GetTo())
		assert.Equal(t, "1701095980", first.GetGasPrice())
		assert.Equal(t, uint64(736379), first.GetNonce().GetValue())
	})

	t.Run("blob sidecars available", func(t *testing.T) {
		commitments, err := block.BlobKZGCommitments()
		require.NoError(t, err)
		require.Len(t, commitments, 1)

		sidecar := &deneb.BlobSidecar{KZGCommitment: commitments[0]}
		hash := ethereum.ConvertKzgCommitmentToVersionedHash(commitments[0][:]).String()

		events, err := deriver.deriveEvents(context.Background(), block, nil, map[string]*deneb.BlobSidecar{hash: sidecar}, testBlockIdentifier(), big.NewInt(1))
		require.NoError(t, err)

		additional := events[blobTx].GetMeta().GetClient().GetEthV2BeaconBlockExecutionTransaction()
		assert.Equal(t, "131072", additional.GetBlobSidecarsSize())
		assert.Equal(t, "131072", additional.GetBlobSidecarsEmptySize())
	})

	t.Run("blob sidecar missing", func(t *testing.T) {
		events, err := deriver.deriveEvents(context.Background(), block, nil, map[string]*deneb.BlobSidecar{}, testBlockIdentifier(), big.NewInt(1))
		require.NoError(t, err)

		additional := events[blobTx].GetMeta().GetClient().GetEthV2BeaconBlockExecutionTransaction()
		assert.Empty(t, additional.GetBlobSidecarsSize())
		assert.Empty(t, additional.GetBlobSidecarsEmptySize())
	})
}
