-- Add Direction column to libp2p_handle_status tables
ALTER TABLE libp2p_handle_status_local ON CLUSTER '{cluster}'
ADD COLUMN IF NOT EXISTS direction LowCardinality(Nullable(String)) CODEC(ZSTD(1))
AFTER protocol;

ALTER TABLE libp2p_handle_status_local ON CLUSTER '{cluster}'
COMMENT COLUMN direction 'Direction of the RPC request (inbound or outbound)';

-- Add Direction column to distributed table
ALTER TABLE libp2p_handle_status ON CLUSTER '{cluster}'
ADD COLUMN IF NOT EXISTS direction LowCardinality(Nullable(String))
AFTER protocol;

ALTER TABLE libp2p_handle_status ON CLUSTER '{cluster}'
COMMENT COLUMN direction 'Direction of the RPC request (inbound or outbound)';

-- Add Direction column to libp2p_handle_metadata tables
ALTER TABLE libp2p_handle_metadata_local ON CLUSTER '{cluster}'
ADD COLUMN IF NOT EXISTS direction LowCardinality(Nullable(String)) CODEC(ZSTD(1))
AFTER protocol;

ALTER TABLE libp2p_handle_metadata_local ON CLUSTER '{cluster}'
COMMENT COLUMN direction 'Direction of the RPC request (inbound or outbound)';

-- Add Direction column to distributed table
ALTER TABLE libp2p_handle_metadata ON CLUSTER '{cluster}'
ADD COLUMN IF NOT EXISTS direction LowCardinality(Nullable(String))
AFTER protocol;

ALTER TABLE libp2p_handle_metadata ON CLUSTER '{cluster}'
COMMENT COLUMN direction 'Direction of the RPC request (inbound or outbound)';