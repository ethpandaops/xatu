package flattener

import "github.com/ethpandaops/xatu/pkg/proto/xatu"

func beaconRoutes() []Flattener {
	return []Flattener{
		NewGenericRoute(TableBeaconApiEthV1EventsHead, []xatu.Event_Name{
			xatu.Event_BEACON_API_ETH_V1_EVENTS_HEAD,
			xatu.Event_BEACON_API_ETH_V1_EVENTS_HEAD_V2,
		}),
		NewGenericRoute(TableBeaconApiEthV1EventsBlock, []xatu.Event_Name{
			xatu.Event_BEACON_API_ETH_V1_EVENTS_BLOCK,
			xatu.Event_BEACON_API_ETH_V1_EVENTS_BLOCK_V2,
		}),
		NewGenericRoute(TableBeaconApiEthV1EventsBlockGossip, []xatu.Event_Name{
			xatu.Event_BEACON_API_ETH_V1_EVENTS_BLOCK_GOSSIP,
		}),
		NewGenericRoute(TableBeaconApiEthV1EventsFinalizedCheckpoint, []xatu.Event_Name{
			xatu.Event_BEACON_API_ETH_V1_EVENTS_FINALIZED_CHECKPOINT,
			xatu.Event_BEACON_API_ETH_V1_EVENTS_FINALIZED_CHECKPOINT_V2,
		}),
		NewGenericRoute(TableBeaconApiEthV1EventsChainReorg, []xatu.Event_Name{
			xatu.Event_BEACON_API_ETH_V1_EVENTS_CHAIN_REORG,
			xatu.Event_BEACON_API_ETH_V1_EVENTS_CHAIN_REORG_V2,
		}),
		NewGenericRoute(TableBeaconApiEthV1BeaconBlob, []xatu.Event_Name{
			xatu.Event_BEACON_API_ETH_V1_BEACON_BLOB,
		}),
		NewGenericRoute(TableBeaconApiEthV1EventsBlobSidecar, []xatu.Event_Name{
			xatu.Event_BEACON_API_ETH_V1_EVENTS_BLOB_SIDECAR,
			xatu.Event_BEACON_API_ETH_V1_EVENTS_DATA_COLUMN_SIDECAR,
		}),
		NewGenericRoute(TableBeaconApiEthV1EventsDataColumnSidecar, []xatu.Event_Name{
			xatu.Event_BEACON_API_ETH_V1_EVENTS_DATA_COLUMN_SIDECAR,
		}),
		NewGenericRoute(TableBeaconApiEthV1EventsVoluntaryExit, []xatu.Event_Name{
			xatu.Event_BEACON_API_ETH_V1_EVENTS_VOLUNTARY_EXIT,
			xatu.Event_BEACON_API_ETH_V1_EVENTS_VOLUNTARY_EXIT_V2,
		}),
		NewGenericRoute(TableBeaconApiEthV1EventsContributionAndProof, []xatu.Event_Name{
			xatu.Event_BEACON_API_ETH_V1_EVENTS_CONTRIBUTION_AND_PROOF,
			xatu.Event_BEACON_API_ETH_V1_EVENTS_CONTRIBUTION_AND_PROOF_V2,
		}),
		NewGenericRoute(TableBeaconApiEthV1EventsAttestation, []xatu.Event_Name{
			xatu.Event_BEACON_API_ETH_V1_EVENTS_ATTESTATION,
			xatu.Event_BEACON_API_ETH_V1_EVENTS_ATTESTATION_V2,
		}),
		NewGenericRoute(TableBeaconApiEthV1ValidatorAttestationData, []xatu.Event_Name{
			xatu.Event_BEACON_API_ETH_V1_VALIDATOR_ATTESTATION_DATA,
		}),
		NewGenericRoute(TableCanonicalBeaconBlobSidecar, []xatu.Event_Name{
			xatu.Event_BEACON_API_ETH_V1_BEACON_BLOB_SIDECAR,
		}),
		NewGenericRoute(
			TableBeaconApiEthV1ProposerDuty,
			[]xatu.Event_Name{xatu.Event_BEACON_API_ETH_V1_PROPOSER_DUTY},
			WithPredicate(stringFromAdditionalData("head", "state_id")),
		),
		NewGenericRoute(
			TableBeaconApiEthV1BeaconCommittee,
			[]xatu.Event_Name{xatu.Event_BEACON_API_ETH_V1_BEACON_COMMITTEE},
			WithPredicate(stringNotFromAdditionalData("finalized", "state_id")),
		),
		NewGenericRoute(
			TableCanonicalBeaconCommittee,
			[]xatu.Event_Name{xatu.Event_BEACON_API_ETH_V1_BEACON_COMMITTEE},
			WithPredicate(stringFromAdditionalData("finalized", "state_id")),
		),
		NewGenericRoute(
			TableCanonicalBeaconProposerDuty,
			[]xatu.Event_Name{xatu.Event_BEACON_API_ETH_V1_PROPOSER_DUTY},
			WithPredicate(stringFromAdditionalData("finalized", "state_id")),
		),
		NewGenericRoute(TableBeaconApiEthV2BeaconBlock, []xatu.Event_Name{
			xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK,
			xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_V2,
		}),
		NewGenericRoute(
			TableCanonicalBeaconBlock,
			[]xatu.Event_Name{xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_V2},
			WithPredicate(boolFromAdditionalData("finalized_when_requested")),
		),
		NewGenericRoute(TableBeaconApiEthV3ValidatorBlock, []xatu.Event_Name{
			xatu.Event_BEACON_API_ETH_V3_VALIDATOR_BLOCK,
		}),
		NewValidatorsFanoutFlattener(TableCanonicalBeaconValidators, ValidatorsFanoutKindValidators),
		NewValidatorsFanoutFlattener(TableCanonicalBeaconValidatorsPubkeys, ValidatorsFanoutKindPubkeys),
		NewValidatorsFanoutFlattener(
			TableCanonicalBeaconValidatorsWithdrawalCredentials,
			ValidatorsFanoutKindWithdrawalCredential,
		),
		NewGenericRoute(TableCanonicalBeaconBlockAttesterSlashing, []xatu.Event_Name{
			xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_ATTESTER_SLASHING,
		}),
		NewGenericRoute(
			TableCanonicalBeaconElaboratedAttestation,
			[]xatu.Event_Name{xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_ELABORATED_ATTESTATION},
			WithAliases(map[string]string{"validators": "validator_indexes"}),
		),
		NewGenericRoute(TableCanonicalBeaconBlockBlsToExecutionChange, []xatu.Event_Name{
			xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_BLS_TO_EXECUTION_CHANGE,
		}),
		NewGenericRoute(TableCanonicalBeaconBlockDeposit, []xatu.Event_Name{
			xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_DEPOSIT,
		}),
		NewGenericRoute(TableCanonicalBeaconBlockExecutionTransaction, []xatu.Event_Name{
			xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_EXECUTION_TRANSACTION,
		}),
		NewGenericRoute(TableCanonicalBeaconBlockProposerSlashing, []xatu.Event_Name{
			xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_PROPOSER_SLASHING,
		}),
		NewGenericRoute(TableCanonicalBeaconBlockVoluntaryExit, []xatu.Event_Name{
			xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_VOLUNTARY_EXIT,
		}),
		NewGenericRoute(TableCanonicalBeaconBlockWithdrawal, []xatu.Event_Name{
			xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_WITHDRAWAL,
		}),
		NewGenericRoute(
			TableCanonicalBeaconSyncCommittee,
			[]xatu.Event_Name{xatu.Event_BEACON_API_ETH_V1_BEACON_SYNC_COMMITTEE},
			WithMutator(syncCommitteeMutator),
		),
		NewGenericRoute(
			TableCanonicalBeaconBlockSyncAggregate,
			[]xatu.Event_Name{xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_SYNC_AGGREGATE},
			WithMutator(syncAggregateMutator),
		),
	}
}
