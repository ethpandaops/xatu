# PRD: Horizon - Head Data Collection Module

## Introduction

Horizon is a new Xatu module for collecting canonical **head** (non-finalized) blockchain data from multiple beacon nodes with high availability (HA) support. Unlike Cannon which processes finalized epochs, Horizon operates at the chain head, subscribing to real-time beacon node SSE events and deriving structured data from head blocks.

The module addresses the challenge of reliably collecting head data across distributed beacon node infrastructure while ensuring:
- **No duplicate events**: When connected to 10 beacon nodes all reporting the same block, only one set of derived events is emitted
- **No missed blocks**: A consistency fill iterator guarantees every slot is processed, even if SSE events are dropped
- **Immediate head tracking**: After downtime, the module immediately resumes head tracking rather than waiting for backfill to complete
- **HA deployment**: Multiple Horizon instances can run safely with coordinator-based state sharing

## Goals

- Derive the same event types as Cannon (beacon blocks, attestations, slashings, deposits, withdrawals, etc.) but for head data instead of finalized data
- **Use identical event types as Cannon** - xatu-server routes events by `MODULE_NAME` (HORIZON vs CANNON), not event type
- Support connecting to multiple upstream beacon nodes simultaneously
- Provide local deduplication to prevent emitting duplicate events when the same block is reported by multiple beacon nodes
- Enable HA deployments where multiple Horizon instances coordinate via the existing Coordinator service
- Implement a dual-iterator design: HEAD iterator for real-time data + FILL iterator for consistency catch-up
- Ensure the FILL iterator never blocks HEAD processing - they operate independently
- Achieve feature parity with Cannon's 13 derivers for head data
- **Refactor derivers into shared package** - both Cannon and Horizon use the same deriver implementations
- **End-to-end validation** - verified working with Kurtosis ethereum-package and all consensus clients

## User Stories

### US-001: Create Horizon module skeleton and CLI command
**Description:** As an operator, I want to run Xatu in "horizon" mode so that I can collect head data from my beacon nodes.

**Acceptance Criteria:**
- [ ] New `pkg/horizon/` directory structure mirrors Cannon's organization
- [ ] `cmd/horizon.go` command added to CLI with `xatu horizon` subcommand
- [ ] Basic configuration loading with YAML support
- [ ] Metrics server starts on configured address
- [ ] Module logs startup message with version and instance ID
- [ ] Graceful shutdown on SIGTERM/SIGINT
- [ ] Typecheck/lint passes

### US-002: Multi-beacon node connection management
**Description:** As an operator, I want Horizon to connect to multiple beacon nodes so that I have redundancy and can see the chain from multiple perspectives.

**Acceptance Criteria:**
- [ ] Configuration accepts array of beacon node URLs with optional headers
- [ ] Each beacon node connection is established independently
- [ ] Health checking per beacon node with configurable interval
- [ ] Failed connections are retried with exponential backoff
- [ ] Metrics track connection status per beacon node
- [ ] At least one healthy beacon node required to operate
- [ ] Typecheck/lint passes

### US-003: SSE event subscription for head blocks
**Description:** As a data collector, I want Horizon to subscribe to beacon node block events so that I receive real-time notifications of new head blocks.

**Acceptance Criteria:**
- [ ] Subscribe to `/eth/v1/events?topics=block` SSE stream on each beacon node
- [ ] Handle SSE reconnection on connection loss
- [ ] Parse block event payload (slot, block root, execution_optimistic flag)
- [ ] Route block events to deduplication layer
- [ ] Metrics track events received per beacon node
- [ ] Typecheck/lint passes

### US-004: Local deduplication by block root
**Description:** As a data collector, I want Horizon to deduplicate block events locally so that the same block reported by multiple beacon nodes only triggers derivation once.

**Acceptance Criteria:**
- [ ] TTL-based cache keyed by block root (configurable TTL, default 2 epochs / ~13 minutes)
- [ ] First block event for a root triggers derivation
- [ ] Subsequent events for the same root within TTL are dropped
- [ ] Cache cleanup runs periodically to prevent memory growth
- [ ] Metrics track cache hits/misses and deduplication rate
- [ ] Typecheck/lint passes

### US-005: Coordinator-based slot location tracking
**Description:** As an operator running multiple Horizon instances, I want them to share state via the Coordinator so that they don't process the same slots.

**Acceptance Criteria:**
- [ ] New `HorizonLocation` protobuf message with HEAD and FILL slot markers
- [ ] `GetHorizonLocation` and `UpsertHorizonLocation` Coordinator RPC methods
- [ ] Location tracked per deriver type and network (similar to Cannon)
- [ ] Atomic location updates to prevent race conditions
- [ ] Metrics expose current HEAD and FILL slot positions
- [ ] Typecheck/lint passes

### US-006: HEAD iterator for real-time slot processing
**Description:** As a data collector, I want a HEAD iterator that processes slots as they arrive so that I immediately capture head data.

**Acceptance Criteria:**
- [ ] HEAD iterator receives slot notifications from SSE deduplication layer
- [ ] HEAD iterator fetches full block data for the slot
- [ ] HEAD iterator passes block to derivers for event extraction
- [ ] HEAD iterator updates coordinator location after successful derivation
- [ ] HEAD iterator operates independently from FILL iterator
- [ ] HEAD iterator can skip slots if they've already been processed (race with FILL)
- [ ] Typecheck/lint passes

### US-007: FILL iterator for consistency catch-up
**Description:** As an operator, I want a FILL iterator that ensures no slots are missed so that I have complete data even if SSE events are dropped.

**Acceptance Criteria:**
- [ ] FILL iterator walks slots from its last position toward HEAD - LAG
- [ ] Configurable LAG distance (default: 32 slots / 1 epoch behind head)
- [ ] FILL iterator checks if slot already processed before fetching
- [ ] FILL iterator has configurable batch size for efficiency
- [ ] FILL iterator has rate limiting to avoid overwhelming beacon nodes
- [ ] FILL iterator updates coordinator location after successful derivation
- [ ] Typecheck/lint passes

### US-008: Dual-iterator coordination
**Description:** As an operator, I want HEAD and FILL iterators to coordinate so that HEAD always takes priority and they don't duplicate work.

**Acceptance Criteria:**
- [ ] HEAD iterator has priority - FILL never blocks HEAD processing
- [ ] Separate location markers in coordinator: `head_slot` and `fill_slot`
- [ ] On startup, HEAD iterator immediately begins tracking new blocks
- [ ] On startup, FILL iterator begins from `fill_slot` toward `HEAD - LAG`
- [ ] Both iterators skip slots marked as processed by the other
- [ ] Configurable bounded range for FILL (e.g., never fill more than N slots back)
- [ ] Typecheck/lint passes

### US-009: Refactor derivers to shared package
**Description:** As a developer, I want derivers shared between Cannon and Horizon so that we maintain a single source of truth for derivation logic.

**Acceptance Criteria:**
- [ ] Create new `pkg/cldata/` package for shared consensus layer data derivation
- [ ] Move block fetching and parsing logic to shared package
- [ ] Move all 13 deriver implementations to shared package
- [ ] Derivers accept an iterator interface (epoch-based for Cannon, slot-based for Horizon)
- [ ] Derivers accept a context provider interface for client metadata
- [ ] Cannon continues to work identically after refactor
- [ ] Comprehensive tests verify no regression in Cannon behavior
- [ ] Typecheck/lint passes

### US-010: BeaconBlock deriver for head data
**Description:** As a data analyst, I want Horizon to derive beacon block events from head data so that I can analyze blocks before finalization.

**Acceptance Criteria:**
- [ ] Use shared `BeaconBlockDeriver` from `pkg/cldata/`
- [ ] Derive `BEACON_API_ETH_V2_BEACON_BLOCK` events (same type as Cannon)
- [ ] Events routed by xatu-server based on `MODULE_NAME: HORIZON`
- [ ] Events include full block data matching Cannon's output format
- [ ] Deriver handles missing blocks (missed slots) gracefully
- [ ] Typecheck/lint passes

### US-011: AttesterSlashing deriver for head data
**Description:** As a data analyst, I want Horizon to derive attester slashing events from head blocks.

**Acceptance Criteria:**
- [ ] Use shared `AttesterSlashingDeriver` from `pkg/cldata/`
- [ ] Derive `BEACON_API_ETH_V2_BEACON_BLOCK_ATTESTER_SLASHING` events
- [ ] Events match Cannon's output format
- [ ] Typecheck/lint passes

### US-012: ProposerSlashing deriver for head data
**Description:** As a data analyst, I want Horizon to derive proposer slashing events from head blocks.

**Acceptance Criteria:**
- [ ] Use shared `ProposerSlashingDeriver` from `pkg/cldata/`
- [ ] Derive `BEACON_API_ETH_V2_BEACON_BLOCK_PROPOSER_SLASHING` events
- [ ] Events match Cannon's output format
- [ ] Typecheck/lint passes

### US-013: Deposit deriver for head data
**Description:** As a data analyst, I want Horizon to derive deposit events from head blocks.

**Acceptance Criteria:**
- [ ] Use shared `DepositDeriver` from `pkg/cldata/`
- [ ] Derive `BEACON_API_ETH_V2_BEACON_BLOCK_DEPOSIT` events
- [ ] Events match Cannon's output format
- [ ] Typecheck/lint passes

### US-014: Withdrawal deriver for head data
**Description:** As a data analyst, I want Horizon to derive withdrawal events from head blocks.

**Acceptance Criteria:**
- [ ] Use shared `WithdrawalDeriver` from `pkg/cldata/`
- [ ] Derive `BEACON_API_ETH_V2_BEACON_BLOCK_WITHDRAWAL` events
- [ ] Events match Cannon's output format
- [ ] Typecheck/lint passes

### US-015: VoluntaryExit deriver for head data
**Description:** As a data analyst, I want Horizon to derive voluntary exit events from head blocks.

**Acceptance Criteria:**
- [ ] Use shared `VoluntaryExitDeriver` from `pkg/cldata/`
- [ ] Derive `BEACON_API_ETH_V2_BEACON_BLOCK_VOLUNTARY_EXIT` events
- [ ] Events match Cannon's output format
- [ ] Typecheck/lint passes

### US-016: BLSToExecutionChange deriver for head data
**Description:** As a data analyst, I want Horizon to derive BLS to execution change events from head blocks.

**Acceptance Criteria:**
- [ ] Use shared `BLSToExecutionChangeDeriver` from `pkg/cldata/`
- [ ] Derive `BEACON_API_ETH_V2_BEACON_BLOCK_BLS_TO_EXECUTION_CHANGE` events
- [ ] Events match Cannon's output format
- [ ] Typecheck/lint passes

### US-017: ExecutionTransaction deriver for head data
**Description:** As a data analyst, I want Horizon to derive execution transaction events from head blocks.

**Acceptance Criteria:**
- [ ] Use shared `ExecutionTransactionDeriver` from `pkg/cldata/`
- [ ] Derive `BEACON_API_ETH_V2_BEACON_BLOCK_EXECUTION_TRANSACTION` events
- [ ] Events match Cannon's output format
- [ ] Typecheck/lint passes

### US-018: ElaboratedAttestation deriver for head data
**Description:** As a data analyst, I want Horizon to derive elaborated attestation events from head blocks.

**Acceptance Criteria:**
- [ ] Use shared `ElaboratedAttestationDeriver` from `pkg/cldata/`
- [ ] Derive `BEACON_API_ETH_V2_BEACON_BLOCK_ELABORATED_ATTESTATION` events
- [ ] Events match Cannon's output format
- [ ] Typecheck/lint passes

### US-019: ProposerDuty deriver for head data
**Description:** As a data analyst, I want Horizon to derive proposer duty events for upcoming epochs.

**Acceptance Criteria:**
- [ ] Use shared `ProposerDutyDeriver` from `pkg/cldata/`
- [ ] Fetch proposer duties for NEXT epoch midway through current epoch
- [ ] Derive `BEACON_API_ETH_V1_PROPOSER_DUTY` events
- [ ] Events match Cannon's output format
- [ ] Typecheck/lint passes

### US-020: BeaconBlob deriver for head data
**Description:** As a data analyst, I want Horizon to derive blob sidecar events from head blocks.

**Acceptance Criteria:**
- [ ] Use shared `BeaconBlobDeriver` from `pkg/cldata/`
- [ ] Derive `BEACON_API_ETH_V1_BEACON_BLOB_SIDECAR` events
- [ ] Events match Cannon's output format
- [ ] Respects fork activation (Deneb+)
- [ ] Typecheck/lint passes

### US-021: BeaconValidators deriver for head data
**Description:** As a data analyst, I want Horizon to derive validator state events for upcoming epochs.

**Acceptance Criteria:**
- [ ] Use shared `BeaconValidatorsDeriver` from `pkg/cldata/`
- [ ] Fetch validator state for NEXT epoch midway through current epoch
- [ ] Derive `BEACON_API_ETH_V1_BEACON_VALIDATORS` events
- [ ] Events match Cannon's output format
- [ ] Typecheck/lint passes

### US-022: BeaconCommittee deriver for head data
**Description:** As a data analyst, I want Horizon to derive committee assignment events for upcoming epochs.

**Acceptance Criteria:**
- [ ] Use shared `BeaconCommitteeDeriver` from `pkg/cldata/`
- [ ] Fetch committee assignments for NEXT epoch midway through current epoch
- [ ] Derive `BEACON_API_ETH_V1_BEACON_COMMITTEE` events
- [ ] Events match Cannon's output format
- [ ] Typecheck/lint passes

### US-023: Reorg handling
**Description:** As a data collector, I want Horizon to handle chain reorgs gracefully so that I can track which blocks were reorged.

**Acceptance Criteria:**
- [ ] Subscribe to chain_reorg SSE events on each beacon node
- [ ] When reorg detected, mark affected slots for re-processing
- [ ] Configurable reorg depth limit (default: 64 slots, ~2 epochs)
- [ ] Derive events for new canonical blocks
- [ ] Add `reorg_detected: true` metadata to events derived after reorg
- [ ] Metrics track reorg frequency and depth
- [ ] Typecheck/lint passes

### US-024: Configuration and example files
**Description:** As an operator, I want comprehensive configuration options and example files so that I can deploy Horizon correctly.

**Acceptance Criteria:**
- [ ] `example_horizon.yaml` with documented configuration
- [ ] Configuration for multiple beacon nodes with failover
- [ ] Configuration for HEAD and FILL iterator behaviors
- [ ] Configuration for deduplication TTL
- [ ] Configuration for LAG distance
- [ ] Configuration for reorg depth limit
- [ ] Configuration for each deriver (enable/disable)
- [ ] Configuration validation on startup
- [ ] Typecheck/lint passes

### US-025: Documentation
**Description:** As an operator, I want documentation for the Horizon module so that I understand how to deploy and operate it.

**Acceptance Criteria:**
- [ ] `docs/horizon.md` with architecture overview
- [ ] Explanation of dual-iterator design
- [ ] Explanation of multi-beacon node connection
- [ ] Explanation of HA deployment with coordinator
- [ ] Comparison with Cannon (when to use which)
- [ ] Troubleshooting guide
- [ ] Metrics reference

### US-026: Local docker-compose E2E testing setup
**Description:** As a developer, I want a local docker-compose setup for Horizon so that I can test the full pipeline locally.

**Acceptance Criteria:**
- [ ] Add Horizon service to `deploy/local/docker-compose.yml`
- [ ] Horizon connects to local beacon node(s)
- [ ] Horizon sends events to local xatu-server
- [ ] xatu-server routes Horizon events to ClickHouse
- [ ] ClickHouse tables receive Horizon-derived data
- [ ] Documentation for running the local E2E test
- [ ] Typecheck/lint passes

### US-027: Kurtosis ethereum-package E2E test
**Description:** As a developer, I want to run Horizon against a Kurtosis ethereum-package network with all consensus clients so that I can validate compatibility across all CLs.

**Acceptance Criteria:**
- [ ] Kurtosis network config with all consensus clients (Lighthouse, Prysm, Teku, Lodestar, Nimbus, Grandine)
- [ ] Horizon configuration to connect to all CL beacon nodes in the network
- [ ] Test script to spin up Kurtosis network + Xatu stack (coordinator, server, Horizon, ClickHouse)
- [ ] Verification script that queries ClickHouse to confirm blocks are landing
- [ ] Test passes with blocks from all CL clients visible in ClickHouse
- [ ] CI integration or documented manual test procedure
- [ ] Test runs for at least 2 epochs (~13 minutes) to verify consistency

### US-028: E2E validation queries
**Description:** As a developer, I want validation queries to confirm Horizon is working correctly so that I can verify the E2E test passes.

**Acceptance Criteria:**
- [ ] Query to count beacon blocks by slot in ClickHouse
- [ ] Query to verify no duplicate blocks for same slot (deduplication working)
- [ ] Query to verify no gaps in slot sequence (FILL iterator working)
- [ ] Query to verify events have `module_name = 'HORIZON'`
- [ ] Query to count events per deriver type
- [ ] All queries return expected results after test run
- [ ] Queries documented in test README

## Functional Requirements

### Core Module
- FR-1: Horizon module MUST start with `xatu horizon --config <path>` CLI command
- FR-2: Horizon module MUST connect to one or more beacon nodes specified in configuration
- FR-3: Horizon module MUST subscribe to SSE block events on all connected beacon nodes
- FR-4: Horizon module MUST maintain connection health and reconnect on failures
- FR-5: Horizon module MUST expose Prometheus metrics on configured address

### Deduplication
- FR-6: Horizon MUST deduplicate block events by block root using a TTL cache
- FR-7: TTL cache MUST be configurable with default of 2 epochs (~13 minutes)
- FR-8: Only the first block event for a given root MUST trigger derivation
- FR-9: Deduplication MUST occur locally before coordinator checks

### Coordinator Integration
- FR-10: Horizon MUST store HEAD slot position in coordinator per deriver type
- FR-11: Horizon MUST store FILL slot position in coordinator per deriver type
- FR-12: Coordinator locations MUST be updated atomically after successful derivation
- FR-13: Multiple Horizon instances MUST coordinate to avoid duplicate processing

### HEAD Iterator
- FR-14: HEAD iterator MUST process slots immediately when SSE events arrive
- FR-15: HEAD iterator MUST fetch full block data from any healthy beacon node
- FR-16: HEAD iterator MUST pass blocks to all enabled derivers
- FR-17: HEAD iterator MUST update coordinator location after derivation
- FR-18: HEAD iterator MUST skip slots already processed by FILL iterator

### FILL Iterator
- FR-19: FILL iterator MUST walk slots from its last position toward (HEAD - LAG)
- FR-20: FILL iterator MUST respect configurable LAG distance (default 32 slots)
- FR-21: FILL iterator MUST check coordinator before processing each slot
- FR-22: FILL iterator MUST NOT block HEAD iterator processing
- FR-23: FILL iterator MUST have configurable rate limiting
- FR-24: FILL iterator MUST have configurable bounded range for catch-up

### Derivers
- FR-25: Horizon MUST support all 13 deriver types from Cannon
- FR-26: Derivers MUST produce events with same types as Cannon (e.g., `BEACON_API_ETH_V2_BEACON_BLOCK`)
- FR-27: Events MUST be distinguishable by `MODULE_NAME: HORIZON` in client metadata
- FR-28: Each deriver MUST be independently enable/disable via configuration
- FR-29: Derivers MUST respect fork activation epochs
- FR-30: Epoch-boundary derivers (validators, committees, proposer duties) MUST fetch for NEXT epoch midway through current epoch

### Reorg Handling
- FR-31: Horizon MUST subscribe to chain_reorg SSE events
- FR-32: On reorg, Horizon MUST mark affected slots for re-derivation
- FR-33: Reorg re-derivation depth MUST be configurable (default: 64 slots)
- FR-34: Reorg-triggered events MUST include reorg metadata

### Output
- FR-35: Horizon MUST support all Cannon output sinks (Xatu server, stdout, etc.)
- FR-36: Events MUST follow the same DecoratedEvent protobuf format as Cannon

### Shared Code
- FR-37: Derivers MUST be refactored to `pkg/cldata/` shared package
- FR-38: Both Cannon and Horizon MUST use shared deriver implementations
- FR-39: Refactoring MUST NOT break existing Cannon functionality

### E2E Testing
- FR-40: Local docker-compose MUST support running full Horizon pipeline
- FR-41: Kurtosis E2E test MUST validate Horizon with all consensus clients
- FR-42: E2E test MUST verify blocks land in ClickHouse
- FR-43: E2E test MUST verify no duplicate or missing slots

## Non-Goals (Out of Scope)

- **Historical backfill beyond bounded range**: Horizon is for head data; use Cannon for deep historical backfill
- **Finality confirmation**: Horizon emits events immediately; finality tracking is not in scope
- **Execution layer data**: Focus on consensus layer data only (matching Cannon scope)
- **Attestation pool monitoring**: Only attestations included in blocks are derived
- **Mempool monitoring**: Out of scope; use Sentry for real-time mempool data
- **Block building/MEV analysis**: Out of scope; use Relay Monitor for MEV data
- **Automatic Cannon handoff**: No automatic transition to Cannon once data is finalized
- **New event types**: Horizon uses identical event types as Cannon; routing is by MODULE_NAME

## Design Considerations

### Shared Deriver Architecture

The derivers will be refactored to a shared `pkg/cldata/` package:

```
pkg/cldata/
├── deriver/
│   ├── interface.go       # Deriver interface definitions
│   ├── beacon_block.go    # BeaconBlockDeriver implementation
│   ├── attester_slashing.go
│   ├── proposer_slashing.go
│   ├── deposit.go
│   ├── withdrawal.go
│   ├── voluntary_exit.go
│   ├── bls_to_execution_change.go
│   ├── execution_transaction.go
│   ├── elaborated_attestation.go
│   ├── proposer_duty.go
│   ├── beacon_blob.go
│   ├── beacon_validators.go
│   └── beacon_committee.go
├── iterator/
│   ├── interface.go       # Iterator interface (epoch-based, slot-based)
│   └── types.go           # Shared types
└── block/
    ├── fetcher.go         # Block fetching logic
    └── parser.go          # Block parsing logic
```

**Key interfaces:**

```go
// Iterator provides the next item to process
type Iterator interface {
    Next(ctx context.Context) (*NextResponse, error)
    UpdateLocation(ctx context.Context, position uint64, direction Direction) error
}

// Deriver extracts events from beacon data
type Deriver interface {
    Start(ctx context.Context) error
    Stop(ctx context.Context) error
    Name() string
    EventType() xatu.EventType
    ActivationFork() spec.DataVersion
    OnEventsDerived(ctx context.Context, fn func(ctx context.Context, events []*xatu.DecoratedEvent) error)
}

// ContextProvider supplies client metadata and network info
type ContextProvider interface {
    CreateClientMeta(ctx context.Context) (*xatu.ClientMeta, error)
    NetworkName() string
    NetworkID() string
    Wallclock() *ethwallclock.EthereumBeaconChain
}
```

### Iterator Architecture

```
                    ┌─────────────────┐
                    │  Beacon Nodes   │
                    │  (1..N)         │
                    └────────┬────────┘
                             │ SSE block events
                             ▼
                    ┌─────────────────┐
                    │  Deduplication  │
                    │  Cache          │
                    │  (by block root)│
                    └────────┬────────┘
                             │ unique blocks
              ┌──────────────┴──────────────┐
              │                             │
              ▼                             ▼
    ┌─────────────────┐          ┌─────────────────┐
    │  HEAD Iterator  │          │  FILL Iterator  │
    │  (real-time)    │          │  (catch-up)     │
    │                 │          │                 │
    │  Priority: HIGH │          │  Priority: LOW  │
    └────────┬────────┘          └────────┬────────┘
             │                            │
             │    ┌─────────────────┐     │
             └───►│   Coordinator   │◄────┘
                  │   (shared state)│
                  │  per-deriver    │
                  └────────┬────────┘
                           │
                           ▼
                  ┌─────────────────┐
                  │ Shared Derivers │
                  │  (pkg/cldata/)  │
                  └────────┬────────┘
                           │
                           ▼
                  ┌─────────────────┐
                  │     Sinks       │
                  │  (outputs)      │
                  └─────────────────┘
```

### Epoch-Boundary Deriver Timing

For derivers that operate on epoch boundaries (validators, committees, proposer duties):

```
Epoch N                              Epoch N+1
├──────────────────────────────────┤├──────────────────────────────────┤
│  slot 0  │  ...  │  slot 16  │   ││  slot 0  │  ...  │  slot 31  │   │
│          │       │  ^        │   ││          │       │           │   │
│          │       │  │        │   ││          │       │           │   │
│          │       │  Fetch    │   ││          │       │           │   │
│          │       │  epoch    │   ││          │       │           │   │
│          │       │  N+1 data │   ││          │       │           │   │
```

- At slot 16 of epoch N (midway), fetch data for epoch N+1
- This ensures data is available before the epoch starts
- Configurable trigger point (default: 50% through epoch)

### Protobuf Changes Required

New messages needed in `pkg/proto/xatu/`:
- `HorizonLocation` - slot-based location marker for HEAD and FILL per deriver
- Coordinator RPC extensions for Horizon location get/upsert

**No new event types** - Horizon uses the same `CannonType` enum values as Cannon. Events are distinguished by `ClientMeta.ModuleName = HORIZON`.

### Beacon Node Selection for Fetching

When fetching full block data:
1. **For HEAD iterator**: Prefer the beacon node that reported the SSE event (block is cached there)
2. **For FILL iterator**: Round-robin across healthy beacon nodes
3. **On failure**: Retry with exponential backoff, try next healthy node
4. **Timeout**: Configurable per-request timeout (default: 10s)

### Metrics

Key metrics to expose:
- `xatu_horizon_head_slot` - current HEAD iterator position
- `xatu_horizon_fill_slot` - current FILL iterator position
- `xatu_horizon_lag_slots` - difference between head and fill
- `xatu_horizon_dedup_cache_size` - current cache entries
- `xatu_horizon_dedup_hits_total` - deduplicated events count
- `xatu_horizon_blocks_derived_total` - blocks processed per deriver
- `xatu_horizon_beacon_node_status` - connection health per node
- `xatu_horizon_reorgs_total` - chain reorgs detected
- `xatu_horizon_reorg_depth` - histogram of reorg depths

### Configuration Structure

```yaml
name: horizon-mainnet-01

ethereum:
  network: mainnet
  beaconNodes:
    - url: http://beacon-1:5052
      headers: {}
    - url: http://beacon-2:5052
      headers: {}
    - url: http://beacon-3:5052
      headers: {}
  healthCheckInterval: 3s

coordinator:
  address: coordinator:8080
  headers:
    authorization: "Bearer xxx"

deduplication:
  ttl: 13m  # ~2 epochs

reorg:
  maxDepth: 64  # slots to re-derive on reorg

iterators:
  head:
    enabled: true
  fill:
    enabled: true
    lagSlots: 32          # 1 epoch behind head
    maxBoundedSlots: 7200 # ~1 day max catch-up
    rateLimit: 10         # slots per second

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
    epochTriggerPercent: 50  # fetch at 50% through epoch
  beaconBlob:
    enabled: true
  beaconValidators:
    enabled: true
    epochTriggerPercent: 50
  beaconCommittee:
    enabled: true
    epochTriggerPercent: 50

outputs:
  - name: xatu-server
    type: xatu
    config:
      address: xatu-server:8080

metricsAddr: ":9090"
```

### E2E Test Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                     Kurtosis ethereum-package                        │
│  ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐       │
│  │Lighthouse│ │  Prysm  │ │  Teku   │ │Lodestar │ │ Nimbus  │ ...   │
│  └────┬────┘ └────┬────┘ └────┬────┘ └────┬────┘ └────┬────┘       │
│       │           │           │           │           │             │
└───────┼───────────┼───────────┼───────────┼───────────┼─────────────┘
        │           │           │           │           │
        └───────────┴───────────┴───────────┴───────────┘
                              │
                    SSE subscriptions
                              │
                              ▼
                    ┌─────────────────┐
                    │     Horizon     │
                    └────────┬────────┘
                             │
                             ▼
                    ┌─────────────────┐
                    │  Xatu Server    │
                    └────────┬────────┘
                             │
                             ▼
                    ┌─────────────────┐
                    │   ClickHouse    │
                    └─────────────────┘
                             │
                             ▼
                    ┌─────────────────┐
                    │ Validation      │
                    │ Queries         │
                    └─────────────────┘
```

**Validation queries:**
```sql
-- Count blocks per slot (should be 1 per slot, no duplicates)
SELECT slot, count(*) as cnt
FROM beacon_api_eth_v2_beacon_block
WHERE meta_client_module = 'HORIZON'
GROUP BY slot
HAVING cnt > 1;

-- Check for gaps in slots
SELECT t1.slot + 1 as missing_slot
FROM beacon_api_eth_v2_beacon_block t1
LEFT JOIN beacon_api_eth_v2_beacon_block t2 ON t1.slot + 1 = t2.slot
WHERE t2.slot IS NULL
  AND t1.meta_client_module = 'HORIZON'
  AND t1.slot < (SELECT max(slot) FROM beacon_api_eth_v2_beacon_block WHERE meta_client_module = 'HORIZON');

-- Count events by deriver type
SELECT meta_event_name, count(*)
FROM xatu_events
WHERE meta_client_module = 'HORIZON'
GROUP BY meta_event_name;
```

## Success Metrics

- HEAD iterator processes new blocks within 500ms of SSE event receipt
- FILL iterator catches up to HEAD - LAG within configured rate limits
- Zero duplicate events emitted for the same block across multiple beacon nodes
- Zero missed slots over 24-hour observation period (with FILL enabled)
- HA deployment with 3 instances shows even load distribution
- Memory usage remains stable (dedup cache bounded)
- CPU usage proportional to derivation workload
- **E2E test passes with all 6 consensus clients**
- **Blocks visible in ClickHouse within 5 seconds of slot time**

## Open Questions

*All questions have been resolved:*

1. ~~Event type naming~~ **Resolved**: Use same event types as Cannon; route by MODULE_NAME
2. ~~Shared deriver refactoring~~ **Resolved**: Yes, refactor to shared `pkg/cldata/` package
3. ~~Reorg depth limit~~ **Resolved**: 64 slots default, configurable
4. ~~Validator/Committee derivers~~ **Resolved**: Fetch for next epoch midway through current epoch
5. ~~Block availability~~ **Resolved**: Retry with exponential backoff, try other healthy nodes
6. ~~Coordinator lock granularity~~ **Resolved**: Per-deriver for parallelism
