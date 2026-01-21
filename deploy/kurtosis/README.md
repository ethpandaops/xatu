# Kurtosis E2E Testing for Horizon

This directory contains configuration files for running E2E tests of the Horizon module using Kurtosis.

## Quick Start (Automated)

The easiest way to run the E2E test is using the automated script:

```bash
# Full test (~15 minutes, 2 epochs)
./scripts/e2e-horizon-test.sh

# Quick test (~8 minutes, 1 epoch)
./scripts/e2e-horizon-test.sh --quick

# Skip image build (use existing image)
./scripts/e2e-horizon-test.sh --skip-build

# Keep resources for debugging (no cleanup on exit)
./scripts/e2e-horizon-test.sh --skip-cleanup
```

The script handles:
- Building the Xatu Docker image
- Starting the docker-compose stack (ClickHouse, Kafka, PostgreSQL, xatu-server)
- Spinning up the Kurtosis Ethereum testnet with all 6 consensus clients
- Connecting networks between Kurtosis and docker-compose
- Generating Horizon configuration with actual beacon node URLs
- Starting Horizon and waiting for data collection
- Running validation queries against ClickHouse
- Reporting pass/fail status
- Cleaning up all resources on exit

## Architecture

The E2E test uses two separate infrastructure components:

1. **Kurtosis Network**: Runs the Ethereum testnet with all consensus clients
2. **Xatu Stack**: Runs via docker-compose (ClickHouse, Kafka, PostgreSQL, xatu-server, Horizon)

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                           Kurtosis Enclave                                   │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐        │
│  │ Lighthouse  │  │   Prysm     │  │    Teku     │  │  Lodestar   │        │
│  │   + Geth    │  │ +Nethermind │  │  + Erigon   │  │   + Reth    │        │
│  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘        │
│         │                │                │                │                │
│  ┌──────┴──────┐  ┌──────┴──────┐                                          │
│  │   Nimbus    │  │  Grandine   │                                          │
│  │   + Besu    │  │   + Geth    │                                          │
│  └──────┬──────┘  └──────┬──────┘                                          │
│         │                │                                                  │
└─────────┼────────────────┼──────────────────────────────────────────────────┘
          │                │
          │ SSE Events     │
          ▼                ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                         Docker Compose Stack                                 │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                          Horizon                                      │   │
│  │  - Connects to all 6 beacon nodes                                    │   │
│  │  - Deduplicates block events                                         │   │
│  │  - Derives canonical data                                            │   │
│  └───────────────────────────┬─────────────────────────────────────────┘   │
│                              │                                              │
│                              ▼                                              │
│  ┌───────────────────────────────────────────────────────────────────────┐ │
│  │                       xatu-server                                      │ │
│  │  - Event ingestion                                                     │ │
│  │  - Coordinator (location tracking)                                     │ │
│  └───────────────────────────┬───────────────────────────────────────────┘ │
│                              │                                              │
│         ┌────────────────────┼────────────────────┐                        │
│         ▼                    ▼                    ▼                        │
│  ┌─────────────┐      ┌─────────────┐      ┌─────────────┐                │
│  │   Kafka     │      │ PostgreSQL  │      │ ClickHouse  │                │
│  │  (events)   │      │ (locations) │      │  (storage)  │                │
│  └─────────────┘      └─────────────┘      └─────────────┘                │
└─────────────────────────────────────────────────────────────────────────────┘
```

## Files

- `horizon-test.yaml`: Kurtosis ethereum-package configuration with all 6 consensus clients
- `xatu-horizon.yaml`: Horizon configuration for connecting to Kurtosis beacon nodes
- `xatu-server.yaml`: Xatu server configuration for E2E testing

## Prerequisites

1. [Kurtosis](https://docs.kurtosis.com/install/) installed
2. Docker and Docker Compose installed
3. Xatu Docker image built: `docker build -t ethpandaops/xatu:local .`

## Running the E2E Test

### Step 1: Start the Xatu Stack

From the repository root:

```bash
# Start all xatu infrastructure (ClickHouse, Kafka, PostgreSQL, etc.)
docker compose up --detach
```

### Step 2: Start the Kurtosis Network

```bash
kurtosis run github.com/ethpandaops/ethereum-package \
  --args-file deploy/kurtosis/horizon-test.yaml \
  --enclave horizon
```

### Step 3: Get Beacon Node URLs

After Kurtosis starts, get the actual service URLs:

```bash
kurtosis enclave inspect horizon | grep -E "cl-.+-http"
```

Update the `xatu-horizon.yaml` file with the actual URLs, or set environment variables.

### Step 4: Connect Networks

Connect the Kurtosis containers to the xatu-net docker network:

```bash
# Get the Kurtosis network name
KURTOSIS_NETWORK=$(docker network ls | grep horizon | awk '{print $2}')

# Connect xatu containers to Kurtosis network (for beacon node access)
docker network connect $KURTOSIS_NETWORK xatu-server
docker network connect $KURTOSIS_NETWORK xatu-horizon
```

Or connect Kurtosis containers to xatu-net:

```bash
for container in $(kurtosis enclave inspect horizon | grep cl- | awk '{print $1}'); do
  docker network connect xatu_xatu-net $container
done
```

### Step 5: Start Horizon

Start Horizon with the Kurtosis configuration:

```bash
docker compose --profile horizon up xatu-horizon
```

Or run locally:

```bash
xatu horizon --config deploy/kurtosis/xatu-horizon.yaml
```

### Step 6: Verify Data in ClickHouse

Query ClickHouse to verify Horizon is producing data:

```bash
docker exec xatu-clickhouse-01 clickhouse-client --query "
  SELECT
    meta_client_name,
    COUNT(*) as events
  FROM default.beacon_api_eth_v2_beacon_block
  WHERE meta_client_module = 'HORIZON'
  GROUP BY meta_client_name
"
```

## Validation Queries

A comprehensive set of validation queries is available in `scripts/e2e-horizon-validate.sql`. Run them with:

```bash
# Run all validation queries
cat scripts/e2e-horizon-validate.sql | docker exec -i xatu-clickhouse-01 clickhouse-client

# Or if clickhouse-client is installed locally
cat scripts/e2e-horizon-validate.sql | clickhouse-client -h localhost
```

### Individual Queries

**Check for beacon blocks:**

```sql
SELECT
  slot,
  block_root,
  COUNT(*) as count
FROM default.beacon_api_eth_v2_beacon_block FINAL
WHERE meta_client_module = 'HORIZON'
GROUP BY slot, block_root
ORDER BY slot DESC
LIMIT 20;
```

**Check for no gaps in slot sequence:**

```sql
WITH slots AS (
  SELECT DISTINCT slot
  FROM default.beacon_api_eth_v2_beacon_block FINAL
  WHERE meta_client_module = 'HORIZON'
)
SELECT
  slot,
  slot - lagInFrame(slot, 1) OVER (ORDER BY slot) as gap
FROM slots
WHERE gap > 1
LIMIT 20;
```

**Count events per deriver:**

```sql
SELECT
  event_name,
  COUNT(*) as count
FROM (
  SELECT 'beacon_block' as event_name, COUNT(*) as c FROM default.beacon_api_eth_v2_beacon_block FINAL WHERE meta_client_module = 'HORIZON'
  UNION ALL
  SELECT 'attester_slashing', COUNT(*) FROM default.beacon_api_eth_v2_beacon_block_attester_slashing FINAL WHERE meta_client_module = 'HORIZON'
  UNION ALL
  SELECT 'proposer_slashing', COUNT(*) FROM default.beacon_api_eth_v2_beacon_block_proposer_slashing FINAL WHERE meta_client_module = 'HORIZON'
  UNION ALL
  SELECT 'deposit', COUNT(*) FROM default.beacon_api_eth_v2_beacon_block_deposit FINAL WHERE meta_client_module = 'HORIZON'
  UNION ALL
  SELECT 'withdrawal', COUNT(*) FROM default.beacon_api_eth_v2_beacon_block_withdrawal FINAL WHERE meta_client_module = 'HORIZON'
  UNION ALL
  SELECT 'voluntary_exit', COUNT(*) FROM default.beacon_api_eth_v2_beacon_block_voluntary_exit FINAL WHERE meta_client_module = 'HORIZON'
  UNION ALL
  SELECT 'bls_to_execution_change', COUNT(*) FROM default.beacon_api_eth_v2_beacon_block_bls_to_execution_change FINAL WHERE meta_client_module = 'HORIZON'
  UNION ALL
  SELECT 'execution_transaction', COUNT(*) FROM default.beacon_api_eth_v2_beacon_block_execution_transaction FINAL WHERE meta_client_module = 'HORIZON'
  UNION ALL
  SELECT 'elaborated_attestation', COUNT(*) FROM default.beacon_api_eth_v2_beacon_block_elaborated_attestation FINAL WHERE meta_client_module = 'HORIZON'
);
```

## Expected Results

After running the E2E test for 2 epochs (~13 minutes), you should see the following results:

### Query 1: Duplicate Blocks Check
**Expected:** Empty result set (no rows returned)

If deduplication is working correctly, there should be no duplicate blocks with the same `slot` and `block_root`. Despite receiving SSE events from multiple beacon nodes, Horizon's dedup cache ensures only one event per block is processed.

### Query 2: Slot Gaps Check
**Expected:** Empty or minimal results

Large gaps (>1 slot) between consecutive blocks indicate the FILL iterator may not be catching up properly. However, some gaps are acceptable:
- Gaps of 1 slot are normal (consecutive slots)
- Gaps may occur for genuinely missed blocks (no proposer)
- During initial sync, gaps are expected until FILL catches up

### Query 3: Module Verification
**Expected:** All rows have `meta_client_module = 'HORIZON'`

This confirms events were generated by the Horizon module and not by Cannon or other sources.

### Query 4: Events Per Deriver
**Expected counts after 2 epochs:**

| Event Type | Expected Count | Notes |
|------------|---------------|-------|
| beacon_block | ~64 | 32 slots/epoch × 2 epochs |
| elaborated_attestation | >1000 | Multiple attestations per block |
| execution_transaction | 0+ | Depends on test network activity |
| proposer_duty | ~64 | One duty per slot |
| beacon_committee | >100 | Multiple committees per epoch |
| beacon_validators | >0 | Chunked validator states |
| attester_slashing | 0 | Rare event, typically 0 |
| proposer_slashing | 0 | Rare event, typically 0 |
| deposit | 0+ | Only if deposits occur |
| withdrawal | 0+ | Only if withdrawals enabled |
| voluntary_exit | 0 | Rare event, typically 0 |
| bls_to_execution_change | 0 | Rare event, typically 0 |
| beacon_blob | 0+ | Deneb+ only, depends on blob txs |

### Query 5: Slot Coverage
**Expected:**
- `coverage_percent` > 90%
- `actual_slots` close to `expected_slots`

If coverage is significantly below 100%, check:
1. The FILL iterator is running
2. The LAG setting isn't too large
3. No beacon node connectivity issues

### Query 6: Block Latency
**Expected:**
- `avg_latency_seconds` < 30s for real-time blocks
- `min_latency_seconds` should be low (< 5s) for HEAD-processed blocks
- `max_latency_seconds` may be higher for FILL-processed blocks

### Query 7: Events by Node
**Expected:** All events attributed to Horizon instance name (e.g., `horizon-e2e-test`)

Events from all beacon nodes should be deduplicated into a single stream. The `meta_client_name` should match the configured Horizon instance name.

### Query 8: Recent Blocks
**Expected:** Shows the 10 most recent slots with block data

Use this for quick visual verification that data is flowing. The `slot` values should be recent and increasing.

### Validation Summary
**Expected:** All columns return `1` (true)

| Check | Expected | Description |
|-------|----------|-------------|
| has_beacon_blocks | 1 | Beacon blocks are being collected |
| no_duplicates | 1 | Deduplication is working |
| has_attestations | 1 | Attestation deriver is working |
| has_proposer_duties | 1 | Proposer duty deriver is working |
| has_committees | 1 | Committee deriver is working |
| good_coverage | 1 | >90% slot coverage |

If any check returns `0`, investigate the specific deriver or component.

## Cleanup

```bash
# Stop Kurtosis network
kurtosis enclave stop horizon
kurtosis enclave rm horizon

# Stop xatu stack
docker compose down -v
```

## Consensus Clients Tested

| Client     | EL Pair    | Beacon API Port |
|------------|------------|-----------------|
| Lighthouse | Geth       | 4000            |
| Prysm      | Nethermind | 3500            |
| Teku       | Erigon     | 4000            |
| Lodestar   | Reth       | 4000            |
| Nimbus     | Besu       | 4000            |
| Grandine   | Geth       | 4000            |

## Manual Test Procedure

For debugging or step-by-step execution, follow this manual procedure:

### Step 1: Build the Xatu Image

```bash
cd /path/to/xatu
docker build -t ethpandaops/xatu:local .
```

### Step 2: Start the Xatu Stack

```bash
docker compose up --detach
```

Wait for all services to be healthy:
```bash
docker compose ps
```

### Step 3: Start the Kurtosis Network

```bash
kurtosis run github.com/ethpandaops/ethereum-package \
  --args-file deploy/kurtosis/horizon-test.yaml \
  --enclave horizon-e2e
```

Wait for the network to start (this may take 2-3 minutes).

### Step 4: Get Beacon Node Containers

```bash
kurtosis enclave inspect horizon-e2e | grep -E "^cl-" | grep -v validator
```

### Step 5: Connect Networks

Connect Kurtosis containers to the xatu network:

```bash
for container in $(kurtosis enclave inspect horizon-e2e | grep -E "^cl-" | grep -v validator | awk '{print $1}'); do
  docker network connect xatu_xatu-net "$container" 2>/dev/null || true
  echo "Connected: $container"
done
```

### Step 6: Generate Horizon Config

Create a config file with actual beacon node URLs:

```bash
# Get container names
LIGHTHOUSE=$(kurtosis enclave inspect horizon-e2e | grep cl-lighthouse | grep -v validator | head -n1 | awk '{print $1}')
PRYSM=$(kurtosis enclave inspect horizon-e2e | grep cl-prysm | grep -v validator | head -n1 | awk '{print $1}')
TEKU=$(kurtosis enclave inspect horizon-e2e | grep cl-teku | grep -v validator | head -n1 | awk '{print $1}')
LODESTAR=$(kurtosis enclave inspect horizon-e2e | grep cl-lodestar | grep -v validator | head -n1 | awk '{print $1}')
NIMBUS=$(kurtosis enclave inspect horizon-e2e | grep cl-nimbus | grep -v validator | head -n1 | awk '{print $1}')
GRANDINE=$(kurtosis enclave inspect horizon-e2e | grep cl-grandine | grep -v validator | head -n1 | awk '{print $1}')

echo "Beacon nodes:"
echo "  Lighthouse: $LIGHTHOUSE"
echo "  Prysm: $PRYSM"
echo "  Teku: $TEKU"
echo "  Lodestar: $LODESTAR"
echo "  Nimbus: $NIMBUS"
echo "  Grandine: $GRANDINE"
```

Update `deploy/kurtosis/xatu-horizon.yaml` with these container names.

### Step 7: Start Horizon

```bash
docker run -d \
  --name xatu-horizon \
  --network xatu_xatu-net \
  -v $(pwd)/deploy/kurtosis/xatu-horizon.yaml:/etc/xatu/config.yaml:ro \
  ethpandaops/xatu:local \
  horizon --config /etc/xatu/config.yaml
```

### Step 8: Monitor Progress

Check Horizon logs:
```bash
docker logs -f xatu-horizon
```

Check block count in ClickHouse:
```bash
docker exec xatu-clickhouse-01 clickhouse-client --query "
  SELECT COUNT(*) as blocks
  FROM beacon_api_eth_v2_beacon_block FINAL
  WHERE meta_client_module = 'HORIZON'
"
```

### Step 9: Run Validation Queries

After waiting 2 epochs (~13 minutes), run the validation queries from the "Validation Queries" section above.

### Step 10: Cleanup

```bash
# Stop Horizon
docker stop xatu-horizon && docker rm xatu-horizon

# Stop Kurtosis
kurtosis enclave stop horizon-e2e && kurtosis enclave rm horizon-e2e

# Stop docker-compose
docker compose down -v
```

## Notes

- The E2E test uses the main docker-compose.yml which includes ClickHouse with full schema migrations
- Horizon connects to all 6 beacon nodes simultaneously, testing the multi-beacon node pool functionality
- Block deduplication ensures only one event per block root despite receiving from multiple beacon nodes
- The coordinator tracks progress, allowing Horizon to resume from where it left off
- The automated script (`scripts/e2e-horizon-test.sh`) is recommended for CI/CD pipelines
