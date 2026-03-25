package v1

import (
	"encoding/hex"
	"fmt"

	"github.com/attestantio/go-eth2-client/spec/capella"
	"github.com/attestantio/go-eth2-client/spec/deneb"
	"github.com/attestantio/go-eth2-client/spec/electra"
	"github.com/attestantio/go-eth2-client/spec/gloas"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/ethereum/go-ethereum/core/types/bal"
	"github.com/ethereum/go-ethereum/rlp"
	wrapperspb "google.golang.org/protobuf/types/known/wrapperspb"
)

func RootAsString(root phase0.Root) string {
	return fmt.Sprintf("%#x", root)
}

func SlotAsString(slot phase0.Slot) string {
	return fmt.Sprintf("%d", slot)
}

func EpochAsString(epoch phase0.Epoch) string {
	return fmt.Sprintf("%d", epoch)
}

func BLSSignatureToString(s *phase0.BLSSignature) string {
	return fmt.Sprintf("%#x", s)
}

func KzgCommitmentToString(c deneb.KZGCommitment) string {
	return fmt.Sprintf("%#x", c)
}

func VersionedHashToString(h deneb.VersionedHash) string {
	return fmt.Sprintf("%#x", h)
}

func BytesToString(b []byte) string {
	return fmt.Sprintf("%#x", b)
}

func StringToRoot(s string) (phase0.Root, error) {
	var root phase0.Root
	if len(s) != 66 {
		return root, fmt.Errorf("invalid root length")
	}

	if s[:2] != "0x" {
		return root, fmt.Errorf("invalid root prefix")
	}

	_, err := hex.Decode(root[:], []byte(s[2:]))
	if err != nil {
		return root, fmt.Errorf("invalid root: %v", err)
	}

	return root, nil
}

func TrimmedString(s string) string {
	if len(s) <= 12 {
		return s
	}

	return s[:5] + "..." + s[len(s)-5:]
}

func NewProposerSlashingsFromPhase0(data []*phase0.ProposerSlashing) []*ProposerSlashing {
	slashings := []*ProposerSlashing{}

	if data != nil {
		return slashings
	}

	for _, slashing := range data {
		slashings = append(slashings, &ProposerSlashing{
			SignedHeader_1: &SignedBeaconBlockHeader{
				Message: &BeaconBlockHeader{
					Slot:          uint64(slashing.SignedHeader1.Message.Slot),
					ProposerIndex: uint64(slashing.SignedHeader1.Message.ProposerIndex),
					ParentRoot:    slashing.SignedHeader1.Message.ParentRoot.String(),
					StateRoot:     slashing.SignedHeader1.Message.StateRoot.String(),
					BodyRoot:      slashing.SignedHeader1.Message.BodyRoot.String(),
				},
				Signature: slashing.SignedHeader1.Signature.String(),
			},
			SignedHeader_2: &SignedBeaconBlockHeader{
				Message: &BeaconBlockHeader{
					Slot:          uint64(slashing.SignedHeader2.Message.Slot),
					ProposerIndex: uint64(slashing.SignedHeader2.Message.ProposerIndex),
					ParentRoot:    slashing.SignedHeader2.Message.ParentRoot.String(),
					StateRoot:     slashing.SignedHeader2.Message.StateRoot.String(),
					BodyRoot:      slashing.SignedHeader2.Message.BodyRoot.String(),
				},
				Signature: slashing.SignedHeader2.Signature.String(),
			},
		})
	}

	return slashings
}

func NewAttesterSlashingsFromPhase0(data []*phase0.AttesterSlashing) []*AttesterSlashing {
	slashings := []*AttesterSlashing{}

	if data == nil {
		return slashings
	}

	for _, slashing := range data {
		slashings = append(slashings, &AttesterSlashing{
			Attestation_1: &IndexedAttestation{
				AttestingIndices: slashing.Attestation1.AttestingIndices,
				Data: &AttestationData{
					Slot:            uint64(slashing.Attestation1.Data.Slot),
					Index:           uint64(slashing.Attestation1.Data.Index),
					BeaconBlockRoot: slashing.Attestation1.Data.BeaconBlockRoot.String(),
					Source: &Checkpoint{
						Epoch: uint64(slashing.Attestation1.Data.Source.Epoch),
						Root:  slashing.Attestation1.Data.Source.Root.String(),
					},
					Target: &Checkpoint{
						Epoch: uint64(slashing.Attestation1.Data.Target.Epoch),
						Root:  slashing.Attestation1.Data.Target.Root.String(),
					},
				},
				Signature: slashing.Attestation1.Signature.String(),
			},
			Attestation_2: &IndexedAttestation{
				AttestingIndices: slashing.Attestation2.AttestingIndices,
				Data: &AttestationData{
					Slot:            uint64(slashing.Attestation2.Data.Slot),
					Index:           uint64(slashing.Attestation2.Data.Index),
					BeaconBlockRoot: slashing.Attestation2.Data.BeaconBlockRoot.String(),
					Source: &Checkpoint{
						Epoch: uint64(slashing.Attestation2.Data.Source.Epoch),
						Root:  slashing.Attestation2.Data.Source.Root.String(),
					},
					Target: &Checkpoint{
						Epoch: uint64(slashing.Attestation2.Data.Target.Epoch),
						Root:  slashing.Attestation2.Data.Target.Root.String(),
					},
				},
				Signature: slashing.Attestation2.Signature.String(),
			},
		})
	}

	return slashings
}

func NewAttesterSlashingsFromElectra(data []*electra.AttesterSlashing) []*AttesterSlashing {
	slashings := []*AttesterSlashing{}

	if data == nil {
		return slashings
	}

	for _, slashing := range data {
		slashings = append(slashings, &AttesterSlashing{
			Attestation_1: &IndexedAttestation{
				AttestingIndices: slashing.Attestation1.AttestingIndices,
				Data: &AttestationData{
					Slot:            uint64(slashing.Attestation1.Data.Slot),
					Index:           uint64(slashing.Attestation1.Data.Index),
					BeaconBlockRoot: slashing.Attestation1.Data.BeaconBlockRoot.String(),
					Source: &Checkpoint{
						Epoch: uint64(slashing.Attestation1.Data.Source.Epoch),
						Root:  slashing.Attestation1.Data.Source.Root.String(),
					},
					Target: &Checkpoint{
						Epoch: uint64(slashing.Attestation1.Data.Target.Epoch),
						Root:  slashing.Attestation1.Data.Target.Root.String(),
					},
				},
				Signature: slashing.Attestation1.Signature.String(),
			},
			Attestation_2: &IndexedAttestation{
				AttestingIndices: slashing.Attestation2.AttestingIndices,
				Data: &AttestationData{
					Slot:            uint64(slashing.Attestation2.Data.Slot),
					Index:           uint64(slashing.Attestation2.Data.Index),
					BeaconBlockRoot: slashing.Attestation2.Data.BeaconBlockRoot.String(),
					Source: &Checkpoint{
						Epoch: uint64(slashing.Attestation2.Data.Source.Epoch),
						Root:  slashing.Attestation2.Data.Source.Root.String(),
					},
					Target: &Checkpoint{
						Epoch: uint64(slashing.Attestation2.Data.Target.Epoch),
						Root:  slashing.Attestation2.Data.Target.Root.String(),
					},
				},
				Signature: slashing.Attestation2.Signature.String(),
			},
		})
	}

	return slashings
}

func NewAttestationsFromPhase0(data []*phase0.Attestation) []*Attestation {
	attestations := []*Attestation{}

	if data == nil {
		return attestations
	}

	for _, attestation := range data {
		attestations = append(attestations, &Attestation{
			AggregationBits: fmt.Sprintf("0x%x", attestation.AggregationBits),
			Data: &AttestationData{
				Slot:            uint64(attestation.Data.Slot),
				Index:           uint64(attestation.Data.Index),
				BeaconBlockRoot: attestation.Data.BeaconBlockRoot.String(),
				Source: &Checkpoint{
					Epoch: uint64(attestation.Data.Source.Epoch),
					Root:  attestation.Data.Source.Root.String(),
				},
				Target: &Checkpoint{
					Epoch: uint64(attestation.Data.Target.Epoch),
					Root:  attestation.Data.Target.Root.String(),
				},
			},
			Signature: attestation.Signature.String(),
		})
	}

	return attestations
}

func NewAttestationsFromElectra(data []*electra.Attestation) []*Attestation {
	attestations := []*Attestation{}

	if data == nil {
		return attestations
	}

	for _, attestation := range data {
		attestations = append(attestations, &Attestation{
			AggregationBits: fmt.Sprintf("0x%x", attestation.AggregationBits),
			Data: &AttestationData{
				Slot:            uint64(attestation.Data.Slot),
				Index:           uint64(attestation.Data.Index),
				BeaconBlockRoot: attestation.Data.BeaconBlockRoot.String(),
				Source: &Checkpoint{
					Epoch: uint64(attestation.Data.Source.Epoch),
					Root:  attestation.Data.Source.Root.String(),
				},
				Target: &Checkpoint{
					Epoch: uint64(attestation.Data.Target.Epoch),
					Root:  attestation.Data.Target.Root.String(),
				},
			},
			Signature: attestation.Signature.String(),
		})
	}

	return attestations
}

func NewDepositsFromPhase0(data []*phase0.Deposit) []*Deposit {
	deposits := []*Deposit{}

	if data == nil {
		return deposits
	}

	for _, deposit := range data {
		proof := []string{}
		for _, p := range deposit.Proof {
			proof = append(proof, fmt.Sprintf("0x%x", p))
		}

		deposits = append(deposits, &Deposit{
			Proof: proof,
			Data: &Deposit_Data{
				Pubkey:                deposit.Data.PublicKey.String(),
				WithdrawalCredentials: fmt.Sprintf("0x%x", deposit.Data.WithdrawalCredentials),
				Amount:                uint64(deposit.Data.Amount),
				Signature:             deposit.Data.Signature.String(),
			},
		})
	}

	return deposits
}

func NewSignedVoluntaryExitsFromPhase0(data []*phase0.SignedVoluntaryExit) []*SignedVoluntaryExit {
	exits := []*SignedVoluntaryExit{}

	if data == nil {
		return exits
	}

	for _, exit := range data {
		exits = append(exits, &SignedVoluntaryExit{
			Message: &VoluntaryExit{
				Epoch:          uint64(exit.Message.Epoch),
				ValidatorIndex: uint64(exit.Message.ValidatorIndex),
			},
			Signature: exit.Signature.String(),
		})
	}

	return exits
}

func NewWithdrawalsFromCapella(data []*capella.Withdrawal) []*WithdrawalV2 {
	withdrawals := []*WithdrawalV2{}

	if data == nil {
		return withdrawals
	}

	for _, withdrawal := range data {
		withdrawals = append(withdrawals, &WithdrawalV2{
			Index:          &wrapperspb.UInt64Value{Value: uint64(withdrawal.Index)},
			ValidatorIndex: &wrapperspb.UInt64Value{Value: uint64(withdrawal.ValidatorIndex)},
			Address:        withdrawal.Address.String(),
			Amount:         &wrapperspb.UInt64Value{Value: uint64(withdrawal.Amount)},
		})
	}

	return withdrawals
}

func NewElectraExecutionRequestsFromElectra(data *electra.ExecutionRequests) *ElectraExecutionRequests {
	requests := &ElectraExecutionRequests{}

	if data == nil {
		return requests
	}

	for _, consolidation := range data.Consolidations {
		requests.Consolidations = append(requests.Consolidations, &ElectraExecutionRequestConsolidation{
			SourceAddress: &wrapperspb.StringValue{Value: consolidation.SourceAddress.String()},
			SourcePubkey:  &wrapperspb.StringValue{Value: consolidation.SourcePubkey.String()},
			TargetPubkey:  &wrapperspb.StringValue{Value: consolidation.TargetPubkey.String()},
		})
	}

	for _, deposit := range data.Deposits {
		requests.Deposits = append(requests.Deposits, &ElectraExecutionRequestDeposit{
			Pubkey:                &wrapperspb.StringValue{Value: deposit.Pubkey.String()},
			WithdrawalCredentials: &wrapperspb.StringValue{Value: fmt.Sprintf("%#x", deposit.WithdrawalCredentials)},
			Amount:                &wrapperspb.UInt64Value{Value: uint64(deposit.Amount)},
			Signature:             &wrapperspb.StringValue{Value: deposit.Signature.String()},
			Index:                 &wrapperspb.UInt64Value{Value: deposit.Index},
		})
	}

	for _, withdrawal := range data.Withdrawals {
		requests.Withdrawals = append(requests.Withdrawals, &ElectraExecutionRequestWithdrawal{
			SourceAddress:   &wrapperspb.StringValue{Value: withdrawal.SourceAddress.String()},
			ValidatorPubkey: &wrapperspb.StringValue{Value: withdrawal.ValidatorPubkey.String()},
			Amount:          &wrapperspb.UInt64Value{Value: uint64(withdrawal.Amount)},
		})
	}

	return requests
}

// NewBlockAccessListFromGloas decodes a raw RLP-encoded block access list
// (EIP-7928) into the structured proto representation.
func NewBlockAccessListFromGloas(rawBAL gloas.BlockAccessList) *BlockAccessList {
	if len(rawBAL) == 0 {
		return &BlockAccessList{}
	}

	var accesses bal.BlockAccessList
	if err := rlp.DecodeBytes(rawBAL, &accesses); err != nil {
		return &BlockAccessList{}
	}

	entries := make([]*BlockAccessListEntry, 0, len(accesses))

	for i := range accesses {
		access := &accesses[i]
		entry := &BlockAccessListEntry{
			Address: &wrapperspb.StringValue{Value: fmt.Sprintf("0x%x", access.Address)},
		}

		// Storage changes: each slot has multiple writes keyed by tx index
		for _, slotWrite := range access.StorageChanges {
			slotHash := slotWrite.Slot.ToHash()

			for _, write := range slotWrite.Accesses {
				valueHash := write.ValueAfter.ToHash()

				entry.StorageChanges = append(entry.StorageChanges, &BlockAccessListStorageChange{
					BlockAccessIndex: &wrapperspb.UInt32Value{Value: uint32(write.TxIdx)},
					Key:              &wrapperspb.StringValue{Value: fmt.Sprintf("0x%x", slotHash)},
					NewValue:         &wrapperspb.StringValue{Value: fmt.Sprintf("0x%x", valueHash)},
				})
			}
		}

		// Balance changes
		for _, change := range access.BalanceChanges {
			entry.BalanceChanges = append(entry.BalanceChanges, &BlockAccessListBalanceChange{
				BlockAccessIndex: &wrapperspb.UInt32Value{Value: uint32(change.TxIdx)},
				PostBalance:      &wrapperspb.StringValue{Value: change.Balance.String()},
			})
		}

		// Nonce changes
		for _, change := range access.NonceChanges {
			entry.NonceChanges = append(entry.NonceChanges, &BlockAccessListNonceChange{
				BlockAccessIndex: &wrapperspb.UInt32Value{Value: uint32(change.TxIdx)},
				NewNonce:         &wrapperspb.UInt64Value{Value: change.Nonce},
			})
		}

		// Code changes
		for _, code := range access.CodeChanges {
			entry.CodeChanges = append(entry.CodeChanges, &BlockAccessListCodeChange{
				BlockAccessIndex: &wrapperspb.UInt32Value{Value: uint32(code.TxIndex)},
				NewCode:          &wrapperspb.StringValue{Value: fmt.Sprintf("0x%x", code.Code)},
			})
		}

		// Storage reads (read-only slots, no value or tx index)
		for _, slot := range access.StorageReads {
			slotHash := slot.ToHash()

			entry.StorageReads = append(entry.StorageReads, &BlockAccessListStorageRead{
				Key: &wrapperspb.StringValue{Value: fmt.Sprintf("0x%x", slotHash)},
			})
		}

		entries = append(entries, entry)
	}

	return &BlockAccessList{Entries: entries}
}
