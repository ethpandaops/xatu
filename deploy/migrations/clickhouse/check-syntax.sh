#!/usr/bin/env bash
# Validate that every ClickHouse migration parses with the real ClickHouse parser.
#
# Runs `clickhouse format` (parse + reformat, no server needed) over each *.sql file
# in check-only mode. This is a syntax gate, complementary to lint.sh: lint.sh
# enforces the database-agnostic rules, this catches malformed SQL before it ever
# reaches the migrator.
#
#   --multiquery     files contain many `;`-separated statements
#   --quiet          only report on failure (no reformatted output on success)
#   --max_query_size the parser caps a query at 256 KiB by default; the xatu set is
#                    larger than that, so raise it well past any single file.
#
# Requires the `clickhouse` binary on PATH (override with CLICKHOUSE_BIN). In CI this
# runs inside the clickhouse/clickhouse-server image; locally any clickhouse works.
#
# Usage: check-syntax.sh [MIGRATIONS_DIR]   (default: dir of this script)
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
MIGRATIONS_DIR="${1:-$SCRIPT_DIR}"
CH="${CLICKHOUSE_BIN:-clickhouse}"
MAX_QUERY_SIZE="${MAX_QUERY_SIZE:-1073741824}"   # 1 GiB; files are well under this.

CI_ANNOTATE=0
[ "${GITHUB_ACTIONS:-}" = "true" ] && CI_ANNOTATE=1

shopt -s nullglob
files=("$MIGRATIONS_DIR"/*/*.sql)
if [ ${#files[@]} -eq 0 ]; then
  echo "no migration .sql files found under ${MIGRATIONS_DIR}" >&2
  exit 1
fi

status=0
for file in "${files[@]}"; do
  if err=$("$CH" format --multiquery --quiet --max_query_size "$MAX_QUERY_SIZE" < "$file" 2>&1); then
    echo "OK   $file"
  else
    msg=$(printf '%s' "$err" | head -1)
    echo "FAIL $file :: $msg" >&2
    [ "$CI_ANNOTATE" -eq 1 ] && echo "::error file=${file}::ClickHouse failed to parse this migration: ${msg}"
    status=1
  fi
done

if [ "$status" -ne 0 ]; then
  echo "" >&2
  echo "ClickHouse migration syntax check failed." >&2
  exit 1
fi

echo "ClickHouse migration syntax check passed (${#files[@]} files)."
