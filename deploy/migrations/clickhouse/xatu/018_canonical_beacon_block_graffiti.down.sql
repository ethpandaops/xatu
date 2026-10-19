-- Drop cannon per-block graffiti tables.

DROP TABLE IF EXISTS canonical_beacon_block_graffiti ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS canonical_beacon_block_graffiti_local ON CLUSTER '{cluster}' SYNC;
