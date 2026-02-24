# Consumoor Delivery Semantics

## Guarantee

Consumoor provides **at-least-once** delivery from Kafka to ClickHouse. Every
message that is successfully consumed will be written to ClickHouse at least
once, but duplicate rows are possible under failure conditions.

## Offset Commit Timeline

```
Kafka ──► Benthos consumer ──► WriteBatch() ──► ClickHouse INSERT
                                    │
                                    ▼
                           return nil (ack)
                                    │
                                    ▼
                          offset commit-eligible
                                    │
                          ┌─────────┴──────────┐
                          │  commit_period (5s) │
                          └─────────┬──────────┘
                                    ▼
                           Kafka offset committed
```

The critical gap is between a successful ClickHouse INSERT (when `WriteBatch`
returns nil) and the next periodic Kafka offset commit. Any failure in that
window causes the consumer to replay messages that were already written.

## When Duplicates Occur

1. **Process crash** -- The process dies after writing rows to ClickHouse but
   before the next `commit_period` tick commits offsets. On restart, the
   consumer replays up to `commit_period` worth of messages.

2. **Kafka consumer group rebalance** -- A rebalance revokes partition
   ownership. If the new owner starts consuming before offsets are committed by
   the previous owner, messages are processed twice.

3. **Pod restart / rolling deploy** -- Same mechanism as a crash: the shutdown
   drain writes remaining buffered rows to ClickHouse, but if offset commits
   do not complete before the pod terminates, the replacement pod replays them.

4. **Organic flush vs. offset commit race** -- Each table writer flushes on its
   own timer (batch size or flush interval). These flushes are independent of
   Kafka offset commits, so rows can land in ClickHouse before their offsets
   are committed.

The duplicate window is bounded by `commitInterval` (default 5 s).

## Why Exactly-Once Is Not Achievable

Kafka offset commits and ClickHouse INSERTs are separate, non-transactional
operations. There is no two-phase commit or transactional outbox linking them.
Benthos commits offsets periodically (not per-INSERT), making it impossible to
guarantee that every successful write has a corresponding committed offset at
all times.

## Mitigation Strategies

- **ClickHouse block-level deduplication** -- `ReplicatedMergeTree` deduplicates
  identical INSERT blocks within `replicated_deduplication_window` (default 100
  blocks). This helps when a replay produces the exact same block, but does not
  cover all scenarios (e.g. different batch boundaries on replay).

- **ReplacingMergeTree** -- Use `ReplacingMergeTree` with a version column to
  collapse duplicates at merge time. Queries should use `FINAL` or apply
  deduplication in the application layer.

- **Idempotent queries** -- Design downstream queries to tolerate duplicates,
  for example by using `DISTINCT`, `argMax`, or `GROUP BY` on a unique key.

- **Shorter commit interval** -- Reducing `commitInterval` shrinks the duplicate
  window at the cost of more frequent offset commits to Kafka.
