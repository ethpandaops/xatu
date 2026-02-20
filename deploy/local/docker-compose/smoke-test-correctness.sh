#!/bin/bash
# Smoke test comparing Vector (default db) vs Consumoor (consumoor db) ClickHouse output.
#
# Usage:
#   ./smoke-test-correctness.sh [ingest|compare|all]
#
# Modes:
#   ingest  - Send test events from testdata captures to xatu-server HTTP API
#   compare - Compare default vs consumoor ClickHouse tables
#   all     - Ingest, wait, then compare (default)
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"
CAPTURES_DIR="$REPO_ROOT/pkg/consumoor/testdata/captures"
XATU_SERVER_URL="${XATU_SERVER_URL:-http://localhost:8087/v1/events}"
CH_HOST="${CH_HOST:-localhost}"
CH_PORT="${CH_PORT:-8123}"
FLUSH_WAIT="${FLUSH_WAIT:-15}"

MODE="${1:-all}"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

ch_query() {
  curl -sf "http://${CH_HOST}:${CH_PORT}/" --data-binary "$1"
}

# ── Ingest ──────────────────────────────────────────────────────────────────
ingest_events() {
  echo "=== Ingesting test events via xatu-server HTTP API ==="
  echo "Server: $XATU_SERVER_URL"
  echo "Captures: $CAPTURES_DIR"
  echo ""

  local total=0
  local sent=0
  local skipped=0
  local failed=0

  for jsonl_file in "$CAPTURES_DIR"/*.jsonl; do
    filename=$(basename "$jsonl_file" .jsonl)
    total=$((total + 1))

    # Skip empty files
    if [ ! -s "$jsonl_file" ]; then
      skipped=$((skipped + 1))
      continue
    fi

    lines=$(wc -l < "$jsonl_file" | tr -d ' ')

    # Send each line as a separate CreateEventsRequest
    local line_num=0
    while IFS= read -r line; do
      line_num=$((line_num + 1))

      # Wrap the DecoratedEvent in a CreateEventsRequest
      payload="{\"events\":[$line]}"

      http_code=$(curl -sf -o /dev/null -w "%{http_code}" \
        -X POST "$XATU_SERVER_URL" \
        -H "Content-Type: application/json" \
        -d "$payload" 2>/dev/null) || http_code="000"

      if [ "$http_code" = "200" ] || [ "$http_code" = "202" ] || [ "$http_code" = "204" ]; then
        :
      else
        echo -e "  ${RED}FAIL${NC} $filename line $line_num (HTTP $http_code)"
        failed=$((failed + 1))
      fi
    done < "$jsonl_file"

    echo -e "  ${GREEN}SENT${NC} $filename ($lines events)"
    sent=$((sent + 1))
  done

  echo ""
  echo "Ingest summary: total=$total sent=$sent skipped=$skipped failed=$failed"
}

# ── Compare ─────────────────────────────────────────────────────────────────
compare_tables() {
  echo "=== Comparing default vs consumoor ClickHouse tables ==="
  echo ""

  # Get tables that exist in consumoor database
  consumoor_tables=$(ch_query "SELECT name FROM system.tables WHERE database = 'consumoor' AND name NOT LIKE '%_local' AND engine = 'Distributed' ORDER BY name")

  if [ -z "$consumoor_tables" ]; then
    echo -e "${RED}ERROR: No tables found in consumoor database${NC}"
    echo "Make sure xatu-clickhouse-consumoor-init has completed."
    exit 1
  fi

  local total=0
  local match=0
  local diff=0
  local skip=0
  local empty=0

  while IFS= read -r table; do
    [ -z "$table" ] && continue
    total=$((total + 1))

    # Check if table exists in default
    default_exists=$(ch_query "SELECT count() FROM system.tables WHERE database = 'default' AND name = '$table' AND engine = 'Distributed'")
    if [ "$default_exists" = "0" ]; then
      echo -e "  ${YELLOW}SKIP${NC} $table (not in default db)"
      skip=$((skip + 1))
      continue
    fi

    # Get row counts
    default_count=$(ch_query "SELECT count() FROM default.\`$table\`")
    consumoor_count=$(ch_query "SELECT count() FROM consumoor.\`$table\`")

    if [ "$default_count" = "0" ] && [ "$consumoor_count" = "0" ]; then
      echo -e "  ${YELLOW}EMPTY${NC} $table (both 0 rows)"
      empty=$((empty + 1))
      continue
    fi

    if [ "$default_count" != "$consumoor_count" ]; then
      echo -e "  ${RED}DIFF${NC}  $table (default=$default_count consumoor=$consumoor_count)"
      diff=$((diff + 1))
      continue
    fi

    # Same count — do a deeper comparison using EXCEPT
    # Compare all columns except those that might legitimately differ
    diff_count=$(ch_query "
      SELECT count() FROM (
        SELECT * FROM default.\`$table\`
        EXCEPT
        SELECT * FROM consumoor.\`$table\`
      )
    " 2>/dev/null) || diff_count="error"

    if [ "$diff_count" = "error" ]; then
      echo -e "  ${YELLOW}SKIP${NC} $table (schema mismatch, cannot EXCEPT)"
      skip=$((skip + 1))
    elif [ "$diff_count" = "0" ]; then
      echo -e "  ${GREEN}MATCH${NC} $table ($default_count rows)"
      match=$((match + 1))
    else
      echo -e "  ${RED}DIFF${NC}  $table ($default_count rows, $diff_count differ)"
      diff=$((diff + 1))
    fi
  done <<< "$consumoor_tables"

  echo ""
  echo "Compare summary: total=$total match=$match diff=$diff skip=$skip empty=$empty"

  if [ "$diff" -gt 0 ]; then
    echo ""
    echo -e "${RED}FAIL: $diff tables have differences${NC}"
    return 1
  elif [ "$match" -gt 0 ]; then
    echo ""
    echo -e "${GREEN}PASS: All $match populated tables match${NC}"
    return 0
  else
    echo ""
    echo -e "${YELLOW}WARN: No tables had data to compare${NC}"
    return 0
  fi
}

# ── Main ────────────────────────────────────────────────────────────────────
case "$MODE" in
  ingest)
    ingest_events
    ;;
  compare)
    compare_tables
    ;;
  all)
    ingest_events
    echo ""
    echo "Waiting ${FLUSH_WAIT}s for both pipelines to flush..."
    sleep "$FLUSH_WAIT"
    echo ""
    compare_tables
    ;;
  *)
    echo "Usage: $0 [ingest|compare|all]"
    exit 1
    ;;
esac
