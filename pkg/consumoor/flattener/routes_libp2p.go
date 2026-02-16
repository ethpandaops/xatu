package flattener

import "github.com/ethpandaops/xatu/pkg/proto/xatu"

func libp2pRoutes() []Flattener {
	return []Flattener{
		NewGenericRoute(
			TableLibp2pPeer,
			[]xatu.Event_Name{
				xatu.Event_LIBP2P_TRACE_CONNECTED,
				xatu.Event_LIBP2P_TRACE_DISCONNECTED,
				xatu.Event_LIBP2P_TRACE_ADD_PEER,
				xatu.Event_LIBP2P_TRACE_REMOVE_PEER,
				xatu.Event_LIBP2P_TRACE_RECV_RPC,
				xatu.Event_LIBP2P_TRACE_SEND_RPC,
				xatu.Event_LIBP2P_TRACE_DROP_RPC,
				xatu.Event_LIBP2P_TRACE_GRAFT,
				xatu.Event_LIBP2P_TRACE_PRUNE,
			},
			WithMutator(peerConvergenceMutator),
		),
		NewGenericRoute(TableLibp2pConnected, []xatu.Event_Name{xatu.Event_LIBP2P_TRACE_CONNECTED}),
		NewGenericRoute(TableLibp2pDisconnected, []xatu.Event_Name{xatu.Event_LIBP2P_TRACE_DISCONNECTED}),
		NewGenericRoute(TableLibp2pAddPeer, []xatu.Event_Name{xatu.Event_LIBP2P_TRACE_ADD_PEER}),
		NewGenericRoute(TableLibp2pRemovePeer, []xatu.Event_Name{xatu.Event_LIBP2P_TRACE_REMOVE_PEER}),
		NewGenericRoute(TableLibp2pRecvRpc, []xatu.Event_Name{xatu.Event_LIBP2P_TRACE_RECV_RPC}),
		NewGenericRoute(TableLibp2pSendRpc, []xatu.Event_Name{xatu.Event_LIBP2P_TRACE_SEND_RPC}),
		NewGenericRoute(TableLibp2pDropRpc, []xatu.Event_Name{xatu.Event_LIBP2P_TRACE_DROP_RPC}),
		NewGenericRoute(TableLibp2pJoin, []xatu.Event_Name{xatu.Event_LIBP2P_TRACE_JOIN}),
		NewGenericRoute(TableLibp2pLeave, []xatu.Event_Name{xatu.Event_LIBP2P_TRACE_LEAVE}),
		NewGenericRoute(TableLibp2pGraft, []xatu.Event_Name{xatu.Event_LIBP2P_TRACE_GRAFT}),
		NewGenericRoute(TableLibp2pPrune, []xatu.Event_Name{xatu.Event_LIBP2P_TRACE_PRUNE}),
		NewGenericRoute(TableLibp2pPublishMessage, []xatu.Event_Name{xatu.Event_LIBP2P_TRACE_PUBLISH_MESSAGE}),
		NewGenericRoute(TableLibp2pRejectMessage, []xatu.Event_Name{xatu.Event_LIBP2P_TRACE_REJECT_MESSAGE}),
		NewGenericRoute(TableLibp2pDuplicateMessage, []xatu.Event_Name{xatu.Event_LIBP2P_TRACE_DUPLICATE_MESSAGE}),
		NewGenericRoute(TableLibp2pDeliverMessage, []xatu.Event_Name{xatu.Event_LIBP2P_TRACE_DELIVER_MESSAGE}),
		NewGenericRoute(TableLibp2pHandleMetadata, []xatu.Event_Name{xatu.Event_LIBP2P_TRACE_HANDLE_METADATA}),
		NewGenericRoute(TableLibp2pHandleStatus, []xatu.Event_Name{xatu.Event_LIBP2P_TRACE_HANDLE_STATUS}),
		NewGenericRoute(TableLibp2pGossipsubBeaconBlock, []xatu.Event_Name{xatu.Event_LIBP2P_TRACE_GOSSIPSUB_BEACON_BLOCK}),
		NewGenericRoute(
			TableLibp2pGossipsubBeaconAttestation,
			[]xatu.Event_Name{xatu.Event_LIBP2P_TRACE_GOSSIPSUB_BEACON_ATTESTATION},
		),
		NewGenericRoute(
			TableLibp2pGossipsubAggregateAndProof,
			[]xatu.Event_Name{xatu.Event_LIBP2P_TRACE_GOSSIPSUB_AGGREGATE_AND_PROOF},
		),
		NewGenericRoute(TableLibp2pGossipsubBlobSidecar, []xatu.Event_Name{xatu.Event_LIBP2P_TRACE_GOSSIPSUB_BLOB_SIDECAR}),
		NewGenericRoute(
			TableLibp2pGossipsubDataColumnSidecar,
			[]xatu.Event_Name{xatu.Event_LIBP2P_TRACE_GOSSIPSUB_DATA_COLUMN_SIDECAR},
		),
		NewGenericRoute(
			TableLibp2pRpcDataColumnCustodyProbe,
			[]xatu.Event_Name{xatu.Event_LIBP2P_TRACE_RPC_DATA_COLUMN_CUSTODY_PROBE},
		),
		NewGenericRoute(TableLibp2pSyntheticHeartbeat, []xatu.Event_Name{xatu.Event_LIBP2P_TRACE_SYNTHETIC_HEARTBEAT}),
		NewGenericRoute(
			TableLibp2pRpcMetaControlIhave,
			[]xatu.Event_Name{xatu.Event_LIBP2P_TRACE_RPC_META_CONTROL_IHAVE},
		),
		NewGenericRoute(
			TableLibp2pRpcMetaControlIwant,
			[]xatu.Event_Name{xatu.Event_LIBP2P_TRACE_RPC_META_CONTROL_IWANT},
		),
		NewGenericRoute(
			TableLibp2pRpcMetaControlIdontwant,
			[]xatu.Event_Name{xatu.Event_LIBP2P_TRACE_RPC_META_CONTROL_IDONTWANT},
		),
		NewGenericRoute(
			TableLibp2pRpcMetaControlGraft,
			[]xatu.Event_Name{xatu.Event_LIBP2P_TRACE_RPC_META_CONTROL_GRAFT},
		),
		NewGenericRoute(
			TableLibp2pRpcMetaControlPrune,
			[]xatu.Event_Name{xatu.Event_LIBP2P_TRACE_RPC_META_CONTROL_PRUNE},
		),
		NewGenericRoute(
			TableLibp2pRpcMetaSubscription,
			[]xatu.Event_Name{xatu.Event_LIBP2P_TRACE_RPC_META_SUBSCRIPTION},
		),
		NewGenericRoute(
			TableLibp2pRpcMetaMessage,
			[]xatu.Event_Name{xatu.Event_LIBP2P_TRACE_RPC_META_MESSAGE},
		),
	}
}
