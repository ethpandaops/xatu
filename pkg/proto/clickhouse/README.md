# ClickHouse read types

Generated, typed read access to the xatu ClickHouse tables. Do not edit by
hand — regenerate with `make proto-clickhouse`, which introspects a local
ClickHouse built from `deploy/migrations/clickhouse` via
[clickhouse-proto-gen](https://github.com/ethpandaops/clickhouse-proto-gen).

Each table gets:

- `<table>.proto` / `<table>.pb.go` — the row message, typed filter requests
  and a List/Get service definition
- `<table>.go` — `BuildList<Table>Query` / `BuildGet<Table>Query` SQL builders
  returning a parameterized `SQLQuery`
- `<table>.row.go` — `<Table>Row`, a ClickHouse-scannable struct matching the
  SELECT expressions of the built queries, with a `ToProto()` converter

Consumers import this package, build queries with typed filters, run them with
clickhouse-go and scan into the generated rows:

```go
req := &clickhouse.ListBeaconApiEthV1EventsBlockRequest{
    MetaNetworkName:   &clickhouse.StringFilter{Filter: &clickhouse.StringFilter_Eq{Eq: "mainnet"}},
    SlotStartDateTime: &clickhouse.UInt32Filter{Filter: &clickhouse.UInt32Filter_Eq{Eq: slotStart}},
    Slot:              &clickhouse.UInt32Filter{Filter: &clickhouse.UInt32Filter_Eq{Eq: slot}},
}

q, _ := clickhouse.BuildListBeaconApiEthV1EventsBlockQuery(req)
rows, _ := conn.Query(ctx, q.Query, q.Args...)

for rows.Next() {
    var row clickhouse.BeaconApiEthV1EventsBlockRow
    _ = rows.ScanStruct(&row)
}
```

Schema changes regenerate this package; downstream consumers pick them up by
bumping their xatu dependency and recompiling.
