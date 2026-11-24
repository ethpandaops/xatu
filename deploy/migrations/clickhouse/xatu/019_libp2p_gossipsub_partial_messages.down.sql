ALTER TABLE libp2p_rpc_meta_subscription ON CLUSTER '{cluster}'
    DROP COLUMN IF EXISTS requests_partial,
    DROP COLUMN IF EXISTS supports_sending_partial;

ALTER TABLE libp2p_rpc_meta_subscription_local ON CLUSTER '{cluster}'
    DROP COLUMN IF EXISTS requests_partial,
    DROP COLUMN IF EXISTS supports_sending_partial;

DROP TABLE IF EXISTS libp2p_gossipsub_partial_data_column_outcome ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS libp2p_gossipsub_partial_data_column_outcome_local ON CLUSTER '{cluster}' SYNC;

DROP TABLE IF EXISTS libp2p_data_column_completed ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS libp2p_data_column_completed_local ON CLUSTER '{cluster}' SYNC;

DROP TABLE IF EXISTS libp2p_gossipsub_partial_data_column ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS libp2p_gossipsub_partial_data_column_local ON CLUSTER '{cluster}' SYNC;
