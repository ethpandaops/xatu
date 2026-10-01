-- Drop cannon Gloas (EIP-7928) per-block block access list summary tables.

DROP TABLE IF EXISTS canonical_beacon_block_access_list_summary ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS canonical_beacon_block_access_list_summary_local ON CLUSTER '{cluster}' SYNC;
