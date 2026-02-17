package table

import "github.com/ethpandaops/xatu/pkg/consumoor/flattener"

func libp2pRoutes() []flattener.Route {
	return []flattener.Route{
		Libp2pPeerRoute{}.Build(),
		Libp2pConnectedRoute{}.Build(),
		Libp2pDisconnectedRoute{}.Build(),
		Libp2pAddPeerRoute{}.Build(),
		Libp2pRemovePeerRoute{}.Build(),
		Libp2pRecvRpcRoute{}.Build(),
		Libp2pSendRpcRoute{}.Build(),
		Libp2pDropRpcRoute{}.Build(),
		Libp2pJoinRoute{}.Build(),
		Libp2pLeaveRoute{}.Build(),
		Libp2pGraftRoute{}.Build(),
		Libp2pPruneRoute{}.Build(),
		Libp2pPublishMessageRoute{}.Build(),
		Libp2pRejectMessageRoute{}.Build(),
		Libp2pDuplicateMessageRoute{}.Build(),
		Libp2pDeliverMessageRoute{}.Build(),
		Libp2pHandleMetadataRoute{}.Build(),
		Libp2pHandleStatusRoute{}.Build(),
		Libp2pGossipsubBeaconBlockRoute{}.Build(),
		Libp2pGossipsubBeaconAttestationRoute{}.Build(),
		Libp2pGossipsubAggregateAndProofRoute{}.Build(),
		Libp2pGossipsubBlobSidecarRoute{}.Build(),
		Libp2pGossipsubDataColumnSidecarRoute{}.Build(),
		Libp2pRpcDataColumnCustodyProbeRoute{}.Build(),
		Libp2pSyntheticHeartbeatRoute{}.Build(),
		Libp2pRpcMetaControlIhaveRoute{}.Build(),
		Libp2pRpcMetaControlIwantRoute{}.Build(),
		Libp2pRpcMetaControlIdontwantRoute{}.Build(),
		Libp2pRpcMetaControlGraftRoute{}.Build(),
		Libp2pRpcMetaControlPruneRoute{}.Build(),
		Libp2pRpcMetaSubscriptionRoute{}.Build(),
		Libp2pRpcMetaMessageRoute{}.Build(),
	}
}
