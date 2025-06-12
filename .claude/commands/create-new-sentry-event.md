# Create New Sentry Event Command

This command provides a step-by-step template for adding a new event type to the xatu sentry pipeline with complete end-to-end data flow.

## Usage
```bash
# Replace {EVENT_NAME} with your actual event name in snake_case
# Replace {EVENT_TYPE} with the event type (e.g., "attestation", "block", "sidecar")
# Replace {API_VERSION} with the API version (e.g., "v1", "v2")
```

## Prerequisites
- Understand the beacon API specification for your event
- Identify the event trigger condition (when it should fire)
- Know the data structure fields required

## Step-by-Step Implementation

### 1. Define Protobuf Schema
**File**: `pkg/proto/eth/{API_VERSION}/events.proto`

Add your event message:
```protobuf
message Event{EventName} {
  google.protobuf.UInt64Value slot = 1 [ json_name = "slot" ];
  // Add other fields based on beacon API spec
  string field_name = 2 [ json_name = "field_name" ];
  repeated string array_field = 3 [ json_name = "array_field" ];
}
```

### 2. Update Event Enum
**File**: `pkg/proto/xatu/event_ingester.proto`

Add to Event enum (use next available number):
```protobuf
BEACON_API_ETH_{API_VERSION}_EVENTS_{EVENT_NAME} = {NEXT_NUMBER};
```

### 3. Add DecoratedEvent Field
**File**: `pkg/proto/xatu/event_ingester.proto`

Add to DecoratedEvent oneof:
```protobuf
xatu.eth.{api_version}.Event{EventName} eth_{api_version}_events_{event_name} = {NEXT_FIELD_NUMBER}
    [ json_name = "BEACON_API_ETH_{API_VERSION}_EVENTS_{EVENT_NAME}" ];
```

### 4. Add Additional Metadata
**File**: `pkg/proto/xatu/event_ingester.proto`

Add additional data message and ClientMeta field:
```protobuf
message AdditionalEth{ApiVersion}Events{EventName}Data {
  EpochV2 epoch = 1;
  SlotV2 slot = 2;
  PropagationV2 propagation = 3;
  // Add event-specific metadata
}

// In ClientMeta oneof additional_data:
AdditionalEth{ApiVersion}Events{EventName}Data eth_{api_version}_events_{event_name} = {NEXT_NUMBER}
    [ json_name = "BEACON_API_ETH_{API_VERSION}_EVENTS_{EVENT_NAME}" ];
```

### 5. Generate Protobuf Code
```bash
buf generate
```

### 6. Create Sentry Event Handler
**File**: `pkg/sentry/event/beacon/eth/{api_version}/events_{event_name}.go`

```go
package event

import (
    "context"
    "fmt"
    "time"
    
    eth2v1 "github.com/attestantio/go-eth2-client/api/v1"
    xatuethv1 "github.com/ethpandaops/xatu/pkg/proto/eth/v1"
    "github.com/ethpandaops/xatu/pkg/proto/xatu"
    "github.com/ethpandaops/xatu/pkg/sentry/ethereum"
    "github.com/google/uuid"
    ttlcache "github.com/jellydator/ttlcache/v3"
    hashstructure "github.com/mitchellh/hashstructure/v2"
    "github.com/sirupsen/logrus"
    "google.golang.org/protobuf/types/known/timestamppb"
    "google.golang.org/protobuf/types/known/wrapperspb"
)

type Events{EventName} struct {
    log            logrus.FieldLogger
    now            time.Time
    event          *eth2v1.{EventName}Event
    beacon         *ethereum.BeaconNode
    duplicateCache *ttlcache.Cache[string, time.Time]
    clientMeta     *xatu.ClientMeta
    id             uuid.UUID
}

func NewEvents{EventName}(log logrus.FieldLogger, event *eth2v1.{EventName}Event, now time.Time, beacon *ethereum.BeaconNode, duplicateCache *ttlcache.Cache[string, time.Time], clientMeta *xatu.ClientMeta) *Events{EventName} {
    return &Events{EventName}{
        log:            log.WithField("event", "BEACON_API_ETH_{API_VERSION}_EVENTS_{EVENT_NAME}"),
        now:            now,
        event:          event,
        beacon:         beacon,
        duplicateCache: duplicateCache,
        clientMeta:     clientMeta,
        id:             uuid.New(),
    }
}

func (e *Events{EventName}) Decorate(ctx context.Context) (*xatu.DecoratedEvent, error) {
    decoratedEvent := &xatu.DecoratedEvent{
        Event: &xatu.Event{
            Name:     xatu.Event_BEACON_API_ETH_{API_VERSION}_EVENTS_{EVENT_NAME},
            DateTime: timestamppb.New(e.now),
            Id:       e.id.String(),
        },
        Meta: &xatu.Meta{
            Client: e.clientMeta,
        },
        Data: &xatu.DecoratedEvent_Eth{ApiVersion}Events{EventName}{
            Eth{ApiVersion}Events{EventName}: &xatuethv1.Event{EventName}{
                Slot: &wrapperspb.UInt64Value{Value: uint64(e.event.Slot)},
                // Map other fields from e.event to protobuf message
            },
        },
    }

    additionalData, err := e.getAdditionalData(ctx)
    if err != nil {
        e.log.WithError(err).Error("Failed to get extra {event_name} data")
    } else {
        decoratedEvent.Meta.Client.AdditionalData = &xatu.ClientMeta_Eth{ApiVersion}Events{EventName}{
            Eth{ApiVersion}Events{EventName}: additionalData,
        }
    }

    return decoratedEvent, nil
}

func (e *Events{EventName}) ShouldIgnore(ctx context.Context) (bool, error) {
    if err := e.beacon.Synced(ctx); err != nil {
        return true, err
    }

    hash, err := hashstructure.Hash(e.event, hashstructure.FormatV2, nil)
    if err != nil {
        return true, err
    }

    item, retrieved := e.duplicateCache.GetOrSet(fmt.Sprint(hash), e.now, ttlcache.WithTTL[string, time.Time](ttlcache.DefaultTTL))
    if retrieved {
        e.log.WithFields(logrus.Fields{
            "hash":                  hash,
            "time_since_first_item": time.Since(item.Value()),
            "slot":                  e.event.Slot,
        }).Debug("Duplicate {event_name} event received")

        return true, nil
    }

    return false, nil
}

func (e *Events{EventName}) getAdditionalData(_ context.Context) (*xatu.ClientMeta_AdditionalEth{ApiVersion}Events{EventName}Data, error) {
    extra := &xatu.ClientMeta_AdditionalEth{ApiVersion}Events{EventName}Data{}

    slot := e.beacon.Metadata().Wallclock().Slots().FromNumber(uint64(e.event.Slot))
    epoch := e.beacon.Metadata().Wallclock().Epochs().FromSlot(uint64(e.event.Slot))

    extra.Slot = &xatu.SlotV2{
        Number:        &wrapperspb.UInt64Value{Value: slot.Number()},
        StartDateTime: timestamppb.New(slot.TimeWindow().Start()),
    }

    extra.Epoch = &xatu.EpochV2{
        Number:        &wrapperspb.UInt64Value{Value: epoch.Number()},
        StartDateTime: timestamppb.New(epoch.TimeWindow().Start()),
    }

    extra.Propagation = &xatu.PropagationV2{
        SlotStartDiff: &wrapperspb.UInt64Value{
            Value: uint64(e.now.Sub(slot.TimeWindow().Start()).Milliseconds()),
        },
    }

    return extra, nil
}
```

### 7. Add Cache to Duplicate Detection
**File**: `pkg/sentry/cache/duplicate.go`

Add cache field:
```go
BeaconEth{ApiVersion}Events{EventName} *ttlcache.Cache[string, time.Time]
```

Initialize in NewDuplicateCache():
```go
BeaconEth{ApiVersion}Events{EventName}: ttlcache.New(
    ttlcache.WithTTL[string, time.Time](defaultTTL),
),
```

Start cache in Start():
```go
go d.BeaconEth{ApiVersion}Events{EventName}.Start()
```

### 8. Register Event Handler in Sentry
**File**: `pkg/sentry/sentry.go`

Add event subscription:
```go
s.beacon.Node().On{EventName}(ctx, func(ctx context.Context, {eventName} *eth2v1.{EventName}Event) error {
    now := time.Now().Add(s.clockDrift)

    meta, err := s.createNewClientMeta(ctx)
    if err != nil {
        return err
    }

    event := v1.NewEvents{EventName}(s.log, {eventName}, now, s.beacon, s.duplicateCache.BeaconEth{ApiVersion}Events{EventName}, meta)

    ignore, err := event.ShouldIgnore(ctx)
    if err != nil {
        return err
    }

    if ignore {
        return nil
    }

    decoratedEvent, err := event.Decorate(ctx)
    if err != nil {
        return err
    }

    return s.handleNewDecoratedEvent(ctx, decoratedEvent)
})
```

### 9. Create Server Event Handler
**File**: `pkg/server/service/event-ingester/event/beacon/eth/{api_version}/events_{event_name}.go`

```go
package v1

import (
    "context"
    "errors"

    "github.com/ethpandaops/xatu/pkg/proto/xatu"
    "github.com/sirupsen/logrus"
)

const (
    Events{EventName}Type = "BEACON_API_ETH_{API_VERSION}_EVENTS_{EVENT_NAME}"
)

type Events{EventName} struct {
    log   logrus.FieldLogger
    event *xatu.DecoratedEvent
}

func NewEvents{EventName}(log logrus.FieldLogger, event *xatu.DecoratedEvent) *Events{EventName} {
    return &Events{EventName}{
        log:   log.WithField("event", Events{EventName}Type),
        event: event,
    }
}

func (e *Events{EventName}) Type() string {
    return Events{EventName}Type
}

func (e *Events{EventName}) Validate(ctx context.Context) error {
    _, ok := e.event.GetData().(*xatu.DecoratedEvent_Eth{ApiVersion}Events{EventName})
    if !ok {
        return errors.New("failed to cast event data")
    }

    return nil
}

func (e *Events{EventName}) Filter(ctx context.Context) bool {
    return false
}

func (e *Events{EventName}) AppendServerMeta(ctx context.Context, meta *xatu.ServerMeta) *xatu.ServerMeta {
    return meta
}
```

### 10. Create ClickHouse Migration
**File**: `deploy/migrations/clickhouse/{NEXT_NUMBER}_{event_name}.up.sql`

```sql
CREATE TABLE beacon_api_eth_{api_version}_events_{event_name}_local on cluster '{cluster}' (
    event_date_time DateTime64(3) Codec(DoubleDelta, ZSTD(1)),
    slot UInt32 Codec(DoubleDelta, ZSTD(1)),
    slot_start_date_time DateTime Codec(DoubleDelta, ZSTD(1)),
    propagation_slot_start_diff UInt32 Codec(ZSTD(1)),
    epoch UInt32 Codec(DoubleDelta, ZSTD(1)),
    epoch_start_date_time DateTime Codec(DoubleDelta, ZSTD(1)),
    
    -- Event-specific columns
    field_name String Codec(ZSTD(1)),
    array_field Array(String) Codec(ZSTD(1)),
    
    -- Standard metadata columns
    meta_client_name LowCardinality(String),
    meta_client_id String CODEC(ZSTD(1)),
    meta_client_version LowCardinality(String),
    meta_client_implementation LowCardinality(String),
    meta_client_os LowCardinality(String),
    meta_client_ip Nullable(IPv6) CODEC(ZSTD(1)),
    meta_client_geo_city LowCardinality(String) CODEC(ZSTD(1)),
    meta_client_geo_country LowCardinality(String) CODEC(ZSTD(1)),
    meta_client_geo_country_code LowCardinality(String) CODEC(ZSTD(1)),
    meta_client_geo_continent_code LowCardinality(String) CODEC(ZSTD(1)),
    meta_client_geo_longitude Nullable(Float64) CODEC(ZSTD(1)),
    meta_client_geo_latitude Nullable(Float64) CODEC(ZSTD(1)),
    meta_client_geo_autonomous_system_number Nullable(UInt32) CODEC(ZSTD(1)),
    meta_client_geo_autonomous_system_organization Nullable(String) CODEC(ZSTD(1)),
    meta_network_id Int32 CODEC(DoubleDelta, ZSTD(1)),
    meta_network_name LowCardinality(String),
    meta_consensus_version LowCardinality(String),
    meta_consensus_version_major LowCardinality(String),
    meta_consensus_version_minor LowCardinality(String),
    meta_consensus_version_patch LowCardinality(String),
    meta_consensus_implementation LowCardinality(String),
    meta_labels Map(String, String) CODEC(ZSTD(1))
) Engine = ReplicatedMergeTree('/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}', '{replica}')
PARTITION BY toStartOfMonth(slot_start_date_time)
ORDER BY (slot_start_date_time, meta_network_name, meta_client_name);

ALTER TABLE default.beacon_api_eth_{api_version}_events_{event_name}_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains beacon API eventstream "{event_name}" data from each sentry client attached to a beacon node.',
COMMENT COLUMN event_date_time 'When the sentry received the event from a beacon node',
COMMENT COLUMN slot 'Slot number in the beacon API event stream payload',
COMMENT COLUMN slot_start_date_time 'The wall clock time when the slot started',
COMMENT COLUMN propagation_slot_start_diff 'The difference between the event_date_time and the slot_start_date_time',
COMMENT COLUMN epoch 'The epoch number in the beacon API event stream payload',
COMMENT COLUMN epoch_start_date_time 'The wall clock time when the epoch started';

CREATE TABLE beacon_api_eth_{api_version}_events_{event_name} on cluster '{cluster}' AS beacon_api_eth_{api_version}_events_{event_name}_local
ENGINE = Distributed('{cluster}', default, beacon_api_eth_{api_version}_events_{event_name}_local, rand());
```

**File**: `deploy/migrations/clickhouse/{NEXT_NUMBER}_{event_name}.down.sql`

```sql
DROP TABLE IF EXISTS beacon_api_eth_{api_version}_events_{event_name} on cluster '{cluster}';
DROP TABLE IF EXISTS beacon_api_eth_{api_version}_events_{event_name}_local on cluster '{cluster}';
```

### 11. Update Vector Configuration
**File**: `deploy/local/docker-compose/vector-kafka-clickhouse.yaml`

Add Kafka source:
```yaml
sources:
  beacon_api_eth_{api_version}_events_{event_name}_kafka:
    type: kafka
    bootstrap_servers: "${KAFKA_BROKERS}"
    auto_offset_reset: earliest
    group_id: xatu-vector-kafka-clickhouse-beacon-api-eth-{api_version}-events-{event_name}
    key_field: "event.id"
    decoding:
      codec: json
    topics:
      - "beacon-api-eth-{api_version}-events-{event_name}"
```

Add to router inputs and route:
```yaml
transforms:
  xatu_server_events_router:
    inputs:
      - beacon_api_eth_{api_version}_events_{event_name}_kafka
    route:
      eth_{api_version}_events_{event_name}: .event.name == "BEACON_API_ETH_{API_VERSION}_EVENTS_{EVENT_NAME}"
```

Add data formatter:
```yaml
  beacon_api_eth_{api_version}_events_{event_name}_formatted:
    type: remap
    inputs:
      - xatu_server_events_router.eth_{api_version}_events_{event_name}
    source: |-
      event_date_time, err = parse_timestamp(.event.date_time, format: "%+");
      if err == null {
        .event_date_time = to_unix_timestamp(event_date_time, unit: "milliseconds")
      }
      .slot = .data.slot
      .field_name = .data.field_name
      .array_field = .data.array_field
      
      # Extract metadata
      .meta_client_name = .meta.client.name
      .meta_client_id = .meta.client.id
      # ... other metadata fields
      
      # Extract timing data
      .slot_start_date_time = to_unix_timestamp(parse_timestamp(.meta.client.additional_data.slot.start_date_time, format: "%+"))
      .propagation_slot_start_diff = .meta.client.additional_data.propagation.slot_start_diff
      .epoch = .meta.client.additional_data.epoch.number
      .epoch_start_date_time = to_unix_timestamp(parse_timestamp(.meta.client.additional_data.epoch.start_date_time, format: "%+"))
      
      del(.event)
      del(.meta)
      del(.data)
```

Add ClickHouse sink:
```yaml
sinks:
  beacon_api_eth_{api_version}_events_{event_name}_clickhouse:
    type: clickhouse
    inputs:
      - beacon_api_eth_{api_version}_events_{event_name}_formatted
    auth:
      strategy: basic
      user: "${CLICKHOUSE_USER}"
      password: "${CLICKHOUSE_PASSWORD}"
    database: default
    endpoint: "${CLICKHOUSE_ENDPOINT}"
    table: beacon_api_eth_{api_version}_events_{event_name}
    batch:
      max_bytes: 52428800
      max_events: 200000
      timeout_secs: 1
    buffer:
      max_events: 200000
    healthcheck:
      enabled: true
    skip_unknown_fields: false
```

## Testing Checklist

- [ ] Unit tests for event handler (Decorate, ShouldIgnore, getAdditionalData)
- [ ] Protobuf serialization/deserialization tests
- [ ] Sentry integration test (event subscription and processing)
- [ ] Server event validation and filtering tests
- [ ] ClickHouse migration up/down tests
- [ ] Vector configuration validation
- [ ] End-to-end pipeline test (sentry → kafka → vector → clickhouse)
- [ ] Performance test with high event volume

## Common Gotchas

1. **Field Numbers**: Always use the next available field number in protobuf
2. **Event Names**: Use consistent naming: BEACON_API_ETH_{VERSION}_EVENTS_{NAME}
3. **Cache Initialization**: Don't forget to add cache to duplicate.go and start it
4. **Vector Routing**: Event name in route must match protobuf enum exactly
5. **ClickHouse Types**: Use appropriate ClickHouse types for your data
6. **Migration Numbers**: Use the next sequential migration number
7. **Kafka Topics**: Follow naming convention: beacon-api-eth-{version}-events-{name}

## File Checklist

- [ ] `pkg/proto/eth/{api_version}/events.proto` - Event message
- [ ] `pkg/proto/xatu/event_ingester.proto` - Enum, oneof, additional data
- [ ] `pkg/sentry/event/beacon/eth/{api_version}/events_{event_name}.go` - Handler
- [ ] `pkg/sentry/cache/duplicate.go` - Cache field and initialization
- [ ] `pkg/sentry/sentry.go` - Event subscription
- [ ] `pkg/server/service/event-ingester/event/beacon/eth/{api_version}/events_{event_name}.go` - Server handler
- [ ] `deploy/migrations/clickhouse/{number}_{event_name}.up.sql` - Schema
- [ ] `deploy/migrations/clickhouse/{number}_{event_name}.down.sql` - Rollback
- [ ] `deploy/local/docker-compose/vector-kafka-clickhouse.yaml` - Pipeline config

## Final Steps

1. Run `buf generate` to generate protobuf code
2. Build and test locally
3. Run database migrations
4. Update Vector configuration
5. Deploy and monitor end-to-end data flow