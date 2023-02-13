# Server

A centralized server running configurable services collecting events from clients and can output them to sinks.

## Table of contents

- [Usage](#usage)
- [Requirements](#requirements)
- [Configuration](#configuration)
  - [Simple example](#simple-example)
  - [Only run the event exporter example](#only-run-the-event-exporter-example)
  - [Complex example](#complex-example)
- [Services](#services)
  - [Coordinator](#coordinator)
  - [Event Ingester](#event-ingester)
    - [Events](#events)
      - [Event field](#event-field)
      - [Meta field](#meta-field)
      - [Data field](#data-field)
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

## Configuration

Server requires a single `yaml` config file. An example file can be found [here](../example_server.yaml)

| Name| Type | Default | Description |
| --- | --- | --- | --- |
| logging | string | `warn` | Log level (`panic`, `fatal`, `warn`, `info`, `debug`, `trace`) |
| metricsAddr | string | `:9090` | The address the metrics server will listen on |
| addr | string | `:8080` | The grpc address for [services](#services) |
| labels | object |  | A key value map of labels to append to every sentry event |
| ntpServer | string | `pool.ntp.org` | NTP server to calculate clock drift for events |
| services | object |  | [Services](#services) to run |
| services.coordinator | object |  | [Coordinator](#coordinator) service |
| services.coordinator.enabled | bool | `false` | Enable the coordinator service |
| services.coordinator.persistence | object |  | Persistence configuration |
| services.coordinator.persistence.driverName | string | `postgres` | Persistence driver name (`postgres`) |
| services.coordinator.persistence.connectionString | string |  | Connection string for the persistence driver |
| services.coordinator.persistence.maxIdleConns | int | `2` | The maximum number of connections in the idle connection pool. `0` means no idle connections are retained. |
| services.coordinator.persistence.maxOpenConns | int | `0` | The maximum number of open connections to the database. `0` means unlimited connections. |
| services.coordinator.persistence.maxQueueSize | int | `51200` | The maximum queue size to buffer items for delayed processing. If the queue gets full it drops the items |
| services.coordinator.persistence.batchTimeout | string | `5s` | The maximum duration for constructing a batch. Processor forcefully sends available items when timeout is reached |
| services.coordinator.persistence.exportTimeout | string | `30s` | The maximum duration for exporting items. If the timeout is reached, the export will be cancelled |
| services.coordinator.persistence.maxExportBatchSize | int | `512` | MaxExportBatchSize is the maximum number of items to process in a single batch. If there are more than one batch worth of items then it processes multiple batches of items one batch after the other without any delay |
| services.eventIngester | object |  | [Event Ingester](#event-ingester) service |
| services.eventIngester.enabled | bool | `false` | Enable the event ingester service |
| services.eventIngester.outputs | array |  | List exampleone batch after the other without any delay |

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

labels:
  ethpandaops: rocks

ntpServer: time.google.com

services:
  coordinator:
    enabled: true
    persistence:
      driverName: postgres
      connectionString: postgres://postgres:password@localhost:5432/xatu?sslmode=disable
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
```

## Services

Xatu server runs a number of services that can be configured to run or not. Each service has its own configuration.

### Coordinator

The coordinator service is responsible for;
- adding/updating [ethereum node records](https://eips.ethereum.org/EIPS/eip-778) into persistence from [Xatu discovery](./discovery.md) clients.
- TODO: mimicry client management

[Persistence](#persistence) is **required** for the coordinator service.

### Event Ingester

The event ingester service is responsible for receiving events from clients (sentries), validating and then forwarding them to sinks.

#### Events

All events output from the ingester contain the following fields;

- `event` - name and date time of the event
- `meta` - metadata of the event, including specific event details as well as sentry/server computed info
- `data` - raw data collected for the event. This could be the Beacon API payload or a transaction.

The following is a sample event output is from a [Xatu sentry block](./sentry.md#block) event;

```json
{
  "event": {
    "name": "BEACON_API_ETH_V1_EVENTS_BLOCK",
    "date_time": "2022-01-01T10:12:10.050Z"
  },
  "meta": {
    "client": {
      "name": "some-client-001",
      "version": "v1.0.1",
      "id": "0697583c-3c65-4f9a-bcd0-b57ef919dc6c",
      "os": "amiga4000-68040",
      "clock_drift": 51,
      "labels": {
        "network-class": "tincan"
      },
      "ethereum": {
        "network": {
          "id": 1,
          "name": "mainnet"
        },
        "consensus": {
          "implementation": "lighthouse",
          "version": "v1.0.0-abc"
        }
      },
      "additional_data": {
        "slot": {
          "start_date_time": "2022-01-01T10:12:10.000Z"
        },
        "epoch": {
          "number": 1
          "start_date_time": "2022-01-01T10:10:10.000Z"
        },
        "propagation": {
          "slot_start_diff": 50
        }
      }
    },
    "server": {
        "event": {
            "received_date_time": "2022-01-01T10:12:10.050Z"
        },
        "client": {
            "ip": "1.1.1.1",
            "geo": {
                "city": "Brisbane",
                "country": "Australia",
                "country_code": "AU",
                "continent_code": "OC",
                "latitude": -27.4698,
                "longitude": 153.0251,
                "autonomous_system_number": 1221,
                "autonomous_system_organization": "Telstra Corporation Ltd",
                "isp": "Telstra Internet",
                "organization": "Telstra Corporation Ltd"
            }
        }
    }
  },
  "data": {
    "slot":"10",
    "block":"0x9a2fefd2fdb57f74993c7780ea5b9030d2897b615b89f808011ca5aebed54eaf",
    "execution_optimistic": false
  }
}
```

##### Event field

| Name | Type | Required | Description |
| --- | --- | --- | --- |
| event.name | string | `required` | Event name |
| event.date_time | string | `required` | When the event occured |

##### Meta field

| Name | Type | Required | Description |
| --- | --- | --- | --- |
| meta.client | object | `required` | Meta data appended from the sentry client |
| meta.client.name | string | `required` | Sentry name |
| meta.client.version | string | `required` | Sentry version |
| meta.client.id | string | `required` | Sentry ID generated on service start |
| meta.client.implementation | string | `required` | Sentry implementation eg. Xatu |
| meta.client.os | string | `optional` | Sentry operating system |
| meta.client.clock_drift | int | `optional` | NTP calculated clock drift of the sentry |
| meta.client.labels | object |  | A key value map of labels |
| meta.client.ethereum.network.name | string | `required` | Ethereum network name eg. `mainnet`, `sepolia` |
| meta.client.ethereum.consensus.implementation | string | `optional` | Ethereum consensus client name upstream from this sentry |
| meta.client.ethereum.consensus.version | string | `optional` | Ethereum consensus client version upstream from this sentry |
| meta.client.additional_data | object | `optional` | Computed additional data to compliment the events upstream raw data eg. calculating slot start date time |
| meta.server | object | `required` | Meta data appended from the server |
| meta.server.event.received_date_time | string | `required` | When the event was received by the server |
| meta.server.client.ip | string | `optional` | IP address of the client |
| meta.server.client.geo | object | `optional` | Geo data of the client |
| meta.server.client.geo.city | string | `optional` | City of the client |
| meta.server.client.geo.country | string | `optional` | Country of the client |
| meta.server.client.geo.country_code | string | `optional` | Country code of the client |
| meta.server.client.geo.continent_code | string | `optional` | Continent code of the client |
| meta.server.client.geo.latitude | float | `optional` | Latitude of the client |
| meta.server.client.geo.longitude | float | `optional` | Longitude of the client |
| meta.server.client.geo.autonomous_system_number | int | `optional` | Autonomous system number of the client |
| meta.server.client.geo.autonomous_system_organization | string | `optional` | Autonomous system organization of the client |
| meta.server.client.geo.isp | string | `optional` | ISP of the client |
| meta.server.client.geo.organization | string | `optional` | Organization of the client |

##### Data field

Depending on the origin of the event the data field will contain different data;
- [Xatu sentry](./sentry.md#events) events

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
