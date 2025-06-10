#!/bin/bash

# Script to update cannon iterator type to a specific epoch on a network
# Usage: ./hack/cannon-at-epoch.sh <network_id> <cannon_type> <epoch>

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1" >&2
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1" >&2
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1" >&2
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1" >&2
}

# Function to show usage
usage() {
    cat << EOF
Usage: $0 <network_id> <cannon_type> <epoch>

Updates a cannon iterator type to a specific epoch on a network.

Arguments:
  network_id    Network identifier (e.g., mainnet, goerli, holesky)
  cannon_type   Cannon type identifier (e.g., BEACON_API_ETH_V2_BEACON_BLOCK)
  epoch         Target epoch number

Examples:
  $0 mainnet BEACON_API_ETH_V2_BEACON_BLOCK 123456
  $0 holesky BEACON_API_ETH_V1_BEACON_BLOB_SIDECAR 789

Database Configuration:
  The script uses docker-compose postgres container with the following defaults:
  - Container: xatu-postgres
  - Database: xatu
  - User: user
  - Password: password

  Set these environment variables to override:
  - POSTGRES_CONTAINER
  - POSTGRES_DB
  - POSTGRES_USER
  - POSTGRES_PASSWORD
EOF
}

# Check if required arguments are provided
if [ $# -ne 3 ]; then
    log_error "Invalid number of arguments"
    usage
    exit 1
fi

NETWORK_ID="$1"
CANNON_TYPE="$2"
EPOCH="$3"

# Validate epoch is a number
if ! [[ "$EPOCH" =~ ^[0-9]+$ ]]; then
    log_error "Epoch must be a positive integer"
    exit 1
fi

# Database configuration - using docker-compose postgres container
POSTGRES_CONTAINER="${POSTGRES_CONTAINER:-xatu-postgres}"
POSTGRES_DB="${POSTGRES_DB:-xatu}"
POSTGRES_USER="${POSTGRES_USER:-user}"
POSTGRES_PASSWORD="${POSTGRES_PASSWORD:-password}"

# Docker command to exec into postgres container
DOCKER_PSQL_CMD="docker exec -i $POSTGRES_CONTAINER psql -U $POSTGRES_USER -d $POSTGRES_DB"

log_info "Starting cannon iterator update process"
log_info "Network ID: $NETWORK_ID"
log_info "Cannon Type: $CANNON_TYPE"
log_info "Target Epoch: $EPOCH"
log_info "Database Container: $POSTGRES_CONTAINER/$POSTGRES_DB"

# Check if docker is available
if ! command -v docker &> /dev/null; then
    log_error "docker command not found. Please install Docker."
    exit 1
fi

# Test database connection
log_info "Testing database connection..."
if ! echo "SELECT 1;" | PGPASSWORD="$POSTGRES_PASSWORD" $DOCKER_PSQL_CMD &> /dev/null; then
    log_error "Failed to connect to database. Please check your connection settings."
    log_error "Make sure docker-compose postgres container is running and accessible."
    log_error "Try running: docker ps | grep $POSTGRES_CONTAINER"
    exit 1
fi
log_success "Database connection successful"

# Check if cannon_location table exists
log_info "Checking if cannon_location table exists..."
if ! echo "SELECT 1 FROM cannon_location LIMIT 1;" | PGPASSWORD="$POSTGRES_PASSWORD" $DOCKER_PSQL_CMD &> /dev/null; then
    log_error "cannon_location table not found. Please run database migrations first."
    exit 1
fi
log_success "cannon_location table found"

# Check if record exists
log_info "Checking for existing cannon location record..."
EXISTING_RECORD=$(echo "SELECT value FROM cannon_location WHERE network_id = '$NETWORK_ID' AND type = '$CANNON_TYPE';" | PGPASSWORD="$POSTGRES_PASSWORD" $DOCKER_PSQL_CMD -t 2>/dev/null | xargs)

if [ -z "$EXISTING_RECORD" ]; then
    log_info "No existing record found. Creating new cannon location record..."
    
    # Create JSON structure for new record (protobuf uses snake_case in JSON)
    NEW_JSON="{\"backfilling_checkpoint_marker\":{\"finalized_epoch\":$EPOCH,\"backfill_epoch\":$EPOCH}}"
    
    # Insert new record
    echo "
        INSERT INTO cannon_location (network_id, type, value, create_time, update_time) 
        VALUES ('$NETWORK_ID', '$CANNON_TYPE', '$NEW_JSON', NOW(), NOW())
        ON CONFLICT (network_id, type) DO UPDATE SET 
            value = EXCLUDED.value,
            update_time = NOW();
    " | PGPASSWORD="$POSTGRES_PASSWORD" $DOCKER_PSQL_CMD &> /dev/null
    
    if [ $? -eq 0 ]; then
        log_success "Successfully created new cannon location record"
    else
        log_error "Failed to create new cannon location record"
        exit 1
    fi
else
    log_info "Found existing record. Current value: $EXISTING_RECORD"
    
    # Parse existing JSON and update the epoch values
    # Handle both camelCase and snake_case formats that exist in the database
    log_info "Updating existing record with new epoch values..."
    
    # Detect JSON format and update accordingly
    if echo "$EXISTING_RECORD" | grep -q "backfilling_checkpoint_marker"; then
        # snake_case format
        log_info "Detected snake_case JSON format"
        echo "
            UPDATE cannon_location 
            SET value = jsonb_set(
                jsonb_set(
                    value::jsonb, 
                    '{backfilling_checkpoint_marker,finalized_epoch}', 
                    '$EPOCH'::jsonb
                ),
                '{backfilling_checkpoint_marker,backfill_epoch}', 
                '$EPOCH'::jsonb
            )::text,
            update_time = NOW()
            WHERE network_id = '$NETWORK_ID' AND type = '$CANNON_TYPE';
        " | PGPASSWORD="$POSTGRES_PASSWORD" $DOCKER_PSQL_CMD &> /dev/null
    elif echo "$EXISTING_RECORD" | grep -q "backfillingCheckpointMarker"; then
        # camelCase format
        log_info "Detected camelCase JSON format"
        echo "
            UPDATE cannon_location 
            SET value = jsonb_set(
                jsonb_set(
                    value::jsonb, 
                    '{backfillingCheckpointMarker,finalizedEpoch}', 
                    '\"$EPOCH\"'::jsonb
                ),
                '{backfillingCheckpointMarker,backfillEpoch}', 
                '\"$EPOCH\"'::jsonb
            )::text,
            update_time = NOW()
            WHERE network_id = '$NETWORK_ID' AND type = '$CANNON_TYPE';
        " | PGPASSWORD="$POSTGRES_PASSWORD" $DOCKER_PSQL_CMD &> /dev/null
    else
        log_error "Unknown JSON format in existing record: $EXISTING_RECORD"
        exit 1
    fi
    
    if [ $? -eq 0 ]; then
        log_success "Successfully updated existing cannon location record"
    else
        log_error "Failed to update existing cannon location record"
        exit 1
    fi
fi

# Verify the update
log_info "Verifying the update..."
UPDATED_RECORD=$(echo "SELECT value FROM cannon_location WHERE network_id = '$NETWORK_ID' AND type = '$CANNON_TYPE';" | PGPASSWORD="$POSTGRES_PASSWORD" $DOCKER_PSQL_CMD -t 2>/dev/null | xargs)

if [ -n "$UPDATED_RECORD" ]; then
    log_success "Update verified. New value: $UPDATED_RECORD"
    
    # Extract and display the epoch values for confirmation
    if echo "$UPDATED_RECORD" | grep -q "backfilling_checkpoint_marker"; then
        # snake_case format - numeric values
        FINALIZED_EPOCH=$(echo "$UPDATED_RECORD" | grep -o '"finalized_epoch":[0-9]*' | cut -d':' -f2)
        BACKFILL_EPOCH=$(echo "$UPDATED_RECORD" | grep -o '"backfill_epoch":[0-9]*' | cut -d':' -f2)
    else
        # camelCase format - string values
        FINALIZED_EPOCH=$(echo "$UPDATED_RECORD" | grep -o '"finalizedEpoch":"[0-9]*"' | cut -d':' -f2 | tr -d '"')
        BACKFILL_EPOCH=$(echo "$UPDATED_RECORD" | grep -o '"backfillEpoch":"[0-9]*"' | cut -d':' -f2 | tr -d '"')
    fi
    
    log_success "Finalized Epoch set to: $FINALIZED_EPOCH"
    log_success "Backfill Epoch set to: $BACKFILL_EPOCH"
else
    log_error "Failed to verify the update"
    exit 1
fi

log_success "Cannon iterator update completed successfully!"
log_info "Network: $NETWORK_ID"
log_info "Type: $CANNON_TYPE" 
log_info "Epoch: $EPOCH"