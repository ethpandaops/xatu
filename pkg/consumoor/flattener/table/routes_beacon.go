package table

import "github.com/ethpandaops/xatu/pkg/consumoor/flattener"

func beaconRoutes() []flattener.Route {
	return []flattener.Route{
		BeaconApiEthV1EventsHeadRoute{}.Build(),
		BeaconApiEthV1EventsBlockRoute{}.Build(),
		BeaconApiEthV1EventsBlockGossipRoute{}.Build(),
		BeaconApiEthV1EventsFinalizedCheckpointRoute{}.Build(),
		BeaconApiEthV1EventsChainReorgRoute{}.Build(),
		BeaconApiEthV1BeaconBlobRoute{}.Build(),
		BeaconApiEthV1EventsBlobSidecarRoute{}.Build(),
		BeaconApiEthV1EventsDataColumnSidecarRoute{}.Build(),
		BeaconApiEthV1EventsVoluntaryExitRoute{}.Build(),
		BeaconApiEthV1EventsContributionAndProofRoute{}.Build(),
		BeaconApiEthV1EventsAttestationRoute{}.Build(),
		BeaconApiEthV1ValidatorAttestationDataRoute{}.Build(),
		CanonicalBeaconBlobSidecarRoute{}.Build(),
		BeaconApiEthV1ProposerDutyRoute{}.Build(),
		BeaconApiEthV1BeaconCommitteeRoute{}.Build(),
		CanonicalBeaconCommitteeRoute{}.Build(),
		CanonicalBeaconProposerDutyRoute{}.Build(),
		BeaconApiEthV2BeaconBlockRoute{}.Build(),
		CanonicalBeaconBlockRoute{}.Build(),
		BeaconApiEthV3ValidatorBlockRoute{}.Build(),
		CanonicalBeaconValidatorsRoute{}.Build(),
		CanonicalBeaconValidatorsPubkeysRoute{}.Build(),
		CanonicalBeaconValidatorsWithdrawalCredentialsRoute{}.Build(),
		CanonicalBeaconBlockAttesterSlashingRoute{}.Build(),
		CanonicalBeaconElaboratedAttestationRoute{}.Build(),
		CanonicalBeaconBlockBlsToExecutionChangeRoute{}.Build(),
		CanonicalBeaconBlockDepositRoute{}.Build(),
		CanonicalBeaconBlockExecutionTransactionRoute{}.Build(),
		CanonicalBeaconBlockProposerSlashingRoute{}.Build(),
		CanonicalBeaconBlockVoluntaryExitRoute{}.Build(),
		CanonicalBeaconBlockWithdrawalRoute{}.Build(),
		CanonicalBeaconSyncCommitteeRoute{}.Build(),
		CanonicalBeaconBlockSyncAggregateRoute{}.Build(),
	}
}
