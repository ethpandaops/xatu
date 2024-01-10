#!/bin/bash
set -e

clickhouse client --user default -n <<-EOSQL
CREATE TABLE default.schema_migrations_local ON CLUSTER '{cluster}'
(
    "version" Int64,
    "dirty" UInt8,
    "sequence" UInt64
) Engine = ReplicatedMergeTree('/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}', '{replica}')
ORDER BY sequence
SETTINGS index_granularity = 81921;

CREATE TABLE schema_migrations on cluster '{cluster}' AS schema_migrations_local
ENGINE = Distributed('{cluster}', default, schema_migrations_local, rand());
EOSQL
