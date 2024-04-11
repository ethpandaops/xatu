-- Creating local and distributed tables for libp2p_publish_message
CREATE TABLE default.libp2p_publish_message_local
(
    unique_key Int64 COMMENT 'Unique identifier for each record',
    updated_date_time DateTime CODEC(DoubleDelta, ZSTD(1)) COMMENT 'Timestamp when the record was last updated',
    event_date_time DateTime64(3) CODEC(DoubleDelta, ZSTD(1)) COMMENT 'Timestamp of the event',
    message_id String CODEC(ZSTD(1)) COMMENT 'Identifier of the message',
    topic String CODEC(ZSTD(1)) COMMENT 'Topic associated with the message',
    peer_id String CODEC(ZSTD(1)) COMMENT 'Identifier of the peer'
) Engine = ReplicatedReplacingMergeTree('/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}', '{replica}', updated_date_time)
PARTITION BY toYYYYMM(event_date_time)
ORDER BY (event_date_time, unique_key);

CREATE TABLE default.libp2p_publish_message AS default.libp2p_publish_message_local
ENGINE = Distributed('{cluster}', default, libp2p_publish_message_local, rand());

-- Creating local and distributed tables for libp2p_reject_message
CREATE TABLE default.libp2p_reject_message_local
(
    unique_key Int64 COMMENT 'Unique identifier for each record',
    updated_date_time DateTime CODEC(DoubleDelta, ZSTD(1)) COMMENT 'Timestamp when the record was last updated',
    event_date_time DateTime64(3) CODEC(DoubleDelta, ZSTD(1)) COMMENT 'Timestamp of the event',
    message_id String CODEC(ZSTD(1)) COMMENT 'Identifier of the message',
    received_from String CODEC(ZSTD(1)) COMMENT 'Identifier of the sender',
    reason String CODEC(ZSTD(1)) COMMENT 'Reason for message rejection',
    topic String CODEC(ZSTD(1)) COMMENT 'Topic associated with the message',
    peer_id String CODEC(ZSTD(1)) COMMENT 'Identifier of the peer'
) Engine = ReplicatedReplacingMergeTree('/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}', '{replica}', updated_date_time)
PARTITION BY toYYYYMM(event_date_time)
ORDER BY (event_date_time, unique_key);

CREATE TABLE default.libp2p_reject_message AS default.libp2p_reject_message_local
ENGINE = Distributed('{cluster}', default, libp2p_reject_message_local, rand());

-- Creating local and distributed tables for libp2p_duplicate_message
CREATE TABLE default.libp2p_duplicate_message_local
(
    unique_key Int64 COMMENT 'Unique identifier for each record',
    updated_date_time DateTime CODEC(DoubleDelta, ZSTD(1)) COMMENT 'Timestamp when the record was last updated',
    event_date_time DateTime64(3) CODEC(DoubleDelta, ZSTD(1)) COMMENT 'Timestamp of the event',
    message_id String CODEC(ZSTD(1)) COMMENT 'Identifier of the message',
    received_from String CODEC(ZSTD(1)) COMMENT 'Identifier of the sender',
    topic String CODEC(ZSTD(1)) COMMENT 'Topic associated with the message',
    peer_id String CODEC(ZSTD(1)) COMMENT 'Identifier of the peer'
) Engine = ReplicatedReplacingMergeTree('/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}', '{replica}', updated_date_time)
PARTITION BY toYYYYMM(event_date_time)
ORDER BY (event_date_time, unique_key);

CREATE TABLE default.libp2p_duplicate_message AS default.libp2p_duplicate_message_local
ENGINE = Distributed('{cluster}', default, libp2p_duplicate_message_local, rand());

-- Creating local and distributed tables for libp2p_deliver_message
CREATE TABLE default.libp2p_deliver_message_local
(
    unique_key Int64 COMMENT 'Unique identifier for each record',
    updated_date_time DateTime CODEC(DoubleDelta, ZSTD(1)) COMMENT 'Timestamp when the record was last updated',
    event_date_time DateTime64(3) CODEC(DoubleDelta, ZSTD(1)) COMMENT 'Timestamp of the event',
    message_id String CODEC(ZSTD(1)) COMMENT 'Identifier of the message',
    topic String CODEC(ZSTD(1)) COMMENT 'Topic associated with the message',
    received_from String CODEC(ZSTD(1)) COMMENT 'Identifier of the sender',
    peer_id String CODEC(ZSTD(1)) COMMENT 'Identifier of the peer'
) Engine = ReplicatedReplacingMergeTree('/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}', '{replica}', updated_date_time)
PARTITION BY toYYYYMM(event_date_time)
ORDER BY (event_date_time, unique_key);

CREATE TABLE default.libp2p_deliver_message AS default.libp2p_deliver_message_local
ENGINE = Distributed('{cluster}', default, libp2p_deliver_message_local, rand());

-- Creating local and distributed tables for libp2p_add_peer
CREATE TABLE default.libp2p_add_peer_local
(
    unique_key Int64 COMMENT 'Unique identifier for each record',
    updated_date_time DateTime CODEC(DoubleDelta, ZSTD(1)) COMMENT 'Timestamp when the record was last updated',
    event_date_time DateTime64(3) CODEC(DoubleDelta, ZSTD(1)) COMMENT 'Timestamp of the event',
    peer_id String CODEC(ZSTD(1)) COMMENT 'Identifier of the peer',
    proto String CODEC(ZSTD(1)) COMMENT 'Protocol used by the peer'
) Engine = ReplicatedReplacingMergeTree('/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}', '{replica}', updated_date_time)
PARTITION BY toYYYYMM(event_date_time)
ORDER BY (event_date_time, unique_key);

CREATE TABLE default.libp2p_add_peer AS default.libp2p_add_peer_local
ENGINE = Distributed('{cluster}', default, libp2p_add_peer_local, rand());

-- Creating local and distributed tables for libp2p_remove_peer
CREATE TABLE default.libp2p_remove_peer_local
(
    unique_key Int64 COMMENT 'Unique identifier for each record',
    updated_date_time DateTime CODEC(DoubleDelta, ZSTD(1)) COMMENT 'Timestamp when the record was last updated',
    event_date_time DateTime64(3) CODEC(DoubleDelta, ZSTD(1)) COMMENT 'Timestamp of the event',
    peer_id String CODEC(ZSTD(1)) COMMENT 'Identifier of the peer'
) Engine = ReplicatedReplacingMergeTree('/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}', '{replica}', updated_date_time)
PARTITION BY toYYYYMM(event_date_time)
ORDER BY (event_date_time, unique_key);

CREATE TABLE default.libp2p_remove_peer AS default.libp2p_remove_peer_local
ENGINE = Distributed('{cluster}', default, libp2p_remove_peer_local, rand());

-- Creating local and distributed tables for libp2p_recv_rpc
CREATE TABLE default.libp2p_recv_rpc_local
(
    unique_key Int64 COMMENT 'Unique identifier for each record',
    updated_date_time DateTime CODEC(DoubleDelta, ZSTD(1)) COMMENT 'Timestamp when the record was last updated',
    event_date_time DateTime64(3) CODEC(DoubleDelta, ZSTD(1)) COMMENT 'Timestamp of the event',
    received_from String CODEC(ZSTD(1)) COMMENT 'Identifier of the sender'
) Engine = ReplicatedReplacingMergeTree('/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}', '{replica}', updated_date_time)
PARTITION BY toYYYYMM(event_date_time)
ORDER BY (event_date_time, unique_key);

CREATE TABLE default.libp2p_recv_rpc AS default.libp2p_recv_rpc_local
ENGINE = Distributed('{cluster}', default, libp2p_recv_rpc_local, rand());

-- Creating local and distributed tables for libp2p_send_rpc
CREATE TABLE default.libp2p_send_rpc_local
(
    unique_key Int64 COMMENT 'Unique identifier for each record',
    updated_date_time DateTime CODEC(DoubleDelta, ZSTD(1)) COMMENT 'Timestamp when the record was last updated',
    event_date_time DateTime64(3) CODEC(DoubleDelta, ZSTD(1)) COMMENT 'Timestamp of the event',
    send_to String CODEC(ZSTD(1)) COMMENT 'Identifier of the receiver'
) Engine = ReplicatedReplacingMergeTree('/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}', '{replica}', updated_date_time)
PARTITION BY toYYYYMM(event_date_time)
ORDER BY (event_date_time, unique_key);

CREATE TABLE default.libp2p_send_rpc AS default.libp2p_send_rpc_local
ENGINE = Distributed('{cluster}', default, libp2p_send_rpc_local, rand());

-- Creating local and distributed tables for libp2p_drop_rpc
CREATE TABLE default.libp2p_drop_rpc_local
(
    unique_key Int64 COMMENT 'Unique identifier for each record',
    updated_date_time DateTime CODEC(DoubleDelta, ZSTD(1)) COMMENT 'Timestamp when the record was last updated',
    event_date_time DateTime64(3) CODEC(DoubleDelta, ZSTD(1)) COMMENT 'Timestamp of the event',
    send_to String CODEC(ZSTD(1)) COMMENT 'Identifier of the receiver'
) Engine = ReplicatedReplacingMergeTree('/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}', '{replica}', updated_date_time)
PARTITION BY toYYYYMM(event_date_time)
ORDER BY (event_date_time, unique_key);

CREATE TABLE default.libp2p_drop_rpc AS default.libp2p_drop_rpc_local
ENGINE = Distributed('{cluster}', default, libp2p_drop_rpc_local, rand());

-- Creating local and distributed tables for libp2p_join
CREATE TABLE default.libp2p_join_local
(
    unique_key Int64 COMMENT 'Unique identifier for each record',
    updated_date_time DateTime CODEC(DoubleDelta, ZSTD(1)) COMMENT 'Timestamp when the record was last updated',
    event_date_time DateTime64(3) CODEC(DoubleDelta, ZSTD(1)) COMMENT 'Timestamp of the event',
    topic String CODEC(ZSTD(1)) COMMENT 'Topic involved in the join event'
) Engine = ReplicatedReplacingMergeTree('/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}', '{replica}', updated_date_time)
PARTITION BY toYYYYMM(event_date_time)
ORDER BY (event_date_time, unique_key);

CREATE TABLE default.libp2p_join AS default.libp2p_join_local
ENGINE = Distributed('{cluster}', default, libp2p_join_local, rand());

-- Creating local and distributed tables for libp2p_leave
CREATE TABLE default.libp2p_leave_local
(
    unique_key Int64 COMMENT 'Unique identifier for each record',
    updated_date_time DateTime CODEC(DoubleDelta, ZSTD(1)) COMMENT 'Timestamp when the record was last updated',
    event_date_time DateTime64(3) CODEC(DoubleDelta, ZSTD(1)) COMMENT 'Timestamp of the event',
    topic String CODEC(ZSTD(1)) COMMENT 'Topic involved in the leave event'
) Engine = ReplicatedReplacingMergeTree('/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}', '{replica}', updated_date_time)
PARTITION BY toYYYYMM(event_date_time)
ORDER BY (event_date_time, unique_key);

CREATE TABLE default.libp2p_leave AS default.libp2p_leave_local
ENGINE = Distributed('{cluster}', default, libp2p_leave_local, rand());

-- Creating local and distributed tables for libp2p_graft
CREATE TABLE default.libp2p_graft_local
(
    unique_key Int64 COMMENT 'Unique identifier for each record',
    updated_date_time DateTime CODEC(DoubleDelta, ZSTD(1)) COMMENT 'Timestamp when the record was last updated',
    event_date_time DateTime64(3) CODEC(DoubleDelta, ZSTD(1)) COMMENT 'Timestamp of the event',
    peer_id String CODEC(ZSTD(1)) COMMENT 'Identifier of the peer',
    topic String CODEC(ZSTD(1)) COMMENT 'Topic involved in the graft event'
) Engine = ReplicatedReplacingMergeTree('/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}', '{replica}', updated_date_time)
PARTITION BY toYYYYMM(event_date_time)
ORDER BY (event_date_time, unique_key);

CREATE TABLE default.libp2p_graft AS default.libp2p_graft_local
ENGINE = Distributed('{cluster}', default, libp2p_graft_local, rand());

-- Creating local and distributed tables for libp2p_prune
CREATE TABLE default.libp2p_prune_local
(
    unique_key Int64 COMMENT 'Unique identifier for each record',
    updated_date_time DateTime CODEC(DoubleDelta, ZSTD(1)) COMMENT 'Timestamp when the record was last updated',
    event_date_time DateTime64(3) CODEC(DoubleDelta, ZSTD(1)) COMMENT 'Timestamp of the event',
    peer_id String CODEC(ZSTD(1)) COMMENT 'Identifier of the peer',
    topic String CODEC(ZSTD(1)) COMMENT 'Topic involved in the prune event'
) Engine = ReplicatedReplacingMergeTree('/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}', '{replica}', updated_date_time)
PARTITION BY toYYYYMM(event_date_time)
ORDER BY (event_date_time, unique_key);

CREATE TABLE default.libp2p_prune AS default.libp2p_prune_local
ENGINE = Distributed('{cluster}', default, libp2p_prune_local, rand());


-- Creating local and distributed tables for libp2p_validate_message
CREATE TABLE default.libp2p_validate_message_local
(
    unique_key Int64 COMMENT 'Unique identifier for each record',
    updated_date_time DateTime CODEC(DoubleDelta, ZSTD(1)) COMMENT 'Timestamp when the record was last updated',
    event_date_time DateTime64(3) CODEC(DoubleDelta, ZSTD(1)) COMMENT 'Timestamp of the event',
    message_id String CODEC(ZSTD(1)) COMMENT 'ID of the message',
    topic String CODEC(ZSTD(1)) COMMENT 'Topic involved in the validate message event',
    peer_id String CODEC(ZSTD(1)) COMMENT 'Identifier of the peer',
    local Bool COMMENT 'Whether the message was local'
) Engine = ReplicatedReplacingMergeTree('/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}', '{replica}', updated_date_time)
PARTITION BY toYYYYMM(event_date_time)
ORDER BY (event_date_time, unique_key);

CREATE TABLE default.libp2p_validate_message AS default.libp2p_validate_message_local
ENGINE = Distributed('{cluster}', default, libp2p_validate_message_local, rand());

-- Creating local and distributed tables for libp2p_throttle_peer
CREATE TABLE default.libp2p_throttle_peer_local
(
    unique_key Int64 COMMENT 'Unique identifier for each record',
    updated_date_time DateTime CODEC(DoubleDelta, ZSTD(1)) COMMENT 'Timestamp when the record was last updated',
    event_date_time DateTime64(3) CODEC(DoubleDelta, ZSTD(1)) COMMENT 'Timestamp of the event',
    peer_id String CODEC(ZSTD(1)) COMMENT 'Identifier of the peer'
) Engine = ReplicatedReplacingMergeTree('/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}', '{replica}', updated_date_time)
PARTITION BY toYYYYMM(event_date_time)
ORDER BY (event_date_time, unique_key);

CREATE TABLE default.libp2p_throttle_peer AS default.libp2p_throttle_peer_local
ENGINE = Distributed('{cluster}', default, libp2p_throttle_peer_local, rand());

-- Creating local and distributed tables for libp2p_undeliverable_message
CREATE TABLE default.libp2p_undeliverable_message_local
(
    unique_key Int64 COMMENT 'Unique identifier for each record',
    updated_date_time DateTime CODEC(DoubleDelta, ZSTD(1)) COMMENT 'Timestamp when the record was last updated',
    event_date_time DateTime64(3) CODEC(DoubleDelta, ZSTD(1)) COMMENT 'Timestamp of the event',
    message_id String CODEC(ZSTD(1)) COMMENT 'ID of the message',
    topic String CODEC(ZSTD(1)) COMMENT 'Topic involved in the undeliverable message event',
    peer_id String CODEC(ZSTD(1)) COMMENT 'Identifier of the peer',
    local Bool COMMENT 'Whether the message was local'
) Engine = ReplicatedReplacingMergeTree('/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}', '{replica}', updated_date_time)
PARTITION BY toYYYYMM(event_date_time)
ORDER BY (event_date_time, unique_key);

CREATE TABLE default.libp2p_undeliverable_message AS default.libp2p_undeliverable_message_local
ENGINE = Distributed('{cluster}', default, libp2p_undeliverable_message_local, rand());

-- Creating local and distributed tables for libp2p_connected
CREATE TABLE default.libp2p_connected_local
(
    unique_key Int64 COMMENT 'Unique identifier for each record',
    updated_date_time DateTime CODEC(DoubleDelta, ZSTD(1)) COMMENT 'Timestamp when the record was last updated',
    event_date_time DateTime64(3) CODEC(DoubleDelta, ZSTD(1)) COMMENT 'Timestamp of the event',
    remote_peer String CODEC(ZSTD(1)) COMMENT 'Remote peer identifier',
    remote_maddrs String CODEC(ZSTD(1)) COMMENT 'Multiaddresses of the remote peer',
    agent_version String CODEC(ZSTD(1)) COMMENT 'Agent version of the remote peer',
    direction String CODEC(ZSTD(1)) COMMENT 'Connection direction',
    opened DateTime CODEC(DoubleDelta, ZSTD(1)) COMMENT 'Timestamp when the connection was opened',
    transient Bool COMMENT 'Whether the connection is transient'
) Engine = ReplicatedReplacingMergeTree('/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}', '{replica}', updated_date_time)
PARTITION BY toYYYYMM(event_date_time)
ORDER BY (event_date_time, unique_key);

CREATE TABLE default.libp2p_connected AS default.libp2p_connected_local
ENGINE = Distributed('{cluster}', default, libp2p_connected_local, rand());

-- Creating local and distributed tables for libp2p_disconnected
CREATE TABLE default.libp2p_disconnected_local
(
    unique_key Int64 COMMENT 'Unique identifier for each record',
    updated_date_time DateTime CODEC(DoubleDelta, ZSTD(1)) COMMENT 'Timestamp when the record was last updated',
    event_date_time DateTime64(3) CODEC(DoubleDelta, ZSTD(1)) COMMENT 'Timestamp of the event',
    remote_peer String CODEC(ZSTD(1)) COMMENT 'Remote peer identifier',
    remote_maddrs String CODEC(ZSTD(1)) COMMENT 'Multiaddresses of the remote peer',
    agent_version String CODEC(ZSTD(1)) COMMENT 'Agent version of the remote peer',
    direction String CODEC(ZSTD(1)) COMMENT 'Connection direction',
    opened DateTime CODEC(DoubleDelta, ZSTD(1)) COMMENT 'Timestamp when the connection was opened',
    transient Bool COMMENT 'Whether the connection is transient'
) Engine = ReplicatedReplacingMergeTree('/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}', '{replica}', updated_date_time)
PARTITION BY toYYYYMM(event_date_time)
ORDER BY (event_date_time, unique_key);

CREATE TABLE default.libp2p_disconnected AS default.libp2p_disconnected_local
ENGINE = Distributed('{cluster}', default, libp2p_disconnected_local, rand());

