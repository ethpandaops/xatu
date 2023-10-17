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
