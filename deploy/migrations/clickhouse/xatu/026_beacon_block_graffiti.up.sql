-- Record the graffiti of every beacon block the sentries fetch, so the newest, unfinalized blocks have it before
-- cannon writes canonical_beacon_block_graffiti at finality.

ALTER TABLE beacon_api_eth_v2_beacon_block_local ON CLUSTER '{cluster}'
    ADD COLUMN IF NOT EXISTS graffiti Nullable(FixedString(66)) COMMENT 'The raw 32 byte graffiti from the beacon block body (hex encoded), NULL for rows written before it was recorded' CODEC(ZSTD(1)) AFTER eth1_data_deposit_root,
    ADD COLUMN IF NOT EXISTS graffiti_text Nullable(String) COMMENT 'The graffiti decoded as UTF-8 with trailing null bytes removed and invalid sequences replaced, NULL for rows written before it was recorded' CODEC(ZSTD(1)) AFTER graffiti;

ALTER TABLE beacon_api_eth_v2_beacon_block ON CLUSTER '{cluster}'
    ADD COLUMN IF NOT EXISTS graffiti Nullable(FixedString(66)) COMMENT 'The raw 32 byte graffiti from the beacon block body (hex encoded), NULL for rows written before it was recorded' CODEC(ZSTD(1)) AFTER eth1_data_deposit_root,
    ADD COLUMN IF NOT EXISTS graffiti_text Nullable(String) COMMENT 'The graffiti decoded as UTF-8 with trailing null bytes removed and invalid sequences replaced, NULL for rows written before it was recorded' CODEC(ZSTD(1)) AFTER graffiti;

