-- Remove Direction column from distributed tables first
DROP TABLE IF EXISTS libp2p_handle_status ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS libp2p_handle_metadata ON CLUSTER '{cluster}' SYNC;

-- Remove Direction column from local tables
ALTER TABLE libp2p_handle_status_local ON CLUSTER '{cluster}'
DROP COLUMN IF EXISTS direction;

ALTER TABLE libp2p_handle_metadata_local ON CLUSTER '{cluster}'
DROP COLUMN IF EXISTS direction;

-- Recreate distributed tables
CREATE TABLE libp2p_handle_status ON CLUSTER '{cluster}' AS libp2p_handle_status_local
ENGINE = Distributed(
    '{cluster}',
    default,
    libp2p_handle_status_local,
    cityHash64(
        event_date_time,
        meta_network_name,
        meta_client_name,
        peer_id_unique_key,
        latency_milliseconds
    )
);

CREATE TABLE libp2p_handle_metadata ON CLUSTER '{cluster}' AS libp2p_handle_metadata_local
ENGINE = Distributed(
    '{cluster}',
    default,
    libp2p_handle_metadata_local,
    cityHash64(
        event_date_time,
        meta_network_name,
        meta_client_name,
        peer_id_unique_key,
        latency_milliseconds
    )
);