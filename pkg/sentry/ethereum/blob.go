package ethereum

import (
	"crypto/sha256"
	"fmt"

	eth2v1 "github.com/attestantio/go-eth2-client/api/v1"
	"github.com/attestantio/go-eth2-client/spec/deneb"
)

const blobCommitmentVersionKZG uint8 = 0x01

// ConvertKzgCommitmentToVersionedHash computes a versioned hash from a KZG commitment.
// Reference: https://github.com/prysmaticlabs/prysm/blob/bfae7f3c9fa30cf0d513b59ad95cc99a5316eacd/beacon-chain/blockchain/execution_engine.go#L413
func ConvertKzgCommitmentToVersionedHash(commitment []byte) deneb.VersionedHash {
	versionedHash := sha256.Sum256(commitment)

	versionedHash[0] = blobCommitmentVersionKZG

	return deneb.VersionedHash(versionedHash)
}

// BlobSidecarToBlobSidecarEvent converts a fetched BlobSidecar to a BlobSidecarEvent format
// that can be used with the existing event handling infrastructure.
func BlobSidecarToBlobSidecarEvent(blob *deneb.BlobSidecar) (*eth2v1.BlobSidecarEvent, error) {
	if blob == nil {
		return nil, fmt.Errorf("blob sidecar is nil")
	}

	if blob.SignedBlockHeader == nil || blob.SignedBlockHeader.Message == nil {
		return nil, fmt.Errorf("blob sidecar has nil signed block header")
	}

	blockRoot, err := blob.SignedBlockHeader.Message.HashTreeRoot()
	if err != nil {
		return nil, fmt.Errorf("failed to compute block root: %w", err)
	}

	versionedHash := ConvertKzgCommitmentToVersionedHash(blob.KZGCommitment[:])

	return &eth2v1.BlobSidecarEvent{
		BlockRoot:     blockRoot,
		Slot:          blob.SignedBlockHeader.Message.Slot,
		Index:         blob.Index,
		KZGCommitment: blob.KZGCommitment,
		VersionedHash: versionedHash,
	}, nil
}
