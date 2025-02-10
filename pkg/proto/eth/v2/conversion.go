package v2

import (
	"github.com/attestantio/go-eth2-client/spec/capella"
)

func NewBLSToExecutionChangesFromCapella(data []*capella.SignedBLSToExecutionChange) []*SignedBLSToExecutionChange {
	changes := []*SignedBLSToExecutionChange{}

	if data == nil {
		return changes
	}

	for _, change := range data {
		changes = append(changes, &SignedBLSToExecutionChange{
			Message: &BLSToExecutionChange{
				ValidatorIndex:     uint64(change.Message.ValidatorIndex),
				FromBlsPubkey:      change.Message.FromBLSPubkey.String(),
				ToExecutionAddress: change.Message.ToExecutionAddress.String(),
			},
			Signature: change.Signature.String(),
		})
	}

	return changes
}
