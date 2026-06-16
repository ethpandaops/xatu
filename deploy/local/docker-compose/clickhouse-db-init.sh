#!/bin/bash
# Create the target databases for the migration matrix before migrations run.
#
# Databases are DISCOVERED from the migration set directories (convention:
# directory name == database name) plus any MIGRATE_DB_OVERRIDES, so adding a new
# migration directory automatically provisions its database. `default` always
# exists and is skipped.
set -euo pipefail

: "${CLICKHOUSE_HOST:=xatu-clickhouse-01}"
: "${CLICKHOUSE_PORT:=9000}"
: "${CLICKHOUSE_USER:=default}"
: "${CLICKHOUSE_PASSWORD:=}"
: "${MIGRATIONS_DIR:=/migrations}"
: "${MIGRATE_DB_OVERRIDES:=}"

# Echo the target database(s) for a set: the override if present, else the set name.
target_dbs() {
  local set="$1" entry
  for entry in $MIGRATE_DB_OVERRIDES; do
    case "$entry" in
      "${set}="*) echo "${entry#*=}" | tr ',' ' '; return 0 ;;
    esac
  done
  echo "$set"
}

dbs=""
for dir in "${MIGRATIONS_DIR}"/*/; do
  [ -d "$dir" ] || continue
  dbs="$dbs $(target_dbs "$(basename "$dir")")"
done

for db in $(printf '%s\n' $dbs | sort -u); do
  [ "$db" = "default" ] && continue
  echo "creating database ${db}"
  clickhouse client --host "$CLICKHOUSE_HOST" --port "$CLICKHOUSE_PORT" \
    --user "$CLICKHOUSE_USER" --password "$CLICKHOUSE_PASSWORD" \
    --query "CREATE DATABASE IF NOT EXISTS ${db} ON CLUSTER '{cluster}'"
done

echo "target databases ready"
