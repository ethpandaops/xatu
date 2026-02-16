package flattener

import "github.com/ethpandaops/xatu/pkg/proto/xatu"

func libp2pRoutes() []TableDefinition {
	return []TableDefinition{
		GenericTable(
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
		GenericTable(TableLibp2pConnected, []xatu.Event_Name{xatu.Event_LIBP2P_TRACE_CONNECTED}),
		GenericTable(TableLibp2pDisconnected, []xatu.Event_Name{xatu.Event_LIBP2P_TRACE_DISCONNECTED}),
		GenericTable(TableLibp2pAddPeer, []xatu.Event_Name{xatu.Event_LIBP2P_TRACE_ADD_PEER}),
		GenericTable(TableLibp2pRemovePeer, []xatu.Event_Name{xatu.Event_LIBP2P_TRACE_REMOVE_PEER}),
		GenericTable(TableLibp2pRecvRpc, []xatu.Event_Name{xatu.Event_LIBP2P_TRACE_RECV_RPC}),
		GenericTable(TableLibp2pSendRpc, []xatu.Event_Name{xatu.Event_LIBP2P_TRACE_SEND_RPC}),
		GenericTable(TableLibp2pDropRpc, []xatu.Event_Name{xatu.Event_LIBP2P_TRACE_DROP_RPC}),
		GenericTable(TableLibp2pJoin, []xatu.Event_Name{xatu.Event_LIBP2P_TRACE_JOIN}),
		GenericTable(TableLibp2pLeave, []xatu.Event_Name{xatu.Event_LIBP2P_TRACE_LEAVE}),
		GenericTable(TableLibp2pGraft, []xatu.Event_Name{xatu.Event_LIBP2P_TRACE_GRAFT}),
		GenericTable(TableLibp2pPrune, []xatu.Event_Name{xatu.Event_LIBP2P_TRACE_PRUNE}),
		GenericTable(TableLibp2pPublishMessage, []xatu.Event_Name{xatu.Event_LIBP2P_TRACE_PUBLISH_MESSAGE}),
		GenericTable(TableLibp2pRejectMessage, []xatu.Event_Name{xatu.Event_LIBP2P_TRACE_REJECT_MESSAGE}),
		GenericTable(TableLibp2pDuplicateMessage, []xatu.Event_Name{xatu.Event_LIBP2P_TRACE_DUPLICATE_MESSAGE}),
		GenericTable(TableLibp2pDeliverMessage, []xatu.Event_Name{xatu.Event_LIBP2P_TRACE_DELIVER_MESSAGE}),
		GenericTable(TableLibp2pHandleMetadata, []xatu.Event_Name{xatu.Event_LIBP2P_TRACE_HANDLE_METADATA}),
		GenericTable(TableLibp2pHandleStatus, []xatu.Event_Name{xatu.Event_LIBP2P_TRACE_HANDLE_STATUS}),
		GenericTable(TableLibp2pGossipsubBeaconBlock, []xatu.Event_Name{xatu.Event_LIBP2P_TRACE_GOSSIPSUB_BEACON_BLOCK}),
		GenericTable(
			TableLibp2pGossipsubBeaconAttestation,
			[]xatu.Event_Name{xatu.Event_LIBP2P_TRACE_GOSSIPSUB_BEACON_ATTESTATION},
		),
		GenericTable(
			TableLibp2pGossipsubAggregateAndProof,
			[]xatu.Event_Name{xatu.Event_LIBP2P_TRACE_GOSSIPSUB_AGGREGATE_AND_PROOF},
		),
		GenericTable(TableLibp2pGossipsubBlobSidecar, []xatu.Event_Name{xatu.Event_LIBP2P_TRACE_GOSSIPSUB_BLOB_SIDECAR}),
		GenericTable(
			TableLibp2pGossipsubDataColumnSidecar,
			[]xatu.Event_Name{xatu.Event_LIBP2P_TRACE_GOSSIPSUB_DATA_COLUMN_SIDECAR},
		),
		GenericTable(
			TableLibp2pRpcDataColumnCustodyProbe,
			[]xatu.Event_Name{xatu.Event_LIBP2P_TRACE_RPC_DATA_COLUMN_CUSTODY_PROBE},
		),
		GenericTable(TableLibp2pSyntheticHeartbeat, []xatu.Event_Name{xatu.Event_LIBP2P_TRACE_SYNTHETIC_HEARTBEAT}),
		GenericTable(
			TableLibp2pRpcMetaControlIhave,
			[]xatu.Event_Name{xatu.Event_LIBP2P_TRACE_RPC_META_CONTROL_IHAVE},
		),
		GenericTable(
			TableLibp2pRpcMetaControlIwant,
			[]xatu.Event_Name{xatu.Event_LIBP2P_TRACE_RPC_META_CONTROL_IWANT},
		),
		GenericTable(
			TableLibp2pRpcMetaControlIdontwant,
			[]xatu.Event_Name{xatu.Event_LIBP2P_TRACE_RPC_META_CONTROL_IDONTWANT},
		),
		GenericTable(
			TableLibp2pRpcMetaControlGraft,
			[]xatu.Event_Name{xatu.Event_LIBP2P_TRACE_RPC_META_CONTROL_GRAFT},
		),
		GenericTable(
			TableLibp2pRpcMetaControlPrune,
			[]xatu.Event_Name{xatu.Event_LIBP2P_TRACE_RPC_META_CONTROL_PRUNE},
		),
		GenericTable(
			TableLibp2pRpcMetaSubscription,
			[]xatu.Event_Name{xatu.Event_LIBP2P_TRACE_RPC_META_SUBSCRIPTION},
		),
		GenericTable(
			TableLibp2pRpcMetaMessage,
			[]xatu.Event_Name{xatu.Event_LIBP2P_TRACE_RPC_META_MESSAGE},
		),
	}
}
