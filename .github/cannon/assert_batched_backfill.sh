#!/bin/bash

# Asserts the batched backfill deriver from seeding.yaml walked from
# SEED_EPOCH-1 down to BACKFILL_TO_EPOCH in multi-epoch steps: its coordinator
# marker stops exactly on the floor, every epoch of the range has rows, nothing
# below the floor was written, and the logged steps carried whole batches.

set -euo pipefail

if [ $# -eq 0 ]; then
    echo "Error: Please provide the location of seeding.yaml as an argument."
    exit 1
fi

SEEDING_YAML="$1"

SEED_EPOCH="${SEED_EPOCH:?SEED_EPOCH environment variable is required}"
BACKFILL_TO_EPOCH="${BACKFILL_TO_EPOCH:?BACKFILL_TO_EPOCH environment variable is required}"
NETWORK_ID="${NETWORK_ID:?NETWORK_ID environment variable is required}"
NETWORK_NAME="${NETWORK_NAME:?NETWORK_NAME environment variable is required}"

TYPE=$(yq '.batched_backfill.type' "$SEEDING_YAML")
TABLE=$(yq '.batched_backfill.table' "$SEEDING_YAML")
EPOCHS=$(yq '.batched_backfill.epochs' "$SEEDING_YAML")
PER_BATCH=$(yq '.batched_backfill.epochs_per_batch' "$SEEDING_YAML")
HIGHEST=$((SEED_EPOCH - 1))

echo "============================================"
echo "Batched backfill assertions"
echo "============================================"
echo "Type: $TYPE ($TABLE)"
echo "Range: $HIGHEST down to $BACKFILL_TO_EPOCH ($EPOCHS epochs, $PER_BATCH per batch)"
echo "============================================"

ch() {
    clickhouse-client -h localhost --port 9000 -u default --password "" -d default -q "$1" </dev/null
}

marker() {
    docker exec xatu-postgres psql -U user -d xatu -tAc \
        "SELECT value::jsonb->'backfillingCheckpointMarker'->>'backfillEpoch' FROM cannon_location WHERE network_id = '$NETWORK_ID' AND type = '$TYPE'" |
        tr -d '[:space:]'
}

show_logs() {
    echo "=== $TYPE cannon logs (last 40) ==="
    docker logs xatu-cannon 2>&1 | grep "$TYPE" | tail -n 40 || true
}

MAX_ATTEMPTS=100

for i in $(seq 1 "$MAX_ATTEMPTS"); do
    CURRENT=$(marker)

    if [ "$CURRENT" = "$BACKFILL_TO_EPOCH" ]; then
        echo "Backfill marker reached $BACKFILL_TO_EPOCH after $i attempt(s)"
        break
    fi

    if [ "$i" -eq "$MAX_ATTEMPTS" ]; then
        echo "::error::Backfill marker is ${CURRENT:-unset}, expected $BACKFILL_TO_EPOCH"
        show_logs
        exit 1
    fi

    echo "  backfill marker at ${CURRENT:-unset}, waiting for $BACKFILL_TO_EPOCH (attempt $i/$MAX_ATTEMPTS)..."
    sleep 5
done

FAILED=0

# The floor is final: the marker must not move below it.
sleep 15

CURRENT=$(marker)
if [ "$CURRENT" != "$BACKFILL_TO_EPOCH" ]; then
    echo "✗ Backfill marker moved past the floor to $CURRENT"
    FAILED=1
fi

echo ""
echo "Rows per backfilled epoch:"
ch "SELECT epoch, count() AS rows FROM $TABLE FINAL WHERE meta_network_name = '$NETWORK_NAME' AND epoch BETWEEN $BACKFILL_TO_EPOCH AND $HIGHEST GROUP BY epoch ORDER BY epoch FORMAT PrettyCompact"

COVERED=$(ch "SELECT uniqExact(epoch) FROM $TABLE FINAL WHERE meta_network_name = '$NETWORK_NAME' AND epoch BETWEEN $BACKFILL_TO_EPOCH AND $HIGHEST")
if [ "$COVERED" -ne "$EPOCHS" ]; then
    echo "✗ $COVERED of $EPOCHS epochs in $BACKFILL_TO_EPOCH..$HIGHEST have rows"
    FAILED=1
fi

MAX_ROWS=$(ch "SELECT max(rows) FROM (SELECT count() AS rows FROM $TABLE FINAL WHERE meta_network_name = '$NETWORK_NAME' AND epoch BETWEEN $BACKFILL_TO_EPOCH AND $HIGHEST GROUP BY epoch)")
if [ "$MAX_ROWS" -gt 32 ]; then
    echo "✗ An epoch has $MAX_ROWS rows, more than one per slot"
    FAILED=1
fi

BELOW=$(ch "SELECT count() FROM $TABLE FINAL WHERE meta_network_name = '$NETWORK_NAME' AND epoch < $BACKFILL_TO_EPOCH")
if [ "$BELOW" -ne 0 ]; then
    echo "✗ $BELOW rows below the backfill floor $BACKFILL_TO_EPOCH"
    FAILED=1
fi

STEPS=$(docker logs xatu-cannon 2>&1 |
    jq -rR --arg t "$TYPE" 'fromjson? | select(.msg == "Derived next backfill epoch to process" and .cannon_type == $t) | "\(.next_epoch) \(.last_epoch)"' |
    sort -u)

echo ""
echo "Backfill steps (first epoch, last epoch):"
echo "$STEPS"

FIRST_BATCH="$HIGHEST $((HIGHEST - PER_BATCH + 1))"
if ! grep -qx "$FIRST_BATCH" <<<"$STEPS"; then
    echo "✗ No step covered the first batch $FIRST_BATCH"
    FAILED=1
fi

if ! grep -q " $BACKFILL_TO_EPOCH\$" <<<"$STEPS"; then
    echo "✗ No step ended on the floor $BACKFILL_TO_EPOCH"
    FAILED=1
fi

if [ "$FAILED" -ne 0 ]; then
    show_logs
    exit 1
fi

echo "✓ $TYPE backfilled $EPOCHS epochs in batches of $PER_BATCH and stopped on $BACKFILL_TO_EPOCH"
