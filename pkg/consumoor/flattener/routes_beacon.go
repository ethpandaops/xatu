package flattener

import "github.com/ethpandaops/xatu/pkg/proto/xatu"

func beaconRoutes() []TableDefinition {
	return []TableDefinition{
		GenericTable(TableBeaconApiEthV1EventsHead, []xatu.Event_Name{
			xatu.Event_BEACON_API_ETH_V1_EVENTS_HEAD,
			xatu.Event_BEACON_API_ETH_V1_EVENTS_HEAD_V2,
		}),
		GenericTable(TableBeaconApiEthV1EventsBlock, []xatu.Event_Name{
			xatu.Event_BEACON_API_ETH_V1_EVENTS_BLOCK,
			xatu.Event_BEACON_API_ETH_V1_EVENTS_BLOCK_V2,
		}),
		GenericTable(TableBeaconApiEthV1EventsBlockGossip, []xatu.Event_Name{
			xatu.Event_BEACON_API_ETH_V1_EVENTS_BLOCK_GOSSIP,
		}),
		GenericTable(TableBeaconApiEthV1EventsFinalizedCheckpoint, []xatu.Event_Name{
			xatu.Event_BEACON_API_ETH_V1_EVENTS_FINALIZED_CHECKPOINT,
			xatu.Event_BEACON_API_ETH_V1_EVENTS_FINALIZED_CHECKPOINT_V2,
		}),
		GenericTable(TableBeaconApiEthV1EventsChainReorg, []xatu.Event_Name{
			xatu.Event_BEACON_API_ETH_V1_EVENTS_CHAIN_REORG,
			xatu.Event_BEACON_API_ETH_V1_EVENTS_CHAIN_REORG_V2,
		}),
		GenericTable(TableBeaconApiEthV1BeaconBlob, []xatu.Event_Name{
			xatu.Event_BEACON_API_ETH_V1_BEACON_BLOB,
		}),
		GenericTable(TableBeaconApiEthV1EventsBlobSidecar, []xatu.Event_Name{
			xatu.Event_BEACON_API_ETH_V1_EVENTS_BLOB_SIDECAR,
			xatu.Event_BEACON_API_ETH_V1_EVENTS_DATA_COLUMN_SIDECAR,
		}),
		GenericTable(TableBeaconApiEthV1EventsDataColumnSidecar, []xatu.Event_Name{
			xatu.Event_BEACON_API_ETH_V1_EVENTS_DATA_COLUMN_SIDECAR,
		}),
		GenericTable(TableBeaconApiEthV1EventsVoluntaryExit, []xatu.Event_Name{
			xatu.Event_BEACON_API_ETH_V1_EVENTS_VOLUNTARY_EXIT,
			xatu.Event_BEACON_API_ETH_V1_EVENTS_VOLUNTARY_EXIT_V2,
		}),
		GenericTable(TableBeaconApiEthV1EventsContributionAndProof, []xatu.Event_Name{
			xatu.Event_BEACON_API_ETH_V1_EVENTS_CONTRIBUTION_AND_PROOF,
			xatu.Event_BEACON_API_ETH_V1_EVENTS_CONTRIBUTION_AND_PROOF_V2,
		}),
		GenericTable(TableBeaconApiEthV1EventsAttestation, []xatu.Event_Name{
			xatu.Event_BEACON_API_ETH_V1_EVENTS_ATTESTATION,
			xatu.Event_BEACON_API_ETH_V1_EVENTS_ATTESTATION_V2,
		}),
		GenericTable(TableBeaconApiEthV1ValidatorAttestationData, []xatu.Event_Name{
			xatu.Event_BEACON_API_ETH_V1_VALIDATOR_ATTESTATION_DATA,
		}),
		GenericTable(TableCanonicalBeaconBlobSidecar, []xatu.Event_Name{
			xatu.Event_BEACON_API_ETH_V1_BEACON_BLOB_SIDECAR,
		}),
		GenericTable(
			TableBeaconApiEthV1ProposerDuty,
			[]xatu.Event_Name{xatu.Event_BEACON_API_ETH_V1_PROPOSER_DUTY},
			WithPredicate(stringFromAdditionalData("head", "state_id")),
		),
		GenericTable(
			TableBeaconApiEthV1BeaconCommittee,
			[]xatu.Event_Name{xatu.Event_BEACON_API_ETH_V1_BEACON_COMMITTEE},
			WithPredicate(stringNotFromAdditionalData("finalized", "state_id")),
		),
		GenericTable(
			TableCanonicalBeaconCommittee,
			[]xatu.Event_Name{xatu.Event_BEACON_API_ETH_V1_BEACON_COMMITTEE},
			WithPredicate(stringFromAdditionalData("finalized", "state_id")),
		),
		GenericTable(
			TableCanonicalBeaconProposerDuty,
			[]xatu.Event_Name{xatu.Event_BEACON_API_ETH_V1_PROPOSER_DUTY},
			WithPredicate(stringFromAdditionalData("finalized", "state_id")),
		),
		GenericTable(TableBeaconApiEthV2BeaconBlock, []xatu.Event_Name{
			xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK,
			xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_V2,
		}),
		GenericTable(
			TableCanonicalBeaconBlock,
			[]xatu.Event_Name{xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_V2},
			WithPredicate(boolFromAdditionalData("finalized_when_requested")),
		),
		GenericTable(TableBeaconApiEthV3ValidatorBlock, []xatu.Event_Name{
			xatu.Event_BEACON_API_ETH_V3_VALIDATOR_BLOCK,
		}),
		ValidatorsFanoutTable(TableCanonicalBeaconValidators, ValidatorsFanoutKindValidators),
		ValidatorsFanoutTable(TableCanonicalBeaconValidatorsPubkeys, ValidatorsFanoutKindPubkeys),
		ValidatorsFanoutTable(
			TableCanonicalBeaconValidatorsWithdrawalCredentials,
			ValidatorsFanoutKindWithdrawalCredential,
		),
		GenericTable(TableCanonicalBeaconBlockAttesterSlashing, []xatu.Event_Name{
			xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_ATTESTER_SLASHING,
		}),
		GenericTable(
			TableCanonicalBeaconElaboratedAttestation,
			[]xatu.Event_Name{xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_ELABORATED_ATTESTATION},
			WithAliases(map[string]string{"validators": "validator_indexes"}),
		),
		GenericTable(TableCanonicalBeaconBlockBlsToExecutionChange, []xatu.Event_Name{
			xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_BLS_TO_EXECUTION_CHANGE,
		}),
		GenericTable(TableCanonicalBeaconBlockDeposit, []xatu.Event_Name{
			xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_DEPOSIT,
		}),
		GenericTable(TableCanonicalBeaconBlockExecutionTransaction, []xatu.Event_Name{
			xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_EXECUTION_TRANSACTION,
		}),
		GenericTable(TableCanonicalBeaconBlockProposerSlashing, []xatu.Event_Name{
			xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_PROPOSER_SLASHING,
		}),
		GenericTable(TableCanonicalBeaconBlockVoluntaryExit, []xatu.Event_Name{
			xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_VOLUNTARY_EXIT,
		}),
		GenericTable(TableCanonicalBeaconBlockWithdrawal, []xatu.Event_Name{
			xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_WITHDRAWAL,
		}),
		GenericTable(
			TableCanonicalBeaconSyncCommittee,
			[]xatu.Event_Name{xatu.Event_BEACON_API_ETH_V1_BEACON_SYNC_COMMITTEE},
			WithMutator(syncCommitteeMutator),
		),
		GenericTable(
			TableCanonicalBeaconBlockSyncAggregate,
			[]xatu.Event_Name{xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_SYNC_AGGREGATE},
			WithMutator(syncAggregateMutator),
		),
	}
}
