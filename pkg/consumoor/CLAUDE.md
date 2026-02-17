# Consumoor

Kafka-to-ClickHouse consumer that replaces Vector's VRL transforms with typed Go flatteners.

## Architecture

Consumoor consumes DecoratedEvent protobufs from Kafka topics and writes flat rows to ClickHouse tables. The pipeline is:

```
Kafka → KafkaConsumer (decode) → Router (event name → routes) → ClickHouseWriter (batched inserts)
```

### Key Components

- **KafkaConsumer** (`kafka.go`): Sarama consumer group that decodes JSON or protobuf messages into `*xatu.DecoratedEvent`.
- **Router** (`router.go`): Maps `Event.Name` to registered `Route` implementations. Handles conditional routing (same event → different tables based on additional_data) and fan-out (one event → multiple tables).
- **ChGoWriter** (`clickhouse_chgo.go`): Per-table batched inserts with configurable batch sizes, byte limits, flush intervals, retries, and pool settings.
- **Route interface** (`flattener/flattener.go`): Each implementation handles one or more event names and produces flat `map[string]any` rows for one ClickHouse table.
- **CommonMetadata** (`metadata/metadata.go`): Shared metadata extraction from DecoratedEvent proto fields. Replaces the 200-line VRL `xatu_server_events_meta` transform.

### Adding a New Event

1. Add or update a table definition in `flattener/routes_*.go` using `GenericTable(...)` or `Table(...)` + `CustomSource(...)`
2. For custom behavior, implement `flattener.Route` (or use `WithPredicate` / `WithMutator` on `GenericTable`)
3. Write a ClickHouse migration for the target table
4. Add or update unit tests in `flattener/registry_test.go`

### Configuration

All operational parameters (batch sizes, flush intervals, Kafka tuning) are YAML-configurable. See `example_consumoor.yaml`. The typed proto→ClickHouse mapping (routes) is in code and compile-time checked.
