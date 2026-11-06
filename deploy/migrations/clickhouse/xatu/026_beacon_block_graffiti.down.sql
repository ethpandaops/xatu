ALTER TABLE beacon_api_eth_v2_beacon_block_local ON CLUSTER '{cluster}'
    DROP COLUMN IF EXISTS graffiti_text,
    DROP COLUMN IF EXISTS graffiti;

ALTER TABLE beacon_api_eth_v2_beacon_block ON CLUSTER '{cluster}'
    DROP COLUMN IF EXISTS graffiti_text,
    DROP COLUMN IF EXISTS graffiti;

