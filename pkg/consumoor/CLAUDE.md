# Consumoor

Kafka-to-ClickHouse consumer that replaces Vector's VRL transforms with typed Go flatteners.

## Architecture

Consumoor consumes DecoratedEvent protobufs from Kafka topics and writes flat rows to ClickHouse tables. The pipeline is:

```
Kafka → KafkaConsumer (decode) → Router (event name → flatteners) → ClickHouseWriter (batched inserts)
```

### Key Components

- **KafkaConsumer** (`kafka.go`): Sarama consumer group that decodes JSON or protobuf messages into `*xatu.DecoratedEvent`.
- **Router** (`router.go`): Maps `Event.Name` to registered `Flattener` implementations. Handles conditional routing (same event → different tables based on additional_data) and fan-out (one event → multiple tables).
- **ClickHouseWriter** (`clickhouse.go`): Per-table batched inserts with configurable batch sizes, byte limits, and flush intervals.
- **Flattener interface** (`flattener/flattener.go`): Each implementation handles one or more event names and produces flat `map[string]any` rows for one ClickHouse table.
- **CommonMetadata** (`metadata/metadata.go`): Shared metadata extraction from DecoratedEvent proto fields. Replaces the 200-line VRL `xatu_server_events_meta` transform.

### Adding a New Event

1. Implement the `flattener.Flattener` interface (EventNames, TableName, Flatten, ShouldProcess)
2. Register it in `buildFlatteners()` in `consumoor.go`
3. Write a ClickHouse migration for the target table
4. Add unit tests

### Configuration

All operational parameters (batch sizes, flush intervals, Kafka tuning) are YAML-configurable. See `example_consumoor.yaml`. The typed proto→ClickHouse mapping (flatteners) is in code and compile-time checked.
