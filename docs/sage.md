# Sage

Client that is run along side an [Armiarma](https://github.com/migalabs/armiarma) instance and collects data from the beacon chain p2p network.

This sage can output events to various sinks and it is **not** a hard requirement to run the [Xatu server](./server.md).

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

sage requires a [config file](#configuration).

```bash
Usage:
  xatu sage [flags]

Flags:
      --config string   config file (default is sage.yaml) (default "sage.yaml")
  -h, --help            help for sage
```

## Requirements

- [Armiarma instance](https://github.com/migalabs/armiarma) with expose SSE server.

## Configuration

Sage requires a single `yaml` config file. An example file can be found [here](../example_sage.yaml)

| Name| Type | Default | Description                                                                                                                                |
| --- | --- | --- |--------------------------------------------------------------------------------------------------------------------------------------------|
| logging | string | `warn` | Log level (`panic`, `fatal`, `warn`, `info`, `debug`, `trace`)                                                                             |
| metricsAddr | string | `:9090` | The address the metrics server will listen on                                                                                              |
| pprofAddr | string | | The address the [pprof](https://github.com/google/pprof) server will listen on. When ommited, the pprof server will not be started         |
| name | string |  | Unique name of the sage                                                                                                                  |
| labels | object |  | A key value map of labels to append to every sage event                                                                                  |
| ethereum.beaconNodeAddress | string |  | [Ethereum consensus client](https://ethereum.org/en/developers/docs/nodes-and-clients/#consensus-clients) http server endpoint             |
| ethereum.beaconNodeAddress | object |  | A key value map of headers                                                                                                                 |
| ethereum.overrideNetworkName | string |  | Override the network name                                                                                                                  |
| armiarmaUrl | string |  | The address of the Armiarma instance                                                                                           |
| workers | int | 1  | The count of workers processing attestations. Will result in more duplicate attestations if count is greater than 1                                                                                          |
| duplicateAttestationThreshold | int | 3 | The amount of times to forward on the same attestation                                                                                   |
| outputs | array<object> |  | List of outputs for the sage to send data to                                                                                             |
| outputs[].name | string |  | Name of the output                                                                                                                         |
| outputs[].type | string |  | Type of output (`xatu`, `http`, `kafka`, `stdout`)                                                                                         |
| outputs[].config | object |  | Output type configuration [`xatu`](#output-xatu-configuration)/[`http`](#output-http-configuration)/[`kafka`](#output-kafka-configuration) |

### Output `xatu` configuration

Output configuration to send sage events to a [Xatu server](./server.md).

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

Output configuration to send sage events to a http server.

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
| outputs[].config.topic          | string |           |                                     | Name of the topic.                                                                                                                      |
| outputs[].config.flushFrequency | string | `3s`      |                                     | The maximum time a single batch can wait before a flush. Producer flushes the batch when the limit is reached.                          |
| outputs[].config.flushMessages  | int    | `500`     |                                     | The maximum number of events in a single batch before a flush. Producer flushes the batch when the limit is reached.                    |
| outputs[].config.flushBytes     | int    | `1000000` |                                     | The maximum size (in bytes) of a single batch before a flush. Producer flushes the batch when the limit is reached.                     |
| outputs[].config.maxRetries     | int    | `3`       |                                     | The maximum retries allowed for a single batch delivery. The batch would be dropped, if the producer fails to flush with-in this limit. |
| outputs[].config.compression    | string | `none`    | `none` `gzip` `snappy` `lz4` `zstd` | Compression to use.                                                                                                                     |
| outputs[].config.requiredAcks   | string | `leader`  | `none` `leader` `all`               | Number of ack's required for a succesful batch delivery.                                                                                |
| outputs[].config.partitioning   | string | `none`    | `none` `random`                     | Paritioning to use for the distribution of messages across the partitions.                                                              |

### Simple example

```yaml
name: xatu-sage

armiarmaUrl: http://localhost:9099/events

ethereum:
  beaconNodeAddress: http://localhost:5052

outputs:
- name: standard-out
  type: stdout
```

### Xatu server output example

```yaml
name: xatu-sage

armiarmaUrl: http://localhost:9099/events

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
name: xatu-sage

armiarmaUrl: http://localhost:9099/events

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

armiarmaUrl: http://localhost:9099/events

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

name: xatu-sage

labels:
  ethpandaops: rocks

ntpServer: time.google.com

armiarmaUrl: http://localhost:9099/events

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
docker run -d --name xatu-sage -v $HOST_DIR_CHANGE_ME/config.yaml:/opt/xatu/config.yaml -p 9090:9090 -it ethpandaops/xatu:latest sage --config /opt/xatu/config.yaml
# build
go build -o dist/xatu main.go
./dist/xatu sage --config sage.yaml
# dev
go run main.go sage --config sage.yaml
```
