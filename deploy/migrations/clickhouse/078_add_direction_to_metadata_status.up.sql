-- Drop distributed tables
DROP TABLE IF EXISTS libp2p_handle_status ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS libp2p_handle_metadata ON CLUSTER '{cluster}' SYNC;

-- Add Direction column to libp2p_handle_status_local
ALTER TABLE libp2p_handle_status_local ON CLUSTER '{cluster}'
ADD COLUMN IF NOT EXISTS direction LowCardinality(Nullable(String)) COMMENT 'Direction of the RPC request (inbound or outbound)' CODEC(ZSTD(1))
AFTER protocol;

-- Add Direction column to libp2p_handle_metadata_local
ALTER TABLE libp2p_handle_metadata_local ON CLUSTER '{cluster}'
ADD COLUMN IF NOT EXISTS direction LowCardinality(Nullable(String)) COMMENT 'Direction of the RPC request (inbound or outbound)' CODEC(ZSTD(1))
AFTER protocol;```

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