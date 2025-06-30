# Data Column Sidecar Event Implementation Plan

## Overview
> This plan details the implementation of a new `data_column_sidecar` event in xatu sentry that captures data column sidecar events as defined in the ethereum/beacon-APIs PR #535. This event supports PeerDAS (EIP-7594) data availability sampling and will be triggered when the node receives a DataColumnSidecar that passes all gossip validations.

## Current State Assessment

- **Existing Implementation**: Xatu sentry currently implements blob sidecar events (`events_blob_sidecar.go`)
- **Pattern**: Event handling follows a consistent pattern: event creation, decoration, duplicate detection, and forwarding
- **Infrastructure**: All necessary infrastructure exists for new events including protobuf definitions, event handlers, and output sinks
- **Limitations**: No data column sidecar support currently exists
- **Dependencies**: Requires go-eth2-client library support for DataColumnSidecar events
- **Preserved Components**: Event handling pattern, duplicate cache system, and protobuf generation pipeline

## Goals

1. **Primary goal**: Implement data_column_sidecar event handling in xatu sentry following ethereum/beacon-APIs specification
2. **Event Structure**: Capture slot, index, block_root, and kzg_commitments from DataColumnSidecar events
3. **Integration**: Seamlessly integrate with existing event handling pipeline and output sinks
4. **Performance**: Maintain duplicate detection and caching consistent with other events
5. **Non-functional requirements**:
   - Follow existing code patterns and conventions
   - Maintain backward compatibility
   - Include comprehensive protobuf definitions
   - Add appropriate logging and error handling
   - Support all configured output formats (kafka, http, stdout, xatu)

## Design Approach

### Architecture Overview
The implementation follows xatu's established event handling architecture:
- Event reception through go-eth2-client library hooks
- Event decoration with metadata (timing, propagation, epoch/slot info)
- Duplicate detection using TTL cache
- Forwarding to configured output sinks via protobuf-defined messages

### Component Breakdown
1. **Protobuf Definitions**
   - Purpose: Define data structures for data column sidecar events
   - Responsibilities: Schema definition, serialization support, type safety
   - Interfaces: Integrates with xatu event ingester and client metadata

2. **Event Handler Implementation** 
   - Purpose: Process incoming data column sidecar events
   - Responsibilities: Event decoration, duplicate detection, metadata enrichment
   - Interfaces: Implements standard event interface, integrates with beacon client

3. **Sentry Integration**
   - Purpose: Wire data column sidecar event handler into main sentry loop
   - Responsibilities: Event subscription, handler registration, cache management
   - Interfaces: Connects beacon client events to processing pipeline

4. **Server Event Ingester**
   - Purpose: Handle incoming data column sidecar events on server side
   - Responsibilities: Event persistence, validation, routing
   - Interfaces: Implements gRPC service interfaces for event ingestion

## Implementation Approach

### 1. Protobuf Schema Definition

#### Specific Changes
- Add `EventDataColumnSidecar` message to `pkg/proto/eth/v1/events.proto`
- Add `BEACON_API_ETH_V1_EVENTS_DATA_COLUMN_SIDECAR` enum value to Event.Name
- Add oneof field to DecoratedEvent for data column sidecar events
- Add additional metadata message for enriched data
- Regenerate protobuf Go code using `buf generate`

#### Sample Implementation
```protobuf
// In pkg/proto/eth/v1/events.proto
message EventDataColumnSidecar {
  google.protobuf.UInt64Value slot = 1 [ json_name = "slot" ];
  google.protobuf.UInt64Value index = 2 [ json_name = "index" ];
  string block_root = 3 [ json_name = "block_root" ];
  repeated string kzg_commitments = 4 [ json_name = "kzg_commitments" ];
}

// In pkg/proto/xatu/event_ingester.proto - Event enum
BEACON_API_ETH_V1_EVENTS_DATA_COLUMN_SIDECAR = 58;

// In DecoratedEvent oneof
xatu.eth.v1.EventDataColumnSidecar eth_v1_events_data_column_sidecar = 60
    [ json_name = "BEACON_API_ETH_V1_EVENTS_DATA_COLUMN_SIDECAR" ];

// In ClientMeta additional data
AdditionalEthV1EventsDataColumnSidecarData eth_v1_events_data_column_sidecar = 61
    [ json_name = "BEACON_API_ETH_V1_EVENTS_DATA_COLUMN_SIDECAR" ];
```

### 2. Event Handler Implementation

#### Specific Changes
- Create `pkg/sentry/event/beacon/eth/v1/events_data_column_sidecar.go`
- Implement EventsDataColumnSidecar struct following existing patterns
- Add methods: NewEventsDataColumnSidecar, Decorate, ShouldIgnore, getAdditionalData
- Include timing, propagation, and epoch/slot metadata

#### Sample Implementation
```go
type EventsDataColumnSidecar struct {
    log            logrus.FieldLogger
    now            time.Time
    event          *eth2v1.DataColumnSidecarEvent
    beacon         *ethereum.BeaconNode  
    duplicateCache *ttlcache.Cache[string, time.Time]
    clientMeta     *xatu.ClientMeta
    id             uuid.UUID
}

func NewEventsDataColumnSidecar(log logrus.FieldLogger, event *eth2v1.DataColumnSidecarEvent, now time.Time, beacon *ethereum.BeaconNode, duplicateCache *ttlcache.Cache[string, time.Time], clientMeta *xatu.ClientMeta) *EventsDataColumnSidecar {
    return &EventsDataColumnSidecar{
        log:            log.WithField("event", "BEACON_API_ETH_V1_EVENTS_DATA_COLUMN_SIDECAR"),
        now:            now,
        event:          event,
        beacon:         beacon,
        duplicateCache: duplicateCache,
        clientMeta:     clientMeta,
        id:             uuid.New(),
    }
}
```

### 3. Sentry Integration

#### Specific Changes
- Add data column sidecar cache to `pkg/sentry/cache/duplicate.go`
- Register OnDataColumnSidecar handler in `pkg/sentry/sentry.go`
- Follow existing blob sidecar pattern for event subscription and processing

#### Sample Implementation
```go
// In cache/duplicate.go
BeaconEthV1EventsDataColumnSidecar *ttlcache.Cache[string, time.Time]

// In sentry.go
s.beacon.Node().OnDataColumnSidecar(ctx, func(ctx context.Context, dataColumnSidecar *eth2v1.DataColumnSidecarEvent) error {
    now := time.Now().Add(s.clockDrift)
    
    meta, err := s.createNewClientMeta(ctx)
    if err != nil {
        return err
    }
    
    event := v1.NewEventsDataColumnSidecar(s.log, dataColumnSidecar, now, s.beacon, s.duplicateCache.BeaconEthV1EventsDataColumnSidecar, meta)
    
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

### 4. Server Event Ingester

#### Specific Changes
- Add data column sidecar event handling to `pkg/server/service/event-ingester/event/beacon/eth/v1/events_data_column_sidecar.go`
- Implement server-side processing, validation, and persistence
- Follow existing server-side event patterns

#### Sample Implementation
```go
const (
    EventsDataColumnSidecarType = "BEACON_API_ETH_V1_EVENTS_DATA_COLUMN_SIDECAR"
)

type EventsDataColumnSidecar struct {
    log   logrus.FieldLogger
    event *xatu.DecoratedEvent
}

func NewEventsDataColumnSidecar(log logrus.FieldLogger, event *xatu.DecoratedEvent) *EventsDataColumnSidecar {
    return &EventsDataColumnSidecar{
        log:   log.WithField("event", EventsDataColumnSidecarType),
        event: event,
    }
}

func (d *EventsDataColumnSidecar) Validate(ctx context.Context) error {
    _, ok := d.event.GetData().(*xatu.DecoratedEvent_EthV1EventsDataColumnSidecar)
    if !ok {
        return errors.New("failed to cast event data")
    }
    return nil
}
```

### 5. ClickHouse Database Schema

#### Specific Changes
- Create new migration file: `058_data_column_sidecar.up.sql`
- Define table schema for storing data column sidecar events
- Include all standard metadata columns and data column specific fields

#### Sample Implementation
```sql
-- File: deploy/migrations/clickhouse/058_data_column_sidecar.up.sql
CREATE TABLE beacon_api_eth_v1_events_data_column_sidecar_local on cluster '{cluster}' (
    event_date_time DateTime64(3) Codec(DoubleDelta, ZSTD(1)),
    slot UInt32 Codec(DoubleDelta, ZSTD(1)),
    slot_start_date_time DateTime Codec(DoubleDelta, ZSTD(1)),
    propagation_slot_start_diff UInt32 Codec(ZSTD(1)),
    epoch UInt32 Codec(DoubleDelta, ZSTD(1)),
    epoch_start_date_time DateTime Codec(DoubleDelta, ZSTD(1)),
    block_root FixedString(66) Codec(ZSTD(1)),
    column_index UInt64 Codec(ZSTD(1)),
    kzg_commitments Array(FixedString(98)) Codec(ZSTD(1)),
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

CREATE TABLE beacon_api_eth_v1_events_data_column_sidecar on cluster '{cluster}' AS beacon_api_eth_v1_events_data_column_sidecar_local
ENGINE = Distributed('{cluster}', default, beacon_api_eth_v1_events_data_column_sidecar_local, rand());
```

### 6. Vector Configuration Updates

#### Specific Changes
- Add Kafka input source for data column sidecar events
- Add routing logic to handle `BEACON_API_ETH_V1_EVENTS_DATA_COLUMN_SIDECAR` events
- Add data transformation/formatting for ClickHouse ingestion
- Add ClickHouse sink for processed events

#### Sample Implementation
```yaml
# In deploy/local/docker-compose/vector-kafka-clickhouse.yaml

# Kafka input for data column sidecar events
sources:
  beacon_api_eth_v1_events_data_column_sidecar_kafka:
    type: kafka
    bootstrap_servers: "${KAFKA_BROKERS}"
    auto_offset_reset: earliest
    group_id: xatu-vector-kafka-clickhouse-beacon-api-eth-v1-events-data-column-sidecar
    key_field: "event.id"
    decoding:
      codec: json
    topics:
      - "beacon-api-eth-v1-events-data-column-sidecar"

# Router updates
transforms:
  xatu_server_events_router:
    type: route
    inputs:
      - beacon_api_eth_v1_events_data_column_sidecar_kafka
      # ... other inputs
    route:
      eth_v1_events_data_column_sidecar: .event.name == "BEACON_API_ETH_V1_EVENTS_DATA_COLUMN_SIDECAR"
      # ... other routes

  # Data formatting for ClickHouse
  beacon_api_eth_v1_events_data_column_sidecar_formatted:
    type: remap
    inputs:
      - xatu_server_events_router.eth_v1_events_data_column_sidecar
    source: |-
      event_date_time, err = parse_timestamp(.event.date_time, format: "%+");
      if err == null {
        .event_date_time = to_unix_timestamp(event_date_time, unit: "milliseconds")
      }
      .slot = .data.slot
      .column_index = .data.index
      .block_root = .data.block_root
      .kzg_commitments = .data.kzg_commitments
      
      # Extract metadata
      .meta_client_name = .meta.client.name
      .meta_client_id = .meta.client.id
      # ... other metadata fields
      
      # Extract timing data from additional_data
      .slot_start_date_time = to_unix_timestamp(parse_timestamp(.meta.client.additional_data.slot.start_date_time, format: "%+"))
      .propagation_slot_start_diff = .meta.client.additional_data.propagation.slot_start_diff
      .epoch = .meta.client.additional_data.epoch.number
      .epoch_start_date_time = to_unix_timestamp(parse_timestamp(.meta.client.additional_data.epoch.start_date_time, format: "%+"))
      
      del(.event)
      del(.meta)
      del(.data)

# ClickHouse sink
sinks:
  beacon_api_eth_v1_events_data_column_sidecar_clickhouse:
    type: clickhouse
    inputs:
      - beacon_api_eth_v1_events_data_column_sidecar_formatted
    auth:
      strategy: basic
      user: "${CLICKHOUSE_USER}"
      password: "${CLICKHOUSE_PASSWORD}"
    database: default
    endpoint: "${CLICKHOUSE_ENDPOINT}"
    table: beacon_api_eth_v1_events_data_column_sidecar
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

### 7. Kafka Topic Configuration

#### Specific Changes
- Add new Kafka topic: `beacon-api-eth-v1-events-data-column-sidecar`
- Configure topic partitioning and replication settings
- Update Kafka output routing in xatu components

#### Sample Implementation
```yaml
# Topic creation (typically handled by infrastructure)
# Topic: beacon-api-eth-v1-events-data-column-sidecar
# Partitions: 3
# Replication Factor: 3
# Retention: 168h (7 days)
```

## Testing Strategy

### Unit Testing
- **Event Handler Tests**: Verify event creation, decoration, and duplicate detection
- **Protobuf Tests**: Validate serialization/deserialization of data structures
- **Cache Tests**: Ensure proper duplicate detection behavior
- **Server Event Processing**: Test event validation and filtering logic

### Integration Testing  
- **Sentry to Kafka**: Verify events are properly published to Kafka topics
- **Vector Processing**: Test data transformation and routing logic
- **ClickHouse Ingestion**: Validate data insertion and schema compliance
- **Multi-sink Validation**: Verify events reach all configured outputs (kafka, http, stdout, xatu)
- **Timing and Metadata**: Validate propagation timing and metadata accuracy

### End-to-End Testing
- **Complete Pipeline**: Test sentry → kafka → vector → clickhouse data flow
- **Data Integrity**: Verify no data loss or corruption through pipeline
- **Error Handling**: Test pipeline behavior during component failures
- **Performance Testing**: Load test with high-volume data column sidecar events
- **Monitoring**: Validate metrics and alerting throughout the pipeline

### Database Testing
- **Schema Validation**: Verify ClickHouse table creation and structure
- **Query Performance**: Test read performance with realistic data volumes
- **Partitioning**: Validate partition pruning and query optimization
- **Migration Testing**: Test up/down migrations in isolated environments

### Infrastructure Testing
- **Kafka Topic Management**: Verify topic creation, partitioning, and replication
- **Vector Configuration**: Test configuration reload and error recovery
- **Network Resilience**: Test behavior during network partitions and timeouts
- **Resource Usage**: Monitor CPU, memory, and disk usage under load

### Validation Criteria
- **Spec Compliance**: Event structure matches ethereum/beacon-APIs specification exactly
- **Performance**: No performance degradation in event processing pipeline
- **Data Accuracy**: 100% data integrity from sentry to ClickHouse
- **Compatibility**: No breaking changes to existing event handling
- **Scalability**: Pipeline handles expected PeerDAS event volumes
- **Reliability**: Pipeline recovers gracefully from component failures
- **Edge Cases**: Handle malformed events, network issues, and high event volumes

## Implementation Dependencies

1. **Phase 1: Protobuf and Core Infrastructure**
   - Add protobuf definitions for data column sidecar events
   - Generate protobuf Go code using `buf generate`
   - Update event type enums and oneof fields
   - Dependencies: Protobuf schema design completion

2. **Phase 2: Sentry Event Handler Implementation**
   - Implement sentry-side event handler and cache integration
   - Add duplicate detection using TTL cache
   - Register event handler with beacon client
   - Dependencies: Phase 1 completion, go-eth2-client DataColumnSidecar support

3. **Phase 3: Server-side Event Processing**
   - Implement server-side event ingester
   - Add event validation and filtering logic
   - Register event handler in server pipeline
   - Dependencies: Phase 2 completion

4. **Phase 4: ClickHouse Database Schema**
   - Create ClickHouse migration files (up/down)
   - Define table schema with proper indexing and partitioning
   - Add column comments and metadata
   - Dependencies: Phase 3 completion

5. **Phase 5: Kafka and Vector Pipeline**
   - Configure new Kafka topic for data column sidecar events
   - Update Vector configuration for routing and transformation
   - Add ClickHouse sink configuration
   - Test data flow from Kafka through Vector to ClickHouse
   - Dependencies: Phase 4 completion, Kafka infrastructure availability

6. **Phase 6: End-to-End Integration Testing**
   - Test complete pipeline: sentry → kafka → vector → clickhouse
   - Validate data integrity and transformation accuracy
   - Load testing and performance validation
   - Dependencies: Phase 5 completion, test environment availability

7. **Phase 7: Production Deployment**
   - Deploy database migrations to production
   - Update Vector configurations
   - Deploy updated sentry and server components
   - Monitor data flow and performance
   - Dependencies: Phase 6 completion, production deployment approval

## Risks and Considerations

### Implementation Risks
- **go-eth2-client Support**: go-eth2-client library may not yet support DataColumnSidecar events - Mitigation: Check library compatibility and contribute upstream if needed
- **Specification Changes**: ethereum/beacon-APIs PR may evolve - Mitigation: Monitor PR status and update implementation accordingly

### Performance Considerations
- **Event Volume**: Data column sidecars may have higher volume than blob sidecars - Mitigation: Leverage existing duplicate cache and efficient processing patterns
- **Memory Usage**: Additional cache for data column events - Mitigation: Use same TTL cache patterns as existing events

### Security Considerations
- **Event Validation**: Ensure proper validation of incoming data column events - Mitigation: Follow existing validation patterns and sanitize inputs
- **Resource Usage**: Prevent resource exhaustion from malicious events - Mitigation: Implement rate limiting and proper error handling

## Expected Outcomes

- Complete end-to-end implementation of data_column_sidecar event handling from beacon client to ClickHouse
- Full compatibility with ethereum/beacon-APIs specification and PeerDAS (EIP-7594) requirements
- Seamless integration with existing xatu event processing pipeline
- Support for all output formats: Kafka, HTTP, stdout, and xatu server
- Robust data persistence in ClickHouse with proper indexing and partitioning
- Production-ready Vector configuration for reliable data transformation
- Comprehensive monitoring and alerting for the complete data pipeline

### Success Metrics
- **Event Processing**: Successfully processes and forwards data column sidecar events through complete pipeline
- **Spec Compliance**: 100% compliance with ethereum/beacon-APIs data_column_sidecar specification
- **Data Pipeline**: Events flow reliably from sentry → kafka → vector → clickhouse
- **Performance**: No measurable performance impact on existing event processing
- **Data Integrity**: 100% data accuracy from event capture to database storage  
- **Integration**: Events successfully reach all configured output sinks
- **Reliability**: Proper duplicate detection, error handling, and pipeline recovery
- **Scalability**: Pipeline handles expected PeerDAS event volumes without degradation
- **Query Performance**: ClickHouse queries perform efficiently on data column sidecar data
- **Monitoring**: Complete observability of event flow and pipeline health