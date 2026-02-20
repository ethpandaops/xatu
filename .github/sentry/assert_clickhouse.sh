#!/bin/bash

# Sentry smoke test assertion script
#
# This script verifies that data has flowed through the entire pipeline:
# Beacon Node -> Sentry -> Xatu Server -> Kafka -> Vector -> ClickHouse
#
# It checks that each event type in the seeding config has data in ClickHouse
# matching the sentry's client name. Uses simple "count > 0" assertions since
# we're testing the pipeline, not specific data values.

# Check if seeding.yaml location is provided
if [ $# -eq 0 ]; then
    echo "Error: Please provide the location of seeding.yaml as an argument."
    exit 1
fi

SEEDING_YAML="$1"

# Environment variables (set by workflow)
SENTRY_NAME="${SENTRY_NAME:?SENTRY_NAME environment variable is required}"

# ClickHouse connection settings
CLICKHOUSE_HOST=${CLICKHOUSE_HOST:-"localhost"}
CLICKHOUSE_PORT=${CLICKHOUSE_PORT:-"9000"}
CLICKHOUSE_USER=${CLICKHOUSE_USER:-"default"}
CLICKHOUSE_PASSWORD=${CLICKHOUSE_PASSWORD:-""}
CLICKHOUSE_DB=${CLICKHOUSE_DB:-"default"}

echo "============================================"
echo "Sentry Smoke Test Assertions"
echo "============================================"
echo "Sentry name: $SENTRY_NAME"
echo "ClickHouse: $CLICKHOUSE_HOST:$CLICKHOUSE_PORT"
echo "============================================"

# Function to execute ClickHouse query
execute_query() {
    if command -v clickhouse-client &> /dev/null; then
        clickhouse-client -h "$CLICKHOUSE_HOST" --port "$CLICKHOUSE_PORT" -u "$CLICKHOUSE_USER" --password "$CLICKHOUSE_PASSWORD" -d "$CLICKHOUSE_DB" -q "$1"
    else
        clickhouse client -h "$CLICKHOUSE_HOST" --port "$CLICKHOUSE_PORT" -u "$CLICKHOUSE_USER" --password "$CLICKHOUSE_PASSWORD" -d "$CLICKHOUSE_DB" -q "$1"
    fi
}

# Check if the seeding.yaml file exists
if [ ! -f "$SEEDING_YAML" ]; then
    echo "Error: seeding.yaml file not found at $SEEDING_YAML"
    exit 1
fi

# Function to assert a single event type
assert_event_type() {
    local NAME="$1"
    local TABLE="$2"
    local DESC="$3"

    echo ""
    echo "Asserting: $NAME"
    echo "  Table: $TABLE"
    echo "  Description: $DESC"

    local MAX_RETRIES=60
    local RETRY_INTERVAL=5
    local RETRY_COUNT=0

    while true; do
        # Check if table exists before querying
        local TABLE_EXISTS
        TABLE_EXISTS=$(execute_query "EXISTS TABLE $TABLE" 2>&1) || true

        if [ "$TABLE_EXISTS" != "1" ]; then
            RETRY_COUNT=$((RETRY_COUNT + 1))
            if [ $RETRY_COUNT -ge $MAX_RETRIES ]; then
                echo "  ✗ FAILED: Table $TABLE does not exist after $MAX_RETRIES retries"
                echo "  Recent sentry logs:"
                docker logs xatu-sentry 2>&1 | tail -n 10 || echo "  (no sentry logs)"
                return 1
            fi

            echo "  Table $TABLE does not exist yet (attempt $RETRY_COUNT/$MAX_RETRIES). Waiting ${RETRY_INTERVAL}s..."
            sleep $RETRY_INTERVAL
            continue
        fi

        # Query for data matching our sentry name
        local QUERY="SELECT COUNT(*) FROM $TABLE WHERE meta_client_name = '$SENTRY_NAME'"
        echo "  Executing: $QUERY"
        local RESULT
        RESULT=$(execute_query "$QUERY" 2>&1) || true

        if [ -n "$RESULT" ] && [ "$RESULT" != "0" ] && [[ "$RESULT" =~ ^[0-9]+$ ]]; then
            echo "  Result: $RESULT rows"
            echo "  ✓ PASSED: Found $RESULT records for $NAME"
            return 0
        else
            RETRY_COUNT=$((RETRY_COUNT + 1))
            if [ $RETRY_COUNT -ge $MAX_RETRIES ]; then
                echo "  ✗ FAILED: No data found for $NAME after $MAX_RETRIES retries"
                echo "  Last result: $RESULT"

                # Show recent sentry logs for debugging
                echo "  Recent sentry logs:"
                docker logs xatu-sentry 2>&1 | tail -n 10 || echo "  (no sentry logs)"

                return 1
            fi

            echo "  No data yet (attempt $RETRY_COUNT/$MAX_RETRIES). Waiting ${RETRY_INTERVAL}s..."
            docker logs xatu-sentry 2>&1 | grep -i "event" | tail -n 3 || true
            sleep $RETRY_INTERVAL
        fi
    done
}

# Track failures
FAILED_TYPES=()

# Get event types from config
EVENT_COUNT=$(yq '.event_types | length' "$SEEDING_YAML")

for ((i=0; i<EVENT_COUNT; i++)); do
    NAME=$(yq ".event_types[$i].name" "$SEEDING_YAML")
    TABLE=$(yq ".event_types[$i].table" "$SEEDING_YAML")
    DESC=$(yq ".event_types[$i].description" "$SEEDING_YAML")

    if ! assert_event_type "$NAME" "$TABLE" "$DESC"; then
        FAILED_TYPES+=("$NAME")
    fi
done

echo ""
echo "============================================"
echo "Summary"
echo "============================================"

if [ ${#FAILED_TYPES[@]} -eq 0 ]; then
    echo "✓ All $EVENT_COUNT assertions passed!"
    exit 0
else
    echo "✗ ${#FAILED_TYPES[@]} of $EVENT_COUNT assertions failed:"
    for TYPE in "${FAILED_TYPES[@]}"; do
        echo "  - $TYPE"
    done
    exit 1
fi
