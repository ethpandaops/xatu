#!/usr/bin/env bash
# Staging correctness test for consumoor.
#
# Builds consumoor as a native binary and connects it to a staging/production
# Kafka cluster via per-broker port-forwards. Each broker pod is forwarded to
# a unique loopback IP (127.0.0.{2,3,4,...}:9092) and /etc/hosts entries map
# the broker's internal K8s DNS name to that IP.
#
# After consuming for a configurable duration, the script compares consumoor's
# ClickHouse output against the reference ClickHouse (staging Vector output).
#
# Prerequisites:
#   - kubectl access to the target cluster
#   - Docker running locally (for ClickHouse)
#   - Go toolchain for building consumoor and running the comparison test
#   - Loopback aliases + /etc/hosts for Kafka brokers (see one-time setup below)
#
# Usage:
#   ./staging-correctness-test.sh [flags]
#
# Flags:
#   --consume-duration SEC   How long to consume from Kafka (default: 300 = 5m)
#   --topics PATTERN         Kafka topic regex pattern (default: all proto topics)
#   --skip-build             Skip building the consumoor binary
#   --skip-portforward       Skip kubectl port-forward setup
#   --compare-only           Skip consuming, just run the comparison test
#   --help                   Show this help message
#
# Environment variables:
#   KUBE_CONTEXT             kubectl context (default: platform-analytics-hel1-staging)
#   KUBE_NAMESPACE           kubectl namespace (default: xatu)
#   CLICKHOUSE_SERVICE       ClickHouse service name (default: svc/chendpoint-xatu-clickhouse)
#   STAGING_CLICKHOUSE_USER  ClickHouse user for staging (default: empty)
#   STAGING_CLICKHOUSE_PASSWORD  ClickHouse password for staging (default: empty)
#   MAX_COMPARE_ROWS         Max rows to compare per table (default: 10000)
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"

# --- Defaults ---
CONSUME_DURATION="${CONSUME_DURATION:-300}"
TOPICS="${TOPICS:-^xatu-protobuf-.+}"
SKIP_BUILD=false
SKIP_PORTFORWARD=false
COMPARE_ONLY=false
KUBE_CONTEXT="${KUBE_CONTEXT:-platform-analytics-hel1-staging}"
KUBE_NAMESPACE="${KUBE_NAMESPACE:-xatu}"
CLICKHOUSE_SERVICE="${CLICKHOUSE_SERVICE:-svc/chendpoint-xatu-clickhouse}"
STAGING_CLICKHOUSE_USER="${STAGING_CLICKHOUSE_USER:-}"
STAGING_CLICKHOUSE_PASSWORD="${STAGING_CLICKHOUSE_PASSWORD:-}"
MAX_COMPARE_ROWS="${MAX_COMPARE_ROWS:-10000}"
RUN_ID="$(date +%s)"

# Broker discovery settings.
BROKER_POD_LABEL="${BROKER_POD_LABEL:-strimzi.io/name=xatu-internal-kafka-kafka}"
BROKER_SVC_SUFFIX="${BROKER_SVC_SUFFIX:-.xatu-internal-kafka-kafka-brokers.xatu.svc}"
BROKER_PORT=9092

# Paths.
XATU_BINARY="/tmp/xatu-staging-test-$$"
CONFIG_FILE="$SCRIPT_DIR/xatu-consumoor-staging.yaml"

# Random port in 21000s for staging ClickHouse.
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
    --skip-build)
      SKIP_BUILD=true
      shift
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
      head -37 "$0" | tail -32
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
      echo "  # Skip rebuilding binary for fast iteration:"
      echo "  ./staging-correctness-test.sh --skip-build --consume-duration 60"
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

# --- PID / resource tracking for cleanup ---
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

  # Docker compose down.
  if [[ "$COMPARE_ONLY" != "true" ]]; then
    echo "Stopping docker compose..."
    cd "$REPO_ROOT"
    docker compose -f docker-compose.yml \
      -f deploy/local/docker-compose/docker-compose.staging.yml \
      down --timeout 10 2>/dev/null || true
  fi

  # Kill port-forwards.
  for pid in "${PF_PIDS[@]}"; do
    if kill -0 "$pid" 2>/dev/null; then
      echo "Killing port-forward (PID $pid)"
      kill "$pid" 2>/dev/null || true
    fi
  done

  # Remove generated files.
  rm -f "$CONFIG_FILE"
  rm -f "$XATU_BINARY"

  echo "Cleanup complete."
}

trap cleanup EXIT

# --- Build binary ---
if [[ "$COMPARE_ONLY" != "true" && "$SKIP_BUILD" != "true" ]]; then
  echo "=== Building consumoor binary ==="
  cd "$REPO_ROOT"
  go build -o "$XATU_BINARY" .
  echo "Binary built: $XATU_BINARY"
elif [[ "$SKIP_BUILD" == "true" && ! -f "$XATU_BINARY" ]]; then
  # If --skip-build but no binary exists, check for a previous build.
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

  # Convert to array.
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
  echo "  Staging ClickHouse → localhost:$CH_LOCAL_PORT"
  kubectl --context "$KUBE_CONTEXT" -n "$KUBE_NAMESPACE" \
    port-forward "$CLICKHOUSE_SERVICE" "${CH_LOCAL_PORT}:8123" &
  PF_PIDS+=($!)

  echo ""
  echo "Waiting for port-forwards to establish..."
  for i in $(seq 1 15); do
    if curl -sf "http://localhost:${CH_LOCAL_PORT}/?query=SELECT+1" >/dev/null 2>&1; then
      echo "Port-forwards established."
      break
    fi
    if [[ "$i" -eq 15 ]]; then
      echo "ERROR: Staging ClickHouse not reachable at localhost:${CH_LOCAL_PORT} after 15s"
      exit 1
    fi
    sleep 1
  done
else
  echo "=== Skipping port-forward setup ==="
  SEED_BROKER="${SEED_BROKER:-127.0.0.2:${BROKER_PORT}}"
fi

echo ""
echo "=== Staging Correctness Test ==="
echo "  Seed broker:            ${SEED_BROKER}"
echo "  Staging CH port-forward: localhost:${CH_LOCAL_PORT}"
echo "  Topics:                 ${TOPICS}"
echo "  Consume duration:       ${CONSUME_DURATION}s"
echo ""

# --- Start docker compose (ClickHouse + init only; consumoor is disabled) ---
if [[ "$COMPARE_ONLY" != "true" ]]; then
  echo "=== Generating consumoor staging config ==="
  echo "  Topics: $TOPICS"
  echo "  Consumer group: correctness-test-consumoor-${RUN_ID}"

  cat > "$CONFIG_FILE" <<YAML
logging: "debug"
metricsAddr: ":9091"

kafka:
  brokers:
    - ${SEED_BROKER}
  topics:
    - "${TOPICS}"
  consumerGroup: correctness-test-consumoor-${RUN_ID}
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

  echo "Config written."

  echo "=== Starting docker compose (ClickHouse + init only) ==="
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

  # --- Run consumoor as native binary ---
  echo "=== Starting consumoor (native binary) ==="
  "$XATU_BINARY" consumoor --config "$CONFIG_FILE" &
  CONSUMOOR_PID=$!

  # Give it a moment to start and check it's still alive.
  sleep 3
  if ! kill -0 "$CONSUMOOR_PID" 2>/dev/null; then
    echo "ERROR: consumoor exited immediately. Check output above for errors."
    exit 1
  fi
  echo "consumoor started (PID $CONSUMOOR_PID)."

  # --- Consume ---
  echo "=== Consuming from staging Kafka for ${CONSUME_DURATION}s ==="
  echo "  (started at $(date))"
  sleep "$CONSUME_DURATION"
  echo "Consume duration elapsed."

  # Stop consumoor.
  echo "Stopping consumoor..."
  kill "$CONSUMOOR_PID" 2>/dev/null || true
  wait "$CONSUMOOR_PID" 2>/dev/null || true
  CONSUMOOR_PID=""
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
