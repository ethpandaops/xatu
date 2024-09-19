package ethereum

import (
	"crypto/sha256"

	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/pkg/errors"
)

func ComputeForkDigest(genesisValidatorsRoot phase0.Root, forkVersion [4]byte) (phase0.ForkDigest, error) {
	if len(genesisValidatorsRoot) != 32 {
		return phase0.ForkDigest{}, errors.New("invalid genesis validators root")
	}

	data := append(genesisValidatorsRoot[:], forkVersion[:]...)

	digest := sha256.Sum256(data)

	return phase0.ForkDigest(digest[:]), nil
}
