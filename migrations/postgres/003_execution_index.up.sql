CREATE INDEX node_record_eth2_idx ON node_record (eth2);
CREATE INDEX node_record_consecutive_dial_attempts_idx ON node_record (consecutive_dial_attempts);
CREATE INDEX node_record_last_connect_time_idx ON node_record (last_connect_time);
CREATE INDEX node_record_eth2_null_last_dial_time_idx ON node_record (last_dial_time) WHERE eth2 IS NULL;

CREATE INDEX node_record_execution_network_id_create_time_idx ON node_record_execution (network_id, create_time DESC);
CREATE INDEX node_record_execution_network_id_fork_id_hash_create_time_idx ON node_record_execution (network_id, fork_id_hash, create_time DESC);
CREATE INDEX node_record_execution_create_time_network_id_fork_id_hash_idx ON node_record_execution (create_time, network_id, fork_id_hash);

CREATE INDEX node_record_activity_update_time_client_id_idx ON node_record_activity (update_time, client_id);
