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
| name | string |  | Unique name of the mimicry |
| labels | object |  | A key value map of labels to append to every mimicry event |
| ntp_server | string | `pool.ntp.org` | NTP server to calculate clock drift for events |
| coordinator.type | string |  | Type of output (`xatu`, `manual`) |
| coordinator.config | object |  | Output type configuration [`xatu`](#coordinator-xatu-configuration)/[`http`](#coordinator-manual-configuration) |
| outputs | array<object> |  | List of outputs for the mimicry to send data to |
| outputs[].name | string |  | Name of the output |
| outputs[].type | string |  | Type of output (`xatu`, `http`, `stdout`) |
| outputs[].config | object |  | Output type configuration [`xatu`](#output-xatu-configuration)/[`http`](#output-http-configuration) |

### Coordinator `xatu` configuration

Coordinator configuration to get peers to connect to from the [Xatu server coordinator](./server.md#coordinator).

| Name| Type | Default | Description |
| --- | --- | --- | --- |
| coordinator.config.address | string |  | The address of the server receiving events |
| coordinator.config.headers | object |  | A key value map of headers to append to requests |
| coordinator.config.max_queue_size | int | `51200` | The maximum queue size to buffer events for delayed processing. If the queue gets full it drops the events |
| coordinator.config.batch_timeout | string | `5s` | The maximum duration for constructing a batch. Processor forcefully sends available events when timeout is reached |
| coordinator.config.export_timeout | string | `30s` | The maximum duration for exporting events. If the timeout is reached, the export will be cancelled |
| coordinator.config.max_export_batch_size | int | `512` | MaxExportBatchSize is the maximum number of events to process in a single batch. If there are more than one batch worth of events then it processes multiple batches of events one batch after the other without any delay |

### Coordinator `manual` configuration

Coordinator configuration to manually specify peers to connect to.

| Name| Type | Default | Description |
| --- | --- | --- | --- |
| coordinator.config.retry_interval | string |  | Interval between trying to connect to a peer |
| coordinator.config.node_records | array<string> |  | List of ENR/ENode peers to connect to |

### Output `xatu` configuration

Output configuration to send mimicry events to a [Xatu server](./server.md).

| Name| Type | Default | Description |
| --- | --- | --- | --- |
| outputs[].config.address | string |  | The address of the server receiving events |
| outputs[].config.headers | object |  | A key value map of headers to append to requests |
| outputs[].config.max_queue_size | int | `51200` | The maximum queue size to buffer events for delayed processing. If the queue gets full it drops the events |
| outputs[].config.batch_timeout | string | `5s` | The maximum duration for constructing a batch. Processor forcefully sends available events when timeout is reached |
| outputs[].config.export_timeout | string | `30s` | The maximum duration for exporting events. If the timeout is reached, the export will be cancelled |
| outputs[].config.max_export_batch_size | int | `512` | MaxExportBatchSize is the maximum number of events to process in a single batch. If there are more than one batch worth of events then it processes multiple batches of events one batch after the other without any delay |

### Output `http` configuration

Output configuration to send mimicry events to a http server.

| Name| Type | Default | Description |
| --- | --- | --- | --- |
| outputs[].config.address | string |  | The address of the server receiving events |
| outputs[].config.headers | object |  | A key value map of headers to append to requests |
| outputs[].config.max_queue_size | int | `51200` | The maximum queue size to buffer events for delayed processing. If the queue gets full it drops the events |
| outputs[].config.batch_timeout | string | `5s` | The maximum duration for constructing a batch. Processor forcefully sends available events when timeout is reached |
| outputs[].config.export_timeout | string | `30s` | The maximum duration for exporting events. If the timeout is reached, the export will be cancelled |
| outputs[].config.max_export_batch_size | int | `512` | MaxExportBatchSize is the maximum number of events to process in a single batch. If there are more than one batch worth of events then it processes multiple batches of events one batch after the other without any delay |

### Simple example

```yaml
name: example-instance-001

coordinator:
  type: xatu
  config:
    address: localhost:8080

outputs:
- name: standard-out
  type: stdout
```

### Simple example with manual coordinator

```yaml
name: example-instance-001

coordinator:
  type: manual
  config:
    retry_interval: 60s
    node_records:
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

outputs:
- name: http-basic-auth
  type: http
  config:
    address: http://localhost:8080
    headers:
      Authorization: "Basic Someb64Value"
```

### Complex example with multiple outputs example

```yaml
logging: "debug"
metricsAddr: ":9090"

name: example-instance

labels:
  ethpandaops: rocks

ntp_server: time.google.com

coordinator:
  type: xatu
  config:
    address: localhost:8080

outputs:
- name: log
  type: stdout
- name: xatu-server
  type: xatu
  config:
    address: localhost:8080
    headers:
      Authorization: Someb64Value
    max_queue_size: 51200
    batch_timeout: 5s
    export_timeout: 30s
    max_export_batch_size: 512
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
    "name": "EXECUTION_TRANSACTION",
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
            "implementation": "Geth",
            "version": "v1.1.1-stable-e5eb32ac",
            "fork_id": {
              "hash": "0xb96cbd13",
              "next": "0"
            },
            "node_record": "enr:-IS4QHCYrYZbAKWCBRlAy5zzaDZXJBGkcnh4MHcBFZntXNFrdvJjX04jRzjzCBOonrkTfj499SZuOh8R33Ls8RRcy5wBgmlkgnY0gmlwhH8AAAGJc2VjcDI1NmsxoQPKY0yuDUmstAHYpMa2_oxVtw0RW_QAdpzBQA8yWM0xOIN1ZHCCdl8"
          }
      },
      "additional_data": {
        "size": "15658",
        "call_data_size": "15567"
      }
    }
  },
  "data": {
    "hash": "0xbcd265d4623d211ad203456e18e3af8f67c5efefd0d97911a3f7f93790d53f2b",
    "from": "0x9beaCC6511ca46b1fe78309b237dCa259Ba5BC1d",
    "nonce": "1020",
    "gas_price": "2500000007",
    "gas": "3215885",
    "value": "0"
  }
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
| meta.client.ethereum.execution.implementation | string | `optional` | Ethereum consensus client name upstream from this mimicry |
| meta.client.ethereum.execution.version | string | `optional` | Ethereum consensus client version upstream from this mimicry |
| meta.client.ethereum.execution.fork_id | object | `optional` | ForkID [EIP-2124](https://eips.ethereum.org/EIPS/eip-2124) |
| meta.client.ethereum.execution.fork_id.hash | string | `optional` | IEEE CRC32 checksum of the genesis hash and fork blocks numbers that already passed |
| meta.client.ethereum.execution.fork_id.next | string | `optional` | Block number of the next upcoming fork, or 0 if no next fork is known |
| meta.client.ethereum.execution.node_record | string | `optional` | [ENR](https://eips.ethereum.org/EIPS/eip-778) or `ENode` record of the peer |
| meta.client.additional_data | object | `optional` | Computed additional data to compliment the events upstream raw data eg. calculating slot start date time |

### Execution layer

When the mimicry is connected to execution layer peer over the [Ethereum Wire Protocol](https://github.com/ethereum/devp2p/blob/master/caps/eth.md), the following events are emitted.

#### Transaction

`meta.event.name` is `EXECUTION_TRANSACTION`

| Name | Type | Required | Description |
| --- | --- | --- | --- |
| meta.additional_data.size | string | `required` | Size of the transaction in bytes |
| meta.additional_data.call_data_size | string | `required` | Size of the transaction call data in bytes |
| data.hash | string | `required` | Transaction hash |
| data.from | string | `required` | Transaction sender address |
| data.nonce | string | `required` | Transaction nonce |
| data.gas_price | string | `required` | Transaction gas price |
| data.gas | string | `required` | Transaction gas limit |
| data.value | string | `required` | Transaction value |
