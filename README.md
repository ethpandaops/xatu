# Xatu

Ethereum network monitoring with a sentry client and a centralized server for event collection and streaming.

## Table of contents

- [Overview](#overview)
- [Sentry](#sentry)
  - [Usage](#usage)
  - [Configuration](#configuration)
    - [Simple Example](#simple-example)
    - [Authorization header example](#authorization-header-example)
  - [Getting Started](#getting-started)
    - [Download a release](#download-a-release)
    - [Docker](#docker)
      - [Images](#images)
    - [Kubernetes via Helm](#kubernetes-via-helm)
- [Server](#server)
- [Events](#events)
  - [Meta](#meta)
  - [Beacon API - Event Stream](#beacon-api---event-stream)
    - [Head](#head)
    - [Block](#Block)
    - [Attestation](#Attestation)
    - [Voluntary exit](#voluntary-exit)
    - [Finalized checkpoint](#finalized-checkpoint)
    - [Chain reorg](#chain-reorg)
    - [Contribution and proof](#contribution-and-proof)
- [Contributing](#contributing)
- [Contact](#contact)

## Overview

```
┌───────────┐   ┌───────────┐   ┌───────────┐
│ CONSENSUS │   │ CONSENSUS │   │ CONSENSUS │
│   CLIENT  │   │   CLIENT  │   │   CLIENT  │
└─────▲┌────┘   └─────▲┌────┘   └─────▲┌────┘
      ││              ││              ││
      ││              ││              ││
  ┌───┘▼───┐      ┌───┘▼───┐      ┌───┘▼───┐
  │  XATU  │      │  XATU  │      │  XATU  │
  │ SENTRY │      │ SENTRY │      │ SENTRY │
  └────┬───┘      └────┬───┘      └───┬────┘
       │               │              │
       │               │              │
       │          ┌────▼───┐          │
       └──────────►  XATU  ◄──────────┘
                  │ SERVER │
                  └────┬───┘
                       │
                       │
                       ▼
                  Data pipeline
```

> **Xatu Server** is currently not implemented

## Sentry

Sentries are run along side a [Ethereum consensus client](https://ethereum.org/en/developers/docs/nodes-and-clients/#consensus-clients) and collect data via the consensus client's [Beacon API](https://ethereum.github.io/beacon-APIs/). *You must run your own consensus client* and this projects sentry will connect to it via the consensus client's http server.

### Usage

Sentry requires a [config file](#configuration).

```bash
Usage:
  xatu senty [flags]

Flags:
      --config string   config file (default is sentry.yaml) (default "sentry.yaml")
  -h, --help            help for sentry
```

### Configuration

Sentry relies entirely on a single `yaml` config file. An example file can be found [here](./example_sentry.yaml)

| Name| Type | Default | Description |
| --- | --- | --- | --- |
| logging | string | `warn` | Log level (`panic`, `fatal`, `warn`, `info`, `debug`, `trace`) |
| metricsAddr | string | `:9090` | The address the metrics server will listen on |
| name | string |  | Unique name of the sentry |
| labels | object |  | A key value map of labels to append to every sentry event |
| ethereum.beacon_node_address | string |  | [Ethereum consensus client](https://ethereum.org/en/developers/docs/nodes-and-clients/#consensus-clients) http server endpoint |
| ntp_server | string | `pool.ntp.org` | NTP server to calculate clock drift for events |
| outputs | [] |  | List of outputs for the sentry to send data to |
| outputs[].name | string |  | Name of the output |
| outputs[].type | string |  | Type of out (`http`) |
| outputs[].config.address | string |  | The address of the server receiving events |
| outputs[].config.headers | object |  | A key value map of headers to append to requests |
| outputs[].config.max_queue_size | int | `51200` | The maximum queue size to buffer events for delayed processing. If the queue gets full it drops the events |
| outputs[].config.batch_timeout | string | `5s` | The maximum duration for constructing a batch. Processor forcefully sends available events when timeout is reached |
| outputs[].config.export_timeout | string | `30s` | The maximum duration for exporting events. If the timeout is reached, the export will be cancelled |
| outputs[].config.max_export_batch_size | int | `512` | MaxExportBatchSize is the maximum number of events to process in a single batch. If there are more than one batch worth of events then it processes multiple batches of events one batch after the other without any delay |

#### Simple example

```yaml
name: example-instance-001

ethereum:
  beacon_node_address: http://localhost:5052

outputs:
- name: basic
  type: http
  config:
    address: http://localhost:8080
```

#### Authorization header example

```yaml
name: example-instance-002

ethereum:
  beacon_node_address: http://localhost:5052

outputs:
- name: basic
  type: http
  config:
    address: http://localhost:8080
    headers:
      Authorization: "Basic Someb64Value"
```
### Getting Started

#### Download a release
Download the latest release from the [Releases page](https://github.com/ethpandaops/xatu/releases). Extract and run with:
```
./xatu sentry --config your-config.yaml
```

#### Docker
Available as a docker image at [ethpandaops/xatu](https://hub.docker.com/r/ethpandaops/xatu/tags)
##### Images
- `latest` - distroless, multiarch
- `latest-debian` - debian, multiarch
- `$version` - distroless, multiarch, pinned to a release (i.e. `0.4.0`)
- `$version-debian` - debian, multiarch, pinned to a release (i.e. `0.4.0-debian`)

**Quick start**
```
docker run -d  --name xatu -v $HOST_DIR_CHANGE_ME/config.yaml:/opt/xatu/config.yaml -p 9090:9090 -it ethpandaops/xatu:latest sentry --config /opt/xatu/config.yaml;
docker logs -f xatu;
```

#### Kubernetes via Helm
[Read more](https://github.com/ethpandaops/ethereum-helm-charts/tree/master/charts/xatu)
```
helm repo add ethereum-helm-charts https://ethpandaops.github.io/ethereum-helm-charts

helm install xatu ethereum-helm-charts/xatu -f your_values.yaml
```

## Server

> **Xatu Server** is currently not implemented

The goals of the server;

- Add additional data to the event;
  - GeoIP
  - Client identifcation data
  - Other meta data
- OAuth for client authentication/authorization
- Event data validation
- Metrics
- Data sink options eg. http/socket

## Events

All events output from a sentry contain **two** root fields; `meta` and `data`.

- `meta` being the metadata of the event, including specific event details as well as sentry info
- `data` being the raw data collected for the event. This could be the Beacon API payload or a transaction.

An example event payload of a Beacon API event stream for the block topic;

```json
{
  "meta": {
    "client": {
      "name": "some-client-001",
      "version": "v1.0.1",
      "id": "0697583c-3c65-4f9a-bcd0-b57ef919dc6c",
      "os": "amiga4000-68040",
      "clock_drift": 5,
      "event": {
        "name": "BEACON_API_ETH_V1_EVENTS_BLOCK",
        "date_time": "2022-01-01T10:12:10.050Z"
      }
      "labels": {
        "network-class": "tincan"
      },
      "ethereum": {
        "network": {
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

### Meta

| Name | Type | Required | Description |
| --- | --- | --- | --- |
| meta.client | object | `required` | Meta data appended from the sentry client |
| meta.name | string | `required` | Sentry name |
| meta.version | string | `required` | Sentry version |
| meta.id | string | `required` | Sentry ID generated on service start |
| meta.implementation | string | `required` | Sentry implementation eg. Xatu |
| meta.os | string | `optional` | Sentry operating system |
| meta.clock_drift | int | `optional` | NTP calculated clock drift of the sentry |
| meta.event.name | string | `required` | Event name |
| meta.event.date_time | string | `required` | When the event occured |
| meta.labels | object |  | A key value map of labels |
| meta.ethereum.network.name | string | `required` | Ethereum network name eg. `mainnet`, `sepolia` |
| meta.ethereum.consensus.implementation | string | `optional` | Ethereum consensus client name upstream from this sentry |
| meta.ethereum.consensus.version | string | `optional` | Ethereum consensus client version upstream from this sentry |
| meta.additional_data | object | `optional` | Computed additional data to compliment the events upstream raw data eg. calculating slot start date time |

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
| meta.additional_data.target.epoch.start_date_time | string | `required` | Epoch start datetime computed from `data.data.target.epoch` |
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

## Contributing

Contributions are greatly appreciated! Pull requests will be reviewed and merged promptly if you're interested in improving Xatu! 

1. Fork the project
2. Create your feature branch:
    - `git checkout -b feat/new-output`
3. Commit your changes:
    - `git commit -m 'feat(sentry): new output`
4. Push to the branch:
    -`git push origin feat/new-output`
5. Open a pull request

### Running locally

#### Sentry
```
go run main.go sentry --config sentry.yaml
```

## Contact

Sam - [@samcmau](https://twitter.com/samcmau)

Andrew - [@savid](https://twitter.com/Savid)
