CREATE DATABASE IF NOT EXISTS logs_internal ON CLUSTER '{cluster}';
CREATE DATABASE IF NOT EXISTS logs_external ON CLUSTER '{cluster}';

CREATE TABLE IF NOT EXISTS logs_internal.logs_local ON CLUSTER '{cluster}'
(
    `Timestamp` DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    `LogDate` Date DEFAULT toDate(Timestamp),
    `IngressUser` LowCardinality(String) DEFAULT '',
    `Namespace` LowCardinality(String) DEFAULT '',
    `Pod` String DEFAULT '' CODEC(ZSTD(1)),
    `Container` LowCardinality(String) DEFAULT '',
    `Node` LowCardinality(String) DEFAULT '',
    `Stream` LowCardinality(String) DEFAULT '',
    `Message` String CODEC(ZSTD(1)),
    INDEX idx_msg Message TYPE tokenbf_v1(30720, 2, 0) GRANULARITY 1,
    INDEX idx_pod Pod TYPE bloom_filter(0.01) GRANULARITY 4,
    INDEX idx_namespace Namespace TYPE set(100) GRANULARITY 4
)
ENGINE = ReplicatedMergeTree('/clickhouse/{installation}/{cluster}/tables/{shard}/logs_internal/logs_local', '{replica}')
PARTITION BY LogDate
ORDER BY (IngressUser, Namespace, Timestamp)
TTL LogDate + INTERVAL 30 DAY
SETTINGS index_granularity = 8192;

CREATE TABLE IF NOT EXISTS logs_internal.logs ON CLUSTER '{cluster}'
AS logs_internal.logs_local
ENGINE = Distributed('{cluster}', 'logs_internal', 'logs_local', rand());

CREATE TABLE IF NOT EXISTS logs_external.logs_local ON CLUSTER '{cluster}'
(
    `Timestamp` DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    `LogDate` Date DEFAULT toDate(Timestamp),
    `IngressUser` LowCardinality(String) DEFAULT '',
    `Namespace` LowCardinality(String) DEFAULT '',
    `Pod` String DEFAULT '' CODEC(ZSTD(1)),
    `Container` LowCardinality(String) DEFAULT '',
    `Node` LowCardinality(String) DEFAULT '',
    `Stream` LowCardinality(String) DEFAULT '',
    `Message` String CODEC(ZSTD(1)),
    INDEX idx_msg Message TYPE tokenbf_v1(30720, 2, 0) GRANULARITY 1,
    INDEX idx_pod Pod TYPE bloom_filter(0.01) GRANULARITY 4,
    INDEX idx_namespace Namespace TYPE set(100) GRANULARITY 4
)
ENGINE = ReplicatedMergeTree('/clickhouse/{installation}/{cluster}/tables/{shard}/logs_external/logs_local', '{replica}')
PARTITION BY LogDate
ORDER BY (IngressUser, Namespace, Timestamp)
TTL LogDate + INTERVAL 30 DAY
SETTINGS index_granularity = 8192;

CREATE TABLE IF NOT EXISTS logs_external.logs ON CLUSTER '{cluster}'
AS logs_external.logs_local
ENGINE = Distributed('{cluster}', 'logs_external', 'logs_local', rand());
