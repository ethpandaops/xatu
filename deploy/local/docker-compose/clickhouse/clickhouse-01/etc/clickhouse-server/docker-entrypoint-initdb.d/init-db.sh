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
      <password>${CLICKHOUSE_PASSWORD}</password>
      <quota>default</quota>
    </${CLICKHOUSE_USER}>
    <readonly>
      <password>${CLICKHOUSE_USER_READONLY_PASSWORD}</password>
    </readonly>
  </users>
</yandex>
EOT

cat <<EOT >> /etc/clickhouse-server/config.d/users.xml
<yandex>

<clickhouse replace="true">
    <remote_servers>
        <cluster_2S_1R>
            <shard>
                <replica>
                    <host>xatu-clickhouse-01</host>
                    $([ -n "${CLICKHOUSE_PASSWORD}" ] && echo "<password>${CLICKHOUSE_PASSWORD}</password>")
                </replica>
            </shard>
            <shard>
                <replica>
                    <host>xatu-clickhouse-02</host>
                    $([ -n "${CLICKHOUSE_PASSWORD}" ] && echo "<password>${CLICKHOUSE_PASSWORD}</password>")
                </replica>
            </shard>
        </cluster_2S_1R>
    </remote_servers>
</yandex>
EOT