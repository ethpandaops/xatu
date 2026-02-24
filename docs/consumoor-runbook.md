# Consumoor Operational Runbook

## Architecture Overview

Consumoor is a Kafka-to-ClickHouse pipeline that reads Ethereum network events from Kafka topics and writes them to ClickHouse tables.

**Data flow:**

```
Kafka topics
    |
    v
Benthos (kafka_franz input)
    |
    v
Decoder (JSON or Protobuf -> DecoratedEvent)
    |
    v
Router Engine (event name -> target table(s))
    |
    v
Per-table buffer (bounded channel, one per table)
    |
    v
Table Writer (organic + coordinated flush)
    |
    v
ClickHouse (ch-go columnar inserts)
```

**Key design decisions:**

- **At-least-once delivery.** Kafka offsets are committed periodically (`commitInterval`, default 5s). On crash, messages since the last commit are replayed. ClickHouse must tolerate duplicate inserts (e.g. via `ReplacingMergeTree`).
- **Per-topic consumer groups.** Each consumoor instance joins a single consumer group. Multiple instances with different consumer groups or topic patterns can run in parallel.
- **Organic vs coordinated flushes.** Each table writer flushes independently when its batch reaches `batchSize` rows or when `flushInterval` elapses (organic). The Benthos output plugin can also trigger a coordinated `FlushAll` after writing a batch of messages, depending on the `deliveryMode`.
- **Backpressure propagation.** Each table's buffer is a bounded channel (`bufferSize`). When a buffer fills (e.g. ClickHouse is slow), `Write()` blocks, which backs up the Benthos output, which backs up Kafka consumption. This prevents unbounded memory growth.
- **Error classification.** Write errors are classified as permanent (schema mismatch, unknown table, type errors) or transient (network, timeout, overload). Permanent errors drop the batch and optionally send messages to a DLQ topic. Transient errors trigger retry with exponential backoff.

## Key Metrics

All metrics use the prefix `xatu_consumoor_`.

### Pipeline Metrics

| Metric | Type | Labels | Description | What to watch |
|--------|------|--------|-------------|---------------|
| `messages_consumed_total` | Counter | `topic` | Kafka messages consumed | Should increase steadily; flat = consumption stalled |
| `messages_routed_total` | Counter | `event_name`, `table` | Messages successfully routed to a table | Flat with consumed rising = routing issue |
| `messages_dropped_total` | Counter | `event_name`, `reason` | Messages dropped (no flattener, disabled, filtered) | Reason `no_flattener` = missing route registration |
| `messages_rejected_total` | Counter | `reason` | Permanently rejected messages | Any increase needs investigation |
| `decode_errors_total` | Counter | `topic` | Failed to decode message payload | Rising = bad data on topic or encoding mismatch |
| `flatten_errors_total` | Counter | `event_name`, `table` | Flattener conversion errors | Rising = schema/code mismatch |

### Write Path Metrics

| Metric | Type | Labels | Description | What to watch |
|--------|------|--------|-------------|---------------|
| `rows_written_total` | Counter | `table` | Rows successfully written to ClickHouse | Should increase; flat = writes stalled |
| `write_errors_total` | Counter | `table` | ClickHouse write errors | Any increase; check permanent vs transient |
| `write_duration_seconds` | Histogram | `table` | Duration of batch inserts | p99 > `queryTimeout` = ClickHouse too slow |
| `batch_size` | Histogram | `table` | Rows per batch write | Very small batches = inefficient; tune `batchSize` |
| `buffer_usage` | Gauge | `table` | Current buffered rows per table | Approaching `bufferSize` = backpressure building |

### DLQ Metrics

| Metric | Type | Labels | Description | What to watch |
|--------|------|--------|-------------|---------------|
| `dlq_writes_total` | Counter | `reason` | Messages written to DLQ topic | Any increase = messages being rejected |
| `dlq_errors_total` | Counter | `reason` | Failed DLQ writes | Rising = DLQ Kafka connectivity issue |

### Connection Pool Metrics

| Metric | Type | Description | What to watch |
|--------|------|-------------|---------------|
| `chgo_pool_acquired_resources` | Gauge | Currently in-use connections | Sustained at `max_resources` = pool exhaustion |
| `chgo_pool_idle_resources` | Gauge | Currently idle connections | 0 with high acquire wait = need more connections |
| `chgo_pool_constructing_resources` | Gauge | Connections being established | Sustained high = ClickHouse slow to accept |
| `chgo_pool_total_resources` | Gauge | Total pool connections | Should stay within min/max bounds |
| `chgo_pool_max_resources` | Gauge | Configured `maxConns` | Reference value |
| `chgo_pool_acquire_duration_seconds` | Gauge | Cumulative acquire time | Rising steeply = pool contention |
| `chgo_pool_empty_acquire_wait_time_seconds` | Gauge | Wait time when pool was empty | Rising = pool undersized |
| `chgo_pool_acquire_total` | Counter | Successful pool acquires | |
| `chgo_pool_empty_acquire_total` | Counter | Acquires that waited for a connection | High ratio to `acquire_total` = pool too small |
| `chgo_pool_canceled_acquire_total` | Counter | Acquire attempts that timed out | Any increase = severe pool exhaustion |

## Common Scenarios

### ClickHouse Down or Slow

**Symptoms:**
- `buffer_usage` climbing toward `bufferSize` for affected tables
- `write_errors_total` increasing
- `rows_written_total` flat
- `write_duration_seconds` p99 increasing (if ClickHouse is slow but reachable)
- Kafka consumer lag growing

**What happens internally:**
1. Table writer flush fails with a transient error (connection refused, timeout, etc.)
2. The table writer enters `flushBlocked` state -- it stops draining its buffer channel and preserves the failed batch for retry
3. Buffer channel fills up, blocking `Write()` calls
4. Benthos output blocks, causing Kafka fetch to stall
5. On next `flushInterval` tick, the table writer retries the pending batch (with exponential backoff via `doWithRetry`: default 3 retries, 100ms base delay, 2s max delay)
6. If retry succeeds, `flushBlocked` clears and normal processing resumes

**Recovery:**
- Fix ClickHouse (restart, add capacity, resolve disk issues)
- Consumoor auto-recovers once ClickHouse is available -- no restart needed
- Monitor `buffer_usage` dropping and `rows_written_total` resuming
- Expect a burst of writes as buffered data flushes

### Kafka Rebalance Storm

**Symptoms:**
- Kafka consumer lag spikes
- Possible duplicate rows in ClickHouse (at-least-once delivery replays uncommitted offsets)
- Log messages about partition reassignment from Benthos

**Causes:**
- Frequent pod restarts
- Network instability between consumoor and Kafka brokers
- `sessionTimeoutMs` too low for the workload (heartbeats missed)
- Slow ClickHouse causing processing delays that exceed session timeout

**Mitigation:**
- Increase `sessionTimeoutMs` (default 30000ms; try 60000ms for heavy workloads)
- Ensure `heartbeatIntervalMs` < `sessionTimeoutMs / 3` (default 3000ms is fine for 30s session)
- Use stable, rolling deployments instead of all-at-once restarts
- If ClickHouse slowness is the root cause, fix ClickHouse first

### OOM / Memory Pressure

**Symptoms:**
- `buffer_usage` high across multiple tables
- Pod OOMKilled restarts
- Memory usage on the node spiking

**Causes:**
- ClickHouse backpressure fills all table buffers simultaneously. Total potential memory = `bufferSize * number_of_active_tables * avg_event_size`
- Very large `bufferSize` values combined with many tables
- Large batch sizes holding many events in memory during flush

**Mitigation:**
- Reduce `bufferSize` in defaults (e.g. from 200000 to 50000)
- Increase ClickHouse write capacity (more replicas, faster disks)
- Tune `batchSize` down to flush more frequently with less memory per flush
- Increase pod memory limits if the workload genuinely requires it
- Monitor `buffer_usage` across all tables to estimate actual memory use

### Flatten Livelock

**Symptoms:**
- One table's `rows_written_total` stops increasing
- `buffer_usage` for that table grows (or stays at max if already full)
- `write_errors_total` for that table climbing
- `flatten_errors_total` increasing for the affected event/table
- Other tables continue writing normally

**Cause:**
- `FlattenTo()` fails for every event in the batch (e.g. schema change, nil field, missing column)
- The error is classified as permanent (`inputPrepError`), so the batch is dropped
- But new events keep arriving with the same problem, so every batch fails

**Resolution:**
1. Check `flatten_errors_total` labels to identify the affected event name and table
2. Fix the underlying issue: apply missing migrations, update the flattener code, or fix the upstream data
3. If the data is genuinely invalid and should be skipped, consider deploying a code fix for the flattener
4. Restart consumoor after the fix is deployed

### Consumer Lag Growing

**Symptoms:**
- Kafka consumer lag increasing (monitor via Kafka tooling / Burrow / etc.)
- `messages_consumed_total` rate lower than production rate
- Everything else looks healthy (no errors, ClickHouse is fine)

**Causes:**
- ClickHouse write throughput is the bottleneck (check `write_duration_seconds`)
- Batch sizes too small causing excessive round-trips (check `batch_size` histogram)
- Single instance hitting connection pool limits (check `chgo_pool_empty_acquire_total`)
- High-cardinality topics spreading events across many tables, each with its own flush cycle

**Tuning:**
- Increase `batchSize` to write more rows per round-trip (e.g. 200000 -> 500000)
- Increase `maxConns` to allow more concurrent ClickHouse writes
- Increase ClickHouse resources (CPU, memory, disk I/O)
- Scale horizontally: run multiple consumoor instances with different topic patterns or consumer groups
- Switch from `deliveryMode: message` to `deliveryMode: batch` if not already (batch mode is significantly faster)

### DLQ Failures

**Symptoms:**
- `dlq_errors_total` rising
- `messages_rejected_total` rising (messages are being rejected but cannot be persisted to DLQ)
- Benthos may retry the entire batch if the DLQ write failure propagates

**Impact:**
- Rejected messages that fail DLQ writes cause a batch error, which Benthos retries
- This can create a retry loop: message is permanently invalid, gets rejected, DLQ write fails, Benthos retries the batch, same message gets rejected again

**Resolution:**
1. Fix DLQ Kafka connectivity (check `rejectedTopic` Kafka broker reachability)
2. If DLQ is not critical, remove `rejectedTopic` from config to use the no-op reject sink (rejected messages are counted but not persisted)
3. Monitor `messages_rejected_total` to understand the volume of rejected messages

### Missing Tables

**Symptoms:**
- `write_errors_total` rising for a specific table
- Logs show ClickHouse error codes: `ErrUnknownTable` or `ErrUnknownDatabase`
- The error is classified as permanent, so batches are dropped (not retried)
- `messages_rejected_total` with reason `write_permanent` increasing

**Cause:**
- ClickHouse migration not applied for a new table
- Wrong `tableSuffix` configuration (e.g. writing to `table_local` but only `table` exists)
- Wrong database in DSN

**Resolution:**
1. Apply the missing ClickHouse migrations
2. Verify DSN points to the correct database
3. Verify `tableSuffix` matches your ClickHouse table naming convention
4. Restart consumoor (table writers are created lazily, so a restart is not strictly required -- new writes will succeed after the table exists)

## Startup Failures

Common causes of startup failure:

| Cause | Symptom | Resolution |
|-------|---------|------------|
| **Invalid config** | Process exits immediately with validation error | Check logs for `invalid config:` prefix; fix YAML |
| **Kafka unreachable** | Benthos stream fails to build/run | Verify broker addresses; Kafka must be reachable at startup (no retry) |
| **ClickHouse unreachable** | `opening ch-go connection pool` error after 3 retries | Verify DSN; ClickHouse must be reachable (retries 3x with backoff, then exits) |
| **Unknown disabled events** | `unknown disabledEvents:` error | Check event names in `disabledEvents` match proto enum names exactly |
| **DSN parse error** | `parsing clickhouse DSN` error | Verify DSN format: `clickhouse://host:port/database` |
| **SASL password file missing** | `reading sasl password file` error | Verify path exists and is readable |

## Graceful Restart Procedure

1. **Send SIGTERM** (or SIGINT) to the consumoor process
2. **Benthos drains in-flight batches.** The `shutdown_timeout` is 30 seconds. During this window:
   - Benthos stops fetching new messages from Kafka
   - In-flight batches complete processing
   - Kafka offsets for completed batches are committed
3. **Writer flushes remaining buffers.** Each table writer drains its buffer channel and performs a final flush to ClickHouse
4. **Connection pool closes.** The ch-go pool is closed after all writers finish
5. **Process exits**

**Monitor for:**
- Clean exit (exit code 0)
- Log line: `Consumoor stopped`
- If the process does not exit within ~35 seconds, it may be stuck in a flush -- check for ClickHouse connectivity issues

**Data loss considerations:**
- Messages consumed but not yet offset-committed will be replayed on restart (at-least-once)
- If the final flush to ClickHouse fails (e.g. ClickHouse is down during shutdown), those buffered rows are lost
- To minimize loss: ensure ClickHouse is healthy before initiating a restart

## Scaling Guidance

### Horizontal Scaling

- Run multiple consumoor instances with the **same consumer group** to partition Kafka topics across instances. Kafka rebalances partitions automatically.
- Run multiple instances with **different consumer groups** if you need independent processing of the same topics (e.g. writing to different ClickHouse clusters).
- Use **different topic patterns** (`kafka.topics` regex) to split event types across instances.

### Vertical Scaling

- Increase `maxConns` to allow more concurrent ClickHouse writes (default: 8)
- Increase `batchSize` to write more rows per INSERT (default: 200000)
- Increase `bufferSize` to absorb more backpressure spikes (default: 200000), at the cost of higher memory usage
- Increase pod CPU/memory to handle more concurrent table writers

## Configuration Tuning

### Key Knobs

| Parameter | Path | Default | When to adjust |
|-----------|------|---------|----------------|
| `batchSize` | `clickhouse.defaults.batchSize` | 200000 | Increase for higher throughput (more rows per INSERT); decrease to reduce memory per flush |
| `flushInterval` | `clickhouse.defaults.flushInterval` | 1s | Decrease for lower latency; increase if batches are too small |
| `bufferSize` | `clickhouse.defaults.bufferSize` | 200000 | Increase to absorb ClickHouse hiccups without backpressure; decrease to limit memory |
| `commitInterval` | `kafka.commitInterval` | 5s | Decrease to reduce duplicate replay window on crash; increase to reduce Kafka commit overhead |
| `maxConns` | `clickhouse.chgo.maxConns` | 8 | Increase when `chgo_pool_empty_acquire_total` is high |
| `queryTimeout` | `clickhouse.chgo.queryTimeout` | 30s | Increase if large batches legitimately take longer to insert |
| `maxRetries` | `clickhouse.chgo.maxRetries` | 3 | Increase if transient ClickHouse errors are frequent but recover quickly |
| `deliveryMode` | `kafka.deliveryMode` | batch | Use `message` for safer per-message delivery; use `batch` for higher throughput |
| `sessionTimeoutMs` | `kafka.sessionTimeoutMs` | 30000 | Increase if rebalances are frequent due to slow processing |
| `tableSuffix` | `clickhouse.tableSuffix` | (empty) | Set to `_local` to bypass Distributed tables in clustered setups |

### Per-Table Overrides

High-volume tables (e.g. attestations, committees) may need larger batch and buffer sizes:

```yaml
clickhouse:
  defaults:
    batchSize: 200000
    bufferSize: 200000
    flushInterval: 1s
  tables:
    beacon_api_eth_v1_events_attestation:
      batchSize: 1000000
      bufferSize: 1000000
      flushInterval: 5s
    beacon_api_eth_v1_beacon_committee:
      batchSize: 1000000
      bufferSize: 1000000
```

### Canonical Tables

Tables with a `canonical_` prefix automatically get `insert_quorum: auto` applied to their INSERT statements (majority quorum). Override this per-table if needed:

```yaml
clickhouse:
  tables:
    canonical_beacon_block:
      insertSettings:
        insert_quorum: 3
        insert_quorum_timeout: 60000
```
