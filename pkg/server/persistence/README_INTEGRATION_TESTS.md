# Integration Tests for Persistence Package

This document describes the integration tests for the `pkg/server/persistence` package.

## Overview

The integration tests use [testcontainers-go](https://golang.testcontainers.org/) to spin up a real PostgreSQL database for testing. This ensures that:

- Database operations work correctly with real PostgreSQL instances
- SQL syntax and constraints are properly validated 
- Migration scripts are tested as part of the setup
- Tests run in complete isolation with fresh database containers

## Prerequisites

### Docker
The integration tests require Docker to be installed and running:

```bash
# Install Docker Desktop or Docker Engine
# Ensure Docker daemon is running
docker --version
```

### Go Dependencies
All required dependencies are included in `go.mod`:

```bash
go mod download
```

## Test Structure

The integration tests are located in `integration_test.go` and cover:

### Test Functions

1. **TestNodeRecordOperations**
   - `InsertNodeRecords` - Insert multiple node records with ON CONFLICT handling
   - `UpdateNodeRecord` - Update existing node records
   - `CheckoutStalledExecutionNodeRecords` - Retrieve stalled execution nodes
   - `CheckoutStalledConsensusNodeRecords` - Retrieve stalled consensus nodes

2. **TestNodeRecordConsensusOperations**
   - `InsertNodeRecordConsensus` - Insert consensus records with foreign key validation
   - `ListNodeRecordConsensus` - List consensus records with filtering by network ID and fork digest

3. **TestNodeRecordExecutionOperations**
   - `InsertNodeRecordExecution` - Insert execution records with foreign key validation
   - `ListNodeRecordExecutions` - List execution records with filtering by network ID and fork ID hash

4. **TestDatabaseConstraints**
   - Foreign key constraint validation for consensus and execution records

### Test Infrastructure

- **TestContainer** - Manages PostgreSQL testcontainer lifecycle
- **setupTestContainer()** - Creates container, applies migrations, initializes client
- **applyMigrations()** - Applies PostgreSQL migration scripts
- **createTestNodeRecord()** - Generates test node.Record data
- **createTestConsensusRecord()** - Generates test node.Consensus data  
- **createTestExecutionRecord()** - Generates test node.Execution data

## Running the Tests

### Run All Integration Tests

```bash
cd pkg/server/persistence
go test -v
```

### Run Specific Test Functions

```bash
# Run only node record operations tests
go test -v -run TestNodeRecordOperations

# Run only consensus operations tests  
go test -v -run TestNodeRecordConsensusOperations

# Run only execution operations tests
go test -v -run TestNodeRecordExecutionOperations

# Run only constraint validation tests
go test -v -run TestDatabaseConstraints
```

### Run with Race Detection

```bash
go test -v -race
```

### Parallel Test Execution

Tests can be run in parallel as each test gets its own isolated database container:

```bash
go test -v -parallel 4
```

## Test Data

The tests use realistic ENR (Ethereum Node Record) strings and network data:

- **ENR Format**: `enr:-IS4Q...` (Base64 encoded Ethereum Node Records)
- **Network IDs**: 1 (Mainnet), 2 (Test networks)
- **Client Names**: "Lighthouse" (consensus), "Geth" (execution)
- **Capabilities**: "eth/66,eth/67" for execution clients

## Database Schema

The tests validate the complete database schema including:

### Tables
- `node_record` - Primary node information (ENR as primary key)
- `node_record_consensus` - Consensus client specific data
- `node_record_execution` - Execution client specific data
- `node_record_activity` - Node activity tracking

### Constraints
- Foreign key relationships between child tables and `node_record.enr`
- Primary key constraints
- Unique constraints where applicable

### Migration System
The integration tests use an **auto-discovery migration system** that:

- **Automatically discovers** all `.up.sql` files in `migrations/postgres/`
- **Sorts migrations** by filename (which includes numeric prefixes)
- **Applies them in correct order** without manual maintenance
- **Logs each migration** application for debugging
- **Skips empty migrations** gracefully

**Current Migrations Auto-Applied:**
- `001_initialize_schema.up.sql` - Initial table creation
- `002_node_record_index.up.sql` - Index creation  
- `003_execution_index.up.sql` - Execution table indexes
- `004_geo.up.sql` - Geographic data fields
- `005_cannon.up.sql` - Cannon service fields
- `006_consensus.up.sql` - Consensus table and fields

**Adding New Migrations:**
Simply add new `.up.sql` files to `migrations/postgres/` with proper numeric prefixes (e.g., `007_new_feature.up.sql`) and they will be automatically discovered and applied in the correct order.

## Troubleshooting

### Docker Issues

**Error**: `rootless Docker not found`
- Ensure Docker Desktop is installed and running
- Check Docker daemon status: `docker ps`

**Error**: `Cannot connect to Docker daemon`
- Start Docker Desktop/Engine
- Verify Docker socket permissions

### Test Failures

**Migration Errors**:
- Check that migration files exist in `../../../migrations/postgres/`
- Verify SQL syntax in migration files
- Ensure migrations are listed in correct order in `applyMigrations()`

**Connection Errors**:
- Container may take time to start - tests wait up to 60 seconds
- Check PostgreSQL logs in container output

**Foreign Key Errors**:
- Ensure parent `node_record` is inserted before child records
- Verify ENR values match between parent and child records

## Performance

- Container startup: ~5-10 seconds
- Test execution: ~1-5 seconds per test function
- Full test suite: ~30-60 seconds (depending on Docker performance)

## CI/CD Integration

These tests are suitable for CI/CD pipelines that support Docker:

```yaml
# GitHub Actions example
- name: Run Integration Tests
  run: |
    cd pkg/server/persistence
    go test -v -timeout 5m
```

**Note**: Some CI environments may require additional Docker setup or service containers.