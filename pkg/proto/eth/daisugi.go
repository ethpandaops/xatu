package eth

import (
	"github.com/ethpandaops/go-eth2-client/spec"
	"github.com/ethpandaops/go-eth2-client/spec/daisugi"
	"github.com/ethpandaops/go-eth2-client/spec/deneb"
	"github.com/ethpandaops/go-eth2-client/spec/electra"
)

// ForkAtLeast reports whether version is fork or a later fork. Daisugi branches
// off Electra but sorts after every canonical fork, so it compares as Electra.
func ForkAtLeast(version, fork spec.DataVersion) bool {
	if version == spec.DataVersionDaisugi && fork != spec.DataVersionDaisugi {
		version = spec.DataVersionElectra
	}

	return version >= fork
}

// ElectraBlockFromDaisugi returns a Daisugi block as the Electra block it
// extends. The execution payload's recursive STARK proof is dropped.
func ElectraBlockFromDaisugi(block *daisugi.BeaconBlock) *electra.BeaconBlock {
	out := &electra.BeaconBlock{
		Slot:          block.Slot,
		ProposerIndex: block.ProposerIndex,
		ParentRoot:    block.ParentRoot,
		StateRoot:     block.StateRoot,
	}

	body := block.Body
	if body == nil {
		return out
	}

	out.Body = &electra.BeaconBlockBody{
		RANDAOReveal:          body.RANDAOReveal,
		ETH1Data:              body.ETH1Data,
		Graffiti:              body.Graffiti,
		ProposerSlashings:     body.ProposerSlashings,
		AttesterSlashings:     body.AttesterSlashings,
		Attestations:          body.Attestations,
		Deposits:              body.Deposits,
		VoluntaryExits:        body.VoluntaryExits,
		SyncAggregate:         body.SyncAggregate,
		BLSToExecutionChanges: body.BLSToExecutionChanges,
		BlobKZGCommitments:    body.BlobKZGCommitments,
		ExecutionRequests:     body.ExecutionRequests,
	}

	if payload := body.ExecutionPayload; payload != nil {
		out.Body.ExecutionPayload = &deneb.ExecutionPayload{
			ParentHash:    payload.ParentHash,
			FeeRecipient:  payload.FeeRecipient,
			StateRoot:     payload.StateRoot,
			ReceiptsRoot:  payload.ReceiptsRoot,
			LogsBloom:     payload.LogsBloom,
			PrevRandao:    payload.PrevRandao,
			BlockNumber:   payload.BlockNumber,
			GasLimit:      payload.GasLimit,
			GasUsed:       payload.GasUsed,
			Timestamp:     payload.Timestamp,
			ExtraData:     payload.ExtraData,
			BaseFeePerGas: payload.BaseFeePerGas,
			BlockHash:     payload.BlockHash,
			Transactions:  payload.Transactions,
			Withdrawals:   payload.Withdrawals,
			BlobGasUsed:   payload.BlobGasUsed,
			ExcessBlobGas: payload.ExcessBlobGas,
		}
	}

	return out
}
