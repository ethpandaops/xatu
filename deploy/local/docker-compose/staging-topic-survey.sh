#!/usr/bin/env bash
# Survey which topics/tables consumoor handles from staging Kafka.
#
# Builds consumoor as a native binary and connects it to staging Kafka via
# per-broker port-forwards (same approach as staging-correctness-test.sh).
# After consuming briefly, reports which ClickHouse tables received data and
# compares row counts against the staging reference.
#
# Usage:
#   ./staging-topic-survey.sh [flags]
#
# Flags:
#   --consume-seconds SEC    How long to consume (default: 10)
#   --topics PATTERN         Kafka topic regex (default: ^xatu-protobuf-.+)
#   --skip-build             Skip building the consumoor binary
#   --skip-portforward       Skip kubectl port-forward setup
#   --compare-only           Skip consuming, just survey existing data
#   --help                   Show this help
#
# Environment:
#   KUBE_CONTEXT             (default: platform-analytics-hel1-staging)
#   KUBE_NAMESPACE           (default: xatu)
#   CLICKHOUSE_SERVICE       (default: svc/chendpoint-xatu-clickhouse)
#   STAGING_CLICKHOUSE_USER  (default: empty)
#   STAGING_CLICKHOUSE_PASSWORD (default: empty)
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"

# --- Defaults ---
CONSUME_SECONDS="${CONSUME_SECONDS:-10}"
TOPICS="${TOPICS:-^xatu-protobuf-.+}"
SKIP_BUILD=false
SKIP_PORTFORWARD=false
COMPARE_ONLY=false
KUBE_CONTEXT="${KUBE_CONTEXT:-platform-analytics-hel1-staging}"
KUBE_NAMESPACE="${KUBE_NAMESPACE:-xatu}"
CLICKHOUSE_SERVICE="${CLICKHOUSE_SERVICE:-svc/chendpoint-xatu-clickhouse}"
STAGING_CLICKHOUSE_USER="${STAGING_CLICKHOUSE_USER:-}"
STAGING_CLICKHOUSE_PASSWORD="${STAGING_CLICKHOUSE_PASSWORD:-}"

# Broker discovery settings.
BROKER_POD_LABEL="${BROKER_POD_LABEL:-strimzi.io/name=xatu-internal-kafka-kafka}"
BROKER_SVC_SUFFIX="${BROKER_SVC_SUFFIX:-.xatu-internal-kafka-kafka-brokers.xatu.svc}"
BROKER_PORT=9092

# Paths.
XATU_BINARY="/tmp/xatu-staging-test-$$"
CONFIG_FILE="$SCRIPT_DIR/xatu-consumoor-staging.yaml"

# Random port in 21000s for staging ClickHouse.
CH_STAGING_PORT=$((21000 + RANDOM % 1000))

RUN_ID="$(date +%s)"

# --- Parse flags ---
while [[ $# -gt 0 ]]; do
  case "$1" in
    --consume-seconds)  CONSUME_SECONDS="$2"; shift 2 ;;
    --topics)           TOPICS="$2"; shift 2 ;;
    --skip-build)       SKIP_BUILD=true; shift ;;
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
CONSUMOOR_PID=""

cleanup() {
  echo ""
  echo "=== Cleaning up ==="

  # Kill consumoor if running.
  if [[ -n "$CONSUMOOR_PID" ]] && kill -0 "$CONSUMOOR_PID" 2>/dev/null; then
    echo "Stopping consumoor (PID $CONSUMOOR_PID)..."
    kill "$CONSUMOOR_PID" 2>/dev/null || true
    wait "$CONSUMOOR_PID" 2>/dev/null || true
  fi

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
  rm -f "$XATU_BINARY"
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

# --- Build binary ---
if [[ "$COMPARE_ONLY" != "true" && "$SKIP_BUILD" != "true" ]]; then
  echo "=== Building consumoor binary ==="
  cd "$REPO_ROOT"
  go build -o "$XATU_BINARY" .
  echo "Binary built: $XATU_BINARY"
elif [[ "$SKIP_BUILD" == "true" && ! -f "$XATU_BINARY" ]]; then
  EXISTING=$(ls /tmp/xatu-staging-test-* 2>/dev/null | head -1 || true)
  if [[ -n "$EXISTING" ]]; then
    XATU_BINARY="$EXISTING"
    echo "Reusing existing binary: $XATU_BINARY"
  else
    echo "ERROR: --skip-build specified but no binary found. Run without --skip-build first."
    exit 1
  fi
fi

# --- Discover broker pods and set up port-forwards ---
if [[ "$SKIP_PORTFORWARD" != "true" ]]; then
  echo "=== Discovering Kafka broker pods (context: $KUBE_CONTEXT, namespace: $KUBE_NAMESPACE) ==="

  BROKER_PODS=$(kubectl --context "$KUBE_CONTEXT" -n "$KUBE_NAMESPACE" \
    get pods -l "$BROKER_POD_LABEL" -o jsonpath='{.items[*].metadata.name}')

  if [[ -z "$BROKER_PODS" ]]; then
    echo "ERROR: No broker pods found with label $BROKER_POD_LABEL"
    exit 1
  fi

  read -ra BROKER_POD_ARRAY <<< "$BROKER_PODS"
  NUM_BROKERS=${#BROKER_POD_ARRAY[@]}
  echo "Found $NUM_BROKERS broker pods: ${BROKER_POD_ARRAY[*]}"

  # Pre-flight: verify loopback aliases and /etc/hosts are set up.
  MISSING_SETUP=false
  for i in "${!BROKER_POD_ARRAY[@]}"; do
    loopback_ip="127.0.0.$((i + 2))"
    if ! ifconfig lo0 | grep -q "inet $loopback_ip "; then
      echo "ERROR: Missing loopback alias for $loopback_ip"
      MISSING_SETUP=true
    fi
  done

  if [[ "$MISSING_SETUP" == "true" ]]; then
    echo ""
    echo "Loopback aliases are missing (they don't survive reboot). Run this once:"
    echo ""
    for i in "${!BROKER_POD_ARRAY[@]}"; do
      echo "  sudo ifconfig lo0 alias 127.0.0.$((i + 2))"
    done
    echo ""
    echo "Also ensure /etc/hosts has entries mapping broker hostnames to these IPs:"
    echo ""
    for i in "${!BROKER_POD_ARRAY[@]}"; do
      pod="${BROKER_POD_ARRAY[$i]}"
      echo "  127.0.0.$((i + 2)) ${pod}${BROKER_SVC_SUFFIX}"
    done
    exit 1
  fi

  echo ""
  echo "=== Starting port-forwards ==="

  # Port-forward each broker to its loopback IP.
  SEED_BROKER=""
  for i in "${!BROKER_POD_ARRAY[@]}"; do
    pod="${BROKER_POD_ARRAY[$i]}"
    loopback_ip="127.0.0.$((i + 2))"

    echo "  $pod → $loopback_ip:$BROKER_PORT"

    kubectl --context "$KUBE_CONTEXT" -n "$KUBE_NAMESPACE" \
      port-forward --address "$loopback_ip" "pod/$pod" "${BROKER_PORT}:${BROKER_PORT}" &
    PF_PIDS+=($!)

    if [[ -z "$SEED_BROKER" ]]; then
      SEED_BROKER="${loopback_ip}:${BROKER_PORT}"
    fi
  done

  # Port-forward staging ClickHouse.
  echo "  Staging ClickHouse → localhost:$CH_STAGING_PORT"
  kubectl --context "$KUBE_CONTEXT" -n "$KUBE_NAMESPACE" \
    port-forward "$CLICKHOUSE_SERVICE" "${CH_STAGING_PORT}:8123" &
  PF_PIDS+=($!)

  echo ""
  echo "Waiting for port-forwards to establish..."
  for i in $(seq 1 15); do
    if curl -sf "http://localhost:${CH_STAGING_PORT}/?query=SELECT+1" >/dev/null 2>&1; then
      echo "Port-forwards ready."
      break
    fi
    if [[ "$i" -eq 15 ]]; then
      echo "ERROR: Staging ClickHouse not reachable at localhost:${CH_STAGING_PORT} after 15s"
      exit 1
    fi
    sleep 1
  done
else
  echo "=== Skipping port-forward setup ==="
  SEED_BROKER="${SEED_BROKER:-127.0.0.2:${BROKER_PORT}}"
  CH_STAGING_PORT="${CH_STAGING_LOCAL_PORT:-$CH_STAGING_PORT}"
fi

# --- Banner ---
echo ""
echo "=== Staging Topic Survey ==="
echo "  Seed broker:            ${SEED_BROKER}"
echo "  Staging CH port-forward: localhost:${CH_STAGING_PORT}"
echo "  Topics pattern:         ${TOPICS}"
echo "  Consume duration:       ${CONSUME_SECONDS}s"
echo ""

# --- Consume (unless --compare-only) ---
if [[ "$COMPARE_ONLY" != "true" ]]; then
  # Generate consumoor config.
  cat > "$CONFIG_FILE" <<YAML
logging: "debug"
metricsAddr: ":9091"

kafka:
  brokers:
    - ${SEED_BROKER}
  topics:
    - "${TOPICS}"
  consumerGroup: survey-${RUN_ID}
  encoding: protobuf
  offsetDefault: earliest
  commitInterval: 1s

clickhouse:
  dsn: "clickhouse://localhost:9000/consumoor"
  tableSuffix: "_local"
  chgo:
    maxConns: 32
    queryTimeout: 120s
  defaults:
    batchSize: 1000
    flushInterval: 1s
    bufferSize: 10000
    insertSettings:
      insert_quorum: 0
YAML

  echo "=== Starting local ClickHouse ==="
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

  # Start consumoor as native binary.
  echo "=== Starting consumoor (native binary) ==="
  "$XATU_BINARY" consumoor --config "$CONFIG_FILE" &
  CONSUMOOR_PID=$!

  sleep 3
  if ! kill -0 "$CONSUMOOR_PID" 2>/dev/null; then
    echo "ERROR: consumoor exited immediately. Check output above for errors."
    exit 1
  fi
  echo "consumoor started (PID $CONSUMOOR_PID)."

  echo "=== Consuming for ${CONSUME_SECONDS}s ==="
  sleep "$CONSUME_SECONDS"

  # Stop consumoor but keep ClickHouse running for the survey.
  echo "Stopping consumoor..."
  kill "$CONSUMOOR_PID" 2>/dev/null || true
  wait "$CONSUMOOR_PID" 2>/dev/null || true
  CONSUMOOR_PID=""
fi

# --- Survey tables ---
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
