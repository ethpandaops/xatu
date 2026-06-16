#!/bin/sh
# Apply every ClickHouse migration set to its target database(s).
#
# Convention: each directory under deploy/migrations/clickhouse/<set>/ is a
# migration set, and is applied to a database of the SAME NAME. So adding a new
# directory "just works" — new directory == new database.
#
# Override the mapping (rename and/or fan a set out to several databases) with
# MIGRATE_DB_OVERRIDES, a space-separated list of "set=db1,db2" entries. e.g.
#   MIGRATE_DB_OVERRIDES="xatu=default,devnets"
# applies the `xatu` set to both the `default` and `devnets` databases.
#
# The sets are database-agnostic (no db qualifiers; Distributed uses
# currentDatabase(); Replicated paths use the {database} macro), so the same
# files apply cleanly to any database — the golang-migrate multi-db idiom.
# Each (set, database) tracks its own state in `schema_migrations_<set>`.
set -eu

: "${CLICKHOUSE_HOST:=xatu-clickhouse-01}"
: "${CLICKHOUSE_PORT:=9000}"
: "${CLICKHOUSE_USER:=default}"
: "${CLICKHOUSE_PASSWORD:=}"
: "${MIGRATIONS_DIR:=/migrations}"
: "${MIGRATE_DB_OVERRIDES:=}"

DSN="clickhouse://${CLICKHOUSE_HOST}:${CLICKHOUSE_PORT}?username=${CLICKHOUSE_USER}&password=${CLICKHOUSE_PASSWORD}&x-multi-statement=true"

# Echo the target database(s) for a set: the override if present, else the set name.
target_dbs() {
  _set="$1"
  for _entry in $MIGRATE_DB_OVERRIDES; do
    case "$_entry" in
      "${_set}="*) echo "${_entry#*=}" | tr ',' ' '; return 0 ;;
    esac
  done
  echo "$_set"
}

for _dir in "${MIGRATIONS_DIR}"/*/; do
  [ -d "$_dir" ] || continue
  _set=$(basename "$_dir")
  for _db in $(target_dbs "$_set"); do
    echo "==> migrate set=${_set} -> database=${_db}"
    migrate \
      -path "${MIGRATIONS_DIR}/${_set}" \
      -database "${DSN}&database=${_db}&x-migrations-table=schema_migrations_${_set}" \
      up
  done
done

echo "ClickHouse migrations complete."
