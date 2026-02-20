# Mimicry

Client that collects data from the execution layer P2P network.

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

Mimicry requires a [config file](#configuration).

```bash
Usage:
  xatu mimicry [flags]

Flags:
      --config string   config file (default is mimicry.yaml) (default "mimicry.yaml")
  -h, --help            help for mimicry
```

## Requirements

- *Optional* [server](./server.md) running with the [Coordinator](./server.md#coordinator) service enabled.

## Configuration

Mimicry requires a single `yaml` config file. An example file can be found [here](../example_mimicry.yaml)

| Name| Type | Default | Description                                                                                                                        |
| --- | --- | --- |------------------------------------------------------------------------------------------------------------------------------------|
| logging | string | `warn` | Log level (`panic`, `fatal`, `warn`, `info`, `debug`, `trace`)                                                                     |
| metricsAddr | string | `:9090` | The address the metrics server will listen on                                                                                      |
| pprofAddr | string | | The address the [pprof](https://github.com/google/pprof) server will listen on. When ommited, the pprof server will not be started |
| probeAddr | string | | The address for health probes. When ommited, the probe server will not be started                                                  |
| name | string |  | Unique name of the mimicry                                                                                                         |
| labels | object |  | A key value map of labels to append to every mimicry event                                                                         |
| ntpServer | string | `pool.ntp.org` | NTP server to calculate clock drift for events                                                                                     |
| captureDelay | string | `3m` | Delay before starting to capture transactions                                                                                      |
| coordinator.type | string |  | Type of output (`xatu`, `static`)                                                                                                  |
| coordinator.config | object |  | Coordinator type configuration [`xatu`](#coordinator-xatu-configuration)/[`static`](#coordinator-static-configuration)             |
| outputs | array<object> |  | List of outputs for the mimicry to send data to                                                                                    |
| outputs[].name | string |  | Name of the output                                                                                                                 |
| outputs[].type | string |  | Type of output (`xatu`, `http`, `kafka`, `stdout`)                                                                                 |
| outputs[].config | object |  | Output type configuration [`xatu`](#output-xatu-configuration)/[`http`](#output-http-configuration)/[`kafka`](#output-kafka-configuration)                                |

### Coordinator `xatu` configuration

Coordinator configuration to get peers to connect to from the [Xatu server coordinator](./server.md#coordinator).

| Name| Type | Default | Description |
| --- | --- | --- | --- |
| coordinator.config.address | string |  | The address of the server receiving events |
| coordinator.config.tls | bool |  | Server requires TLS |
| coordinator.config.headers | object |  | A key value map of headers to append to requests |
| coordinator.config.maxQueueSize | int | `51200` | The maximum queue size to buffer events for delayed processing. If the queue gets full it drops the events |
| coordinator.config.batchTimeout | string | `5s` | The maximum duration for constructing a batch. Processor forcefully sends available events when timeout is reached |
| coordinator.config.exportTimeout | string | `30s` | The maximum duration for exporting events. If the timeout is reached, the export will be cancelled |
| coordinator.config.maxExportBatchSize | int | `512` | MaxExportBatchSize is the maximum number of events to process in a single batch. If there are more than one batch worth of events then it processes multiple batches of events one batch after the other without any delay |

### Coordinator `static` configuration

Coordinator configuration to statically specify peers to connect to.

| Name| Type | Default | Description |
| --- | --- | --- | --- |
| coordinator.config.retryInterval | string |  | Interval between trying to connect to a peer |
| coordinator.config.nodeRecords | array<string> |  | List of ENR/ENode peers to connect to |

### Output `xatu` configuration

Output configuration to send mimicry events to a [Xatu server](./server.md).

| Name| Type | Default | Description |
| --- | --- | --- | --- |
| outputs[].config.address | string |  | The address of the server receiving events |
| outputs[].config.tls | bool |  | Server requires TLS |
| outputs[].config.headers | object |  | A key value map of headers to append to requests |
| outputs[].config.maxQueueSize | int | `51200` | The maximum queue size to buffer events for delayed processing. If the queue gets full it drops the events |
| outputs[].config.batchTimeout | string | `5s` | The maximum duration for constructing a batch. Processor forcefully sends available events when timeout is reached |
| outputs[].config.exportTimeout | string | `30s` | The maximum duration for exporting events. If the timeout is reached, the export will be cancelled |
| outputs[].config.maxExportBatchSize | int | `512` | MaxExportBatchSize is the maximum number of events to process in a single batch. If there are more than one batch worth of events then it processes multiple batches of events one batch after the other without any delay |
| outputs[].config.networkIds | array<string> |  | List of network ids to connect to (decimal format, eg. '1' for mainnet) |
| outputs[].config.forkIdHashes | array<string> |  | List of [Fork ID hash](https://eips.ethereum.org/EIPS/eip-2124) to connect to (hex string) |
| outputs[].config.maxPeers | int | `100` | Max number of peers to attempt to connect to simultaneously |

### Output `http` configuration

Output configuration to send mimicry events to a http server.

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
name: example-instance-001

coordinator:
  type: xatu
  config:
    address: localhost:8080
    networkIds: [1]
    forkIdHashes: [0xf0afd0e3]
    maxPeers: 100

outputs:
- name: standard-out
  type: stdout
```

### Simple example with static coordinator

```yaml
name: example-instance-001

coordinator:
  type: static
  config:
    retryInterval: 60s
    nodeRecords:
      - enr:-IS4QHCYrYZbAKWCBRlAy5zzaDZXJBGkcnh4MHcBFZntXNFrdvJjX04jRzjzCBOonrkTfj499SZuOh8R33Ls8RRcy5wBgmlkgnY0gmlwhH8AAAGJc2VjcDI1NmsxoQPKY0yuDUmstAHYpMa2_oxVtw0RW_QAdpzBQA8yWM0xOIN1ZHCCdl8

outputs:
- name: standard-out
  type: stdout
```

### Xatu server output example

```yaml
name: example-instance-002

coordinator:
  type: xatu
  config:
    address: localhost:8080
    networkIds: [1]
    forkIdHashes: [0xf0afd0e3]

outputs:
- name: xatu-output
  type: xatu
  config:
    address: localhost:8080
```

### http server output example

```yaml
name: example-instance-003

coordinator:
  type: xatu
  config:
    address: localhost:8080
    networkIds: [1]
    forkIdHashes: [0xf0afd0e3]

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
probeAddr: ":8080"

name: example-instance

labels:
  ethpandaops: rocks

ntpServer: time.google.com

captureDelay: 3m

coordinator:
  type: xatu
  config:
    address: localhost:8080
    networkIds: [1]
    forkIdHashes: [0xf0afd0e3]

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
docker run -d --name xatu-mimicry -v $HOST_DIR_CHANGE_ME/config.yaml:/opt/xatu/config.yaml -p 9090:9090 -it ethpandaops/xatu:latest mimicry --config /opt/xatu/config.yaml
# build
go build -o dist/xatu main.go
./dist/xatu mimicry --config mimicry.yaml
# dev
go run main.go mimicry --config mimicry.yaml
```
