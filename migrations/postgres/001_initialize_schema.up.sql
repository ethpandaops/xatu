CREATE TABLE node_record (
  enr VARCHAR(304) PRIMARY KEY,
  signature BYTEA,
  seq BIGINT,
  create_time TIMESTAMPTZ NOT NULL DEFAULT now(),
  last_dial_time TIMESTAMPTZ,
  consecutive_dial_attempts INT NOT NULL DEFAULT 0,
  last_connect_time TIMESTAMPTZ,
  id VARCHAR(10) NOT NULL,
  secp256k1 BYTEA,
  ip4 INET,
  ip6 INET,
  tcp4 INT,
  udp4 INT,
  tcp6 INT,
  udp6 INT,
  eth2 BYTEA,
  attnets BYTEA,
  syncnets BYTEA,
  node_id VARCHAR(128),
  peer_id VARCHAR(128)
);

CREATE TABLE node_record_execution (
  execution_id SERIAL PRIMARY KEY,
  enr VARCHAR(304) NOT NULL,
  create_time TIMESTAMPTZ NOT NULL DEFAULT now(),
  name VARCHAR(256),
  capabilities VARCHAR(256),
  protocol_version VARCHAR(256),
  network_id VARCHAR(256),
  total_difficulty VARCHAR(256),
  head BYTEA,
  genesis BYTEA,
  fork_id_hash BYTEA,
  fork_id_next VARCHAR(256),
  CONSTRAINT fk_node_record FOREIGN KEY (enr) REFERENCES node_record(enr)
);

CREATE TABLE node_record_activity (
  activity_id SERIAL PRIMARY KEY,
  enr VARCHAR(304) NOT NULL,
  client_id VARCHAR(128) NOT NULL,
  create_time TIMESTAMPTZ NOT NULL DEFAULT now(),
  update_time TIMESTAMPTZ NOT NULL DEFAULT now(),
  connected BOOLEAN NOT NULL DEFAULT FALSE,
  CONSTRAINT c_unique UNIQUE (enr, client_id),
  CONSTRAINT fk_node_record FOREIGN KEY (enr) REFERENCES node_record(enr)
);
