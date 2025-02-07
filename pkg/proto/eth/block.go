package eth

import (
	"fmt"
	"math/big"

	"github.com/attestantio/go-eth2-client/api"
	"github.com/attestantio/go-eth2-client/spec"
	"github.com/attestantio/go-eth2-client/spec/altair"
	"github.com/attestantio/go-eth2-client/spec/bellatrix"
	"github.com/attestantio/go-eth2-client/spec/capella"
	"github.com/attestantio/go-eth2-client/spec/deneb"
	"github.com/attestantio/go-eth2-client/spec/electra"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	v1 "github.com/ethpandaops/xatu/pkg/proto/eth/v1"
	v2 "github.com/ethpandaops/xatu/pkg/proto/eth/v2"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func NewEventBlockV2FromVersionedProposal(proposal *api.VersionedProposal) (*v2.EventBlockV2, error) {
	var data *v2.EventBlockV2

	switch proposal.Version {
	case spec.DataVersionPhase0:
		data = NewEventBlockFromPhase0(proposal.Phase0, nil)
	case spec.DataVersionAltair:
		data = NewEventBlockFromAltair(proposal.Altair, nil)
	case spec.DataVersionBellatrix:
		data = NewEventBlockFromBellatrix(proposal.Bellatrix, nil)
	case spec.DataVersionCapella:
		data = NewEventBlockFromCapella(proposal.Capella, nil)
	case spec.DataVersionDeneb:
		data = NewEventBlockFromDeneb(proposal.Deneb.Block, nil)
	case spec.DataVersionElectra:
		data = NewEventBlockFromElectra(proposal.Electra.Block, nil)
	default:
		return nil, fmt.Errorf("unsupported block version: %v", proposal.Version)
	}

	return data, nil
}

func NewEventBlockV2FromVersionSignedBeaconBlock(block *spec.VersionedSignedBeaconBlock) (*v2.EventBlockV2, error) {
	var data *v2.EventBlockV2

	switch block.Version {
	case spec.DataVersionPhase0:
		data = NewEventBlockFromPhase0(block.Phase0.Message, &block.Phase0.Signature)
	case spec.DataVersionAltair:
		data = NewEventBlockFromAltair(block.Altair.Message, &block.Altair.Signature)
	case spec.DataVersionBellatrix:
		data = NewEventBlockFromBellatrix(block.Bellatrix.Message, &block.Bellatrix.Signature)
	case spec.DataVersionCapella:
		data = NewEventBlockFromCapella(block.Capella.Message, &block.Capella.Signature)
	case spec.DataVersionDeneb:
		data = NewEventBlockFromDeneb(block.Deneb.Message, &block.Deneb.Signature)
	case spec.DataVersionElectra:
		data = NewEventBlockFromElectra(block.Electra.Message, &block.Electra.Signature)
	default:
		return nil, fmt.Errorf("unsupported block version: %v", block.Version)
	}

	return data, nil
}

func NewEventBlockFromPhase0(block *phase0.BeaconBlock, signature *phase0.BLSSignature) *v2.EventBlockV2 {
	event := &v2.EventBlockV2{
		Version: v2.BlockVersion_PHASE0,
		Message: &v2.EventBlockV2_Phase0Block{
			Phase0Block: &v1.BeaconBlockV2{
				Slot:          &wrapperspb.UInt64Value{Value: uint64(block.Slot)},
				ProposerIndex: &wrapperspb.UInt64Value{Value: uint64(block.ProposerIndex)},
				ParentRoot:    block.ParentRoot.String(),
				StateRoot:     block.StateRoot.String(),
				Body: &v1.BeaconBlockBody{
					RandaoReveal: block.Body.RANDAOReveal.String(),
					Eth1Data: &v1.Eth1Data{
						DepositRoot:  block.Body.ETH1Data.DepositRoot.String(),
						DepositCount: block.Body.ETH1Data.DepositCount,
						BlockHash:    fmt.Sprintf("0x%x", block.Body.ETH1Data.BlockHash),
					},
					Graffiti:          fmt.Sprintf("0x%x", block.Body.Graffiti[:]),
					ProposerSlashings: v1.NewProposerSlashingsFromPhase0(block.Body.ProposerSlashings),
					AttesterSlashings: v1.NewAttesterSlashingsFromPhase0(block.Body.AttesterSlashings),
					Attestations:      v1.NewAttestationsFromPhase0(block.Body.Attestations),
					Deposits:          v1.NewDepositsFromPhase0(block.Body.Deposits),
					VoluntaryExits:    v1.NewSignedVoluntaryExitsFromPhase0(block.Body.VoluntaryExits),
				},
			},
		},
	}

	if signature != nil && !signature.IsZero() {
		event.Signature = signature.String()
	}

	return event
}

func NewEventBlockFromAltair(block *altair.BeaconBlock, signature *phase0.BLSSignature) *v2.EventBlockV2 {
	event := &v2.EventBlockV2{
		Version: v2.BlockVersion_ALTAIR,
		Message: &v2.EventBlockV2_AltairBlock{
			AltairBlock: &v2.BeaconBlockAltairV2{
				Slot:          &wrapperspb.UInt64Value{Value: uint64(block.Slot)},
				ProposerIndex: &wrapperspb.UInt64Value{Value: uint64(block.ProposerIndex)},
				ParentRoot:    block.ParentRoot.String(),
				StateRoot:     block.StateRoot.String(),
				Body: &v2.BeaconBlockBodyAltairV2{
					RandaoReveal: block.Body.RANDAOReveal.String(),
					Eth1Data: &v1.Eth1Data{
						DepositRoot:  block.Body.ETH1Data.DepositRoot.String(),
						DepositCount: block.Body.ETH1Data.DepositCount,
						BlockHash:    fmt.Sprintf("0x%x", block.Body.ETH1Data.BlockHash),
					},
					Graffiti:          fmt.Sprintf("0x%x", block.Body.Graffiti[:]),
					ProposerSlashings: v1.NewProposerSlashingsFromPhase0(block.Body.ProposerSlashings),
					AttesterSlashings: v1.NewAttesterSlashingsFromPhase0(block.Body.AttesterSlashings),
					Attestations:      v1.NewAttestationsFromPhase0(block.Body.Attestations),
					Deposits:          v1.NewDepositsFromPhase0(block.Body.Deposits),
					VoluntaryExits:    v1.NewSignedVoluntaryExitsFromPhase0(block.Body.VoluntaryExits),
					SyncAggregate: &v1.SyncAggregate{
						SyncCommitteeBits:      fmt.Sprintf("0x%x", block.Body.SyncAggregate.SyncCommitteeBits),
						SyncCommitteeSignature: block.Body.SyncAggregate.SyncCommitteeSignature.String(),
					},
				},
			},
		},
	}

	if signature != nil && !signature.IsZero() {
		event.Signature = signature.String()
	}

	return event
}

func NewEventBlockFromBellatrix(block *bellatrix.BeaconBlock, signature *phase0.BLSSignature) *v2.EventBlockV2 {
	event := &v2.EventBlockV2{
		Version: v2.BlockVersion_BELLATRIX,
		Message: &v2.EventBlockV2_BellatrixBlock{
			BellatrixBlock: &v2.BeaconBlockBellatrixV2{
				Slot:          &wrapperspb.UInt64Value{Value: uint64(block.Slot)},
				ProposerIndex: &wrapperspb.UInt64Value{Value: uint64(block.ProposerIndex)},
				ParentRoot:    block.ParentRoot.String(),
				StateRoot:     block.StateRoot.String(),
				Body: &v2.BeaconBlockBodyBellatrixV2{
					RandaoReveal: block.Body.RANDAOReveal.String(),
					Eth1Data: &v1.Eth1Data{
						DepositRoot:  block.Body.ETH1Data.DepositRoot.String(),
						DepositCount: block.Body.ETH1Data.DepositCount,
						BlockHash:    fmt.Sprintf("0x%x", block.Body.ETH1Data.BlockHash),
					},
					Graffiti:          fmt.Sprintf("0x%x", block.Body.Graffiti[:]),
					ProposerSlashings: v1.NewProposerSlashingsFromPhase0(block.Body.ProposerSlashings),
					AttesterSlashings: v1.NewAttesterSlashingsFromPhase0(block.Body.AttesterSlashings),
					Attestations:      v1.NewAttestationsFromPhase0(block.Body.Attestations),
					Deposits:          v1.NewDepositsFromPhase0(block.Body.Deposits),
					VoluntaryExits:    v1.NewSignedVoluntaryExitsFromPhase0(block.Body.VoluntaryExits),
					SyncAggregate: &v1.SyncAggregate{
						SyncCommitteeBits:      fmt.Sprintf("0x%x", block.Body.SyncAggregate.SyncCommitteeBits),
						SyncCommitteeSignature: block.Body.SyncAggregate.SyncCommitteeSignature.String(),
					},
					ExecutionPayload: &v1.ExecutionPayloadV2{
						ParentHash:    block.Body.ExecutionPayload.ParentHash.String(),
						FeeRecipient:  block.Body.ExecutionPayload.FeeRecipient.String(),
						StateRoot:     fmt.Sprintf("0x%x", block.Body.ExecutionPayload.StateRoot[:]),
						ReceiptsRoot:  fmt.Sprintf("0x%x", block.Body.ExecutionPayload.ReceiptsRoot[:]),
						LogsBloom:     fmt.Sprintf("0x%x", block.Body.ExecutionPayload.LogsBloom[:]),
						PrevRandao:    fmt.Sprintf("0x%x", block.Body.ExecutionPayload.PrevRandao[:]),
						BlockNumber:   &wrapperspb.UInt64Value{Value: block.Body.ExecutionPayload.BlockNumber},
						GasLimit:      &wrapperspb.UInt64Value{Value: block.Body.ExecutionPayload.GasLimit},
						GasUsed:       &wrapperspb.UInt64Value{Value: block.Body.ExecutionPayload.GasUsed},
						Timestamp:     &wrapperspb.UInt64Value{Value: block.Body.ExecutionPayload.Timestamp},
						ExtraData:     fmt.Sprintf("0x%x", block.Body.ExecutionPayload.ExtraData),
						BaseFeePerGas: new(big.Int).SetBytes(block.Body.ExecutionPayload.BaseFeePerGas[:]).String(),
						BlockHash:     block.Body.ExecutionPayload.BlockHash.String(),
						Transactions:  getTransactions(block.Body.ExecutionPayload.Transactions),
					},
				},
			},
		},
	}

	if signature != nil && !signature.IsZero() {
		event.Signature = signature.String()
	}

	return event
}

func NewEventBlockFromCapella(block *capella.BeaconBlock, signature *phase0.BLSSignature) *v2.EventBlockV2 {
	event := &v2.EventBlockV2{
		Version: v2.BlockVersion_CAPELLA,
		Message: &v2.EventBlockV2_CapellaBlock{
			CapellaBlock: &v2.BeaconBlockCapellaV2{
				Slot:          &wrapperspb.UInt64Value{Value: uint64(block.Slot)},
				ProposerIndex: &wrapperspb.UInt64Value{Value: uint64(block.ProposerIndex)},
				ParentRoot:    block.ParentRoot.String(),
				StateRoot:     block.StateRoot.String(),
				Body: &v2.BeaconBlockBodyCapellaV2{
					RandaoReveal: block.Body.RANDAOReveal.String(),
					Eth1Data: &v1.Eth1Data{
						DepositRoot:  block.Body.ETH1Data.DepositRoot.String(),
						DepositCount: block.Body.ETH1Data.DepositCount,
						BlockHash:    fmt.Sprintf("0x%x", block.Body.ETH1Data.BlockHash),
					},
					Graffiti:          fmt.Sprintf("0x%x", block.Body.Graffiti[:]),
					ProposerSlashings: v1.NewProposerSlashingsFromPhase0(block.Body.ProposerSlashings),
					AttesterSlashings: v1.NewAttesterSlashingsFromPhase0(block.Body.AttesterSlashings),
					Attestations:      v1.NewAttestationsFromPhase0(block.Body.Attestations),
					Deposits:          v1.NewDepositsFromPhase0(block.Body.Deposits),
					VoluntaryExits:    v1.NewSignedVoluntaryExitsFromPhase0(block.Body.VoluntaryExits),
					SyncAggregate: &v1.SyncAggregate{
						SyncCommitteeBits:      fmt.Sprintf("0x%x", block.Body.SyncAggregate.SyncCommitteeBits),
						SyncCommitteeSignature: block.Body.SyncAggregate.SyncCommitteeSignature.String(),
					},
					ExecutionPayload: &v1.ExecutionPayloadCapellaV2{
						ParentHash:    block.Body.ExecutionPayload.ParentHash.String(),
						FeeRecipient:  block.Body.ExecutionPayload.FeeRecipient.String(),
						StateRoot:     fmt.Sprintf("0x%x", block.Body.ExecutionPayload.StateRoot[:]),
						ReceiptsRoot:  fmt.Sprintf("0x%x", block.Body.ExecutionPayload.ReceiptsRoot[:]),
						LogsBloom:     fmt.Sprintf("0x%x", block.Body.ExecutionPayload.LogsBloom[:]),
						PrevRandao:    fmt.Sprintf("0x%x", block.Body.ExecutionPayload.PrevRandao[:]),
						BlockNumber:   &wrapperspb.UInt64Value{Value: block.Body.ExecutionPayload.BlockNumber},
						GasLimit:      &wrapperspb.UInt64Value{Value: block.Body.ExecutionPayload.GasLimit},
						GasUsed:       &wrapperspb.UInt64Value{Value: block.Body.ExecutionPayload.GasUsed},
						Timestamp:     &wrapperspb.UInt64Value{Value: block.Body.ExecutionPayload.Timestamp},
						ExtraData:     fmt.Sprintf("0x%x", block.Body.ExecutionPayload.ExtraData),
						BaseFeePerGas: new(big.Int).SetBytes(block.Body.ExecutionPayload.BaseFeePerGas[:]).String(),
						BlockHash:     block.Body.ExecutionPayload.BlockHash.String(),
						Transactions:  getTransactions(block.Body.ExecutionPayload.Transactions),
						Withdrawals:   v1.NewWithdrawalsFromCapella(block.Body.ExecutionPayload.Withdrawals),
					},
					BlsToExecutionChanges: v2.NewBLSToExecutionChangesFromCapella(block.Body.BLSToExecutionChanges),
				},
			},
		},
	}

	if signature != nil && !signature.IsZero() {
		event.Signature = signature.String()
	}

	return event
}

func NewEventBlockFromDeneb(block *deneb.BeaconBlock, signature *phase0.BLSSignature) *v2.EventBlockV2 {
	kzgCommitments := make([]string, 0)

	for _, commitment := range block.Body.BlobKZGCommitments {
		kzgCommitments = append(kzgCommitments, commitment.String())
	}

	event := &v2.EventBlockV2{
		Version: v2.BlockVersion_DENEB,
		Message: &v2.EventBlockV2_DenebBlock{
			DenebBlock: &v2.BeaconBlockDeneb{
				Slot:          &wrapperspb.UInt64Value{Value: uint64(block.Slot)},
				ProposerIndex: &wrapperspb.UInt64Value{Value: uint64(block.ProposerIndex)},
				ParentRoot:    block.ParentRoot.String(),
				StateRoot:     block.StateRoot.String(),
				Body: &v2.BeaconBlockBodyDeneb{
					RandaoReveal: block.Body.RANDAOReveal.String(),
					Eth1Data: &v1.Eth1Data{
						DepositRoot:  block.Body.ETH1Data.DepositRoot.String(),
						DepositCount: block.Body.ETH1Data.DepositCount,
						BlockHash:    fmt.Sprintf("0x%x", block.Body.ETH1Data.BlockHash),
					},
					Graffiti:          fmt.Sprintf("0x%x", block.Body.Graffiti[:]),
					ProposerSlashings: v1.NewProposerSlashingsFromPhase0(block.Body.ProposerSlashings),
					AttesterSlashings: v1.NewAttesterSlashingsFromPhase0(block.Body.AttesterSlashings),
					Attestations:      v1.NewAttestationsFromPhase0(block.Body.Attestations),
					Deposits:          v1.NewDepositsFromPhase0(block.Body.Deposits),
					VoluntaryExits:    v1.NewSignedVoluntaryExitsFromPhase0(block.Body.VoluntaryExits),
					SyncAggregate: &v1.SyncAggregate{
						SyncCommitteeBits:      fmt.Sprintf("0x%x", block.Body.SyncAggregate.SyncCommitteeBits),
						SyncCommitteeSignature: block.Body.SyncAggregate.SyncCommitteeSignature.String(),
					},
					ExecutionPayload: &v1.ExecutionPayloadDeneb{
						ParentHash:    block.Body.ExecutionPayload.ParentHash.String(),
						FeeRecipient:  block.Body.ExecutionPayload.FeeRecipient.String(),
						StateRoot:     fmt.Sprintf("0x%x", block.Body.ExecutionPayload.StateRoot[:]),
						ReceiptsRoot:  fmt.Sprintf("0x%x", block.Body.ExecutionPayload.ReceiptsRoot[:]),
						LogsBloom:     fmt.Sprintf("0x%x", block.Body.ExecutionPayload.LogsBloom[:]),
						PrevRandao:    fmt.Sprintf("0x%x", block.Body.ExecutionPayload.PrevRandao[:]),
						BlockNumber:   &wrapperspb.UInt64Value{Value: block.Body.ExecutionPayload.BlockNumber},
						GasLimit:      &wrapperspb.UInt64Value{Value: block.Body.ExecutionPayload.GasLimit},
						GasUsed:       &wrapperspb.UInt64Value{Value: block.Body.ExecutionPayload.GasUsed},
						Timestamp:     &wrapperspb.UInt64Value{Value: block.Body.ExecutionPayload.Timestamp},
						ExtraData:     fmt.Sprintf("0x%x", block.Body.ExecutionPayload.ExtraData),
						BaseFeePerGas: block.Body.ExecutionPayload.BaseFeePerGas.String(),
						BlockHash:     block.Body.ExecutionPayload.BlockHash.String(),
						Transactions:  getTransactions(block.Body.ExecutionPayload.Transactions),
						Withdrawals:   v1.NewWithdrawalsFromCapella(block.Body.ExecutionPayload.Withdrawals),
						BlobGasUsed:   &wrapperspb.UInt64Value{Value: block.Body.ExecutionPayload.BlobGasUsed},
						ExcessBlobGas: &wrapperspb.UInt64Value{Value: block.Body.ExecutionPayload.ExcessBlobGas},
					},
					BlsToExecutionChanges: v2.NewBLSToExecutionChangesFromCapella(block.Body.BLSToExecutionChanges),
					BlobKzgCommitments:    kzgCommitments,
				},
			},
		},
	}

	if signature != nil && !signature.IsZero() {
		event.Signature = signature.String()
	}

	return event
}

func getTransactions(data []bellatrix.Transaction) []string {
	transactions := make([]string, 0)

	for _, tx := range data {
		transactions = append(transactions, fmt.Sprintf("0x%x", tx))
	}

	return transactions
}

func NewEventBlockFromElectra(block *electra.BeaconBlock, signature *phase0.BLSSignature) *v2.EventBlockV2 {
	kzgCommitments := []string{}

	for _, commitment := range block.Body.BlobKZGCommitments {
		kzgCommitments = append(kzgCommitments, commitment.String())
	}

	event := &v2.EventBlockV2{
		Version: v2.BlockVersion_ELECTRA,
		Message: &v2.EventBlockV2_ElectraBlock{
			ElectraBlock: &v2.BeaconBlockElectra{
				Slot:          &wrapperspb.UInt64Value{Value: uint64(block.Slot)},
				ProposerIndex: &wrapperspb.UInt64Value{Value: uint64(block.ProposerIndex)},
				ParentRoot:    block.ParentRoot.String(),
				StateRoot:     block.StateRoot.String(),
				Body: &v2.BeaconBlockBodyElectra{
					RandaoReveal: block.Body.RANDAOReveal.String(),
					Eth1Data: &v1.Eth1Data{
						DepositRoot:  block.Body.ETH1Data.DepositRoot.String(),
						DepositCount: block.Body.ETH1Data.DepositCount,
						BlockHash:    fmt.Sprintf("0x%x", block.Body.ETH1Data.BlockHash),
					},
					Graffiti:          fmt.Sprintf("0x%x", block.Body.Graffiti[:]),
					ProposerSlashings: v1.NewProposerSlashingsFromPhase0(block.Body.ProposerSlashings),
					AttesterSlashings: v1.NewAttesterSlashingsFromElectra(block.Body.AttesterSlashings),
					Attestations:      v1.NewAttestationsFromElectra(block.Body.Attestations),
					Deposits:          v1.NewDepositsFromPhase0(block.Body.Deposits),
					VoluntaryExits:    v1.NewSignedVoluntaryExitsFromPhase0(block.Body.VoluntaryExits),
					SyncAggregate: &v1.SyncAggregate{
						SyncCommitteeBits:      fmt.Sprintf("0x%x", block.Body.SyncAggregate.SyncCommitteeBits),
						SyncCommitteeSignature: block.Body.SyncAggregate.SyncCommitteeSignature.String(),
					},
					ExecutionPayload: &v1.ExecutionPayloadElectra{
						ParentHash:    block.Body.ExecutionPayload.ParentHash.String(),
						FeeRecipient:  block.Body.ExecutionPayload.FeeRecipient.String(),
						StateRoot:     fmt.Sprintf("0x%x", block.Body.ExecutionPayload.StateRoot[:]),
						ReceiptsRoot:  fmt.Sprintf("0x%x", block.Body.ExecutionPayload.ReceiptsRoot[:]),
						LogsBloom:     fmt.Sprintf("0x%x", block.Body.ExecutionPayload.LogsBloom[:]),
						PrevRandao:    fmt.Sprintf("0x%x", block.Body.ExecutionPayload.PrevRandao[:]),
						BlockNumber:   &wrapperspb.UInt64Value{Value: block.Body.ExecutionPayload.BlockNumber},
						GasLimit:      &wrapperspb.UInt64Value{Value: block.Body.ExecutionPayload.GasLimit},
						GasUsed:       &wrapperspb.UInt64Value{Value: block.Body.ExecutionPayload.GasUsed},
						Timestamp:     &wrapperspb.UInt64Value{Value: block.Body.ExecutionPayload.Timestamp},
						ExtraData:     fmt.Sprintf("0x%x", block.Body.ExecutionPayload.ExtraData),
						BaseFeePerGas: block.Body.ExecutionPayload.BaseFeePerGas.String(),
						BlockHash:     block.Body.ExecutionPayload.BlockHash.String(),
						Transactions:  getTransactions(block.Body.ExecutionPayload.Transactions),
						Withdrawals:   v1.NewWithdrawalsFromCapella(block.Body.ExecutionPayload.Withdrawals),
						BlobGasUsed:   &wrapperspb.UInt64Value{Value: block.Body.ExecutionPayload.BlobGasUsed},
						ExcessBlobGas: &wrapperspb.UInt64Value{Value: block.Body.ExecutionPayload.ExcessBlobGas},
					},
					BlsToExecutionChanges: v2.NewBLSToExecutionChangesFromCapella(block.Body.BLSToExecutionChanges),
					BlobKzgCommitments:    kzgCommitments,
					ExecutionRequests:     v1.NewElectraExecutionRequestsFromElectra(block.Body.ExecutionRequests),
				},
			},
		},
	}

	if signature != nil && !signature.IsZero() {
		event.Signature = signature.String()
	}

	return event
}
