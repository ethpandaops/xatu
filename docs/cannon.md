# Cannon

Client that is run along side an [Ethereum consensus client](https://ethereum.org/en/developers/docs/nodes-and-clients/#consensus-clients) and collects canonical data via the consensus client's [Beacon API](https://ethereum.github.io/beacon-APIs/). *You must run your own consensus client* and this cannon will connect to it via the consensus client's http server.

This cannon can output events to various sinks and it is **not** a hard requirement to run the [Xatu server](./server.md).

## Table of contents

- [Usage](#usage)
- [Requirements](#requirements)
- [Configuration](#configuration)
  - [`xatu` output](#output-xatu-configuration)
  - [`http` output](#output-http-configuration)
  - [Simple example](#simple-example)
  - [Xatu server output example](#xatu-server-output-example)
  - [HTTP server output example](#http-server-output-example)
  - [Complex example with multiple outputs example](#complex-example-with-multiple-outputs-example)
- [Running locally](#running-locally)

## Usage

Cannon requires a [config file](#configuration).

```bash
Usage:
  xatu cannon [flags]

Flags:
      --config string   config file (default is cannon.yaml) (default "cannon.yaml")
  -h, --help            help for cannon
```

## Requirements

- [Ethereum consensus client](https://ethereum.org/en/developers/docs/nodes-and-clients/#consensus-clients) with exposed [http server](https://ethereum.github.io/beacon-APIs/).
- [Server](./server.md) running with the [Coordinator](./server.md#coordinator) service enabled.

## Configuration

Cannon requires a single `yaml` config file. An example file can be found [here](../example_cannon.yaml)

| Name| Type | Default | Description                                                                                                                                |
| --- | --- | --- |--------------------------------------------------------------------------------------------------------------------------------------------|
| logging | string | `warn` | Log level (`panic`, `fatal`, `warn`, `info`, `debug`, `trace`)                                                                             |
| metricsAddr | string | `:9090` | The address the metrics server will listen on                                                                                              |
| pprofAddr | string | | The address the [pprof](https://github.com/google/pprof) server will listen on. When ommited, the pprof server will not be started         |
| name | string |  | Unique name of the cannon                                                                                                                  |
| labels | object |  | A key value map of labels to append to every cannon event                                                                                  |
| ethereum.beaconNodeAddress | string |  | [Ethereum consensus client](https://ethereum.org/en/developers/docs/nodes-and-clients/#consensus-clients) http server endpoint             |
| ethereum.beaconNodeAddress | object |  | A key value map of headers                                                                                                                 |
| ethereum.overrideNetworkName | string |  | Override the network name                                                                                                                  |
| ethereum.blockCacheSize | int | `1000` | The maximum number of blocks to cache                                                                                                      |
| ethereum.blockCacheTtl | string | `1h` | The maximum duration to cache blocks                                                                                                       |
| ethereum.blockPreloadWorkers | int | `5` | The number of workers to use for preloading blocks                                                                                         |
| ethereum.blockPreloadQueueSize | int | `5000` | The maximum number of blocks to queue for preloading                                                                                       |
| coordinator.address | string |  | The address of the [Xatu server](./server.md)                                                                                              |
| coordinator.tls | bool |  | Server requires TLS                                                                                                                        |
| coordinator.headers | object |  | A key value map of headers to append to requests                                                                                           |
| derivers.attesterSlashing.enabled | bool | `true` | Enable the attester slashing deriver                                                                                                       |
| derivers.attesterSlashing.headSlotLag | int | `5` | The number of slots to lag behind the head                                                                                                 |
| derivers.blsToExecutionChange.enabled | bool | `true` | Enable the BLS to execution change deriver                                                                                                 |
| derivers.blsToExecutionChange.headSlotLag | int | `5` | The number of slots to lag behind the head                                                                                                 |
| derivers.deposit.enabled | bool | `true` | Enable the deposit deriver                                                                                                                 |
| derivers.deposit.headSlotLag | int | `5` | The number of slots to lag behind the head                                                                                                 |
| derivers.withdrawal.enabled | bool | `true` | Enable the withdrawal deriver                                                                                                              |
| derivers.withdrawal.headSlotLag | int | `5` | The number of slots to lag behind the head                                                                                                 |
| derivers.executionTransaction.enabled | bool | `true` | Enable the execution transaction deriver                                                                                                   |
| derivers.executionTransaction.headSlotLag | int | `5` | The number of slots to lag behind the head                                                                                                 |
| derivers.proposerSlashing.enabled | bool | `true` | Enable the proposer slashing deriver                                                                                                       |
| derivers.proposerSlashing.headSlotLag | int | `5` | The number of slots to lag behind the head                                                                                                 |
| derivers.voluntaryExit.enabled | bool | `true` | Enable the voluntary exit deriver                                                                                                          |
| derivers.voluntaryExit.headSlotLag | int | `5` | The number of slots to lag behind the head                                                                                                 |
| ntpServer | string | `pool.ntp.org` | NTP server to calculate clock drift for events                                                                                             |
| outputs | array<object> |  | List of outputs for the cannon to send data to                                                                                             |
| outputs[].name | string |  | Name of the output                                                                                                                         |
| outputs[].type | string |  | Type of output (`xatu`, `http`, `kafka`, `stdout`)                                                                                         |
| outputs[].config | object |  | Output type configuration [`xatu`](#output-xatu-configuration)/[`http`](#output-http-configuration)/[`kafka`](#output-kafka-configuration) |

### Output `xatu` configuration

Output configuration to send cannon events to a [Xatu server](./server.md).

| Name| Type | Default | Description |
| --- | --- | --- | --- |
| outputs[].config.address | string |  | The address of the server receiving events |
| outputs[].config.tls | bool |  | Server requires TLS |
| outputs[].config.headers | object |  | A key value map of headers to append to requests |
| outputs[].config.maxQueueSize | int | `51200` | The maximum queue size to buffer events for delayed processing. If the queue gets full it drops the events |
| outputs[].config.batchTimeout | string | `5s` | The maximum duration for constructing a batch. Processor forcefully sends available events when timeout is reached |
| outputs[].config.exportTimeout | string | `30s` | The maximum duration for exporting events. If the timeout is reached, the export will be cancelled |
| outputs[].config.maxExportBatchSize | int | `512` | MaxExportBatchSize is the maximum number of events to process in a single batch. If there are more than one batch worth of events then it processes multiple batches of events one batch after the other without any delay |

### Output `http` configuration

Output configuration to send cannon events to a http server.

| Name| Type | Default | Description |
| --- | --- | --- | --- |
| outputs[].config.address | string |  | The address of the server receiving events |
| outputs[].config.headers | object |  | A key value map of headers to append to requests |
| outputs[].config.maxQueueSize | int | `51200` | The maximum queue size to buffer events for delayed processing. If the queue gets full it drops the events |
| outputs[].config.batchTimeout | string | `5s` | The maximum duration for constructing a batch. Processor forcefully sends available events when timeout is reached |
| outputs[].config.exportTimeout | string | `30s` | The maximum duration for exporting events. If the timeout is reached, the export will be cancelled |
| outputs[].config.maxExportBatchSize | int | `512` | MaxExportBatchSize is the maximum number of events to process in a single batch. If there are more than one batch worth of events then it processes multiple batches of events one batch after the other without any delay |

### Output `kafka` configuration

Output configuration to send sentry events to a kafka server.

| Name                            | Type   | Default   | Allowed Values                      | Description                                                                                                                             |
| ------------------------------- | ------ |-----------|-------------------------------------|-----------------------------------------------------------------------------------------------------------------------------------------|
| outputs[].config.brokers        | string |           |                                     | Comma delimited list of brokers. Eg: `localhost:19091,localhost:19092`                                                                  |
| outputs[].config.topic          | string |           |                                     | Name of the topic. Mutually exclusive with `topicPattern`.                                                                              |
| outputs[].config.topicPattern   | string |           |                                     | Topic template with variables `${EVENT_NAME}` or `${event-name}`. Mutually exclusive with `topic`.                                      |
| outputs[].config.flushFrequency | string | `3s`      |                                     | The maximum time a single batch can wait before a flush. Producer flushes the batch when the limit is reached.                          |
| outputs[].config.flushMessages  | int    | `500`     |                                     | The maximum number of events in a single batch before a flush. Producer flushes the batch when the limit is reached.                    |
| outputs[].config.flushBytes     | int    | `1000000` |                                     | The maximum size (in bytes) of a single batch before a flush. Producer flushes the batch when the limit is reached.                     |
| outputs[].config.maxRetries     | int    | `3`       |                                     | The maximum retries allowed for a single batch delivery. The batch would be dropped, if the producer fails to flush with-in this limit. |
| outputs[].config.compression    | string | `none`    | `none` `gzip` `snappy` `lz4` `zstd` | Compression to use.                                                                                                                     |
| outputs[].config.requiredAcks   | string | `leader`  | `none` `leader` `all`               | Number of ack's required for a succesful batch delivery.                                                                                |
| outputs[].config.partitioning   | string | `none`    | `none` `random`                     | Paritioning to use for the distribution of messages across the partitions.                                                              |
| outputs[].config.encoding       | string | `json`    | `json` `protobuf`                   | Serialization format for message values.                                                                                                |

### Simple example

```yaml
name: xatu-cannon

coordinator:
  address: http://localhost:8080

ethereum:
  beaconNodeAddress: http://localhost:5052

outputs:
- name: standard-out
  type: stdout
```

### Xatu server output example

```yaml
name: xatu-cannon

coordinator:
  address: http://localhost:8080

ethereum:
  beaconNodeAddress: http://localhost:5052

outputs:
- name: xatu-output
  type: xatu
  config:
    address: localhost:8080
```

### http server output example

```yaml
name: xatu-cannon

coordinator:
  address: http://localhost:8080

ethereum:
  beaconNodeAddress: http://localhost:5052

outputs:
- name: http-basic-auth
  type: http
  config:
    address: http://localhost:8080
    headers:
      authorization: "Basic Someb64Value"
```

### kafka server output example

```yaml
name: example-instance-004

ethereum:
  beaconNodeAddress: http://localhost:5052

outputs:
- name: kafka-sink
  type: kafka
  config:
    brokers: localhost:19092
    topic: events
```
### Complex example with multiple outputs example

```yaml
logging: "debug"
metricsAddr: ":9090"
pprofAddr: ":6060"

name: xatu-cannon

labels:
  ethpandaops: rocks

ntpServer: time.google.com

coordinator:
  address: http://localhost:8080

ethereum:
  beaconNodeAddress: http://localhost:5052

outputs:
- name: log
  type: stdout
- name: xatu-server
  type: xatu
  config:
    address: localhost:8080
    headers:
      authorization: Someb64Value
    maxQueueSize: 51200
    batchTimeout: 5s
    exportTimeout: 30s
    maxExportBatchSize: 512
- name: kafka-sink
  type: kafka
  config:
    brokers: localhost:19092
    topic: events
```

## Running locally

```bash
# docker
docker run -d --name xatu-cannon -v $HOST_DIR_CHANGE_ME/config.yaml:/opt/xatu/config.yaml -p 9090:9090 -it ethpandaops/xatu:latest cannon --config /opt/xatu/config.yaml
# build
go build -o dist/xatu main.go
./dist/xatu cannon --config cannon.yaml
# dev
go run main.go cannon --config cannon.yaml
```
