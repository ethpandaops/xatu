package eth

import (
	"fmt"
	"testing"

	"github.com/attestantio/go-eth2-client/api"
	apiv1deneb "github.com/attestantio/go-eth2-client/api/v1/deneb"
	"github.com/attestantio/go-eth2-client/spec"
	"github.com/attestantio/go-eth2-client/spec/altair"
	"github.com/attestantio/go-eth2-client/spec/bellatrix"
	"github.com/attestantio/go-eth2-client/spec/capella"
	"github.com/attestantio/go-eth2-client/spec/deneb"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	v2 "github.com/ethpandaops/xatu/pkg/proto/eth/v2"
	"github.com/holiman/uint256"
)

func TestNewEventBlockV2FromVersionedProposal(t *testing.T) {
	tests := []struct {
		name            string
		proposal        *api.VersionedProposal
		expectedError   bool
		expectedVersion v2.BlockVersion
	}{
		{
			name: "Unsupported proposal version",
			proposal: &api.VersionedProposal{
				Version: spec.DataVersionUnknown,
			},
			expectedError: true,
		},
		{
			name: "phase0 proposal",
			proposal: &api.VersionedProposal{
				Version: spec.DataVersionPhase0,
				Phase0:  mockPhase0Block(),
			},
			expectedError:   false,
			expectedVersion: v2.BlockVersion_PHASE0,
		},
		{
			name: "altair proposal",
			proposal: &api.VersionedProposal{
				Version: spec.DataVersionAltair,
				Altair:  mockAltairBlock(),
			},
			expectedError:   false,
			expectedVersion: v2.BlockVersion_ALTAIR,
		},
		{
			name: "bellatrix proposal",
			proposal: &api.VersionedProposal{
				Version:   spec.DataVersionBellatrix,
				Bellatrix: mockBellatrixBlock(),
			},
			expectedError:   false,
			expectedVersion: v2.BlockVersion_BELLATRIX,
		},
		{
			name: "capella proposal",
			proposal: &api.VersionedProposal{
				Version: spec.DataVersionCapella,
				Capella: mockCapellaBlock(),
			},
			expectedError:   false,
			expectedVersion: v2.BlockVersion_CAPELLA,
		},
		{
			name: "deneb proposal",
			proposal: &api.VersionedProposal{
				Version: spec.DataVersionDeneb,
				Deneb: &apiv1deneb.BlockContents{
					Block: mockDenebBlock(),
				},
			},
			expectedError:   false,
			expectedVersion: v2.BlockVersion_DENEB,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := NewEventBlockV2FromVersionedProposal(tt.proposal)

			if (err != nil) != tt.expectedError {
				t.Fatalf("expected error: %v, got: %v", tt.expectedError, err)
			}

			if result != nil && result.Version != tt.expectedVersion {
				t.Errorf("expected version %v, got %v", tt.expectedVersion, result.Version)
			}
		})
	}
}

func TestSignatureHandling(t *testing.T) {
	block := mockPhase0Block()
	signature := phase0.BLSSignature{0x01} // Any non-zero sig will do for test.
	resultWithSig := NewEventBlockFromPhase0(block, &signature)

	if resultWithSig.Signature == "" {
		t.Error("expected signature to be set")
	}

	resultNoSig := NewEventBlockFromPhase0(block, nil)

	if resultNoSig.Signature != "" {
		t.Error("expected no signature")
	}
}

func TestExecutionPayloadFormatting(t *testing.T) {
	tests := []struct {
		name          string
		expectedField string
		getField      func(event *v2.EventBlockV2) string
		mockValue     [32]byte
	}{
		{
			name:          "ParentHash",
			expectedField: "ParentHash",
			getField: func(event *v2.EventBlockV2) string {
				payload, ok := event.Message.(*v2.EventBlockV2_BellatrixBlock)
				if !ok {
					t.Fatalf("expected *v2.EventBlockV2_BellatrixBlock, got %T", event.Message)
				}

				return payload.BellatrixBlock.Body.ExecutionPayload.ParentHash
			},
			mockValue: [32]byte{0xde, 0xad, 0xbe, 0xef}, // no functional significance, simply to test formatting.
		},
		{
			name:          "StateRoot",
			expectedField: "StateRoot",
			getField: func(event *v2.EventBlockV2) string {
				payload, ok := event.Message.(*v2.EventBlockV2_BellatrixBlock)
				if !ok {
					t.Fatalf("expected *v2.EventBlockV2_BellatrixBlock, got %T", event.Message)
				}

				return payload.BellatrixBlock.Body.ExecutionPayload.StateRoot
			},
			mockValue: [32]byte{0xca, 0xfe, 0xba, 0xbe}, // no functional significance, simply to test formatting.
		},
		{
			name:          "ReceiptsRoot",
			expectedField: "ReceiptsRoot",
			getField: func(event *v2.EventBlockV2) string {
				payload, ok := event.Message.(*v2.EventBlockV2_BellatrixBlock)
				if !ok {
					t.Fatalf("expected *v2.EventBlockV2_BellatrixBlock, got %T", event.Message)
				}

				return payload.BellatrixBlock.Body.ExecutionPayload.ReceiptsRoot
			},
			mockValue: [32]byte{0xba, 0xad, 0xf0, 0x0d}, // no functional significance, simply to test formatting.
		},
		{
			name:          "PrevRandao",
			expectedField: "PrevRandao",
			getField: func(event *v2.EventBlockV2) string {
				payload, ok := event.Message.(*v2.EventBlockV2_BellatrixBlock)
				if !ok {
					t.Fatalf("expected *v2.EventBlockV2_BellatrixBlock, got %T", event.Message)
				}

				return payload.BellatrixBlock.Body.ExecutionPayload.PrevRandao
			},
			mockValue: [32]byte{0x12, 0x34, 0x56, 0x78}, // no functional significance, simply to test formatting.
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			block := mockBellatrixBlock()
			block.Body.ExecutionPayload = &bellatrix.ExecutionPayload{
				ParentHash:   tt.mockValue,
				StateRoot:    tt.mockValue,
				ReceiptsRoot: tt.mockValue,
				PrevRandao:   tt.mockValue,
			}

			result := NewEventBlockFromBellatrix(block, nil)
			expectedHex := fmt.Sprintf("0x%x", tt.mockValue[:])

			if field := tt.getField(result); field != expectedHex {
				t.Errorf("Expected %s to be %v, got %v", tt.expectedField, expectedHex, field)
			}
		})
	}
}

func mockPhase0Block() *phase0.BeaconBlock {
	return &phase0.BeaconBlock{
		Slot:          phase0.Slot(1),
		ProposerIndex: phase0.ValidatorIndex(1),
		ParentRoot:    [32]byte{},
		StateRoot:     [32]byte{},
		Body: &phase0.BeaconBlockBody{
			RANDAOReveal: [96]byte{},
			ETH1Data: &phase0.ETH1Data{
				DepositRoot:  [32]byte{},
				DepositCount: 1,
				BlockHash:    []byte{},
			},
			Graffiti:          [32]byte{},
			ProposerSlashings: []*phase0.ProposerSlashing{},
			AttesterSlashings: []*phase0.AttesterSlashing{},
			Attestations:      []*phase0.Attestation{},
			Deposits:          []*phase0.Deposit{},
			VoluntaryExits:    []*phase0.SignedVoluntaryExit{},
		},
	}
}

func mockAltairBlock() *altair.BeaconBlock {
	return &altair.BeaconBlock{
		Slot:          phase0.Slot(1),
		ProposerIndex: phase0.ValidatorIndex(1),
		ParentRoot:    [32]byte{},
		StateRoot:     [32]byte{},
		Body: &altair.BeaconBlockBody{
			RANDAOReveal: [96]byte{},
			ETH1Data: &phase0.ETH1Data{
				DepositRoot:  [32]byte{},
				DepositCount: 1,
				BlockHash:    []byte{},
			},
			Graffiti:          [32]byte{},
			ProposerSlashings: []*phase0.ProposerSlashing{},
			AttesterSlashings: []*phase0.AttesterSlashing{},
			Attestations:      []*phase0.Attestation{},
			Deposits:          []*phase0.Deposit{},
			VoluntaryExits:    []*phase0.SignedVoluntaryExit{},
			SyncAggregate: &altair.SyncAggregate{
				SyncCommitteeBits: []byte{},
			},
		},
	}
}

func mockBellatrixBlock() *bellatrix.BeaconBlock {
	return &bellatrix.BeaconBlock{
		Slot:          phase0.Slot(1),
		ProposerIndex: phase0.ValidatorIndex(1),
		ParentRoot:    [32]byte{},
		StateRoot:     [32]byte{},
		Body: &bellatrix.BeaconBlockBody{
			RANDAOReveal: [96]byte{},
			ETH1Data: &phase0.ETH1Data{
				DepositRoot:  [32]byte{},
				DepositCount: 1,
				BlockHash:    []byte{},
			},
			Graffiti:          [32]byte{},
			ProposerSlashings: []*phase0.ProposerSlashing{},
			AttesterSlashings: []*phase0.AttesterSlashing{},
			Attestations:      []*phase0.Attestation{},
			Deposits:          []*phase0.Deposit{},
			VoluntaryExits:    []*phase0.SignedVoluntaryExit{},
			SyncAggregate: &altair.SyncAggregate{
				SyncCommitteeBits: []byte{},
			},
			ExecutionPayload: &bellatrix.ExecutionPayload{},
		},
	}
}

func mockCapellaBlock() *capella.BeaconBlock {
	return &capella.BeaconBlock{
		Slot:          phase0.Slot(1),
		ProposerIndex: phase0.ValidatorIndex(1),
		ParentRoot:    [32]byte{},
		StateRoot:     [32]byte{},
		Body: &capella.BeaconBlockBody{
			RANDAOReveal: [96]byte{},
			ETH1Data: &phase0.ETH1Data{
				DepositRoot:  [32]byte{},
				DepositCount: 1,
				BlockHash:    []byte{},
			},
			Graffiti:          [32]byte{},
			ProposerSlashings: []*phase0.ProposerSlashing{},
			AttesterSlashings: []*phase0.AttesterSlashing{},
			Attestations:      []*phase0.Attestation{},
			Deposits:          []*phase0.Deposit{},
			VoluntaryExits:    []*phase0.SignedVoluntaryExit{},
			SyncAggregate: &altair.SyncAggregate{
				SyncCommitteeBits: []byte{},
			},
			ExecutionPayload:      &capella.ExecutionPayload{},
			BLSToExecutionChanges: []*capella.SignedBLSToExecutionChange{},
		},
	}
}

func mockDenebBlock() *deneb.BeaconBlock {
	return &deneb.BeaconBlock{
		Slot:          phase0.Slot(1),
		ProposerIndex: phase0.ValidatorIndex(1),
		ParentRoot:    [32]byte{},
		StateRoot:     [32]byte{},
		Body: &deneb.BeaconBlockBody{
			RANDAOReveal: [96]byte{},
			ETH1Data: &phase0.ETH1Data{
				DepositRoot:  [32]byte{},
				DepositCount: 1,
				BlockHash:    []byte{},
			},
			Graffiti:          [32]byte{},
			ProposerSlashings: []*phase0.ProposerSlashing{},
			AttesterSlashings: []*phase0.AttesterSlashing{},
			Attestations:      []*phase0.Attestation{},
			Deposits:          []*phase0.Deposit{},
			VoluntaryExits:    []*phase0.SignedVoluntaryExit{},
			SyncAggregate: &altair.SyncAggregate{
				SyncCommitteeBits: []byte{},
			},
			ExecutionPayload: &deneb.ExecutionPayload{
				BaseFeePerGas: uint256.NewInt(1),
			},
			BLSToExecutionChanges: []*capella.SignedBLSToExecutionChange{},
			BlobKZGCommitments:    []deneb.KZGCommitment{},
		},
	}
}
