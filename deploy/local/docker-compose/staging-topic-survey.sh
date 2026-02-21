#!/usr/bin/env bash
# Survey which topics/tables consumoor handles from staging Kafka.
#
# Consumes from staging proto topics briefly, then reports which ClickHouse
# tables received data and compares row counts against the staging reference.
#
# Usage:
#   ./staging-topic-survey.sh [flags]
#
# Flags:
#   --consume-seconds SEC    How long to consume (default: 10)
#   --topics PATTERN         Kafka topic regex (default: ^xatu-protobuf-.+)
#   --skip-portforward       Skip kubectl port-forward setup
#   --compare-only           Skip consuming, just survey existing data
#   --help                   Show this help
#
# Environment:
#   KUBE_CONTEXT             (default: platform-analytics-hel1-staging)
#   KUBE_NAMESPACE           (default: xatu)
#   KAFKA_SERVICE            (default: svc/xatu-internal-kafka-kafka-bootstrap)
#   CLICKHOUSE_SERVICE       (default: svc/chendpoint-xatu-clickhouse)
#   STAGING_CLICKHOUSE_USER  (default: empty)
#   STAGING_CLICKHOUSE_PASSWORD (default: empty)
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"

# --- Defaults ---
CONSUME_SECONDS="${CONSUME_SECONDS:-10}"
TOPICS="${TOPICS:-^xatu-protobuf-.+}"
SKIP_PORTFORWARD=false
COMPARE_ONLY=false
KUBE_CONTEXT="${KUBE_CONTEXT:-platform-analytics-hel1-staging}"
KUBE_NAMESPACE="${KUBE_NAMESPACE:-xatu}"
KAFKA_SERVICE="${KAFKA_SERVICE:-svc/xatu-internal-kafka-kafka-bootstrap}"
CLICKHOUSE_SERVICE="${CLICKHOUSE_SERVICE:-svc/chendpoint-xatu-clickhouse}"
STAGING_CLICKHOUSE_USER="${STAGING_CLICKHOUSE_USER:-}"
STAGING_CLICKHOUSE_PASSWORD="${STAGING_CLICKHOUSE_PASSWORD:-}"

# Random ports in 20000s to avoid conflicts.
KAFKA_PORT=$((20000 + RANDOM % 1000))
CH_STAGING_PORT=$((21000 + RANDOM % 1000))

RUN_ID="$(date +%s)"
CONFIG_FILE="$SCRIPT_DIR/xatu-consumoor-staging.yaml"

# --- Parse flags ---
while [[ $# -gt 0 ]]; do
  case "$1" in
    --consume-seconds)  CONSUME_SECONDS="$2"; shift 2 ;;
    --topics)           TOPICS="$2"; shift 2 ;;
    --skip-portforward) SKIP_PORTFORWARD=true; shift ;;
    --compare-only)     COMPARE_ONLY=true; shift ;;
    --help)
      sed -n '2,/^set -/p' "$0" | head -n -1
      exit 0
      ;;
    *) echo "Unknown flag: $1"; exit 1 ;;
  esac
done

# --- Cleanup ---
PF_PIDS=()

cleanup() {
  echo ""
  echo "=== Cleaning up ==="
  if [[ "$COMPARE_ONLY" != "true" ]]; then
    cd "$REPO_ROOT"
    docker compose -f docker-compose.yml \
      -f deploy/local/docker-compose/docker-compose.staging.yml \
      down --timeout 5 2>/dev/null || true
  fi
  for pid in "${PF_PIDS[@]}"; do
    kill "$pid" 2>/dev/null || true
  done
  rm -f "$CONFIG_FILE"
  echo "Done."
}
trap cleanup EXIT

# --- Helpers ---
ch_local() {
  curl -sf "http://localhost:8123/" --data-binary "$1"
}

ch_staging() {
  if [[ -n "$STAGING_CLICKHOUSE_USER" ]]; then
    curl -sf "http://localhost:${CH_STAGING_PORT}/" \
      --user "${STAGING_CLICKHOUSE_USER}:${STAGING_CLICKHOUSE_PASSWORD}" \
      --data-binary "$1"
  else
    curl -sf "http://localhost:${CH_STAGING_PORT}/" --data-binary "$1"
  fi
}

# Infer the proto topic name(s) from a ClickHouse table name.
# e.g. beacon_api_eth_v1_events_head → xatu-protobuf-beacon-api-eth-v1-events-head
infer_topic() {
  local base="$1"
  echo "xatu-protobuf-$(echo "$base" | tr '_' '-')"
}

# --- Banner ---
echo "=== Staging Topic Survey ==="
echo "  Kafka port-forward:     localhost:${KAFKA_PORT}"
echo "  Staging CH port-forward: localhost:${CH_STAGING_PORT}"
echo "  Topics pattern:         ${TOPICS}"
echo "  Consume duration:       ${CONSUME_SECONDS}s"
echo ""

# --- 1. Port-forwards ---
if [[ "$SKIP_PORTFORWARD" != "true" ]]; then
  echo "=== Starting port-forwards (context: $KUBE_CONTEXT) ==="

  kubectl --context "$KUBE_CONTEXT" -n "$KUBE_NAMESPACE" \
    port-forward "$KAFKA_SERVICE" "${KAFKA_PORT}:9092" &
  PF_PIDS+=($!)

  kubectl --context "$KUBE_CONTEXT" -n "$KUBE_NAMESPACE" \
    port-forward "$CLICKHOUSE_SERVICE" "${CH_STAGING_PORT}:8123" &
  PF_PIDS+=($!)

  sleep 3

  if ! curl -sf "http://localhost:${CH_STAGING_PORT}/?query=SELECT+1" >/dev/null 2>&1; then
    echo "ERROR: Staging ClickHouse not reachable at localhost:${CH_STAGING_PORT}"
    exit 1
  fi
  echo "Port-forwards ready."
else
  echo "=== Skipping port-forward setup ==="
  # If user manages port-forwards, they need to tell us the ports.
  KAFKA_PORT="${KAFKA_LOCAL_PORT:-$KAFKA_PORT}"
  CH_STAGING_PORT="${CH_STAGING_LOCAL_PORT:-$CH_STAGING_PORT}"
fi

# --- 2. Consume (unless --compare-only) ---
if [[ "$COMPARE_ONLY" != "true" ]]; then
  # Generate consumoor config.
  cat > "$CONFIG_FILE" <<YAML
logging: "debug"
metricsAddr: ":9091"

kafka:
  brokers:
    - host.docker.internal:${KAFKA_PORT}
  topics:
    - "${TOPICS}"
  consumerGroup: survey-${RUN_ID}
  encoding: protobuf
  offsetDefault: oldest
  commitInterval: 1s
  deliveryMode: message

clickhouse:
  dsn: "clickhouse://xatu-clickhouse-01:9000/consumoor"
  tableSuffix: "_local"
  chgo:
    maxConns: 32
    queryTimeout: 120s
  defaults:
    batchSize: 1000
    batchBytes: 10485760
    flushInterval: 1s
    bufferSize: 10000
    insertSettings:
      insert_quorum: 0
YAML

  echo "=== Starting local ClickHouse + consumoor ==="
  cd "$REPO_ROOT"
  docker compose -f docker-compose.yml \
    -f deploy/local/docker-compose/docker-compose.staging.yml \
    up -d --build

  # Wait for local ClickHouse.
  echo "Waiting for local ClickHouse..."
  for i in $(seq 1 60); do
    if curl -sf "http://localhost:8123/?query=SELECT+1" >/dev/null 2>&1; then
      echo "Local ClickHouse healthy."
      break
    fi
    if [[ "$i" -eq 60 ]]; then
      echo "ERROR: Local ClickHouse not healthy after 120s."
      exit 1
    fi
    sleep 2
  done

  # Wait for consumoor-init to finish (creates consumoor database).
  echo "Waiting for consumoor-init..."
  for i in $(seq 1 30); do
    init_status=$(docker inspect -f '{{.State.Status}}' xatu-clickhouse-consumoor-init 2>/dev/null || echo "unknown")
    if [[ "$init_status" == "exited" ]]; then
      echo "consumoor-init completed."
      break
    fi
    if [[ "$i" -eq 30 ]]; then
      echo "WARNING: consumoor-init may not have completed. Continuing anyway."
    fi
    sleep 2
  done

  echo "=== Consuming for ${CONSUME_SECONDS}s ==="
  sleep "$CONSUME_SECONDS"

  # Stop consumoor but keep ClickHouse running.
  echo "Stopping consumoor..."
  docker compose -f docker-compose.yml \
    -f deploy/local/docker-compose/docker-compose.staging.yml \
    stop xatu-consumoor 2>/dev/null || true
fi

# --- 3. Survey tables ---
echo ""
echo "=== Survey Results ==="
echo ""

# Known V2 events: these have V1 counterparts that are deprecated/empty.
# Data lands in the same table from the V2 topic.
# Format: v2_topic_suffix → v1_topic_suffix
declare -A V2_EVENTS=(
  ["beacon-api-eth-v1-events-attestation-v2"]="beacon-api-eth-v1-events-attestation"
  ["beacon-api-eth-v1-events-block-v2"]="beacon-api-eth-v1-events-block"
  ["beacon-api-eth-v1-events-chain-reorg-v2"]="beacon-api-eth-v1-events-chain-reorg"
  ["beacon-api-eth-v1-events-finalized-checkpoint-v2"]="beacon-api-eth-v1-events-finalized-checkpoint"
  ["beacon-api-eth-v1-events-head-v2"]="beacon-api-eth-v1-events-head"
  ["beacon-api-eth-v1-events-voluntary-exit-v2"]="beacon-api-eth-v1-events-voluntary-exit"
  ["beacon-api-eth-v1-events-contribution-and-proof-v2"]="beacon-api-eth-v1-events-contribution-and-proof"
  ["beacon-api-eth-v1-debug-fork-choice-v2"]="beacon-api-eth-v1-debug-fork-choice"
  ["beacon-api-eth-v1-debug-fork-choice-reorg-v2"]="beacon-api-eth-v1-debug-fork-choice-reorg"
  ["beacon-api-eth-v2-beacon-block-v2"]="beacon-api-eth-v2-beacon-block"
  ["mempool-transaction-v2"]="mempool-transaction"
)

# Get all consumoor _local tables.
tables=$(ch_local "SELECT name FROM system.tables WHERE database='consumoor' AND name LIKE '%_local' AND engine NOT IN ('MaterializedView') ORDER BY name")

if [[ -z "$tables" ]]; then
  echo "ERROR: No consumoor _local tables found. Did consumoor-init run?"
  exit 1
fi

# Print header.
printf "%-60s │ %7s │ %7s │ %-30s │ %s\n" "TABLE" "ROWS" "STG" "TOPIC" "NOTES"
printf "%-60s─┼─%7s─┼─%7s─┼─%-30s─┼─%s\n" \
  "────────────────────────────────────────────────────────────" \
  "───────" "───────" "──────────────────────────────" "──────────────"

matched=0
empty=0
no_stg=0
total=0

while IFS= read -r local_table; do
  [[ -z "$local_table" ]] && continue
  total=$((total + 1))

  base="${local_table%_local}"

  # Count consumoor rows.
  rows=$(ch_local "SELECT count() FROM consumoor.\`${local_table}\`" 2>/dev/null) || rows="0"

  # Infer topic name.
  topic=$(infer_topic "$base")

  # Check if this is a V2 event table.
  topic_short="${topic#xatu-protobuf-}"
  notes=""

  # Check if the inferred topic has a V2 variant.
  if [[ -n "${V2_EVENTS[${topic_short}-v2]+x}" ]] || [[ -n "${V2_EVENTS[${topic_short}]+x}" ]]; then
    if [[ -n "${V2_EVENTS[${topic_short}-v2]+x}" ]]; then
      notes="V1+V2 → same table"
      topic="${topic}-v2 (+ v1)"
    fi
  fi

  if [[ "$rows" == "0" ]]; then
    printf "%-60s │ %7s │ %7s │ %-30s │ %s\n" "$base" "0" "-" "$topic_short" "${notes:-no data}"
    empty=$((empty + 1))
    continue
  fi

  # Get time range for staging comparison.
  time_range=$(ch_local "SELECT min(event_date_time), max(event_date_time) FROM consumoor.\`${local_table}\`")
  min_time=$(echo "$time_range" | cut -f1)
  max_time=$(echo "$time_range" | cut -f2)

  # Check staging table (try distributed, then _local).
  stg_table="$base"
  stg_exists=$(ch_staging "SELECT count() FROM system.tables WHERE database='default' AND name='${stg_table}'" 2>/dev/null) || stg_exists="0"

  if [[ "$stg_exists" == "0" ]]; then
    stg_table="${local_table}"
    stg_exists=$(ch_staging "SELECT count() FROM system.tables WHERE database='default' AND name='${stg_table}'" 2>/dev/null) || stg_exists="0"
  fi

  if [[ "$stg_exists" == "0" ]]; then
    printf "%-60s │ %7s │ %7s │ %-30s │ %s\n" "$base" "$rows" "?" "$topic_short" "no staging table"
    no_stg=$((no_stg + 1))
    continue
  fi

  # Count staging rows in same time window.
  stg_rows=$(ch_staging "SELECT count() FROM default.\`${stg_table}\` WHERE event_date_time >= '${min_time}' AND event_date_time <= '${max_time}'" 2>/dev/null) || stg_rows="err"

  printf "%-60s │ %7s │ %7s │ %-30s │ %s\n" "$base" "$rows" "$stg_rows" "$topic_short" "$notes"
  matched=$((matched + 1))

done <<< "$tables"

echo ""
echo "=== Summary ==="
echo "  Tables with data:     $matched"
echo "  Tables empty:         $empty"
echo "  No staging table:     $no_stg"
echo "  Total tables checked: $total"
