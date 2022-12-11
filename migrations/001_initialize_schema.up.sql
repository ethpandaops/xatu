CREATE TABLE node_record (
  enr VARCHAR(304) PRIMARY KEY,
  signature BYTEA,
  seq BIGINT,
  created_timestamp TIMESTAMPTZ NOT NULL,
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
