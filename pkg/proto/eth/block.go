package eth

import (
	"fmt"
	"math/big"

	"github.com/attestantio/go-eth2-client/spec"
	"github.com/attestantio/go-eth2-client/spec/bellatrix"
	v1 "github.com/ethpandaops/xatu/pkg/proto/eth/v1"
	v2 "github.com/ethpandaops/xatu/pkg/proto/eth/v2"
	wrapperspb "google.golang.org/protobuf/types/known/wrapperspb"
)

func NewEventBlockV2FromVersionSignedBeaconBlock(block *spec.VersionedSignedBeaconBlock) (*v2.EventBlockV2, error) {
	var data *v2.EventBlockV2

	switch block.Version {
	case spec.DataVersionPhase0:
		data = NewEventBlockFromPhase0(block)
	case spec.DataVersionAltair:
		data = NewEventBlockFromAltair(block)
	case spec.DataVersionBellatrix:
		data = NewEventBlockFromBellatrix(block)
	case spec.DataVersionCapella:
		data = NewEventBlockFromCapella(block)
	case spec.DataVersionDeneb:
		data = NewEventBlockFromDeneb(block)
	default:
		return nil, fmt.Errorf("unsupported block version: %v", block.Version)
	}

	return data, nil
}

func getTransactions(data []bellatrix.Transaction) []string {
	transactions := []string{}

	for _, tx := range data {
		transactions = append(transactions, fmt.Sprintf("0x%x", tx))
	}

	return transactions
}

func NewEventBlockFromPhase0(block *spec.VersionedSignedBeaconBlock) *v2.EventBlockV2 {
	return &v2.EventBlockV2{
		Version: v2.BlockVersion_PHASE0,
		Message: &v2.EventBlockV2_Phase0Block{
			Phase0Block: &v1.BeaconBlockV2{
				Slot:          &wrapperspb.UInt64Value{Value: uint64(block.Phase0.Message.Slot)},
				ProposerIndex: &wrapperspb.UInt64Value{Value: uint64(block.Phase0.Message.ProposerIndex)},
				ParentRoot:    block.Phase0.Message.ParentRoot.String(),
				StateRoot:     block.Phase0.Message.StateRoot.String(),
				Body: &v1.BeaconBlockBody{
					RandaoReveal: block.Phase0.Message.Body.RANDAOReveal.String(),
					Eth1Data: &v1.Eth1Data{
						DepositRoot:  block.Phase0.Message.Body.ETH1Data.DepositRoot.String(),
						DepositCount: block.Phase0.Message.Body.ETH1Data.DepositCount,
						BlockHash:    fmt.Sprintf("0x%x", block.Phase0.Message.Body.ETH1Data.BlockHash),
					},
					Graffiti:          fmt.Sprintf("0x%x", block.Phase0.Message.Body.Graffiti[:]),
					ProposerSlashings: v1.NewProposerSlashingsFromPhase0(block.Phase0.Message.Body.ProposerSlashings),
					AttesterSlashings: v1.NewAttesterSlashingsFromPhase0(block.Phase0.Message.Body.AttesterSlashings),
					Attestations:      v1.NewAttestationsFromPhase0(block.Phase0.Message.Body.Attestations),
					Deposits:          v1.NewDepositsFromPhase0(block.Phase0.Message.Body.Deposits),
					VoluntaryExits:    v1.NewSignedVoluntaryExitsFromPhase0(block.Phase0.Message.Body.VoluntaryExits),
				},
			},
		},
		Signature: block.Phase0.Signature.String(),
	}
}

func NewEventBlockFromAltair(block *spec.VersionedSignedBeaconBlock) *v2.EventBlockV2 {
	return &v2.EventBlockV2{
		Version: v2.BlockVersion_ALTAIR,
		Message: &v2.EventBlockV2_AltairBlock{
			AltairBlock: &v2.BeaconBlockAltairV2{
				Slot:          &wrapperspb.UInt64Value{Value: uint64(block.Altair.Message.Slot)},
				ProposerIndex: &wrapperspb.UInt64Value{Value: uint64(block.Altair.Message.ProposerIndex)},
				ParentRoot:    block.Altair.Message.ParentRoot.String(),
				StateRoot:     block.Altair.Message.StateRoot.String(),
				Body: &v2.BeaconBlockBodyAltairV2{
					RandaoReveal: block.Altair.Message.Body.RANDAOReveal.String(),
					Eth1Data: &v1.Eth1Data{
						DepositRoot:  block.Altair.Message.Body.ETH1Data.DepositRoot.String(),
						DepositCount: block.Altair.Message.Body.ETH1Data.DepositCount,
						BlockHash:    fmt.Sprintf("0x%x", block.Altair.Message.Body.ETH1Data.BlockHash),
					},
					Graffiti:          fmt.Sprintf("0x%x", block.Altair.Message.Body.Graffiti[:]),
					ProposerSlashings: v1.NewProposerSlashingsFromPhase0(block.Altair.Message.Body.ProposerSlashings),
					AttesterSlashings: v1.NewAttesterSlashingsFromPhase0(block.Altair.Message.Body.AttesterSlashings),
					Attestations:      v1.NewAttestationsFromPhase0(block.Altair.Message.Body.Attestations),
					Deposits:          v1.NewDepositsFromPhase0(block.Altair.Message.Body.Deposits),
					VoluntaryExits:    v1.NewSignedVoluntaryExitsFromPhase0(block.Altair.Message.Body.VoluntaryExits),
					SyncAggregate: &v1.SyncAggregate{
						SyncCommitteeBits:      fmt.Sprintf("0x%x", block.Altair.Message.Body.SyncAggregate.SyncCommitteeBits),
						SyncCommitteeSignature: block.Altair.Message.Body.SyncAggregate.SyncCommitteeSignature.String(),
					},
				},
			},
		},
		Signature: block.Altair.Signature.String(),
	}
}

func NewEventBlockFromBellatrix(block *spec.VersionedSignedBeaconBlock) *v2.EventBlockV2 {
	return &v2.EventBlockV2{
		Version: v2.BlockVersion_BELLATRIX,
		Message: &v2.EventBlockV2_BellatrixBlock{
			BellatrixBlock: &v2.BeaconBlockBellatrixV2{
				Slot:          &wrapperspb.UInt64Value{Value: uint64(block.Bellatrix.Message.Slot)},
				ProposerIndex: &wrapperspb.UInt64Value{Value: uint64(block.Bellatrix.Message.ProposerIndex)},
				ParentRoot:    block.Bellatrix.Message.ParentRoot.String(),
				StateRoot:     block.Bellatrix.Message.StateRoot.String(),
				Body: &v2.BeaconBlockBodyBellatrixV2{
					RandaoReveal: block.Bellatrix.Message.Body.RANDAOReveal.String(),
					Eth1Data: &v1.Eth1Data{
						DepositRoot:  block.Bellatrix.Message.Body.ETH1Data.DepositRoot.String(),
						DepositCount: block.Bellatrix.Message.Body.ETH1Data.DepositCount,
						BlockHash:    fmt.Sprintf("0x%x", block.Bellatrix.Message.Body.ETH1Data.BlockHash),
					},
					Graffiti:          fmt.Sprintf("0x%x", block.Bellatrix.Message.Body.Graffiti[:]),
					ProposerSlashings: v1.NewProposerSlashingsFromPhase0(block.Bellatrix.Message.Body.ProposerSlashings),
					AttesterSlashings: v1.NewAttesterSlashingsFromPhase0(block.Bellatrix.Message.Body.AttesterSlashings),
					Attestations:      v1.NewAttestationsFromPhase0(block.Bellatrix.Message.Body.Attestations),
					Deposits:          v1.NewDepositsFromPhase0(block.Bellatrix.Message.Body.Deposits),
					VoluntaryExits:    v1.NewSignedVoluntaryExitsFromPhase0(block.Bellatrix.Message.Body.VoluntaryExits),
					SyncAggregate: &v1.SyncAggregate{
						SyncCommitteeBits:      fmt.Sprintf("0x%x", block.Bellatrix.Message.Body.SyncAggregate.SyncCommitteeBits),
						SyncCommitteeSignature: block.Bellatrix.Message.Body.SyncAggregate.SyncCommitteeSignature.String(),
					},
					ExecutionPayload: &v1.ExecutionPayloadV2{
						ParentHash:    block.Bellatrix.Message.Body.ExecutionPayload.ParentHash.String(),
						FeeRecipient:  block.Bellatrix.Message.Body.ExecutionPayload.FeeRecipient.String(),
						StateRoot:     fmt.Sprintf("0x%x", block.Bellatrix.Message.Body.ExecutionPayload.StateRoot[:]),
						ReceiptsRoot:  fmt.Sprintf("0x%x", block.Bellatrix.Message.Body.ExecutionPayload.ReceiptsRoot[:]),
						LogsBloom:     fmt.Sprintf("0x%x", block.Bellatrix.Message.Body.ExecutionPayload.LogsBloom[:]),
						PrevRandao:    fmt.Sprintf("0x%x", block.Bellatrix.Message.Body.ExecutionPayload.PrevRandao[:]),
						BlockNumber:   &wrapperspb.UInt64Value{Value: block.Bellatrix.Message.Body.ExecutionPayload.BlockNumber},
						GasLimit:      &wrapperspb.UInt64Value{Value: block.Bellatrix.Message.Body.ExecutionPayload.GasLimit},
						GasUsed:       &wrapperspb.UInt64Value{Value: block.Bellatrix.Message.Body.ExecutionPayload.GasUsed},
						Timestamp:     &wrapperspb.UInt64Value{Value: block.Bellatrix.Message.Body.ExecutionPayload.Timestamp},
						ExtraData:     fmt.Sprintf("0x%x", block.Bellatrix.Message.Body.ExecutionPayload.ExtraData),
						BaseFeePerGas: new(big.Int).SetBytes(block.Capella.Message.Body.ExecutionPayload.BaseFeePerGas[:]).String(),
						BlockHash:     block.Bellatrix.Message.Body.ExecutionPayload.BlockHash.String(),
						Transactions:  getTransactions(block.Bellatrix.Message.Body.ExecutionPayload.Transactions),
					},
				},
			},
		},
		Signature: block.Bellatrix.Signature.String(),
	}
}

func NewEventBlockFromCapella(block *spec.VersionedSignedBeaconBlock) *v2.EventBlockV2 {
	return &v2.EventBlockV2{
		Version: v2.BlockVersion_CAPELLA,
		Message: &v2.EventBlockV2_CapellaBlock{
			CapellaBlock: &v2.BeaconBlockCapellaV2{
				Slot:          &wrapperspb.UInt64Value{Value: uint64(block.Capella.Message.Slot)},
				ProposerIndex: &wrapperspb.UInt64Value{Value: uint64(block.Capella.Message.ProposerIndex)},
				ParentRoot:    block.Capella.Message.ParentRoot.String(),
				StateRoot:     block.Capella.Message.StateRoot.String(),
				Body: &v2.BeaconBlockBodyCapellaV2{
					RandaoReveal: block.Capella.Message.Body.RANDAOReveal.String(),
					Eth1Data: &v1.Eth1Data{
						DepositRoot:  block.Capella.Message.Body.ETH1Data.DepositRoot.String(),
						DepositCount: block.Capella.Message.Body.ETH1Data.DepositCount,
						BlockHash:    fmt.Sprintf("0x%x", block.Capella.Message.Body.ETH1Data.BlockHash),
					},
					Graffiti:          fmt.Sprintf("0x%x", block.Capella.Message.Body.Graffiti[:]),
					ProposerSlashings: v1.NewProposerSlashingsFromPhase0(block.Capella.Message.Body.ProposerSlashings),
					AttesterSlashings: v1.NewAttesterSlashingsFromPhase0(block.Capella.Message.Body.AttesterSlashings),
					Attestations:      v1.NewAttestationsFromPhase0(block.Capella.Message.Body.Attestations),
					Deposits:          v1.NewDepositsFromPhase0(block.Capella.Message.Body.Deposits),
					VoluntaryExits:    v1.NewSignedVoluntaryExitsFromPhase0(block.Capella.Message.Body.VoluntaryExits),
					SyncAggregate: &v1.SyncAggregate{
						SyncCommitteeBits:      fmt.Sprintf("0x%x", block.Capella.Message.Body.SyncAggregate.SyncCommitteeBits),
						SyncCommitteeSignature: block.Capella.Message.Body.SyncAggregate.SyncCommitteeSignature.String(),
					},
					ExecutionPayload: &v1.ExecutionPayloadCapellaV2{
						ParentHash:    block.Capella.Message.Body.ExecutionPayload.ParentHash.String(),
						FeeRecipient:  block.Capella.Message.Body.ExecutionPayload.FeeRecipient.String(),
						StateRoot:     fmt.Sprintf("0x%x", block.Capella.Message.Body.ExecutionPayload.StateRoot[:]),
						ReceiptsRoot:  fmt.Sprintf("0x%x", block.Capella.Message.Body.ExecutionPayload.ReceiptsRoot[:]),
						LogsBloom:     fmt.Sprintf("0x%x", block.Capella.Message.Body.ExecutionPayload.LogsBloom[:]),
						PrevRandao:    fmt.Sprintf("0x%x", block.Capella.Message.Body.ExecutionPayload.PrevRandao[:]),
						BlockNumber:   &wrapperspb.UInt64Value{Value: block.Capella.Message.Body.ExecutionPayload.BlockNumber},
						GasLimit:      &wrapperspb.UInt64Value{Value: block.Capella.Message.Body.ExecutionPayload.GasLimit},
						GasUsed:       &wrapperspb.UInt64Value{Value: block.Capella.Message.Body.ExecutionPayload.GasUsed},
						Timestamp:     &wrapperspb.UInt64Value{Value: block.Capella.Message.Body.ExecutionPayload.Timestamp},
						ExtraData:     fmt.Sprintf("0x%x", block.Capella.Message.Body.ExecutionPayload.ExtraData),
						BaseFeePerGas: new(big.Int).SetBytes(block.Capella.Message.Body.ExecutionPayload.BaseFeePerGas[:]).String(),
						BlockHash:     block.Capella.Message.Body.ExecutionPayload.BlockHash.String(),
						Transactions:  getTransactions(block.Capella.Message.Body.ExecutionPayload.Transactions),
						Withdrawals:   v1.NewWithdrawalsFromCapella(block.Capella.Message.Body.ExecutionPayload.Withdrawals),
					},
					BlsToExecutionChanges: v2.NewBLSToExecutionChangesFromCapella(block.Capella.Message.Body.BLSToExecutionChanges),
				},
			},
		},
		Signature: block.Capella.Signature.String(),
	}
}

func NewEventBlockFromDeneb(block *spec.VersionedSignedBeaconBlock) *v2.EventBlockV2 {
	kzgCommitments := []string{}

	for _, commitment := range block.Deneb.Message.Body.BlobKZGCommitments {
		kzgCommitments = append(kzgCommitments, commitment.String())
	}

	return &v2.EventBlockV2{
		Version: v2.BlockVersion_DENEB,
		Message: &v2.EventBlockV2_DenebBlock{
			DenebBlock: &v2.BeaconBlockDeneb{
				Slot:          &wrapperspb.UInt64Value{Value: uint64(block.Deneb.Message.Slot)},
				ProposerIndex: &wrapperspb.UInt64Value{Value: uint64(block.Deneb.Message.ProposerIndex)},
				ParentRoot:    block.Deneb.Message.ParentRoot.String(),
				StateRoot:     block.Deneb.Message.StateRoot.String(),
				Body: &v2.BeaconBlockBodyDeneb{
					RandaoReveal: block.Deneb.Message.Body.RANDAOReveal.String(),
					Eth1Data: &v1.Eth1Data{
						DepositRoot:  block.Deneb.Message.Body.ETH1Data.DepositRoot.String(),
						DepositCount: block.Deneb.Message.Body.ETH1Data.DepositCount,
						BlockHash:    fmt.Sprintf("0x%x", block.Deneb.Message.Body.ETH1Data.BlockHash),
					},
					Graffiti:          fmt.Sprintf("0x%x", block.Deneb.Message.Body.Graffiti[:]),
					ProposerSlashings: v1.NewProposerSlashingsFromPhase0(block.Deneb.Message.Body.ProposerSlashings),
					AttesterSlashings: v1.NewAttesterSlashingsFromPhase0(block.Deneb.Message.Body.AttesterSlashings),
					Attestations:      v1.NewAttestationsFromPhase0(block.Deneb.Message.Body.Attestations),
					Deposits:          v1.NewDepositsFromPhase0(block.Deneb.Message.Body.Deposits),
					VoluntaryExits:    v1.NewSignedVoluntaryExitsFromPhase0(block.Deneb.Message.Body.VoluntaryExits),
					SyncAggregate: &v1.SyncAggregate{
						SyncCommitteeBits:      fmt.Sprintf("0x%x", block.Deneb.Message.Body.SyncAggregate.SyncCommitteeBits),
						SyncCommitteeSignature: block.Deneb.Message.Body.SyncAggregate.SyncCommitteeSignature.String(),
					},
					ExecutionPayload: &v1.ExecutionPayloadDeneb{
						ParentHash:    block.Deneb.Message.Body.ExecutionPayload.ParentHash.String(),
						FeeRecipient:  block.Deneb.Message.Body.ExecutionPayload.FeeRecipient.String(),
						StateRoot:     fmt.Sprintf("0x%x", block.Deneb.Message.Body.ExecutionPayload.StateRoot[:]),
						ReceiptsRoot:  fmt.Sprintf("0x%x", block.Deneb.Message.Body.ExecutionPayload.ReceiptsRoot[:]),
						LogsBloom:     fmt.Sprintf("0x%x", block.Deneb.Message.Body.ExecutionPayload.LogsBloom[:]),
						PrevRandao:    fmt.Sprintf("0x%x", block.Deneb.Message.Body.ExecutionPayload.PrevRandao[:]),
						BlockNumber:   &wrapperspb.UInt64Value{Value: block.Deneb.Message.Body.ExecutionPayload.BlockNumber},
						GasLimit:      &wrapperspb.UInt64Value{Value: block.Deneb.Message.Body.ExecutionPayload.GasLimit},
						GasUsed:       &wrapperspb.UInt64Value{Value: block.Deneb.Message.Body.ExecutionPayload.GasUsed},
						Timestamp:     &wrapperspb.UInt64Value{Value: block.Deneb.Message.Body.ExecutionPayload.Timestamp},
						ExtraData:     fmt.Sprintf("0x%x", block.Deneb.Message.Body.ExecutionPayload.ExtraData),
						BaseFeePerGas: block.Deneb.Message.Body.ExecutionPayload.BaseFeePerGas.String(),
						BlockHash:     block.Deneb.Message.Body.ExecutionPayload.BlockHash.String(),
						Transactions:  getTransactions(block.Deneb.Message.Body.ExecutionPayload.Transactions),
						Withdrawals:   v1.NewWithdrawalsFromCapella(block.Deneb.Message.Body.ExecutionPayload.Withdrawals),
						BlobGasUsed:   &wrapperspb.UInt64Value{Value: block.Deneb.Message.Body.ExecutionPayload.BlobGasUsed},
						ExcessBlobGas: &wrapperspb.UInt64Value{Value: block.Deneb.Message.Body.ExecutionPayload.ExcessBlobGas},
					},
					BlsToExecutionChanges: v2.NewBLSToExecutionChangesFromCapella(block.Deneb.Message.Body.BLSToExecutionChanges),
					BlobKzgCommitments:    kzgCommitments,
				},
			},
		},
		Signature: block.Deneb.Signature.String(),
	}
}
