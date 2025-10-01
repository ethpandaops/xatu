-- Drop distributed tables
DROP TABLE IF EXISTS libp2p_handle_status ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS libp2p_handle_metadata ON CLUSTER '{cluster}' SYNC;

-- Add Direction column to libp2p_handle_status_local
ALTER TABLE libp2p_handle_status_local ON CLUSTER '{cluster}'
ADD COLUMN IF NOT EXISTS direction LowCardinality(Nullable(String)) COMMENT 'Direction of the RPC request (inbound or outbound)' CODEC(ZSTD(1))
AFTER protocol;

-- Add request_earliest_available_slot column to libp2p_handle_status_local
ALTER TABLE libp2p_handle_status_local ON CLUSTER '{cluster}'
ADD COLUMN IF NOT EXISTS request_earliest_available_slot Nullable(UInt32) COMMENT 'Requested earliest available slot' CODEC(ZSTD(1))
AFTER request_head_slot;

-- Add response_earliest_available_slot column to libp2p_handle_status_local
ALTER TABLE libp2p_handle_status_local ON CLUSTER '{cluster}'
ADD COLUMN IF NOT EXISTS response_earliest_available_slot Nullable(UInt32) COMMENT 'Response earliest available slot' CODEC(ZSTD(1))
AFTER response_head_slot;

-- Add Direction column to libp2p_handle_metadata_local
ALTER TABLE libp2p_handle_metadata_local ON CLUSTER '{cluster}'
ADD COLUMN IF NOT EXISTS direction LowCardinality(Nullable(String)) COMMENT 'Direction of the RPC request (inbound or outbound)' CODEC(ZSTD(1))
AFTER protocol;

-- Add custody_group_count column to libp2p_handle_metadata_local
ALTER TABLE libp2p_handle_metadata_local ON CLUSTER '{cluster}'
ADD COLUMN IF NOT EXISTS custody_group_count Nullable(UInt8) COMMENT 'Number of custody groups (0-127)' CODEC(ZSTD(1))
AFTER syncnets;

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