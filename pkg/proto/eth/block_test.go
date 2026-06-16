package eth

import (
	"fmt"
	"testing"

	"github.com/OffchainLabs/go-bitfield"
	"github.com/ethpandaops/go-eth2-client/api"
	apiv1deneb "github.com/ethpandaops/go-eth2-client/api/v1/deneb"
	apiv1electra "github.com/ethpandaops/go-eth2-client/api/v1/electra"
	apiv1fulu "github.com/ethpandaops/go-eth2-client/api/v1/fulu"
	"github.com/ethpandaops/go-eth2-client/spec"
	"github.com/ethpandaops/go-eth2-client/spec/altair"
	"github.com/ethpandaops/go-eth2-client/spec/bellatrix"
	"github.com/ethpandaops/go-eth2-client/spec/capella"
	"github.com/ethpandaops/go-eth2-client/spec/deneb"
	"github.com/ethpandaops/go-eth2-client/spec/electra"
	"github.com/ethpandaops/go-eth2-client/spec/gloas"
	"github.com/ethpandaops/go-eth2-client/spec/phase0"
	"github.com/holiman/uint256"

	v2 "github.com/ethpandaops/xatu/pkg/proto/eth/v2"
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
		{
			name: "electra proposal",
			proposal: &api.VersionedProposal{
				Version: spec.DataVersionElectra,
				Electra: &apiv1electra.BlockContents{
					Block: mockElectraBlock(),
				},
			},
			expectedError:   false,
			expectedVersion: v2.BlockVersion_ELECTRA,
		},
		{
			name: "fulu proposal",
			proposal: &api.VersionedProposal{
				Version: spec.DataVersionFulu,
				Fulu: &apiv1fulu.BlockContents{
					Block: mockFuluBlock(),
				},
			},
			expectedError:   false,
			expectedVersion: v2.BlockVersion_FULU,
		},
		{
			name: "gloas proposal",
			proposal: &api.VersionedProposal{
				Version: spec.DataVersionGloas,
				Gloas:   mockGloasBlock(),
			},
			expectedError:   false,
			expectedVersion: v2.BlockVersion_GLOAS,
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

func mockElectraBlock() *electra.BeaconBlock {
	return &electra.BeaconBlock{
		Slot:          phase0.Slot(1),
		ProposerIndex: phase0.ValidatorIndex(1),
		ParentRoot:    [32]byte{},
		StateRoot:     [32]byte{},
		Body: &electra.BeaconBlockBody{
			RANDAOReveal: [96]byte{},
			ETH1Data: &phase0.ETH1Data{
				DepositRoot:  [32]byte{},
				DepositCount: 1,
				BlockHash:    []byte{},
			},
			Graffiti:          [32]byte{},
			ProposerSlashings: []*phase0.ProposerSlashing{},
			AttesterSlashings: []*electra.AttesterSlashing{},
			Attestations:      []*electra.Attestation{},
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
			ExecutionRequests: &electra.ExecutionRequests{
				Deposits:       []*electra.DepositRequest{},
				Withdrawals:    []*electra.WithdrawalRequest{},
				Consolidations: []*electra.ConsolidationRequest{},
			},
		},
	}
}

func mockFuluBlock() *electra.BeaconBlock {
	return &electra.BeaconBlock{
		Slot:          phase0.Slot(1),
		ProposerIndex: phase0.ValidatorIndex(1),
		ParentRoot:    [32]byte{},
		StateRoot:     [32]byte{},
		Body: &electra.BeaconBlockBody{
			RANDAOReveal: [96]byte{},
			ETH1Data: &phase0.ETH1Data{
				DepositRoot:  [32]byte{},
				DepositCount: 1,
				BlockHash:    []byte{},
			},
			Graffiti:          [32]byte{},
			ProposerSlashings: []*phase0.ProposerSlashing{},
			AttesterSlashings: []*electra.AttesterSlashing{},
			Attestations:      []*electra.Attestation{},
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
			ExecutionRequests: &electra.ExecutionRequests{
				Deposits:       []*electra.DepositRequest{},
				Withdrawals:    []*electra.WithdrawalRequest{},
				Consolidations: []*electra.ConsolidationRequest{},
			},
		},
	}
}

func mockGloasBlock() *gloas.BeaconBlock {
	return &gloas.BeaconBlock{
		Slot:          phase0.Slot(1),
		ProposerIndex: phase0.ValidatorIndex(1),
		ParentRoot:    [32]byte{},
		StateRoot:     [32]byte{},
		Body: &gloas.BeaconBlockBody{
			RANDAOReveal: [96]byte{},
			ETH1Data: &phase0.ETH1Data{
				DepositRoot:  [32]byte{},
				DepositCount: 1,
				BlockHash:    []byte{},
			},
			Graffiti:          [32]byte{},
			ProposerSlashings: []*phase0.ProposerSlashing{},
			AttesterSlashings: []*electra.AttesterSlashing{},
			Attestations:      []*electra.Attestation{},
			Deposits:          []*phase0.Deposit{},
			VoluntaryExits:    []*phase0.SignedVoluntaryExit{},
			SyncAggregate: &altair.SyncAggregate{
				SyncCommitteeBits: []byte{},
			},
			BLSToExecutionChanges: []*capella.SignedBLSToExecutionChange{},
			SignedExecutionPayloadBid: &gloas.SignedExecutionPayloadBid{
				Message: &gloas.ExecutionPayloadBid{
					BlobKZGCommitments: []deneb.KZGCommitment{},
				},
			},
			PayloadAttestations: []*gloas.PayloadAttestation{},
			ParentExecutionRequests: &electra.ExecutionRequests{
				Deposits:       []*electra.DepositRequest{},
				Withdrawals:    []*electra.WithdrawalRequest{},
				Consolidations: []*electra.ConsolidationRequest{},
			},
		},
	}
}

// TestNewEventBlockFromGloas verifies the EIP-7732 (ePBS) conversion path:
// SignedExecutionPayloadBid populates field 14 with the builder bid, and
// PayloadAttestations populates field 15 with PTC aggregations. Pre-ePBS
// proto fields (10/12/13 — execution_payload, blob_kzg_commitments,
// execution_requests) are absent from BeaconBlockBodyGloas — that's enforced
// at compile time by their `reserved` declaration in the proto.
func TestNewEventBlockFromGloas(t *testing.T) {
	bidBlockHash := phase0.Hash32{0xaa, 0xbb}
	feeRecipient := bellatrix.ExecutionAddress{0xfe, 0xfe}
	commitment := deneb.KZGCommitment{0xc0, 0xc1}

	aggBits := bitfield.NewBitvector512()
	aggBits.SetBitAt(7, true)

	block := &gloas.BeaconBlock{
		Slot:          phase0.Slot(42),
		ProposerIndex: phase0.ValidatorIndex(7),
		ParentRoot:    phase0.Root{0x01, 0x02},
		StateRoot:     phase0.Root{0x03, 0x04},
		Body: &gloas.BeaconBlockBody{
			RANDAOReveal: phase0.BLSSignature{},
			ETH1Data: &phase0.ETH1Data{
				DepositRoot:  phase0.Root{},
				DepositCount: 1,
				BlockHash:    []byte{},
			},
			Graffiti:          [32]byte{},
			ProposerSlashings: []*phase0.ProposerSlashing{},
			AttesterSlashings: []*electra.AttesterSlashing{},
			Attestations:      []*electra.Attestation{},
			Deposits:          []*phase0.Deposit{},
			VoluntaryExits:    []*phase0.SignedVoluntaryExit{},
			SyncAggregate: &altair.SyncAggregate{
				SyncCommitteeBits: []byte{},
			},
			BLSToExecutionChanges: []*capella.SignedBLSToExecutionChange{},
			SignedExecutionPayloadBid: &gloas.SignedExecutionPayloadBid{
				Message: &gloas.ExecutionPayloadBid{
					BlockHash:          bidBlockHash,
					FeeRecipient:       feeRecipient,
					GasLimit:           30_000_000,
					BuilderIndex:       gloas.BuilderIndex(99),
					Slot:               phase0.Slot(42),
					Value:              phase0.Gwei(1234),
					ExecutionPayment:   phase0.Gwei(56),
					BlobKZGCommitments: []deneb.KZGCommitment{commitment},
				},
			},
			PayloadAttestations: []*gloas.PayloadAttestation{
				{
					AggregationBits: aggBits,
					Data: &gloas.PayloadAttestationData{
						BeaconBlockRoot:   phase0.Root{0x05},
						Slot:              phase0.Slot(42),
						PayloadPresent:    true,
						BlobDataAvailable: true,
					},
					Signature: phase0.BLSSignature{},
				},
			},
			ParentExecutionRequests: &electra.ExecutionRequests{},
		},
	}

	result := NewEventBlockFromGloas(block, nil)

	if result == nil {
		t.Fatal("expected non-nil result")
	}

	if result.GetVersion() != v2.BlockVersion_GLOAS {
		t.Errorf("expected GLOAS version, got %v", result.GetVersion())
	}

	gloasMsg, ok := result.GetMessage().(*v2.EventBlockV2_GloasBlock)
	if !ok {
		t.Fatalf("expected *v2.EventBlockV2_GloasBlock, got %T", result.GetMessage())
	}

	body := gloasMsg.GloasBlock.GetBody()
	if body == nil {
		t.Fatal("expected non-nil body")
	}

	bid := body.GetSignedExecutionPayloadBid()
	if bid == nil || bid.GetMessage() == nil {
		t.Fatal("expected non-nil signed_execution_payload_bid (field 14)")
	}

	if got := bid.GetMessage().GetBuilderIndex().GetValue(); got != 99 {
		t.Errorf("bid.builder_index = %d, want 99", got)
	}

	if got := bid.GetMessage().GetValue().GetValue(); got != 1234 {
		t.Errorf("bid.value = %d, want 1234", got)
	}

	if got := len(bid.GetMessage().GetBlobKzgCommitments()); got != 1 {
		t.Errorf("bid.blob_kzg_commitments len = %d, want 1", got)
	}

	atts := body.GetPayloadAttestations()
	if len(atts) != 1 {
		t.Fatalf("expected 1 payload_attestation (field 15), got %d", len(atts))
	}

	if !atts[0].GetData().GetPayloadPresent() {
		t.Error("expected payload_attestation data.payload_present=true")
	}

	if got := atts[0].GetData().GetSlot().GetValue(); got != 42 {
		t.Errorf("payload_attestation data.slot = %d, want 42", got)
	}

	// parent_execution_requests is the parent block's deferred-payload
	// requests, processed in this block's state transition. It must
	// round-trip through the conversion even when empty.
	if body.GetParentExecutionRequests() == nil {
		t.Error("expected non-nil parent_execution_requests")
	}
}
