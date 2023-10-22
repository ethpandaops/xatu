# Sentry

Client that is run along side an [Ethereum consensus client](https://ethereum.org/en/developers/docs/nodes-and-clients/#consensus-clients) and collects data via the consensus client's [Beacon API](https://ethereum.github.io/beacon-APIs/). *You must run your own consensus client* and this sentry will connect to it via the consensus client's http server.

This sentry can output events to various sinks and it is **not** a hard requirement to run the [Xatu server](./server.md).

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

Sentry requires a [config file](#configuration).

```bash
Usage:
  xatu sentry [flags]

Flags:
      --config string   config file (default is sentry.yaml) (default "sentry.yaml")
  -h, --help            help for sentry
```

## Requirements

- [Ethereum consensus client](https://ethereum.org/en/developers/docs/nodes-and-clients/#consensus-clients) with exposed [http server](https://ethereum.github.io/beacon-APIs/).
- *Optional* [server](./server.md) running with the [Coordinator](./server.md#coordinator) service enabled.

## Configuration

Sentry requires a single `yaml` config file. An example file can be found [here](../example_sentry.yaml)

| Name| Type | Default | Description |
| --- | --- | --- | --- |
| logging | string | `warn` | Log level (`panic`, `fatal`, `warn`, `info`, `debug`, `trace`) |
| metricsAddr | string | `:9090` | The address the metrics server will listen on |
| pprofAddr | string | | The address the [pprof](https://github.com/google/pprof) server will listen on. When ommited, the pprof server will not be started |
| name | string |  | Unique name of the sentry |
| labels | object |  | A key value map of labels to append to every sentry event |
| ethereum.beaconNodeAddress | string |  | [Ethereum consensus client](https://ethereum.org/en/developers/docs/nodes-and-clients/#consensus-clients) http server endpoint |
| ethereum.beaconSubscriptions | array<string> |  | List of [topcis](https://ethereum.github.io/beacon-APIs/#/Events/eventstream) to subscribe to. If empty, all subscriptions are subscribed to.
| ntpServer | string | `pool.ntp.org` | NTP server to calculate clock drift for events |
| outputs | array<object> |  | List of outputs for the sentry to send data to |
| outputs[].name | string |  | Name of the output |
| outputs[].type | string |  | Type of output (`xatu`, `http`, `stdout`) |
| outputs[].config | object |  | Output type configuration [`xatu`](#output-xatu-configuration)/[`http`](#output-http-configuration) |

### Output `xatu` configuration

Output configuration to send sentry events to a [Xatu server](./server.md).

| Name| Type | Default | Description |
| --- | --- | --- | --- |
| outputs[].config.address | string |  | The address of the server receiving events |
| outputs[].config.tls | bool |  | Server requires TLS |
| outputs[].config.headers | object |  | A key value map of headers to append to requests |
| outputs[].config.maxQueueSize | int | `51200` | The maximum queue size to buffer events for delayed processing. If the queue gets full it drops the events |
| outputs[].config.batchTimeout | string | `5s` | The maximum duration for constructing a batch. Processor forcefully sends available events when timeout is reached |
| outputs[].config.exportTimeout | string | `30s` | The maximum duration for exporting events. If the timeout is reached, the export will be cancelled |
| outputs[].config.maxExportBatchSize | int | `512` | MaxExportBatchSize is the maximum number of events to process in a single batch. If there are more than one batch worth of events then it processes multiple batches of events one batch after the other without any delay |
| outputs[].config.connections | int | `1` | Connections is the number of simultaneous connections to xatu to use |

### Output `http` configuration

Output configuration to send sentry events to a http server.

| Name| Type | Default | Description |
| --- | --- | --- | --- |
| outputs[].config.address | string |  | The address of the server receiving events |
| outputs[].config.headers | object |  | A key value map of headers to append to requests |
| outputs[].config.maxQueueSize | int | `51200` | The maximum queue size to buffer events for delayed processing. If the queue gets full it drops the events |
| outputs[].config.batchTimeout | string | `5s` | The maximum duration for constructing a batch. Processor forcefully sends available events when timeout is reached |
| outputs[].config.exportTimeout | string | `30s` | The maximum duration for exporting events. If the timeout is reached, the export will be cancelled |
| outputs[].config.maxExportBatchSize | int | `512` | MaxExportBatchSize is the maximum number of events to process in a single batch. If there are more than one batch worth of events then it processes multiple batches of events one batch after the other without any delay |

### Simple example

```yaml
name: example-instance-001

ethereum:
  beaconNodeAddress: http://localhost:5052

outputs:
- name: standard-out
  type: stdout
```

### Xatu server output example

```yaml
name: example-instance-002

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
name: example-instance-003

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

### Complex example with multiple outputs example

```yaml
logging: "debug"
metricsAddr: ":9090"
pprofAddr: ":6060"

name: example-instance

labels:
  ethpandaops: rocks

ntpServer: time.google.com

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
    connections: 5
```

## Running locally

```bash
# docker
docker run -d --name xatu-sentry -v $HOST_DIR_CHANGE_ME/config.yaml:/opt/xatu/config.yaml -p 9090:9090 -it ethpandaops/xatu:latest sentry --config /opt/xatu/config.yaml
# build
go build -o dist/xatu main.go
./dist/xatu sentry --config sentry.yaml
# dev
go run main.go sentry --config sentry.yaml
```
