ALTER TABLE default.libp2p_rpc_meta_control_prune ON CLUSTER '{cluster}'
    MODIFY COLUMN `graft_peer_id_unique_key` Nullable(Int64) COMMENT 'Unique key associated with the identifier of the graft peer involved in the Prune control';

ALTER TABLE default.libp2p_rpc_meta_control_prune_local ON CLUSTER '{cluster}'
    MODIFY COLUMN `graft_peer_id_unique_key` Nullable(Int64) COMMENT 'Unique key associated with the identifier of the graft peer involved in the Prune control';

