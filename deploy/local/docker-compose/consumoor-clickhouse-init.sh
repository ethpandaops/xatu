#!/bin/bash
# Initialize the consumoor ClickHouse database by copying all table schemas from default.
set -e

CH_HOST="${CH_HOST:-xatu-clickhouse-01}"

ch() {
  clickhouse-client --host="$CH_HOST" "$@"
}

echo "Creating consumoor database..."
ch --query="CREATE DATABASE IF NOT EXISTS consumoor ON CLUSTER '{cluster}'"

echo "Getting local tables from default database..."
tables=$(ch --query="SELECT name FROM system.tables WHERE database = 'default' AND name LIKE '%_local' AND engine LIKE 'Replicated%' ORDER BY name")

for table in $tables; do
  echo "Copying: default.$table -> consumoor.$table"

  # Get the CREATE TABLE DDL and modify it for consumoor database:
  # 1. Replace database qualifier: default. -> consumoor.
  # 2. Add ON CLUSTER after table name
  # 3. Append /consumoor to the ZK path (first quoted arg in MergeTree())
  #    to ensure unique paths regardless of path pattern variant
  ch --format=TSVRaw --query="SHOW CREATE TABLE default.\`$table\`" \
    | sed '1s/^CREATE TABLE default\./CREATE TABLE IF NOT EXISTS consumoor./' \
    | sed "1s/\$/ ON CLUSTER '{cluster}'/" \
    | sed "s|MergeTree('/\([^']*\)'|MergeTree('/\1/consumoor'|" \
    | ch --multiquery

  # Create corresponding distributed table
  dist_table="${table%_local}"
  echo "Creating distributed: consumoor.$dist_table"
  ch --query="CREATE TABLE IF NOT EXISTS consumoor.\`$dist_table\` ON CLUSTER '{cluster}' AS consumoor.\`$table\` ENGINE = Distributed('{cluster}', 'consumoor', '$table', rand())"
done

echo ""
table_count=$(echo "$tables" | wc -w | tr -d ' ')
echo "Consumoor database initialization complete! ($table_count tables copied)"
