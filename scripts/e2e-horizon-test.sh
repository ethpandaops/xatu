#!/bin/bash
#
# Horizon E2E Test Script
#
# This script runs an end-to-end test of the Horizon module using Kurtosis
# to spin up a local Ethereum testnet with all consensus clients.
#
# The test verifies that data flows through the entire pipeline:
# Beacon Nodes (via SSE) -> Horizon -> Xatu Server -> Kafka -> Vector -> ClickHouse
#
# Usage:
#   ./scripts/e2e-horizon-test.sh [--quick] [--skip-build] [--skip-cleanup]
#
# Options:
#   --quick        Run quick test (1 epoch, ~7 minutes instead of ~15 minutes)
#   --skip-build   Skip building the xatu image (use existing image)
#   --skip-cleanup Don't cleanup on exit (useful for debugging)
#
# Prerequisites:
#   - Docker and Docker Compose
#   - Kurtosis CLI (https://docs.kurtosis.com/install/)
#   - clickhouse-client (optional, will use docker exec if not available)

set -euo pipefail

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
ENCLAVE_NAME="horizon-e2e"
XATU_IMAGE="ethpandaops/xatu:local"
DOCKER_NETWORK="xatu_xatu-net"

# Timing configuration
QUICK_MODE=false
SKIP_BUILD=false
SKIP_CLEANUP=false
WAIT_EPOCHS=2
SECONDS_PER_SLOT=12
SLOTS_PER_EPOCH=32

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --quick)
            QUICK_MODE=true
            WAIT_EPOCHS=1
            shift
            ;;
        --skip-build)
            SKIP_BUILD=true
            shift
            ;;
        --skip-cleanup)
            SKIP_CLEANUP=true
            shift
            ;;
        *)
            echo "Unknown option: $1"
            exit 1
            ;;
    esac
done

# Calculate wait time
EPOCH_DURATION=$((SLOTS_PER_EPOCH * SECONDS_PER_SLOT))
WAIT_TIME=$((WAIT_EPOCHS * EPOCH_DURATION + 60))  # Add 60s buffer for processing

# Color output helpers
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_header() {
    echo ""
    echo -e "${BLUE}============================================${NC}"
    echo -e "${BLUE}  $1${NC}"
    echo -e "${BLUE}============================================${NC}"
}

# Cleanup function
cleanup() {
    if [ "$SKIP_CLEANUP" = true ]; then
        log_warn "Skipping cleanup (--skip-cleanup specified)"
        log_info "To clean up manually:"
        log_info "  kurtosis enclave stop $ENCLAVE_NAME && kurtosis enclave rm $ENCLAVE_NAME"
        log_info "  docker stop xatu-horizon && docker rm xatu-horizon"
        log_info "  docker compose -f $REPO_ROOT/docker-compose.yml down -v"
        return
    fi

    log_header "Cleaning up"

    # Stop and remove Horizon container
    if docker ps -a --format '{{.Names}}' | grep -q "^xatu-horizon$"; then
        log_info "Stopping Horizon container..."
        docker stop xatu-horizon 2>/dev/null || true
        docker rm xatu-horizon 2>/dev/null || true
    fi

    # Stop Kurtosis enclave
    if kurtosis enclave ls 2>/dev/null | grep -q "$ENCLAVE_NAME"; then
        log_info "Stopping Kurtosis enclave..."
        kurtosis enclave stop "$ENCLAVE_NAME" 2>/dev/null || true
        kurtosis enclave rm "$ENCLAVE_NAME" 2>/dev/null || true
    fi

    # Stop docker-compose
    log_info "Stopping docker-compose stack..."
    docker compose -f "$REPO_ROOT/docker-compose.yml" down -v 2>/dev/null || true

    log_success "Cleanup complete"
}

# Set up trap for cleanup on exit
trap cleanup EXIT

# Execute ClickHouse query
execute_query() {
    local query="$1"
    if command -v clickhouse-client &> /dev/null; then
        clickhouse-client -h localhost --port 9000 -u default -d default -q "$query" 2>/dev/null
    else
        docker exec xatu-clickhouse-01 clickhouse-client -q "$query" 2>/dev/null
    fi
}

# Wait for ClickHouse to be ready
wait_for_clickhouse() {
    log_info "Waiting for ClickHouse to be ready..."
    local max_attempts=60
    local attempt=0

    while ! execute_query "SELECT 1" &>/dev/null; do
        attempt=$((attempt + 1))
        if [ $attempt -ge $max_attempts ]; then
            log_error "ClickHouse not ready after $max_attempts attempts"
            return 1
        fi
        sleep 2
    done
    log_success "ClickHouse is ready"
}

# Wait for Postgres to be ready and run migrations
wait_for_postgres() {
    log_info "Waiting for PostgreSQL to be ready..."
    local max_attempts=60
    local attempt=0

    while ! docker exec xatu-postgres pg_isready -U user &>/dev/null; do
        attempt=$((attempt + 1))
        if [ $attempt -ge $max_attempts ]; then
            log_error "PostgreSQL not ready after $max_attempts attempts"
            return 1
        fi
        sleep 2
    done

    # Wait for horizon_location table to be created
    log_info "Waiting for horizon_location table..."
    attempt=0
    while ! docker exec xatu-postgres psql -U user -d xatu -c "SELECT 1 FROM horizon_location LIMIT 1" &>/dev/null; do
        attempt=$((attempt + 1))
        if [ $attempt -ge $max_attempts ]; then
            log_error "horizon_location table not created after $max_attempts attempts"
            return 1
        fi
        sleep 2
    done

    log_success "PostgreSQL is ready with horizon_location table"
}

# Get beacon node container names from Kurtosis
get_beacon_nodes() {
    kurtosis enclave inspect "$ENCLAVE_NAME" 2>/dev/null | \
        grep -E "^cl-" | \
        grep -v validator | \
        awk '{print $1}' | \
        head -n 6
}

# Connect Kurtosis containers to xatu network
connect_networks() {
    log_info "Connecting Kurtosis beacon nodes to xatu network..."

    local beacon_nodes
    beacon_nodes=$(get_beacon_nodes)

    for container in $beacon_nodes; do
        if docker network connect "$DOCKER_NETWORK" "$container" 2>/dev/null; then
            log_info "  Connected: $container"
        else
            log_warn "  Already connected or failed: $container"
        fi
    done
}

# Generate Horizon config with actual beacon node URLs
generate_horizon_config() {
    local config_file="$1"

    log_info "Generating Horizon configuration..."

    # Get beacon node info from Kurtosis
    local lighthouse_container prysm_container teku_container lodestar_container nimbus_container grandine_container

    lighthouse_container=$(kurtosis enclave inspect "$ENCLAVE_NAME" 2>/dev/null | grep "cl-lighthouse" | grep -v validator | head -n1 | awk '{print $1}')
    prysm_container=$(kurtosis enclave inspect "$ENCLAVE_NAME" 2>/dev/null | grep "cl-prysm" | grep -v validator | head -n1 | awk '{print $1}')
    teku_container=$(kurtosis enclave inspect "$ENCLAVE_NAME" 2>/dev/null | grep "cl-teku" | grep -v validator | head -n1 | awk '{print $1}')
    lodestar_container=$(kurtosis enclave inspect "$ENCLAVE_NAME" 2>/dev/null | grep "cl-lodestar" | grep -v validator | head -n1 | awk '{print $1}')
    nimbus_container=$(kurtosis enclave inspect "$ENCLAVE_NAME" 2>/dev/null | grep "cl-nimbus" | grep -v validator | head -n1 | awk '{print $1}')
    grandine_container=$(kurtosis enclave inspect "$ENCLAVE_NAME" 2>/dev/null | grep "cl-grandine" | grep -v validator | head -n1 | awk '{print $1}')

    cat > "$config_file" <<EOF
# Auto-generated Horizon config for E2E test
logging: "info"
metricsAddr: ":9098"

name: horizon-e2e-test

labels:
  environment: e2e-test
  network: kurtosis

ntpServer: time.google.com

coordinator:
  address: xatu-server:8080
  tls: false

ethereum:
  beaconNodes:
EOF

    # Add beacon nodes that exist
    [ -n "$lighthouse_container" ] && cat >> "$config_file" <<EOF
    - name: lighthouse
      address: http://${lighthouse_container}:4000
EOF
    [ -n "$prysm_container" ] && cat >> "$config_file" <<EOF
    - name: prysm
      address: http://${prysm_container}:3500
EOF
    [ -n "$teku_container" ] && cat >> "$config_file" <<EOF
    - name: teku
      address: http://${teku_container}:4000
EOF
    [ -n "$lodestar_container" ] && cat >> "$config_file" <<EOF
    - name: lodestar
      address: http://${lodestar_container}:4000
EOF
    [ -n "$nimbus_container" ] && cat >> "$config_file" <<EOF
    - name: nimbus
      address: http://${nimbus_container}:4000
EOF
    [ -n "$grandine_container" ] && cat >> "$config_file" <<EOF
    - name: grandine
      address: http://${grandine_container}:4000
EOF

    cat >> "$config_file" <<EOF

  healthCheckInterval: 3s
  blockCacheSize: 1000
  blockCacheTtl: 1h
  blockPreloadWorkers: 5
  blockPreloadQueueSize: 5000

dedupCache:
  ttl: 13m

subscription:
  bufferSize: 1000

reorg:
  enabled: true
  maxDepth: 64
  bufferSize: 100

epochIterator:
  enabled: true
  triggerPercent: 0.5

derivers:
  beaconBlock:
    enabled: true
  attesterSlashing:
    enabled: true
  proposerSlashing:
    enabled: true
  deposit:
    enabled: true
  withdrawal:
    enabled: true
  voluntaryExit:
    enabled: true
  blsToExecutionChange:
    enabled: true
  executionTransaction:
    enabled: true
  elaboratedAttestation:
    enabled: true
  proposerDuty:
    enabled: true
  beaconBlob:
    enabled: true
  beaconValidators:
    enabled: true
    chunkSize: 100
  beaconCommittee:
    enabled: true

outputs:
  - name: xatu
    type: xatu
    config:
      address: xatu-server:8080
      tls: false
      maxQueueSize: 51200
      batchTimeout: 0.5s
      exportTimeout: 30s
      maxExportBatchSize: 32
      workers: 50
EOF

    log_success "Generated config: $config_file"
}

# Run validation queries
run_validation() {
    log_header "Running Validation Queries"

    local failed=0
    local total=0

    # Query 1: Check for beacon blocks
    log_info "Checking for beacon blocks..."
    total=$((total + 1))
    local block_count
    block_count=$(execute_query "SELECT COUNT(*) FROM beacon_api_eth_v2_beacon_block FINAL WHERE meta_client_module = 'HORIZON'")
    if [ -n "$block_count" ] && [ "$block_count" -gt 0 ]; then
        log_success "  Found $block_count beacon blocks"
    else
        log_error "  No beacon blocks found"
        failed=$((failed + 1))
    fi

    # Query 2: Verify no duplicate block roots per slot
    log_info "Checking for deduplication (no duplicate block roots per slot)..."
    total=$((total + 1))
    local duplicates
    duplicates=$(execute_query "
        SELECT COUNT(*) FROM (
            SELECT slot, block_root, COUNT(*) as cnt
            FROM beacon_api_eth_v2_beacon_block FINAL
            WHERE meta_client_module = 'HORIZON'
            GROUP BY slot, block_root
            HAVING cnt > 1
        )
    ")
    if [ -n "$duplicates" ] && [ "$duplicates" -eq 0 ]; then
        log_success "  No duplicate blocks found (deduplication working)"
    else
        log_error "  Found $duplicates duplicate block entries"
        failed=$((failed + 1))
    fi

    # Query 3: Check for slot gaps (if we have enough blocks)
    log_info "Checking for slot gaps..."
    total=$((total + 1))
    local min_slot max_slot expected_count actual_count
    min_slot=$(execute_query "SELECT MIN(slot) FROM beacon_api_eth_v2_beacon_block FINAL WHERE meta_client_module = 'HORIZON'")
    max_slot=$(execute_query "SELECT MAX(slot) FROM beacon_api_eth_v2_beacon_block FINAL WHERE meta_client_module = 'HORIZON'")

    if [ -n "$min_slot" ] && [ -n "$max_slot" ] && [ "$min_slot" != "$max_slot" ]; then
        expected_count=$((max_slot - min_slot + 1))
        actual_count=$(execute_query "SELECT COUNT(DISTINCT slot) FROM beacon_api_eth_v2_beacon_block FINAL WHERE meta_client_module = 'HORIZON'")

        if [ "$actual_count" -ge "$((expected_count - 2))" ]; then  # Allow 2 slot tolerance for missed blocks
            log_success "  Slots coverage: $actual_count / $expected_count (min: $min_slot, max: $max_slot)"
        else
            log_warn "  Potential gaps: $actual_count / $expected_count slots covered"
        fi
    else
        log_warn "  Not enough data to check for gaps"
    fi

    # Query 4: Check execution transactions
    log_info "Checking for execution transactions..."
    total=$((total + 1))
    local tx_count
    tx_count=$(execute_query "SELECT COUNT(*) FROM beacon_api_eth_v2_beacon_block_execution_transaction FINAL WHERE meta_client_module = 'HORIZON'")
    if [ -n "$tx_count" ] && [ "$tx_count" -gt 0 ]; then
        log_success "  Found $tx_count execution transactions"
    else
        log_warn "  No execution transactions (may be normal for empty blocks)"
    fi

    # Query 5: Check elaborated attestations
    log_info "Checking for elaborated attestations..."
    total=$((total + 1))
    local attestation_count
    attestation_count=$(execute_query "SELECT COUNT(*) FROM beacon_api_eth_v2_beacon_block_elaborated_attestation FINAL WHERE meta_client_module = 'HORIZON'")
    if [ -n "$attestation_count" ] && [ "$attestation_count" -gt 0 ]; then
        log_success "  Found $attestation_count elaborated attestations"
    else
        log_error "  No elaborated attestations found"
        failed=$((failed + 1))
    fi

    # Query 6: Check proposer duties
    log_info "Checking for proposer duties..."
    total=$((total + 1))
    local duty_count
    duty_count=$(execute_query "SELECT COUNT(*) FROM beacon_api_eth_v1_proposer_duty FINAL WHERE meta_client_module = 'HORIZON'")
    if [ -n "$duty_count" ] && [ "$duty_count" -gt 0 ]; then
        log_success "  Found $duty_count proposer duties"
    else
        log_error "  No proposer duties found"
        failed=$((failed + 1))
    fi

    # Query 7: Check beacon committees
    log_info "Checking for beacon committees..."
    total=$((total + 1))
    local committee_count
    committee_count=$(execute_query "SELECT COUNT(*) FROM beacon_api_eth_v1_beacon_committee FINAL WHERE meta_client_module = 'HORIZON'")
    if [ -n "$committee_count" ] && [ "$committee_count" -gt 0 ]; then
        log_success "  Found $committee_count beacon committees"
    else
        log_error "  No beacon committees found"
        failed=$((failed + 1))
    fi

    # Summary
    log_header "Validation Summary"

    local passed=$((total - failed))
    if [ $failed -eq 0 ]; then
        log_success "All $total checks passed!"
        return 0
    else
        log_error "$failed of $total checks failed"
        return 1
    fi
}

# Main execution
main() {
    log_header "Horizon E2E Test"
    log_info "Mode: $([ "$QUICK_MODE" = true ] && echo 'Quick (1 epoch)' || echo 'Full (2 epochs)')"
    log_info "Wait time: ~$((WAIT_TIME / 60)) minutes"

    cd "$REPO_ROOT"

    # Step 1: Build xatu image
    if [ "$SKIP_BUILD" = false ]; then
        log_header "Building Xatu Image"
        docker build -t "$XATU_IMAGE" .
        log_success "Image built: $XATU_IMAGE"
    else
        log_warn "Skipping build (--skip-build specified)"
    fi

    # Step 2: Start docker-compose stack
    log_header "Starting Xatu Stack"
    docker compose up --detach --quiet-pull
    wait_for_clickhouse
    wait_for_postgres
    log_success "Xatu stack is running"

    # Step 3: Start Kurtosis network
    log_header "Starting Kurtosis Network"
    kurtosis run github.com/ethpandaops/ethereum-package \
        --args-file "$REPO_ROOT/deploy/kurtosis/horizon-test.yaml" \
        --enclave "$ENCLAVE_NAME"
    log_success "Kurtosis network started"

    # Step 4: Wait for genesis
    log_info "Waiting for genesis (120 seconds based on genesis_delay)..."
    sleep 130

    # Step 5: Connect networks
    log_header "Connecting Networks"
    connect_networks

    # Step 6: Generate and start Horizon
    log_header "Starting Horizon"
    local horizon_config="/tmp/horizon-e2e-config.yaml"
    generate_horizon_config "$horizon_config"

    docker run -d \
        --name xatu-horizon \
        --network "$DOCKER_NETWORK" \
        -v "$horizon_config:/etc/xatu/config.yaml:ro" \
        "$XATU_IMAGE" \
        horizon --config /etc/xatu/config.yaml

    log_info "Waiting for Horizon to start..."
    sleep 10

    # Show Horizon logs
    log_info "Horizon initial logs:"
    docker logs xatu-horizon 2>&1 | head -n 20

    # Step 7: Wait for data collection
    log_header "Collecting Data"
    log_info "Waiting $((WAIT_TIME / 60)) minutes for $WAIT_EPOCHS epoch(s)..."

    local elapsed=0
    local check_interval=30
    while [ $elapsed -lt $WAIT_TIME ]; do
        sleep $check_interval
        elapsed=$((elapsed + check_interval))

        # Show progress
        local remaining=$((WAIT_TIME - elapsed))
        log_info "Progress: $((elapsed / 60))m elapsed, ~$((remaining / 60))m remaining"

        # Quick check for blocks
        local current_blocks
        current_blocks=$(execute_query "SELECT COUNT(*) FROM beacon_api_eth_v2_beacon_block FINAL WHERE meta_client_module = 'HORIZON'" 2>/dev/null || echo "0")
        log_info "  Current block count: $current_blocks"

        # Show recent Horizon logs if no blocks yet
        if [ "$current_blocks" = "0" ]; then
            log_info "  Recent Horizon logs:"
            docker logs --tail 5 xatu-horizon 2>&1 | sed 's/^/    /'
        fi
    done

    # Step 8: Run validation
    if run_validation; then
        log_header "TEST PASSED"
        exit 0
    else
        log_header "TEST FAILED"

        # Show debugging info
        log_info "Horizon logs (last 50 lines):"
        docker logs --tail 50 xatu-horizon 2>&1

        log_info "xatu-server logs (last 20 lines):"
        docker logs --tail 20 xatu-server 2>&1

        exit 1
    fi
}

main
