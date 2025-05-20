# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Xatu is an Ethereum network monitoring tool with collection clients and a centralized server for data pipelining. The project is written in Go and follows a modular architecture where each component can run independently.

### Key Components

- **Server**: Centralized server collecting events from various clients, with outputs to various sinks
- **Sentry**: Client that runs alongside an Ethereum consensus client and collects data via the Beacon API
- **Discovery**: Client that uses Node Discovery Protocol to discover nodes on the network
- **Mimicry**: Client that collects data from the execution layer P2P network
- **CL Mimicry**: Client that collects data from the consensus layer P2P network using Hermes
- **Cannon**: Client that collects canonical finalized data via the Beacon API
- **Relay Monitor**: Client that monitors Flashbots relay blocks and validator registrations

## Development Commands

### Build and Run

```bash
# Build the project
go build -o xatu

# Run a specific mode
./xatu <server|sentry|discovery|mimicry|cannon|relay-monitor|cl-mimicry> --config your-config.yaml

# Run locally with docker-compose (server only)
docker compose up --detach

# Build and run with changes
docker compose up -d --build

# Run only the clickhouse cluster (for working with Xatu data)
docker compose --profile clickhouse up --detach
```

### Testing

```bash
# Run all tests
go test ./...

# Run tests with JSON output (for CI)
go test -json ./...

# Run tests for a specific package
go test ./pkg/processor/...

# Run tests with verbose output
go test -v ./...
```

### Code Quality

```bash
# Run linting
golangci-lint run --timeout=5m
```

### Protocol Buffers

```bash
# Generate protocol buffer code
buf generate
```

### Docker

```bash
# Build the docker image
docker build -t ethpandaops/xatu .

# Clean up docker volumes
docker compose down -v
```

## Project Structure

- `/cmd`: Command-line entry points for different modes (server, sentry, etc.)
- `/pkg`: Core packages for the application
  - `/cannon`: Implementation for the Cannon mode
  - `/clmimicry`: Implementation for CL Mimicry mode 
  - `/discovery`: Implementation for the Discovery mode
  - `/ethereum`: Ethereum client interfaces
  - `/mimicry`: Implementation for the Mimicry mode
  - `/networks`: Network configuration
  - `/observability`: Tracing and monitoring
  - `/output`: Output sinks (HTTP, Kafka, stdout, etc.)
  - `/processor`: Batch processing functions
  - `/proto`: Protocol buffer definitions
  - `/relaymonitor`: Implementation for monitoring MEV relays
  - `/sentry`: Implementation for the Sentry mode
  - `/server`: Implementation for the Server mode
- `/docs`: Documentation for each mode
- `/deploy`: Deployment configurations
  - `/local`: Local deployment configurations with docker-compose
  - `/migrations`: Database migration scripts

## Configuration

Each mode has its own configuration file. Examples are provided as:
- `example_server.yaml`: Server configuration
- `example_sentry.yaml`: Sentry configuration
- `example_discovery.yaml`: Discovery configuration
- `example_mimicry.yaml`: Mimicry configuration
- `example_cl_mimicry.yaml`: CL Mimicry configuration
- `example_cannon.yaml`: Cannon configuration
- `example_relay_monitor.yaml`: Relay monitor configuration

## Dependencies

The project uses Go 1.24 and various dependencies including:
- Protocol buffers for serialization
- gRPC for communication
- Prometheus for metrics
- Logrus for logging
- OpenTelemetry for tracing
- Cobra for CLI commands

## Architecture and Data Flow

The Xatu system follows a hub-and-spoke architecture:

1. **Data Collection** - Multiple client components collect different types of data:
   - Sentry: Captures consensus client events via Beacon API
   - Discovery: Discovers and monitors network nodes 
   - Mimicry: Captures execution layer P2P network data
   - CL Mimicry: Captures consensus layer P2P network data
   - Cannon: Extracts canonical finalized blockchain data
   - Relay Monitor: Tracks MEV relay information

2. **Data Aggregation** - The Server component:
   - Receives events from all clients
   - Processes and enriches the data
   - Optionally stores data in persistence layer

3. **Data Distribution** - Server outputs processed data to various sinks:
   - Kafka for high-throughput event streaming
   - HTTP endpoints for direct consumption
   - ClickHouse for analytics and data warehousing
   - Standard output for debugging

This pipeline architecture allows for flexible deployment scenarios, from single-node setups to distributed systems across multiple regions.
