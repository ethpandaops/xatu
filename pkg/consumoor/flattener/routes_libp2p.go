package flattener

import "github.com/ethpandaops/xatu/pkg/proto/xatu"

func libp2pRoutes() []Route {
	return []Route{
		RouteTo(
			TableLibp2pPeer,
			xatu.Event_LIBP2P_TRACE_CONNECTED,
			xatu.Event_LIBP2P_TRACE_DISCONNECTED,
			xatu.Event_LIBP2P_TRACE_ADD_PEER,
			xatu.Event_LIBP2P_TRACE_REMOVE_PEER,
			xatu.Event_LIBP2P_TRACE_RECV_RPC,
			xatu.Event_LIBP2P_TRACE_SEND_RPC,
			xatu.Event_LIBP2P_TRACE_DROP_RPC,
			xatu.Event_LIBP2P_TRACE_GRAFT,
			xatu.Event_LIBP2P_TRACE_PRUNE,
		).
			CommonMetadata().
			RuntimeColumns().
			EventData().
			ClientAdditionalData().
			ServerAdditionalData().
			TableAliases().
			RouteAliases().
			NormalizeDateTimes().
			CommonEnrichment().
			Mutator(peerConvergenceMutator).
			Build(),
		RouteTo(TableLibp2pConnected, xatu.Event_LIBP2P_TRACE_CONNECTED).
			CommonMetadata().
			RuntimeColumns().
			EventData().
			ClientAdditionalData().
			ServerAdditionalData().
			TableAliases().
			RouteAliases().
			NormalizeDateTimes().
			CommonEnrichment().
			Build(),
		RouteTo(TableLibp2pDisconnected, xatu.Event_LIBP2P_TRACE_DISCONNECTED).
			CommonMetadata().
			RuntimeColumns().
			EventData().
			ClientAdditionalData().
			ServerAdditionalData().
			TableAliases().
			RouteAliases().
			NormalizeDateTimes().
			CommonEnrichment().
			Build(),
		RouteTo(TableLibp2pAddPeer, xatu.Event_LIBP2P_TRACE_ADD_PEER).
			CommonMetadata().
			RuntimeColumns().
			EventData().
			ClientAdditionalData().
			ServerAdditionalData().
			TableAliases().
			RouteAliases().
			NormalizeDateTimes().
			CommonEnrichment().
			Build(),
		RouteTo(TableLibp2pRemovePeer, xatu.Event_LIBP2P_TRACE_REMOVE_PEER).
			CommonMetadata().
			RuntimeColumns().
			EventData().
			ClientAdditionalData().
			ServerAdditionalData().
			TableAliases().
			RouteAliases().
			NormalizeDateTimes().
			CommonEnrichment().
			Build(),
		RouteTo(TableLibp2pRecvRpc, xatu.Event_LIBP2P_TRACE_RECV_RPC).
			CommonMetadata().
			RuntimeColumns().
			EventData().
			ClientAdditionalData().
			ServerAdditionalData().
			TableAliases().
			RouteAliases().
			NormalizeDateTimes().
			CommonEnrichment().
			Build(),
		RouteTo(TableLibp2pSendRpc, xatu.Event_LIBP2P_TRACE_SEND_RPC).
			CommonMetadata().
			RuntimeColumns().
			EventData().
			ClientAdditionalData().
			ServerAdditionalData().
			TableAliases().
			RouteAliases().
			NormalizeDateTimes().
			CommonEnrichment().
			Build(),
		RouteTo(TableLibp2pDropRpc, xatu.Event_LIBP2P_TRACE_DROP_RPC).
			CommonMetadata().
			RuntimeColumns().
			EventData().
			ClientAdditionalData().
			ServerAdditionalData().
			TableAliases().
			RouteAliases().
			NormalizeDateTimes().
			CommonEnrichment().
			Build(),
		RouteTo(TableLibp2pJoin, xatu.Event_LIBP2P_TRACE_JOIN).
			CommonMetadata().
			RuntimeColumns().
			EventData().
			ClientAdditionalData().
			ServerAdditionalData().
			TableAliases().
			RouteAliases().
			NormalizeDateTimes().
			CommonEnrichment().
			Build(),
		RouteTo(TableLibp2pLeave, xatu.Event_LIBP2P_TRACE_LEAVE).
			CommonMetadata().
			RuntimeColumns().
			EventData().
			ClientAdditionalData().
			ServerAdditionalData().
			TableAliases().
			RouteAliases().
			NormalizeDateTimes().
			CommonEnrichment().
			Build(),
		RouteTo(TableLibp2pGraft, xatu.Event_LIBP2P_TRACE_GRAFT).
			CommonMetadata().
			RuntimeColumns().
			EventData().
			ClientAdditionalData().
			ServerAdditionalData().
			TableAliases().
			RouteAliases().
			NormalizeDateTimes().
			CommonEnrichment().
			Build(),
		RouteTo(TableLibp2pPrune, xatu.Event_LIBP2P_TRACE_PRUNE).
			CommonMetadata().
			RuntimeColumns().
			EventData().
			ClientAdditionalData().
			ServerAdditionalData().
			TableAliases().
			RouteAliases().
			NormalizeDateTimes().
			CommonEnrichment().
			Build(),
		RouteTo(TableLibp2pPublishMessage, xatu.Event_LIBP2P_TRACE_PUBLISH_MESSAGE).
			CommonMetadata().
			RuntimeColumns().
			EventData().
			ClientAdditionalData().
			ServerAdditionalData().
			TableAliases().
			RouteAliases().
			NormalizeDateTimes().
			CommonEnrichment().
			Build(),
		RouteTo(TableLibp2pRejectMessage, xatu.Event_LIBP2P_TRACE_REJECT_MESSAGE).
			CommonMetadata().
			RuntimeColumns().
			EventData().
			ClientAdditionalData().
			ServerAdditionalData().
			TableAliases().
			RouteAliases().
			NormalizeDateTimes().
			CommonEnrichment().
			Build(),
		RouteTo(TableLibp2pDuplicateMessage, xatu.Event_LIBP2P_TRACE_DUPLICATE_MESSAGE).
			CommonMetadata().
			RuntimeColumns().
			EventData().
			ClientAdditionalData().
			ServerAdditionalData().
			TableAliases().
			RouteAliases().
			NormalizeDateTimes().
			CommonEnrichment().
			Build(),
		RouteTo(TableLibp2pDeliverMessage, xatu.Event_LIBP2P_TRACE_DELIVER_MESSAGE).
			CommonMetadata().
			RuntimeColumns().
			EventData().
			ClientAdditionalData().
			ServerAdditionalData().
			TableAliases().
			RouteAliases().
			NormalizeDateTimes().
			CommonEnrichment().
			Build(),
		RouteTo(TableLibp2pHandleMetadata, xatu.Event_LIBP2P_TRACE_HANDLE_METADATA).
			CommonMetadata().
			RuntimeColumns().
			EventData().
			ClientAdditionalData().
			ServerAdditionalData().
			TableAliases().
			RouteAliases().
			NormalizeDateTimes().
			CommonEnrichment().
			Build(),
		RouteTo(TableLibp2pHandleStatus, xatu.Event_LIBP2P_TRACE_HANDLE_STATUS).
			CommonMetadata().
			RuntimeColumns().
			EventData().
			ClientAdditionalData().
			ServerAdditionalData().
			TableAliases().
			RouteAliases().
			NormalizeDateTimes().
			CommonEnrichment().
			Build(),
		RouteTo(TableLibp2pGossipsubBeaconBlock, xatu.Event_LIBP2P_TRACE_GOSSIPSUB_BEACON_BLOCK).
			CommonMetadata().
			RuntimeColumns().
			EventData().
			ClientAdditionalData().
			ServerAdditionalData().
			TableAliases().
			RouteAliases().
			NormalizeDateTimes().
			CommonEnrichment().
			Build(),
		RouteTo(TableLibp2pGossipsubBeaconAttestation, xatu.Event_LIBP2P_TRACE_GOSSIPSUB_BEACON_ATTESTATION).
			CommonMetadata().
			RuntimeColumns().
			EventData().
			ClientAdditionalData().
			ServerAdditionalData().
			TableAliases().
			RouteAliases().
			NormalizeDateTimes().
			CommonEnrichment().
			Build(),
		RouteTo(TableLibp2pGossipsubAggregateAndProof, xatu.Event_LIBP2P_TRACE_GOSSIPSUB_AGGREGATE_AND_PROOF).
			CommonMetadata().
			RuntimeColumns().
			EventData().
			ClientAdditionalData().
			ServerAdditionalData().
			TableAliases().
			RouteAliases().
			NormalizeDateTimes().
			CommonEnrichment().
			Build(),
		RouteTo(TableLibp2pGossipsubBlobSidecar, xatu.Event_LIBP2P_TRACE_GOSSIPSUB_BLOB_SIDECAR).
			CommonMetadata().
			RuntimeColumns().
			EventData().
			ClientAdditionalData().
			ServerAdditionalData().
			TableAliases().
			RouteAliases().
			NormalizeDateTimes().
			CommonEnrichment().
			Build(),
		RouteTo(TableLibp2pGossipsubDataColumnSidecar, xatu.Event_LIBP2P_TRACE_GOSSIPSUB_DATA_COLUMN_SIDECAR).
			CommonMetadata().
			RuntimeColumns().
			EventData().
			ClientAdditionalData().
			ServerAdditionalData().
			TableAliases().
			RouteAliases().
			NormalizeDateTimes().
			CommonEnrichment().
			Build(),
		RouteTo(TableLibp2pRpcDataColumnCustodyProbe, xatu.Event_LIBP2P_TRACE_RPC_DATA_COLUMN_CUSTODY_PROBE).
			CommonMetadata().
			RuntimeColumns().
			EventData().
			ClientAdditionalData().
			ServerAdditionalData().
			TableAliases().
			RouteAliases().
			NormalizeDateTimes().
			CommonEnrichment().
			Build(),
		RouteTo(TableLibp2pSyntheticHeartbeat, xatu.Event_LIBP2P_TRACE_SYNTHETIC_HEARTBEAT).
			CommonMetadata().
			RuntimeColumns().
			EventData().
			ClientAdditionalData().
			ServerAdditionalData().
			TableAliases().
			RouteAliases().
			NormalizeDateTimes().
			CommonEnrichment().
			Build(),
		RouteTo(TableLibp2pRpcMetaControlIhave, xatu.Event_LIBP2P_TRACE_RPC_META_CONTROL_IHAVE).
			CommonMetadata().
			RuntimeColumns().
			EventData().
			ClientAdditionalData().
			ServerAdditionalData().
			TableAliases().
			RouteAliases().
			NormalizeDateTimes().
			CommonEnrichment().
			Build(),
		RouteTo(TableLibp2pRpcMetaControlIwant, xatu.Event_LIBP2P_TRACE_RPC_META_CONTROL_IWANT).
			CommonMetadata().
			RuntimeColumns().
			EventData().
			ClientAdditionalData().
			ServerAdditionalData().
			TableAliases().
			RouteAliases().
			NormalizeDateTimes().
			CommonEnrichment().
			Build(),
		RouteTo(TableLibp2pRpcMetaControlIdontwant, xatu.Event_LIBP2P_TRACE_RPC_META_CONTROL_IDONTWANT).
			CommonMetadata().
			RuntimeColumns().
			EventData().
			ClientAdditionalData().
			ServerAdditionalData().
			TableAliases().
			RouteAliases().
			NormalizeDateTimes().
			CommonEnrichment().
			Build(),
		RouteTo(TableLibp2pRpcMetaControlGraft, xatu.Event_LIBP2P_TRACE_RPC_META_CONTROL_GRAFT).
			CommonMetadata().
			RuntimeColumns().
			EventData().
			ClientAdditionalData().
			ServerAdditionalData().
			TableAliases().
			RouteAliases().
			NormalizeDateTimes().
			CommonEnrichment().
			Build(),
		RouteTo(TableLibp2pRpcMetaControlPrune, xatu.Event_LIBP2P_TRACE_RPC_META_CONTROL_PRUNE).
			CommonMetadata().
			RuntimeColumns().
			EventData().
			ClientAdditionalData().
			ServerAdditionalData().
			TableAliases().
			RouteAliases().
			NormalizeDateTimes().
			CommonEnrichment().
			Build(),
		RouteTo(TableLibp2pRpcMetaSubscription, xatu.Event_LIBP2P_TRACE_RPC_META_SUBSCRIPTION).
			CommonMetadata().
			RuntimeColumns().
			EventData().
			ClientAdditionalData().
			ServerAdditionalData().
			TableAliases().
			RouteAliases().
			NormalizeDateTimes().
			CommonEnrichment().
			Build(),
		RouteTo(TableLibp2pRpcMetaMessage, xatu.Event_LIBP2P_TRACE_RPC_META_MESSAGE).
			CommonMetadata().
			RuntimeColumns().
			EventData().
			ClientAdditionalData().
			ServerAdditionalData().
			TableAliases().
			RouteAliases().
			NormalizeDateTimes().
			CommonEnrichment().
			Build(),
	}
}
