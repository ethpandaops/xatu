# Consumoor

Kafka-to-ClickHouse consumer that replaces Vector's VRL transforms with typed Go flatteners.

## Architecture

Consumoor consumes DecoratedEvent protobufs from Kafka topics and writes flat rows to ClickHouse tables. The pipeline is:

```
Kafka (Benthos kafka_franz) → xatu_clickhouse output (decode + route + classify) → ChGoWriter (batched inserts)
```

### Key Components

- **Source** (`source/benthos.go`): Kafka ingestion via `kafka_franz` with per-message or batch delivery boundaries and optional rejected-message DLQ.
- **ClickHouse Transform Engine** (`sinks/clickhouse/transform/engine.go`): Maps `Event.Name` to registered `Route` implementations. Handles conditional routing (same event → different tables based on additional_data) and fan-out (one event → multiple tables).
- **ClickHouse Sink** (`sinks/clickhouse/writer.go`): Per-table batched inserts with configurable batch sizes, byte limits, flush intervals, retries, and pool settings.
- **Telemetry** (`telemetry/metrics.go`): Shared Prometheus metric registry used across source/router/sink packages.
- **Route interface** (`sinks/clickhouse/transform/flattener/flattener.go`): Each implementation handles one or more event names and produces flat `map[string]any` rows for one ClickHouse table.
- **CommonMetadata** (`sinks/clickhouse/transform/metadata/metadata.go`): Shared metadata extraction from DecoratedEvent proto fields. Replaces the 200-line VRL `xatu_server_events_meta` transform.

### Adding a New Event

1. Generate a typed row struct using `chgo-rowgen` (see `tables/<domain>/cmd/`) — produces a `.gen.go` file with `ToMap()`, `SetMetadata()`, `SetAnyColumn()`, `GetColumn()`, etc.
2. Add a hand-written `$table.go` in `sinks/clickhouse/transform/flattener/tables/<domain>/` with:
   - `flattenXxx()` function returning `[]map[string]any`
   - `newXxxRow()` constructor calling `setRuntime()`, `SetMetadata()`, `setPayload()`, and optionally `setClientAdditionalData()` / `setServerAdditionalData()`
   - `init()` registering via `flattener.MustRegister(flattener.NewStaticRoute(tableName, eventNames, flattenFn))`
3. For conditional routing, use `flattener.WithStaticRoutePredicate(...)` option
4. Write a ClickHouse migration for the target table
5. Add or update unit tests in `sinks/clickhouse/transform/flattener/routes_test.go`

### Configuration

All operational parameters (batch sizes, flush intervals, Kafka tuning) are YAML-configurable. See `example_consumoor.yaml`. The typed proto→ClickHouse mapping (routes) is in code and compile-time checked.
