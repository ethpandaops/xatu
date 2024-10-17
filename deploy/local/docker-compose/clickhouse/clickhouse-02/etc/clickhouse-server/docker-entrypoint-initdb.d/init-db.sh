#!/bin/bash
set -e
cat /etc/clickhouse-server/users.d/users.xml

cat <<EOT >> /etc/clickhouse-server/users.d/default.xml
<yandex>
  <users>
    <${CLICKHOUSE_USER}>
      <profile>default</profile>
      <networks>
        <ip>::/0</ip>
      </networks>
      $([ -n "${CLICKHOUSE_PASSWORD}" ] && echo "<password>${CLICKHOUSE_PASSWORD}</password>")
      <quota>default</quota>
    </${CLICKHOUSE_USER}>
    <readonly>
      <password>${CLICKHOUSE_USER_READONLY_PASSWORD}</password>
    </readonly>
  </users>
</yandex>
EOT

cat <<EOT >> /etc/clickhouse-server/config.d/users.xml
<clickhouse>
    <remote_servers>
        <cluster_2S_1R>
            <shard>
                <replica>
                    <host>xatu-clickhouse-01</host>
                    $([ -n "${CLICKHOUSE_PASSWORD}" ] && echo "<password replace=\"true\">${CLICKHOUSE_PASSWORD}</password>")
                </replica>
            </shard>
            <shard>
                <replica>
                    <host>xatu-clickhouse-02</host>
                    $([ -n "${CLICKHOUSE_PASSWORD}" ] && echo "<password replace=\"true\">${CLICKHOUSE_PASSWORD}</password>")
                </replica>
            </shard>
        </cluster_2S_1R>
    </remote_servers>
</clickhouse>
EOT


PASSWORD=${CLICKHOUSE_PASSWORD}

clickhouse client --user default --password ${PASSWORD} -n <<-EOSQL
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

echo "ClickHouse schema initialized"