
ALTER TABLE node_record ALTER COLUMN enr TYPE VARCHAR(1000);
ALTER TABLE node_record_execution ALTER COLUMN enr TYPE VARCHAR(1000);
ALTER TABLE node_record_activity ALTER COLUMN enr TYPE VARCHAR(1000);
ALTER TABLE node_record ADD COLUMN cgc BYTEA;
ALTER TABLE node_record ADD COLUMN next_fork_digest BYTEA;

CREATE TABLE node_record_consensus (
  consensus_id SERIAL PRIMARY KEY,
  enr VARCHAR(1000) NOT NULL,
  node_id VARCHAR(128),
  peer_id VARCHAR(128),
  create_time TIMESTAMPTZ NOT NULL DEFAULT now(),
  name VARCHAR(256),
  fork_digest BYTEA,
  next_fork_digest BYTEA,
  finalized_root BYTEA,
  finalized_epoch BYTEA,
  head_root BYTEA,
  head_slot BYTEA,
  cgc BYTEA,
  network_id VARCHAR(256),
  CONSTRAINT fk_node_record FOREIGN KEY (enr) REFERENCES node_record(enr)
);

CREATE INDEX node_record_consensus_network_id_create_time_idx ON node_record_consensus (network_id, create_time DESC);
CREATE INDEX node_record_consensus_network_id_fork_digest_create_time_idx ON node_record_consensus (network_id, fork_digest, create_time DESC);
CREATE INDEX node_record_consensus_create_time_network_id_fork_digest_idx ON node_record_consensus (create_time, network_id, fork_digest);

ALTER TABLE node_record ALTER COLUMN enr TYPE VARCHAR(1000);
ALTER TABLE node_record_execution ALTER COLUMN enr TYPE VARCHAR(1000);
ALTER TABLE node_record_activity ALTER COLUMN enr TYPE VARCHAR(1000);
