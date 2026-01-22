# Horizon

Horizon is a HEAD data collection module with multi-beacon node support, high-availability coordination, and shared derivers. Unlike [Cannon](./cannon.md) which focuses on backfilling historical data, Horizon is optimized for real-time HEAD tracking of the Ethereum beacon chain.

This module can output events to various sinks and it is **not** a hard requirement to run the [Xatu server](./server.md), though it is required for high-availability deployments.

## Table of contents

- [Architecture Overview](#architecture-overview)
- [Dual-Iterator Design](#dual-iterator-design)
- [Multi-Beacon Node Support](#multi-beacon-node-support)
- [High Availability Deployment](#high-availability-deployment)
- [Horizon vs Cannon: When to Use Which](#horizon-vs-cannon-when-to-use-which)
- [Usage](#usage)
- [Requirements](#requirements)
- [Configuration](#configuration)
  - [Beacon Nodes](#beacon-nodes-configuration)
  - [Coordinator](#coordinator-configuration)
  - [Derivers](#derivers-configuration)
  - [Output Sinks](#output-sink-configuration)
- [Metrics Reference](#metrics-reference)
- [Running Locally](#running-locally)

## Architecture Overview

Horizon follows a modular architecture designed for reliability and real-time data collection:

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              HORIZON MODULE                                  │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                      BEACON NODE POOL                                │   │
│  │  ┌──────────┐  ┌──────────┐  ┌──────────┐                          │   │
│  │  │Lighthouse│  │  Prysm   │  │   Teku   │  ...                     │   │
│  │  │   :5052  │  │  :3500   │  │  :5051   │                          │   │
│  │  └────┬─────┘  └────┬─────┘  └────┬─────┘                          │   │
│  │       │             │             │                                 │   │
│  │       └─────────────┼─────────────┘                                 │   │
│  │                     │                                               │   │
│  │            ┌────────┴────────┐                                      │   │
│  │            │  Health Checker │                                      │   │
│  │            │  + Failover     │                                      │   │
│  │            └────────┬────────┘                                      │   │
│  └─────────────────────┼───────────────────────────────────────────────┘   │
│                        │                                                    │
│  ┌─────────────────────┼───────────────────────────────────────────────┐   │
│  │                     ▼                                               │   │
│  │  ┌──────────────────────────────┐    ┌─────────────────────────┐   │   │
│  │  │    SSE Block Subscription    │    │  SSE Reorg Subscription │   │   │
│  │  │  /eth/v1/events?topics=block │    │   chain_reorg events    │   │   │
│  │  └──────────────┬───────────────┘    └───────────┬─────────────┘   │   │
│  │                 │                                │                 │   │
│  │                 ▼                                │                 │   │
│  │  ┌──────────────────────────────┐               │                 │   │
│  │  │     Deduplication Cache      │◄──────────────┘                 │   │
│  │  │   (TTL-based block roots)    │   (clears reorged blocks)       │   │
│  │  └──────────────┬───────────────┘                                 │   │
│  │                 │                                                  │   │
│  │                 ▼                                                  │   │
│  │  ┌──────────────────────────────────────────────────────────────┐ │   │
│  │  │                    HEAD ITERATOR                              │ │   │
│  │  │  • Receives real-time SSE block events                       │ │   │
│  │  │  • Processes slots immediately as they arrive                │ │   │
│  │  │  • Updates head_slot in coordinator                          │ │   │
│  │  └──────────────┬───────────────────────────────────────────────┘ │   │
│  │                 │                                                  │   │
│  │                 ├──────────────────────────────────────┐          │   │
│  │                 │                                      │          │   │
│  │                 ▼                                      ▼          │   │
│  │  ┌─────────────────────────┐          ┌─────────────────────────┐│   │
│  │  │    Block-Based          │          │    Epoch-Based          ││   │
│  │  │    Derivers             │          │    Derivers             ││   │
│  │  │  • BeaconBlock          │          │  • ProposerDuty         ││   │
│  │  │  • AttesterSlashing     │          │  • BeaconBlob           ││   │
│  │  │  • ProposerSlashing     │          │  • BeaconValidators     ││   │
│  │  │  • Deposit              │          │  • BeaconCommittee      ││   │
│  │  │  • Withdrawal           │          │                         ││   │
│  │  │  • VoluntaryExit        │          │  (Triggered midway      ││   │
│  │  │  • BLSToExecutionChange │          │   through each epoch)   ││   │
│  │  │  • ExecutionTransaction │          │                         ││   │
│  │  │  • ElaboratedAttestation│          │                         ││   │
│  │  └───────────┬─────────────┘          └───────────┬─────────────┘│   │
│  │              │                                    │              │   │
│  │              └────────────────┬───────────────────┘              │   │
│  │                               │                                  │   │
│  └───────────────────────────────┼──────────────────────────────────┘   │
│                                  │                                       │
│                                  ▼                                       │
│  ┌───────────────────────────────────────────────────────────────────┐  │
│  │                        OUTPUT SINKS                                │  │
│  │    ┌─────────┐    ┌─────────┐    ┌─────────┐    ┌─────────┐      │  │
│  │    │  Xatu   │    │  HTTP   │    │  Kafka  │    │ Stdout  │      │  │
│  │    │ Server  │    │ Server  │    │ Brokers │    │         │      │  │
│  │    └─────────┘    └─────────┘    └─────────┘    └─────────┘      │  │
│  └───────────────────────────────────────────────────────────────────┘  │
│                                                                          │
└─────────────────────────────────────────────────────────────────────────┘
                                  │
                                  ▼
              ┌───────────────────────────────────────┐
              │          COORDINATOR SERVER           │
              │  (Tracks head_slot / fill_slot per    │
              │   deriver for HA coordination)        │
              └───────────────────────────────────────┘
```

## Dual-Iterator Design

Horizon uses a dual-iterator architecture to ensure both real-time data collection and consistency:

### HEAD Iterator
- **Purpose**: Real-time processing of new blocks as they are produced
- **Mechanism**: Subscribes to SSE `/eth/v1/events?topics=block` on all beacon nodes
- **Priority**: Highest - never blocks, processes events immediately
- **Location Tracking**: Updates `head_slot` in coordinator after processing each slot

### FILL Iterator
- **Purpose**: Catches up on any missed slots between restarts
- **Mechanism**: Walks slots from `fill_slot` toward `HEAD - LAG`
- **Configuration**:
  - `lagSlots`: Number of slots to stay behind HEAD (default: 32)
  - `maxBoundedSlots`: Maximum slots to process in one cycle (default: 7200)
  - `rateLimit`: Maximum slots per second (default: 10.0)
- **Location Tracking**: Updates `fill_slot` in coordinator after processing each slot

### Coordination
Both iterators coordinate through the Coordinator service to avoid duplicate processing:
- HEAD checks both `head_slot` and `fill_slot` before processing
- FILL checks both markers to skip slots already processed by HEAD
- On restart, HEAD immediately begins tracking new blocks while FILL catches up from its last position

```
Timeline:
─────────────────────────────────────────────────────────────►
                                                         HEAD
        fill_slot              HEAD - LAG               (real-time)
            │                      │                        │
            ▼                      ▼                        ▼
────────────[FILL ITERATOR RANGE]──────────[LAG BUFFER]────►

FILL processes historical slots ─┐
                                 │
                    Never overlaps with HEAD ─► LAG ensures separation
```

### Epoch Iterator
For epoch-based derivers (ProposerDuty, BeaconBlob, BeaconValidators, BeaconCommittee):
- **Trigger**: Fires at a configurable percentage through each epoch (default: 50%)
- **Purpose**: Pre-fetches data for the NEXT epoch before it starts
- **Configuration**: `triggerPercent` (0.0 to 1.0, default: 0.5)

## Multi-Beacon Node Support

Horizon connects to multiple beacon nodes simultaneously for redundancy and reliability:

### Features
- **Health Checking**: Periodic health checks per node (configurable interval)
- **Automatic Failover**: Falls back to healthy nodes when primary is unavailable
- **Exponential Backoff Retry**: Failed connections retry with backoff (1s initial, 30s max)
- **SSE Aggregation**: Receives block events from all nodes, deduplicates locally
- **Shared Block Cache**: Single cache across all nodes with singleflight deduplication
- **Shared Services**: Metadata and Duties services initialized from first healthy node

### Configuration Example
```yaml
ethereum:
  beaconNodes:
    - name: lighthouse-1
      address: http://lighthouse:5052
      headers:
        authorization: Bearer token1
    - name: prysm-1
      address: http://prysm:3500
    - name: teku-1
      address: http://teku:5051
  healthCheckInterval: 3s
  blockCacheSize: 1000
  blockCacheTtl: 1h
```

### Node Selection
- `GetHealthyNode()`: Returns any healthy node (round-robin)
- `PreferNode(address)`: Prefers specific node, falls back to healthy if unavailable
- All nodes receive SSE subscriptions for redundancy

## High Availability Deployment

For production deployments requiring high availability:

### Single Instance Mode
Run one Horizon instance per network. Suitable for:
- Development/testing
- Non-critical data collection
- Networks with low stakes

### Multi-Instance HA Mode
Run multiple Horizon instances with Coordinator:

```
┌─────────────────────────────────────────────────────────────────┐
│                         Network                                  │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐          │
│  │  Horizon-1   │  │  Horizon-2   │  │  Horizon-3   │          │
│  │  (Primary)   │  │  (Standby)   │  │  (Standby)   │          │
│  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘          │
│         │                 │                 │                   │
│         └─────────────────┼─────────────────┘                   │
│                           │                                     │
│                           ▼                                     │
│              ┌────────────────────────┐                         │
│              │   Coordinator Server   │                         │
│              │  (PostgreSQL backend)  │                         │
│              └────────────────────────┘                         │
└─────────────────────────────────────────────────────────────────┘
```

**How it works:**
1. All instances track the same `HorizonLocation` records in the Coordinator
2. When processing a slot, each instance checks if `slot <= head_slot OR slot <= fill_slot`
3. First instance to process a slot updates the location, others skip
4. On failover, another instance picks up where the failed one left off

**Requirements:**
- [Server](./server.md) running with Coordinator service enabled
- PostgreSQL database for location persistence
- All instances configured with same `networkId` and deriver types

## Horizon vs Cannon: When to Use Which

| Feature | Horizon | Cannon |
|---------|---------|--------|
| **Primary Focus** | Real-time HEAD tracking | Historical backfilling |
| **Beacon Nodes** | Multiple (pool with failover) | Single |
| **Data Direction** | Forward (new blocks) | Backward (historical) |
| **Use Case** | Live monitoring, real-time analytics | Data backfilling, gap filling |
| **Latency** | Sub-second (SSE events) | Variable (backfill pace) |
| **HA Support** | Built-in (multi-node pool + coordinator) | Via coordinator |
| **Reorg Handling** | Native (SSE reorg events) | Relies on canonical chain |

### When to Use Horizon
- You need real-time data as blocks are produced
- You want redundancy across multiple beacon nodes
- You're building live dashboards or monitoring systems
- You need automatic failover and high availability

### When to Use Cannon
- You need to backfill historical data from genesis
- You're processing data at your own pace (rate-limited)
- You have a single reliable beacon node
- You're doing one-time historical analysis

### Using Both Together
For complete data coverage, run both:
1. **Horizon**: Tracks HEAD in real-time, ensures no new data is missed
2. **Cannon**: Backfills historical data at a controlled pace

Both modules share the same deriver implementations (`pkg/cldata/deriver/`) ensuring data consistency.

## Usage

Horizon requires a [config file](#configuration).

```bash
Usage:
  xatu horizon [flags]

Flags:
      --config string   config file (default is horizon.yaml) (default "horizon.yaml")
  -h, --help            help for horizon
```

## Requirements

- Multiple [Ethereum consensus clients](https://ethereum.org/en/developers/docs/nodes-and-clients/#consensus-clients) with exposed HTTP servers (recommended: 2+ for redundancy)
- [Server](./server.md) running with the [Coordinator](./server.md#coordinator) service enabled (required for HA)
- PostgreSQL database (for coordinator persistence)

## Configuration

Horizon requires a single `yaml` config file. An example file can be found [here](../example_horizon.yaml).

### General Configuration

| Name | Type | Default | Description |
|------|------|---------|-------------|
| logging | string | `info` | Log level (`panic`, `fatal`, `warn`, `info`, `debug`, `trace`) |
| metricsAddr | string | `:9090` | The address the metrics server will listen on |
| pprofAddr | string | | The address the pprof server will listen on (disabled if omitted) |
| name | string | | **Required.** Unique name of the Horizon instance |
| labels | object | | Key-value map of labels to append to every event |
| ntpServer | string | `time.google.com` | NTP server for clock drift correction |

### Beacon Nodes Configuration

| Name | Type | Default | Description |
|------|------|---------|-------------|
| ethereum.beaconNodes | array | | **Required.** List of beacon node configurations |
| ethereum.beaconNodes[].name | string | | **Required.** Unique name for this beacon node |
| ethereum.beaconNodes[].address | string | | **Required.** HTTP endpoint of the beacon node |
| ethereum.beaconNodes[].headers | object | | Key-value map of headers to append to requests |
| ethereum.overrideNetworkName | string | | Override auto-detected network name |
| ethereum.startupTimeout | duration | `60s` | Max time to wait for a healthy beacon node on startup |
| ethereum.healthCheckInterval | duration | `3s` | Interval between health checks |
| ethereum.blockCacheSize | int | `1000` | Maximum number of blocks to cache |
| ethereum.blockCacheTtl | duration | `1h` | TTL for cached blocks |
| ethereum.blockPreloadWorkers | int | `5` | Number of workers for block preloading |
| ethereum.blockPreloadQueueSize | int | `5000` | Size of block preload queue |

### Coordinator Configuration

| Name | Type | Default | Description |
|------|------|---------|-------------|
| coordinator.address | string | | **Required.** Address of the Xatu Coordinator server |
| coordinator.tls | bool | `false` | Server requires TLS |
| coordinator.headers | object | | Key-value map of headers (e.g., authorization) |

### Deduplication Cache Configuration

| Name | Type | Default | Description |
|------|------|---------|-------------|
| dedupCache.ttl | duration | `13m` | TTL for cached block roots (should exceed 1 epoch) |

### Subscription Configuration

| Name | Type | Default | Description |
|------|------|---------|-------------|
| subscription.bufferSize | int | `1000` | Size of the block events channel buffer |

### Reorg Configuration

| Name | Type | Default | Description |
|------|------|---------|-------------|
| reorg.enabled | bool | `true` | Enable chain reorg handling |
| reorg.maxDepth | int | `64` | Maximum reorg depth to handle (deeper reorgs ignored) |
| reorg.bufferSize | int | `100` | Size of the reorg events channel buffer |

### Epoch Iterator Configuration

| Name | Type | Default | Description |
|------|------|---------|-------------|
| epochIterator.enabled | bool | `true` | Enable epoch-based derivers |
| epochIterator.triggerPercent | float | `0.5` | Trigger point within epoch (0.0-1.0, 0.5 = midway) |

### Derivers Configuration

#### Block-Based Derivers

| Name | Type | Default | Description |
|------|------|---------|-------------|
| derivers.beaconBlock.enabled | bool | `true` | Enable beacon block deriver |
| derivers.attesterSlashing.enabled | bool | `true` | Enable attester slashing deriver |
| derivers.proposerSlashing.enabled | bool | `true` | Enable proposer slashing deriver |
| derivers.deposit.enabled | bool | `true` | Enable deposit deriver |
| derivers.withdrawal.enabled | bool | `true` | Enable withdrawal deriver (Capella+) |
| derivers.voluntaryExit.enabled | bool | `true` | Enable voluntary exit deriver |
| derivers.blsToExecutionChange.enabled | bool | `true` | Enable BLS to execution change deriver (Capella+) |
| derivers.executionTransaction.enabled | bool | `true` | Enable execution transaction deriver (Bellatrix+) |
| derivers.elaboratedAttestation.enabled | bool | `true` | Enable elaborated attestation deriver |

#### Epoch-Based Derivers

| Name | Type | Default | Description |
|------|------|---------|-------------|
| derivers.proposerDuty.enabled | bool | `true` | Enable proposer duty deriver |
| derivers.beaconBlob.enabled | bool | `true` | Enable beacon blob deriver (Deneb+) |
| derivers.beaconValidators.enabled | bool | `true` | Enable beacon validators deriver |
| derivers.beaconValidators.chunkSize | int | `100` | Validators per event chunk |
| derivers.beaconCommittee.enabled | bool | `true` | Enable beacon committee deriver |

### Output Sink Configuration

| Name | Type | Default | Description |
|------|------|---------|-------------|
| outputs | array | | **Required.** List of output sinks |
| outputs[].name | string | | Name of the output |
| outputs[].type | string | | Type: `xatu`, `http`, `kafka`, `stdout` |
| outputs[].config | object | | Output-specific configuration |
| outputs[].filter | object | | Event filtering configuration |

See [Cannon documentation](./cannon.md#output-xatu-configuration) for detailed output sink configuration options.

## Metrics Reference

All Horizon metrics use the `xatu_horizon` namespace.

### Core Metrics

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `decorated_event_total` | Counter | type, network | Total decorated events created |
| `head_slot` | Gauge | deriver, network | Current HEAD slot position |
| `fill_slot` | Gauge | deriver, network | Current FILL slot position |
| `lag_slots` | Gauge | deriver, network | Slots FILL is behind HEAD |
| `blocks_derived_total` | Counter | deriver, network, iterator | Total blocks derived |

### Beacon Node Pool Metrics

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `beacon_node_status` | Gauge | node, status | Node health status (1=active) |
| `beacon_blocks_fetched_total` | Counter | node, network | Blocks fetched per node |
| `beacon_block_cache_hits_total` | Counter | network | Block cache hits |
| `beacon_block_cache_misses_total` | Counter | network | Block cache misses |
| `beacon_block_fetch_errors_total` | Counter | node, network | Block fetch errors |
| `beacon_health_check_total` | Counter | node, status | Health checks per node |
| `beacon_health_check_duration_seconds` | Histogram | node | Health check duration |

### SSE Subscription Metrics

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `sse_events_total` | Counter | node, topic, network | SSE events received |
| `sse_connection_status` | Gauge | node | SSE connection status (1=connected) |
| `sse_reconnects_total` | Counter | node | SSE reconnection attempts |
| `sse_last_event_received_at` | Gauge | node, topic | Unix timestamp of last event |
| `sse_event_processing_delay_seconds` | Histogram | node, topic | Event processing delay |

### Reorg Metrics

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `reorg_events_total` | Counter | node, network | Reorg events received |
| `reorg_depth` | Histogram | node, network | Reorg depth distribution |
| `reorg_ignored_total` | Counter | node, network | Reorgs ignored (too deep) |
| `reorg_last_event_at` | Gauge | node, network | Timestamp of last reorg |
| `reorg_last_depth` | Gauge | node, network | Depth of last reorg |
| `reorg_last_slot` | Gauge | node, network | Slot of last reorg |

### Deduplication Cache Metrics

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `dedup_hits_total` | Counter | | Duplicate events dropped |
| `dedup_misses_total` | Counter | | New events processed |
| `dedup_cache_size` | Gauge | | Current cache entries |

### Iterator Metrics

#### HEAD Iterator

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `head_iterator_processed_total` | Counter | deriver, network | Slots processed |
| `head_iterator_skipped_total` | Counter | deriver, network, reason | Slots skipped |
| `head_iterator_position_slot` | Gauge | deriver, network | Current slot position |

#### FILL Iterator

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `fill_iterator_processed_total` | Counter | deriver, network | Slots processed |
| `fill_iterator_skipped_total` | Counter | deriver, network, reason | Slots skipped |
| `fill_iterator_position_slot` | Gauge | deriver, network | Current slot position |
| `fill_iterator_target_slot` | Gauge | deriver, network | Target slot (HEAD - LAG) |
| `fill_iterator_slots_remaining` | Gauge | deriver, network | Slots until caught up |
| `fill_iterator_rate_limit_wait_total` | Counter | | Rate limit wait events |
| `fill_iterator_cycles_complete_total` | Counter | deriver, network | Fill cycles completed |

#### Epoch Iterator

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `epoch_iterator_processed_total` | Counter | deriver, network | Epochs processed |
| `epoch_iterator_skipped_total` | Counter | deriver, network, reason | Epochs skipped |
| `epoch_iterator_position_epoch` | Gauge | deriver, network | Current epoch position |
| `epoch_iterator_trigger_wait_total` | Counter | deriver, network | Trigger point waits |

## Running Locally

```bash
# Docker
docker run -d --name xatu-horizon \
  -v $HOST_DIR_CHANGE_ME/config.yaml:/opt/xatu/config.yaml \
  -p 9090:9090 \
  -it ethpandaops/xatu:latest horizon --config /opt/xatu/config.yaml

# Build from source
go build -o dist/xatu main.go
./dist/xatu horizon --config horizon.yaml

# Development
go run main.go horizon --config horizon.yaml
```

### Minimal Configuration Example

```yaml
name: my-horizon

coordinator:
  address: localhost:8080

ethereum:
  beaconNodes:
    - name: local-beacon
      address: http://localhost:5052

outputs:
  - name: stdout
    type: stdout
```

### Production Configuration Example

```yaml
logging: info
metricsAddr: ":9090"
name: horizon-mainnet-1

labels:
  environment: production
  region: us-east-1

coordinator:
  address: coordinator.example.com:8080
  tls: true
  headers:
    authorization: Bearer mytoken

ethereum:
  beaconNodes:
    - name: lighthouse-1
      address: http://lighthouse-1:5052
    - name: prysm-1
      address: http://prysm-1:3500
    - name: teku-1
      address: http://teku-1:5051
  healthCheckInterval: 3s

dedupCache:
  ttl: 13m

reorg:
  enabled: true
  maxDepth: 64

outputs:
  - name: xatu-server
    type: xatu
    config:
      address: xatu.example.com:8080
      tls: true
      headers:
        authorization: Bearer mytoken
      maxQueueSize: 51200
      batchTimeout: 5s
      maxExportBatchSize: 512
```
