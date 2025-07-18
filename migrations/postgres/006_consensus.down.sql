-- Drop indexes first
DROP INDEX IF EXISTS node_record_consensus_create_time_network_id_fork_digest_idx;
DROP INDEX IF EXISTS node_record_consensus_network_id_fork_digest_create_time_idx;
DROP INDEX IF EXISTS node_record_consensus_network_id_create_time_idx;

-- Drop the table
DROP TABLE IF EXISTS node_record_consensus;

-- Remove added columns from node_record
ALTER TABLE node_record DROP COLUMN IF EXISTS next_fork_digest;
ALTER TABLE node_record DROP COLUMN IF EXISTS cgc;

-- Revert ENR column type changes
ALTER TABLE node_record_activity ALTER COLUMN enr TYPE VARCHAR(304);
ALTER TABLE node_record_execution ALTER COLUMN enr TYPE VARCHAR(304);
ALTER TABLE node_record ALTER COLUMN enr TYPE VARCHAR(304);
