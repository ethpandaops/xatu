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
- [Events](#events)
  - [Event field](#event-field)
  - [Meta field](#meta-field)
  - [Beacon API - Event Stream](#beacon-api---event-stream)
    - [Head](#head)
    - [Block](#Block)
    - [Attestation](#Attestation)
    - [Voluntary exit](#voluntary-exit)
    - [Finalized checkpoint](#finalized-checkpoint)
    - [Chain reorg](#chain-reorg)
    - [Contribution and proof](#contribution-and-proof)

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

| Name| Type | Default | Description |
| --- | --- | --- | --- |
| logging | string | `warn` | Log level (`panic`, `fatal`, `warn`, `info`, `debug`, `trace`) |
| metricsAddr | string | `:9090` | The address the metrics server will listen on |
| pprofAddr | string | | The address the [pprof](https://github.com/google/pprof) server will listen on. When ommited, the pprof server will not be started |
| name | string |  | Unique name of the mimicry |
| labels | object |  | A key value map of labels to append to every mimicry event |
| ntpServer | string | `pool.ntp.org` | NTP server to calculate clock drift for events |
| coordinator.type | string |  | Type of output (`xatu`, `static`) |
| coordinator.config | object |  | Coordinator type configuration [`xatu`](#coordinator-xatu-configuration)/[`static`](#coordinator-static-configuration) |
| outputs | array<object> |  | List of outputs for the mimicry to send data to |
| outputs[].name | string |  | Name of the output |
| outputs[].type | string |  | Type of output (`xatu`, `http`, `stdout`) |
| outputs[].config | object |  | Output type configuration [`xatu`](#output-xatu-configuration)/[`http`](#output-http-configuration) |

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

### Simple example

```yaml
name: example-instance-001

coordinator:
  type: xatu
  config:
    address: localhost:8080
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
    forkIdHashes: [0xf0afd0e3]

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

coordinator:
  type: xatu
  config:
    address: localhost:8080
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

## Events

All events output from a **mimicry** contain the following fields;

- `event` - name and date time of the event
- `meta` - metadata of the event, including specific event details as well as mimicry/server computed info
- `data` - raw data collected for the event. This could be the Beacon API payload or a transaction.

An example event payload of a Beacon API event stream for the block topic;

```json
{
  "event": {
    "name": "MEMPOOL_TRANSACTION",
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
          "execution": {
            "fork_id": {
              "hash": "0xf0afd0e3",
              "next": "0"
            }
          }
      },
      "additional_data": {
        "call_data_size":"0",
        "from":"0x1d9e028b88A1638Dfd60D0d8d1476433D9307d8E",
        "gas":"116900",
        "gas_price":"35062538307",
        "hash":"0x26d92491babaf51d5604347fb0dfdad323218bd11f762e34223dddd18db9adcd",
        "size":"111",
        "to":"0x1d9e028b44A1638Dfd60D0a8a1476433D9307e8E",
        "value":"0"
      }
    }
  },
  "data": "0x02f86d01808420d85580850829e3e0438301d8a4941d9e028b88a1638dfd60d0d8d1476433d9307d8e8080d001a0d8ff2f83dd91fa1d0535a8a0d113ede5301065d9b0aa7b77f9b479412deda893a0700a3f95b0456e6ff2e0794e242a058daad1a5774fef9d078dfae3fb61ffa955"
}
```

### Event field

| Name | Type | Required | Description |
| --- | --- | --- | --- |
| event.name | string | `required` | Event name |
| event.date_time | string | `required` | When the event occured |

### Meta field

| Name | Type | Required | Description |
| --- | --- | --- | --- |
| meta.client | object | `required` | Meta data appended from the mimicry client |
| meta.client.name | string | `required` | Mimicry name |
| meta.client.version | string | `required` | Mimicry version |
| meta.client.id | string | `required` | Mimicry ID generated on service start |
| meta.client.implementation | string | `required` | Mimicry implementation eg. Xatu |
| meta.client.os | string | `optional` | Mimicry operating system |
| meta.client.clock_drift | int | `optional` | NTP calculated clock drift of the mimicry |
| meta.client.labels | object |  | A key value map of labels |
| meta.client.ethereum.network.name | string | `required` | Ethereum network name eg. `mainnet`, `sepolia` |
| meta.client.ethereum.execution.fork_id | object | `optional` | ForkID [EIP-2124](https://eips.ethereum.org/EIPS/eip-2124) |
| meta.client.ethereum.execution.fork_id.hash | string | `optional` | IEEE CRC32 checksum of the genesis hash and fork blocks numbers that already passed |
| meta.client.ethereum.execution.fork_id.next | string | `optional` | Block number of the next upcoming fork, or 0 if no next fork is known |
| meta.client.additional_data | object | `optional` | Computed additional data to compliment the events upstream raw data eg. calculating slot start date time |

### Execution layer

When the mimicry is connected to execution layer peer over the [Ethereum Wire Protocol](https://github.com/ethereum/devp2p/blob/master/caps/eth.md), the following events are emitted.

#### Transaction

`meta.event.name` is `MEMPOOL_TRANSACTION`

| Name | Type | Required | Description |
| --- | --- | --- | --- |
| data| string | `required` | Raw transaction data |
| meta.additional_data.size | string | `required` | Size of the transaction in bytes |
| meta.additional_data.call_data_size | string | `required` | Size of the transaction call data in bytes |
| meta.additional_data.hash | string | `required` | Transaction hash |
| meta.additional_data.from | string | `required` | Transaction sender address |
| meta.additional_data.nonce | string | `required` | Transaction nonce |
| meta.additional_data.gas_price | string | `required` | Transaction gas price |
| meta.additional_data.gas | string | `required` | Transaction gas limit |
| meta.additional_data.value | string | `required` | Transaction value |
