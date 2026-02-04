# Server

A centralized server running configurable services collecting events from clients and can output them to sinks.

## Table of contents

- [Usage](#usage)
- [Requirements](#requirements)
- [Configuration](#configuration)
  - [Store `redis-server` configuration](#store-redis-server-configuration)
  - [Store `redis-cluster` configuration](#store-redis-cluster-configuration)
  - [GeoIP `maxmind` configuration](#geoip-maxmind-configuration)
  - [Simple example](#simple-example)
  - [Only run the event exporter example](#only-run-the-event-exporter-example)
  - [Complex example](#complex-example)
- [Services](#services)
  - [Coordinator](#coordinator)
  - [Event Ingester](#event-ingester)
- [HTTP Ingester](#http-ingester)
- [Persistence](#persistence)
  - [Migrations](#migrations)
- [Running locally](#running-locally)
  - [Running locally with persistence (native)](#running-locally-with-persistence-native)
  - [Running locally with persistence (docker)](#running-locally-with-persistence-docker)

## Usage

Server requires a [config file](#configuration).

```bash
Usage:
  xatu server [flags]

Flags:
      --config string   config file (default is server.yaml) (default "server.yaml")
  -h, --help            help for server
```

## Requirements

- [PostgreSQL](https://www.postgresql.org/) database **only if** running the [Coordinator](./server.md#coordinator) service.
- [Redis server/cluster](https://redis.io/) if using the [`redis-server`](#store-redis-server-configuration)/[`redis-cluster`](#store-redis-cluster-configuration) `store` type.
- [MaxMind GeoLite2 City/ASN database](https://dev.maxmind.com/geoip/geolite2-free-geolocation-data) if using the [`maxmind`](#geoip-maxmind-configuration) `geoip` type.

## Configuration

Server requires a single `yaml` config file. An example file can be found [here](../example_server.yaml)

| Name| Type | Default | Description |
| --- | --- | --- | --- |
| logging | string | `warn` | Log level (`panic`, `fatal`, `warn`, `info`, `debug`, `trace`) |
| metricsAddr | string | `:9090` | The address the metrics server will listen on |
| pprofAddr | string | | The address the [pprof](https://github.com/google/pprof) server will listen on. When ommited, the pprof server will not be started |
| addr | string | `:8080` | The grpc address for [services](#services) |
| labels | object |  | A key value map of labels to append to every sentry event |
| ntpServer | string | `pool.ntp.org` | NTP server to calculate clock drift for events |
| persistence.enabled | bool | `false` | Enable persistence |
| persistence.driverName | string | `postgres` | Persistence driver name (`postgres`) |
| persistence.connectionString | string |  | Connection string for the persistence driver |
| persistence.maxIdleConns | int | `2` | The maximum number of connections in the idle connection pool. `0` means no idle connections are retained. |
| persistence.maxOpenConns | int | `0` | The maximum number of open connections to the database. `0` means unlimited connections. |
| store.type | string | `memory` | Type of store (`memory`, `redis-server`, `redis-cluster`) |
| store.config | object |  | Store type configuration [`redis-server`](#store-redis-server-configuration)/[`redis-cluster`](#store-redis-cluster-configuration) |
| geoip.enabled | bool | `false` | Enable the geoip provider |
| geoip.type | string | `maxmind` | Type of store (`maxmind`) |
| geoip.config | object |  | GeoIP Provider type configuration [`maxmind`](#geoip-maxmind-configuration) |
| httpIngester | object |  | [HTTP Ingester](#http-ingester) configuration |
| httpIngester.enabled | bool | `false` | Enable the HTTP ingester |
| httpIngester.addr | string | `:8081` | The address to listen on for HTTP requests |
| services | object |  | [Services](#services) to run |
| services.coordinator | object |  | [Coordinator](#coordinator) service |
| services.coordinator.enabled | bool | `false` | Enable the coordinator service |
| services.coordinator.nodeRecord.maxQueueSize | int | `51200` | The maximum queue size to buffer node records for delayed processing. If the queue gets full it drops the items |
| services.coordinator.nodeRecord.batchTimeout | string | `5s` | The maximum duration for constructing a batch. Processor forcefully sends available node records when timeout is reached |
| services.coordinator.nodeRecord.exportTimeout | string | `30s` | The maximum duration for exporting node records. If the timeout is reached, the export will be cancelled |
| services.coordinator.nodeRecord.maxExportBatchSize | int | `512` | MaxExportBatchSize is the maximum number of node records to process in a single batch. If there are more than one batch worth of items then it processes multiple batches of items one batch after the other without any delay |
| services.eventIngester | object |  | [Event Ingester](#event-ingester) service |
| services.eventIngester.enabled | bool | `false` | Enable the event ingester service |
| services.eventIngester.outputs | array |  | List exampleone batch after the other without any delay |

### Store `redis-server` configuration

Store configuration for Redis server.

| Name| Type | Default | Description |
| --- | --- | --- | --- |
| store.config.address | string |  | The address of the redis server. [Details on the address format](https://github.com/redis/go-redis/blob/97b491aaceafb0078fa31ee23d26b439bdc78387/options.go#L222) |
| store.config.prefix | string | `xatu` | The redis key prefix to use |

### Store `redis-cluster` configuration

Store configuration for Redis cluster.

| Name| Type | Default | Description |
| --- | --- | --- | --- |
| store.config.address | string |  | The address of the redis cluster. [Details on the address format](https://github.com/redis/go-redis/blob/97b491aaceafb0078fa31ee23d26b439bdc78387/cluster.go#L137) |
| store.config.prefix | string | `xatu` | The redis key prefix to use |

### GeoIP `maxmind` configuration

GeoIP configuration for MaxMind.

| Name| Type | Default | Description |
| --- | --- | --- | --- |
| geoip.config.database.city | string | | The path to the [MaxMind GeoLite2 City database](https://dev.maxmind.com/geoip/geolite2-free-geolocation-data) file |
| geoip.config.database.asn | string |  | The path to the [MaxMind GeoLite2 ASN database](https://dev.maxmind.com/geoip/geolite2-free-geolocation-data) file |

### Simple Example

Simple example that runs on default ports and enables the [Coordinator](./server.md#coordinator) and [Event Ingester](./server.md#event-ingester) services.

```yaml
services:
  coordinator:
    enabled: true
    persistence:
      driverName: postgres
      connectionString: postgres://postgres:admin@localhost:5432/xatu?sslmode=disable
  eventIngester:
    enabled: true
    outputs:
    - name: stdout
      type: stdout
```

### Only run the event exporter example

```yaml
services:
  eventIngester:
    enabled: true
    outputs:
    - name: http-sink
      type: http
      config:
        address: http://localhost:8081
        headers:
          authorization: "Basic Someb64Value"
```

### Complex example

```yaml
logging: "info"
addr: ":8080"
metricsAddr: ":9090"
pprofAddr: ":6060"

labels:
  ethpandaops: rocks

ntpServer: time.google.com

persistence:
  enabled: true
  driverName: postgres
  connectionString: postgres://postgres:password@localhost:5432/xatu?sslmode=disable
  maxIdleConns: 2 # 0 = no idle connections are retained
  maxOpenConns: 0 # 0 = unlimited

store:
  type: redis-server
  config:
    address: "redis://localhost:6379"

geoip:
  enabled: true
  type: maxmind
  config:
    database:
      city: ./GeoLite2-City.mmdb
      asn: ./GeoLite2-ASN.mmdb

services:
  coordinator:
    enabled: true
    nodeRecord:
      maxQueueSize: 51200
      batchTimeout: 5s
      exportTimeout: 30s
      maxExportBatchSize: 512
  eventIngester:
    enabled: true
    outputs:
    - name: logs
      type: stdout
    - name: http-sink
      type: http
      config:
        address: http://localhost:8081
        headers:
          authorization: "Basic Someb64Value"
    - name: kafka-sink
      type: kafka
      config:
        brokers: localhost:19092
        topic: events
```

## Services

Xatu server runs a number of services that can be configured to run or not. Each service has its own configuration.

### Coordinator

The coordinator service is responsible for;
- adding/updating [ethereum node records](https://eips.ethereum.org/EIPS/eip-778) into persistence from [Xatu discovery](./discovery.md) clients.

[Persistence](#persistence) is **required** for the coordinator service.

### Event Ingester

The event ingester service is responsible for receiving events from clients (sentries) via gRPC, validating and then forwarding them to sinks.

## HTTP Ingester

The HTTP ingester provides an HTTP endpoint for receiving events, as an alternative to the gRPC event ingester. This is useful for clients that cannot use gRPC, such as [Vector](https://vector.dev)-based log collectors like [Sentry Logs](./sentry-logs.md).

The HTTP ingester reuses the `services.eventIngester` configuration for authorization, outputs, and other settings.

| Name | Type | Default | Description |
| --- | --- | --- | --- |
| httpIngester.enabled | bool | `false` | Enable the HTTP ingester |
| httpIngester.addr | string | `:8081` | The address to listen on for HTTP requests |

### HTTP Ingester Example

```yaml
# HTTP ingester reuses services.eventIngester config
httpIngester:
  enabled: true
  addr: ":8081"

services:
  eventIngester:
    enabled: true
    clientNameSalt: "your-salt-here"
    authorization:
      enabled: true
      groups:
        default:
          users:
            myuser:
              password: mypassword
    outputs:
      - name: http-sink
        type: http
        config:
          address: http://localhost:9005
```

### HTTP Endpoint

- **POST** `/v1/events` - Submit events
  - Content-Type: `application/json` or `application/x-protobuf`
  - Content-Encoding: `gzip` (optional)
  - Authorization: `Basic <base64(username:password)>`

## Persistence

The server uses persistence layer to store data from various services. Currently only [PostgreSQL](https://www.postgresql.org/) is supported.

### Migrations

Xatu server uses the [golange-migrate](https://github.com/golang-migrate/migrate) tool for simple manual file based migrations. Xatu tries to follow the [Best practices: How to write migrations](https://github.com/golang-migrate/migrate/blob/master/MIGRATIONS.md) when adding new migrations.

An example to run migrations locally against a local postgres instance;

```bash
migrate -database "postgres://postgres:password@localhost:5432/xatu?sslmode=disable" -path ./migrations/postgres up
```

## Running locally

```bash
# docker
docker run -d --name xatu-server -v $HOST_DIR_CHANGE_ME/config.yaml:/opt/xatu/config.yaml -p 9090:9090 -p 8080:8080 -it ethpandaops/xatu:latest server --config /opt/xatu/config.yaml
# build
go build -o dist/xatu main.go
./dist/xatu server --config server.yaml
# dev
go run main.go server --config server.yaml
```

### Running locally with persistence (native)

Requires postgres and [golange-migrate](https://github.com/golang-migrate/migrate) to be installed locally.

```bash
# Create the xatu database
psql -h localhost -U postgres -W -c "create database xatu;"
# Run migrations (requires https://github.com/golang-migrate/migrate tool)
migrate -database "postgres://postgres:password@localhost:5432/xatu?sslmode=disable" -path ./migrations/postgres up
# Run the server
go run main.go server --config server.yaml
```

### Running locally with persistence (docker)

Requires docker to be installed locally.

```bash
# Start a postgres instance
docker run -p 5432:5432 --name xatupostgres -e POSTGRES_PASSWORD=password -d postgres
# Create the xatu database. "password" is the password
docker exec -it xatupostgres psql -U postgres -W -c "create database xatu;"
# Run migrations
docker run -v "$(pwd)/migrations/postgres":/migrations --network host migrate/migrate -path=/migrations/ -database "postgres://postgres:password@localhost:5432/xatu?sslmode=disable" up
# Run the server
go run main.go server --config server.yaml
```
