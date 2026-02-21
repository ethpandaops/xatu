#!/usr/bin/env bash
# Staging correctness test for consumoor.
#
# Connects consumoor (local docker) to a staging/production Kafka cluster,
# lets it consume for a configurable duration, then compares its ClickHouse
# output against the reference ClickHouse (staging Vector output).
#
# Prerequisites:
#   - kubectl access to the target cluster
#   - Docker running locally
#   - Go toolchain for running the comparison test
#
# Usage:
#   ./staging-correctness-test.sh [flags]
#
# Flags:
#   --consume-duration SEC   How long to consume from Kafka (default: 300 = 5m)
#   --topics PATTERN         Kafka topic regex pattern (default: all proto topics)
#   --skip-portforward       Skip kubectl port-forward setup
#   --compare-only           Skip consuming, just run the comparison test
#   --help                   Show this help message
#
# Environment variables:
#   KUBE_CONTEXT             kubectl context (default: platform-analytics-hel1-staging)
#   KUBE_NAMESPACE           kubectl namespace (default: xatu)
#   KAFKA_SERVICE            Kafka service name (default: xatu-internal-kafka-kafka-bootstrap)
#   CLICKHOUSE_SERVICE       ClickHouse service name (default: chendpoint-xatu-clickhouse)
#   STAGING_CLICKHOUSE_USER  ClickHouse user for staging (default: empty)
#   STAGING_CLICKHOUSE_PASSWORD  ClickHouse password for staging (default: empty)
#   MAX_COMPARE_ROWS         Max rows to compare per table (default: 10000)
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"

# --- Defaults ---
CONSUME_DURATION="${CONSUME_DURATION:-300}"
TOPICS="${TOPICS:-^xatu-protobuf-.+}"
SKIP_PORTFORWARD=false
COMPARE_ONLY=false
KUBE_CONTEXT="${KUBE_CONTEXT:-platform-analytics-hel1-staging}"
KUBE_NAMESPACE="${KUBE_NAMESPACE:-xatu}"
KAFKA_SERVICE="${KAFKA_SERVICE:-svc/xatu-internal-kafka-kafka-bootstrap}"
CLICKHOUSE_SERVICE="${CLICKHOUSE_SERVICE:-svc/chendpoint-xatu-clickhouse}"
STAGING_CLICKHOUSE_USER="${STAGING_CLICKHOUSE_USER:-}"
STAGING_CLICKHOUSE_PASSWORD="${STAGING_CLICKHOUSE_PASSWORD:-}"
MAX_COMPARE_ROWS="${MAX_COMPARE_ROWS:-10000}"
RUN_ID="$(date +%s)"

# Random ports in 20000s to avoid conflicts.
KAFKA_LOCAL_PORT="${KAFKA_LOCAL_PORT:-$((20000 + RANDOM % 1000))}"
CH_LOCAL_PORT="${CH_LOCAL_PORT:-$((21000 + RANDOM % 1000))}"

# --- Parse flags ---
while [[ $# -gt 0 ]]; do
  case "$1" in
    --consume-duration)
      CONSUME_DURATION="$2"
      shift 2
      ;;
    --topics)
      TOPICS="$2"
      shift 2
      ;;
    --skip-portforward)
      SKIP_PORTFORWARD=true
      shift
      ;;
    --compare-only)
      COMPARE_ONLY=true
      shift
      ;;
    --help)
      head -35 "$0" | tail -30
      echo ""
      echo "Examples:"
      echo "  # Run against staging with default settings (all topics, 5m consume):"
      echo "  ./staging-correctness-test.sh"
      echo ""
      echo "  # Quick test with only head events for 2 minutes:"
      echo '  ./staging-correctness-test.sh --topics "^xatu-protobuf-beacon-api-eth-v1-events-head" --consume-duration 120'
      echo ""
      echo "  # Run comparison only (consumoor already consumed):"
      echo "  ./staging-correctness-test.sh --compare-only"
      echo ""
      echo "  # Run against production:"
      echo "  KUBE_CONTEXT=platform-analytics-hel1-production ./staging-correctness-test.sh"
      exit 0
      ;;
    *)
      echo "Unknown flag: $1"
      exit 1
      ;;
  esac
done

echo "=== Staging Correctness Test ==="
echo "  Kafka port-forward:     localhost:${KAFKA_LOCAL_PORT}"
echo "  Staging CH port-forward: localhost:${CH_LOCAL_PORT}"
echo "  Topics:                 ${TOPICS}"
echo "  Consume duration:       ${CONSUME_DURATION}s"
echo ""

# --- PID tracking for cleanup ---
PF_PIDS=()

cleanup() {
  echo ""
  echo "=== Cleaning up ==="

  if [[ "$COMPARE_ONLY" != "true" ]]; then
    echo "Stopping docker compose..."
    cd "$REPO_ROOT"
    docker compose -f docker-compose.yml \
      -f deploy/local/docker-compose/docker-compose.staging.yml \
      down --timeout 10 2>/dev/null || true
  fi

  for pid in "${PF_PIDS[@]}"; do
    if kill -0 "$pid" 2>/dev/null; then
      echo "Killing port-forward (PID $pid)"
      kill "$pid" 2>/dev/null || true
    fi
  done

  rm -f "$SCRIPT_DIR/xatu-consumoor-staging.yaml"
  echo "Cleanup complete."
}

trap cleanup EXIT

# --- Port forwarding ---
if [[ "$SKIP_PORTFORWARD" != "true" ]]; then
  echo "=== Starting port-forwards (context: $KUBE_CONTEXT, namespace: $KUBE_NAMESPACE) ==="

  kubectl --context "$KUBE_CONTEXT" -n "$KUBE_NAMESPACE" \
    port-forward "$KAFKA_SERVICE" "${KAFKA_LOCAL_PORT}:9092" &
  PF_PIDS+=($!)

  kubectl --context "$KUBE_CONTEXT" -n "$KUBE_NAMESPACE" \
    port-forward "$CLICKHOUSE_SERVICE" "${CH_LOCAL_PORT}:8123" &
  PF_PIDS+=($!)

  echo "Waiting for port-forwards to establish..."
  sleep 3

  # Verify staging ClickHouse is reachable.
  if ! curl -sf "http://localhost:${CH_LOCAL_PORT}/?query=SELECT+1" >/dev/null 2>&1; then
    echo "ERROR: Staging ClickHouse not reachable at localhost:${CH_LOCAL_PORT}"
    exit 1
  fi

  echo "Port-forwards established."
else
  echo "=== Skipping port-forward setup ==="
fi

# --- Generate consumoor config ---
if [[ "$COMPARE_ONLY" != "true" ]]; then
  echo "=== Generating consumoor staging config ==="
  echo "  Topics: $TOPICS"
  echo "  Consumer group: correctness-test-consumoor-${RUN_ID}"

  cat > "$SCRIPT_DIR/xatu-consumoor-staging.yaml" <<YAML
logging: "debug"
metricsAddr: ":9091"

kafka:
  brokers:
    - host.docker.internal:${KAFKA_LOCAL_PORT}
  topics:
    - "${TOPICS}"
  consumerGroup: correctness-test-consumoor-${RUN_ID}
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

  echo "Config written."

  # --- Start docker compose ---
  echo "=== Starting docker compose (staging overlay) ==="
  cd "$REPO_ROOT"
  docker compose -f docker-compose.yml \
    -f deploy/local/docker-compose/docker-compose.staging.yml \
    up -d

  # Wait for ClickHouse to be healthy.
  echo "Waiting for local ClickHouse to be healthy..."

  for i in $(seq 1 30); do
    if curl -sf "http://localhost:8123/?query=SELECT+1" >/dev/null 2>&1; then
      echo "Local ClickHouse is healthy."
      break
    fi

    if [[ "$i" -eq 30 ]]; then
      echo "ERROR: Local ClickHouse not healthy after 30 attempts."
      exit 1
    fi

    sleep 2
  done

  # --- Consume ---
  echo "=== Consuming from staging Kafka for ${CONSUME_DURATION}s ==="
  echo "  (started at $(date))"
  sleep "$CONSUME_DURATION"
  echo "Consume duration elapsed."
fi

# --- Run comparison test ---
echo "=== Running comparison test ==="
cd "$REPO_ROOT"

CONSUMOOR_STAGING_TEST=true \
CLICKHOUSE_URL="http://localhost:8123" \
STAGING_CLICKHOUSE_URL="http://localhost:${CH_LOCAL_PORT}" \
STAGING_CLICKHOUSE_USER="${STAGING_CLICKHOUSE_USER}" \
STAGING_CLICKHOUSE_PASSWORD="${STAGING_CLICKHOUSE_PASSWORD}" \
MAX_COMPARE_ROWS="${MAX_COMPARE_ROWS}" \
go test ./pkg/consumoor/sinks/clickhouse/transform/flattener/ \
  -run TestStagingCorrectness -v -timeout 300s

echo "=== Done ==="
