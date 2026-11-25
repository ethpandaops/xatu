package v2

import (
	"fmt"

	"github.com/ethpandaops/go-eth2-client/spec/capella"
	"github.com/ethpandaops/go-eth2-client/spec/gloas"
	"github.com/ethpandaops/go-eth2-client/spec/phase0"
	"github.com/ethpandaops/go-eth2-client/spec/pq"
	"google.golang.org/protobuf/types/known/wrapperspb"

	v1 "github.com/ethpandaops/xatu/pkg/proto/eth/v1"
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

func NewAttesterSlashingsFromPQ(data []*pq.AttesterSlashing) []*AttesterSlashingPq {
	slashings := make([]*AttesterSlashingPq, 0, len(data))

	for _, slashing := range data {
		slashings = append(slashings, &AttesterSlashingPq{
			Attestation_1: newIndexedAttestationFromPQ(slashing.Attestation1),
			Attestation_2: newIndexedAttestationFromPQ(slashing.Attestation2),
		})
	}

	return slashings
}

func newIndexedAttestationFromPQ(attestation *pq.IndexedAttestation) *IndexedAttestationPq {
	if attestation == nil {
		return nil
	}

	return &IndexedAttestationPq{
		AttestingIndices: attestation.AttestingIndices,
		Data:             newAttestationDataFromPhase0(attestation.Data),
		Proof:            fmt.Sprintf("%#x", attestation.Proof),
	}
}

func NewAttestationsFromPQ(data []*pq.Attestation) []*v1.Attestation {
	attestations := make([]*v1.Attestation, 0, len(data))

	for _, attestation := range data {
		attestations = append(attestations, &v1.Attestation{
			AggregationBits: fmt.Sprintf("%#x", []byte(attestation.AggregationBits)),
			Data:            newAttestationDataFromPhase0(attestation.Data),
		})
	}

	return attestations
}

func newAttestationDataFromPhase0(data *phase0.AttestationData) *v1.AttestationData {
	if data == nil {
		return nil
	}

	out := &v1.AttestationData{
		Slot:            uint64(data.Slot),
		Index:           uint64(data.Index),
		BeaconBlockRoot: data.BeaconBlockRoot.String(),
	}

	if data.Source != nil {
		out.Source = &v1.Checkpoint{Epoch: uint64(data.Source.Epoch), Root: data.Source.Root.String()}
	}

	if data.Target != nil {
		out.Target = &v1.Checkpoint{Epoch: uint64(data.Target.Epoch), Root: data.Target.Root.String()}
	}

	return out
}

func NewProposerSlashingsFromPQ(data []*pq.ProposerSlashing) []*v1.ProposerSlashing {
	slashings := make([]*v1.ProposerSlashing, 0, len(data))

	for _, slashing := range data {
		slashings = append(slashings, &v1.ProposerSlashing{
			SignedHeader_1: newSignedBeaconBlockHeaderFromPQ(slashing.SignedHeader1),
			SignedHeader_2: newSignedBeaconBlockHeaderFromPQ(slashing.SignedHeader2),
		})
	}

	return slashings
}

func newSignedBeaconBlockHeaderFromPQ(header *pq.SignedBeaconBlockHeader) *v1.SignedBeaconBlockHeader {
	if header == nil || header.Message == nil {
		return nil
	}

	return &v1.SignedBeaconBlockHeader{
		Message: &v1.BeaconBlockHeader{
			Slot:          uint64(header.Message.Slot),
			ProposerIndex: uint64(header.Message.ProposerIndex),
			ParentRoot:    header.Message.ParentRoot.String(),
			StateRoot:     header.Message.StateRoot.String(),
			BodyRoot:      header.Message.BodyRoot.String(),
		},
		Signature: fmt.Sprintf("%#x", header.XMSSSignature[:]),
	}
}

func NewSignedExecutionPayloadBidFromPQ(bid *pq.SignedExecutionPayloadBid) *v1.SignedExecutionPayloadBid {
	if bid == nil {
		return nil
	}

	out := v1.NewSignedExecutionPayloadBidFromGloas(&gloas.SignedExecutionPayloadBid{Message: bid.Message})
	if out != nil {
		out.Signature = fmt.Sprintf("%#x", bid.XMSSSignature[:])
	}

	return out
}

func NewExecutionRequestsFromPQ(data *pq.ExecutionRequests) *ExecutionRequestsPq {
	if data == nil {
		return &ExecutionRequestsPq{}
	}

	shared := v1.NewGloasExecutionRequestsFromGloas(&gloas.ExecutionRequests{
		Deposits:        data.Deposits,
		Withdrawals:     data.Withdrawals,
		Consolidations:  data.Consolidations,
		BuilderDeposits: data.BuilderDeposits,
		BuilderExits:    data.BuilderExits,
	})

	requests := &ExecutionRequestsPq{
		Deposits:        shared.GetDeposits(),
		Withdrawals:     shared.GetWithdrawals(),
		Consolidations:  shared.GetConsolidations(),
		BuilderDeposits: shared.GetBuilderDeposits(),
		BuilderExits:    shared.GetBuilderExits(),
	}

	for _, r := range data.ValidatorPreregistrations {
		requests.ValidatorPreregistrations = append(requests.ValidatorPreregistrations, &ValidatorPreregistrationRequestPq{
			Pubkey:                wrapperspb.String(fmt.Sprintf("%#x", r.PublicKey[:])),
			WithdrawalCredentials: wrapperspb.String(fmt.Sprintf("%#x", r.WithdrawalCredentials)),
			VrfRoot:               wrapperspb.String(r.VRFRoot.String()),
		})
	}

	for _, r := range data.BuilderPreregistrations {
		requests.BuilderPreregistrations = append(requests.BuilderPreregistrations, &BuilderPreregistrationRequestPq{
			Pubkey:                wrapperspb.String(fmt.Sprintf("%#x", r.PublicKey[:])),
			WithdrawalCredentials: wrapperspb.String(fmt.Sprintf("%#x", r.WithdrawalCredentials)),
		})
	}

	for _, r := range data.VoluntaryExits {
		requests.VoluntaryExits = append(requests.VoluntaryExits, &VoluntaryExitRequestPq{
			Pubkey: wrapperspb.String(fmt.Sprintf("%#x", r.PublicKey[:])),
		})
	}

	return requests
}

// VRFProofToString hex encodes a VRF proof's 32 roots as one string.
func VRFProofToString(proof *pq.VRFProof) string {
	b := make([]byte, 0, len(proof)*len(proof[0]))
	for _, root := range proof {
		b = append(b, root[:]...)
	}

	return fmt.Sprintf("%#x", b)
}
