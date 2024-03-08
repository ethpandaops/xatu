package ethereum

import (
	"crypto/sha256"

	"github.com/ethereum/go-ethereum/common"
)

const blobCommitmentVersionKZG uint8 = 0x01

// Reference: https://github.com/prysmaticlabs/prysm/blob/bfae7f3c9fa30cf0d513b59ad95cc99a5316eacd/beacon-chain/blockchain/execution_engine.go#L413
func ConvertKzgCommitmentToVersionedHash(commitment []byte) common.Hash {
	versionedHash := sha256.Sum256(commitment)

	versionedHash[0] = blobCommitmentVersionKZG

	return versionedHash
}

func CountConsecutiveEmptyBytes(byteArray []byte, threshold int) int {
	count := 0
	consecutiveZeros := 0

	for _, b := range byteArray {
		if b == 0 {
			consecutiveZeros++
		} else {
			if consecutiveZeros > threshold {
				count += consecutiveZeros
			}

			consecutiveZeros = 0
		}
	}

	// Check if the last sequence in the array is longer than the threshold and hasn't been counted yet
	if consecutiveZeros > threshold {
		count += consecutiveZeros
	}

	return count
}
