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

## Events

All events output from a **sentry** contain the following fields;

- `event` - name and date time of the event
- `meta` - metadata of the event, including specific event details as well as sentry/server computed info
- `data` - raw data collected for the event. This could be the Beacon API payload or a transaction.

An example event payload of a Beacon API event stream for the block topic;

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
    }
  },
  "data": {
    "slot":"10",
    "block":"0x9a2fefd2fdb57f74993c7780ea5b9030d2897b615b89f808011ca5aebed54eaf",
    "execution_optimistic": false
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

### Beacon API - Event Stream

When the sentry is connected to an upstream Ethereum consensus client, the Beacon API [Event stream](https://ethereum.github.io/beacon-APIs/#/Events/eventstream) will be subcribed to and events will be generated for each topic.

#### Head

`meta.event.name` is `BEACON_API_ETH_V1_EVENTS_HEAD`

| Name | Type | Required | Description |
| --- | --- | --- | --- |
| meta.additional_data.slot.start_date_time | string | `required` | Slot start datetime computed from `data.slot` |
| meta.additional_data.propagation.slot_start_diff | string | `required` | Difference in milliseconds between the `meta.event.date_time` and `meta.additional_data.slot.start_date_time` |
| meta.additional_data.epoch.number | string | `required` | Epoch number computed from `data.slot` |
| meta.additional_data.epoch.start_date_time | string | `required` | Epoch start datetime computed from `meta.additional_data.epoch.number` |
| data.slot | string | `required` |  |
| data.block | string | `required` |  |
| data.state | string | `required` |  |
| data.epoch_transition | string | `required` |  |
| data.previous_duty_dependent_root | string | `required` |  |
| data.current_duty_dependent_root | string | `required` |  |

#### Block

`meta.event.name` is `BEACON_API_ETH_V1_EVENTS_BLOCK`

| Name | Type | Required | Description |
| --- | --- | --- | --- |
| meta.additional_data.slot.start_date_time | string | `required` | Slot start datetime computed from `data.slot` |
| meta.additional_data.propagation.slot_start_diff | string | `required` | Difference in milliseconds between the `meta.event.date_time` and `meta.additional_data.slot.start_date_time` |
| meta.additional_data.epoch.number | string | `required` | Epoch number computed from `data.slot` |
| meta.additional_data.epoch.start_date_time | string | `required` | Epoch start datetime computed from `meta.additional_data.epoch.number` |
| data.slot | string | `required` |  |
| data.block | string | `required` |  |
| data.execution_optimistic | bool | `required` |  |

#### Attestation

`meta.event.name` is `BEACON_API_ETH_V1_EVENTS_ATTESTATION`

| Name | Type | Required | Description |
| --- | --- | --- | --- |
| meta.additional_data.slot.start_date_time | string | `required` | Slot start datetime computed from `data.data.slot` |
| meta.additional_data.propagation.slot_start_diff | string | `required` | Difference in milliseconds between the `meta.event.date_time` and `meta.additional_data.slot.start_date_time` |
| meta.additional_data.epoch.number | string | `required` | Epoch number computed from `data.data.slot` |
| meta.additional_data.epoch.start_date_time | string | `required` | Epoch start datetime computed from `meta.additional_data.epoch.number` |
| meta.additional_data.source.epoch.start_date_time | string | `required` | Epoch start datetime computed from `data.data.source.epoch` |
| meta.additexecutablesional_data.target.epoch.start_date_time | string | `required` | Epoch start datetime computed from `data.data.target.epoch` |
| data.aggregation_bits | string | `required` |  |
| data.data.beacon_block_root | string | `required` |  |
| data.data.index | string | `required` |  |
| data.data.slot | string | `required` |  |
| data.data.source.epoch | string | `required` |  |
| data.data.source.root | string | `required` |  |
| data.data.target.epoch | string | `required` |  |
| data.data.target.root | string | `required` |  |
| data.signature | string | `required` |  |

#### Voluntary exit

`meta.event.name` is `BEACON_API_ETH_V1_EVENTS_VOLUNTARY_EXIT`

| Name | Type | Required | Description |
| --- | --- | --- | --- |
| meta.additional_data.epoch.start_date_time | string | `required` | Epoch start datetime computed from `data.message.epoch` |
| data.message.epoch | string | `required` |  |
| data.message.validator_index | string | `required` |  |
| data.signature | string | `required` |  |

#### Finalized checkpoint

`meta.event.name` is `BEACON_API_ETH_V1_EVENTS_FINALIZED_CHECKPOINT`

| Name | Type | Required | Description |
| --- | --- | --- | --- |
| meta.additional_data.epoch.start_date_time | string | `required` | Epoch start datetime computed from `data.epoch` |
| data.block | string | `required` |  |
| data.epoch | string | `required` |  |
| data.execution_optimistic | bool | `required` |  |
| data.state | string | `required` |  |

#### Chain reorg

`meta.event.name` is `BEACON_API_ETH_V1_EVENTS_CHAIN_REORG`

| Name | Type | Required | Description |
| --- | --- | --- | --- |
| meta.additional_data.slot.start_date_time | string | `required` | Slot start datetime computed from `data.slot` |
| meta.additional_data.propagation.slot_start_diff | string | `required` | Difference in milliseconds between the `meta.event.date_time` and `meta.additional_data.slot.start_date_time` |
| meta.additional_data.epoch.number | string | `required` | Epoch number computed from `data.slot` |
| meta.additional_data.epoch.start_date_time | string | `required` | Epoch start datetime computed from `meta.additional_data.epoch.number` |
| data.depth | string | `required` |  |
| data.epoch | string | `required` |  |
| data.execution_optimistic | bool | `required` |  |
| data.new_head_block | string | `required` |  |
| data.new_head_state | string | `required` |  |
| data.old_head_block | string | `required` |  |
| data.old_head_state | string | `required` |  |
| data.slot | string | `required` |  |

#### Contribution and proof

`meta.event.name` is `BEACON_API_ETH_V1_EVENTS_CONTRIBUTION_AND_PROOF`

| Name | Type | Required | Description |
| --- | --- | --- | --- |
| meta.additional_data.contribution.slot.start_date_time | string | `required` | Slot start datetime computed from `data.message.contribution.slot` |
| meta.additional_data.contribution.propagation.slot_start_diff | string | `required` | Difference in milliseconds between the `meta.event.date_time` and `meta.additional_data.slot.start_date_time` |
| meta.additional_data.contribution.epoch.number | string | `required` | Epoch number computed from `data.message.contribution.slot` |
| meta.additional_data.contribution.epoch.start_date_time | string | `required` | Epoch start datetime computed from `meta.additional_data.epoch.number` |
| data.message.aggregator_index | string | `required` |  |
| data.message.contribution.aggregation_bits | string | `required` |  |
| data.message.contribution.beacon_block_root | string | `required` |  |
| data.message.contribution.signature | string | `required` |  |
| data.message.contribution.slot | string | `required` |  |
| data.message.contribution.subcommittee_index | string | `required` |  |
| data.message.selection_proof | string | `required` |  |
| data.signature | string | `required` |  |
