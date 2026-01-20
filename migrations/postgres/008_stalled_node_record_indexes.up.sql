-- Partial indexes for CheckoutStalledConsensusNodeRecords and CheckoutStalledExecutionNodeRecords queries.
-- These indexes pre-filter rows matching common conditions to avoid full table scans on 56M+ rows.
-- Note: Using ascending index (default) allows Index Scan Backward for ORDER BY DESC.

-- Consensus nodes (eth2 IS NOT NULL)
CREATE INDEX idx_node_record_stalled_consensus
ON node_record (last_dial_time)
WHERE eth2 IS NOT NULL AND consecutive_dial_attempts < 1000;

-- Execution nodes (eth2 IS NULL)
CREATE INDEX idx_node_record_stalled_execution
ON node_record (last_dial_time)
WHERE eth2 IS NULL AND consecutive_dial_attempts < 1000;
