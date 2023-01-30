# Discovery

Client that uses [go-ethereum's](https://github.com/ethereum/go-ethereum/p2p/discover) implementation of [Node Discovery Protocol v5](https://github.com/ethereum/devp2p/blob/master/discv5/discv5.md) and [Node Discovery Protocol v4](https://github.com/ethereum/devp2p/blob/master/discv4.md) to discover ethereum nodes around the network and send them to a [xatu server](./server.md).

The client also requests a list of execution layer node records from the [xatu server](./server.md) and tries to connect. If the connection is successful it sends data back to the [xatu server](./server.md) with a summary of the [RLPx hello](https://github.com/ethereum/devp2p/blob/master/rlpx.md#hello-0x00) and [Etherum Wire Protocol status](https://github.com/ethereum/devp2p/blob/master/caps/eth.md#status-0x00) responses from the peer.

## Table of contents

- [Usage](#usage)
- [Requirements](#requirements)
- [Configuration](#configuration)
  - [Simple example](#simple-example)
- [Running locally](#running-locally)

## Usage

Discovery requires a [config file](#configuration).

```bash
Usage:
  xatu discovery [flags]

Flags:
      --config string   config file (default is discovery.yaml) (default "discovery.yaml")
  -h, --help            help for discovery
```

## Requirements

At least one [Server](./server.md) running with the [Coordinator](./server.md#coordinator) service enabled.

## Configuration

Discovery requires a single `yaml` config file. An example file can be found [here](../example_discovery.yaml)

| Name| Type | Default | Description |
| --- | --- | --- | --- |
| logging | string | `warn` | Log level (`panic`, `fatal`, `warn`, `info`, `debug`, `trace`) |
| metricsAddr | string | `:9090` | The address the metrics server will listen on |
| coordinator | object |  | [Coordinator](./server.md#coordinator) configuration |
| coordinator.address | string |  | The address of the [xatu server](./server.md) |
| coordinator.max_queue_size | int | `51200` | The maximum queue size to buffer node records for delayed processing. If the queue gets full it drops the node records |
| coordinator.batch_timeout | string | `5s` | The maximum duration for constructing a batch. Processor forcefully sends available node records when timeout is reached |
| coordinator.export_timeout | string | `30s` | The maximum duration for exporting node records. If the timeout is reached, the export will be cancelled |
| coordinator.max_export_batch_size | int | `512` | MaxExportBatchSize is the maximum number of node records to process in a single batch. If there are more than one batch worth of node records then it processes multiple batches of node records one batch after the other without any delay |
| p2p | object |  | peer to peer configuration |
| p2p.discV4 | bool | `true` | enable Node Discovery Protocol v4 *Note: both Node Discovery Protocol v4 and v5 can be enabled at the same time* |
| p2p.discV5 | bool | `true` | enable Node Discovery Protocol v5 *Note: both Node Discovery Protocol v4 and v5 can be enabled at the same time* |
| p2p.restart | string | `2m` | Time between initiating discovery scans, will generate a fresh private key each time |
| p2p.boot_nodes | array<string> |  | List of boot nodes to connect to |

### Simple Example

```yaml
logging: info
metricsAddr: :9090

coordinator:
  address: localhost:8080

p2p:
  discV4: true
  discV5: true
  restart: 2m
  boot_nodes:
    - enr:-Jq4QItoFUuug_n_qbYbU0OY04-np2wT8rUCauOOXNi0H3BWbDj-zbfZb7otA7jZ6flbBpx1LNZK2TDebZ9dEKx84LYBhGV0aDKQtTA_KgEAAAD__________4JpZIJ2NIJpcISsaa0ZiXNlY3AyNTZrMaEDHAD2JKYevx89W0CcFJFiskdcEzkH_Wdv9iW42qLK79ODdWRwgiMo
```

## Running locally

```bash
# docker
docker run -d --name xatu-discovery -v $HOST_DIR_CHANGE_ME/config.yaml:/opt/xatu/config.yaml -p 9090:9090 -it ethpandaops/xatu:latest discovery --config /opt/xatu/config.yaml
# build
go build -o dist/xatu main.go
./dist/xatu discovery --config discovery.yaml
# dev
go run main.go discovery --config discovery.yaml
```