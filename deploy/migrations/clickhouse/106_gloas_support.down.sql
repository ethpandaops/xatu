-- Drop canonical_beacon_block_access_list tables
DROP TABLE IF EXISTS default.canonical_beacon_block_access_list ON CLUSTER '{cluster}';
DROP TABLE IF EXISTS default.canonical_beacon_block_access_list_local ON CLUSTER '{cluster}';

-- Remove columns from canonical_beacon_block
ALTER TABLE default.canonical_beacon_block ON CLUSTER '{cluster}'
    DROP COLUMN IF EXISTS execution_payload_block_access_list_root,
    DROP COLUMN IF EXISTS execution_payload_slot_number;

ALTER TABLE default.canonical_beacon_block_local ON CLUSTER '{cluster}'
    DROP COLUMN IF EXISTS execution_payload_block_access_list_root,
    DROP COLUMN IF EXISTS execution_payload_slot_number;

-- Remove columns from beacon_api_eth_v2_beacon_block
ALTER TABLE default.beacon_api_eth_v2_beacon_block ON CLUSTER '{cluster}'
    DROP COLUMN IF EXISTS execution_payload_slot_number;

ALTER TABLE default.beacon_api_eth_v2_beacon_block_local ON CLUSTER '{cluster}'
    DROP COLUMN IF EXISTS execution_payload_slot_number;
