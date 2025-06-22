package mocks

import (
	"github.com/attestantio/go-eth2-client/spec"
	"github.com/attestantio/go-eth2-client/spec/altair"
	"github.com/attestantio/go-eth2-client/spec/capella"
	"github.com/attestantio/go-eth2-client/spec/phase0"
)

// GenerateTestBeaconBlock creates a test beacon block for the specified slot
func GenerateTestBeaconBlock(slot uint64, parentRoot []byte) *spec.VersionedSignedBeaconBlock {
	if parentRoot == nil {
		parentRoot = make([]byte, 32)
		for i := range parentRoot {
			parentRoot[i] = byte(i)
		}
	}

	return &spec.VersionedSignedBeaconBlock{
		Version: spec.DataVersionCapella,
		Capella: &capella.SignedBeaconBlock{
			Message: &capella.BeaconBlock{
				Slot:       phase0.Slot(slot),
				ParentRoot: phase0.Root(parentRoot),
				StateRoot:  phase0.Root(make([]byte, 32)),
				Body: &capella.BeaconBlockBody{
					RANDAOReveal:      phase0.BLSSignature{},
					ETH1Data:          &phase0.ETH1Data{},
					Graffiti:          [32]byte{},
					ProposerSlashings: []*phase0.ProposerSlashing{},
					AttesterSlashings: []*phase0.AttesterSlashing{},
					Attestations:      []*phase0.Attestation{},
					Deposits:          []*phase0.Deposit{},
					VoluntaryExits:    []*phase0.SignedVoluntaryExit{},
					SyncAggregate:     &altair.SyncAggregate{},
					ExecutionPayload:  &capella.ExecutionPayload{},
				},
			},
			Signature: phase0.BLSSignature{},
		},
	}
}

// TODO: Implement GenerateTestDecoratedEvent when we need it
// func GenerateTestDecoratedEvent(eventType string) *xatu.DecoratedEvent {
//     // Implementation pending proper proto field mapping
// }

// GenerateTestBlockClassification creates a test block classification
func GenerateTestBlockClassification(slot uint64, blockprint, blockHash string, blockNumber uint64) map[uint64]*BlockClassification {
	return map[uint64]*BlockClassification{
		slot: {
			Slot:        slot,
			Blockprint:  blockprint,
			BlockHash:   blockHash,
			BlockNumber: blockNumber,
		},
	}
}

// TestConstants provides common test values
var TestConstants = struct {
	NetworkID   string
	NetworkName string
	TestSlot    uint64
	TestEpoch   uint64
	ParentRoot  []byte
}{
	NetworkID:   "1",
	NetworkName: "mainnet",
	TestSlot:    12345,
	TestEpoch:   386,
	ParentRoot:  []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32},
}
