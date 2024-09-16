-- Drop the distributed table
DROP TABLE IF EXISTS default.mev_relay_bid_trace ON CLUSTER '{cluster}' SYNC;

-- Drop the local table
DROP TABLE IF EXISTS default.mev_relay_bid_trace_local ON CLUSTER '{cluster}' SYNC;
