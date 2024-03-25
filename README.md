# Xatu

Ethereum network monitoring with collection clients and a centralized server for data pipelining.

## Overview

Xatu can run in multiple modes. Each mode can be run independently. The following diagram shows the different modes and how they interact with each other and other services.

```

             ┌───────────┐
             │ CONSENSUS │
             │P2P NETWORK│
             └─────▲─────┘
                   │
      ┌────────────┘─────────────┐
      │                          │
┌─────▲─────┐              ┌─────▲─────┐       ┌───────────┐
│ CONSENSUS │              │ ARMIARMA  │       │ EXECUTION │
│   CLIENT  ◄─────┐        │           │       │P2P NETWORK│
└─────▲─────┘     │        └─────▲─────┘       └─────▲─────┘
      │           │              │             ┌─────┘───────┐
      │           │              │             │             │
  ┌───▼────┐ ┌────▼─────┐  ┌─────▼─────┐ ┌─────▼─────┐ ┌─────▼─────┐
  │  XATU  │ │   XATU   │  │   XATU    │ │   XATU    │ │   XATU    │
  │ SENTRY │ │  CANNON  │  │   SAGE    │ │  MIMICRY  │ │ DISCOVERY │
  └───┬────┘ └─────┬────┘  └─────┬─────┘ └─────┬─────┘ └─────┬─────┘
      │            │             │             │             │
      │            │             │             │             │
      │       ┌────▼─────┐       │             │             │
      └───────►          ◄───────┘─────────────┘─────────────┘
              │   XATU   │
              │  SERVER  │    ┌─────────────┐
              │          ◄────► PERSISTENCE │
              │          │    └─────────────┘
              └─────┬────┘
                    │
                    │
                    ▼
              DATA PIPELINE
```

### Modes

Follow the links for more information on each mode.

- [**Server**](./docs/server.md) - Centralized server collecting events from various clients and can output them to various sinks.
- [**Sentry**](./docs/sentry.md) - Client that runs along side a [Ethereum consensus client](https://ethereum.org/en/developers/docs/nodes-and-clients/#consensus-clients) and collects data via the consensus client's [Beacon API](https://ethereum.github.io/beacon-APIs/). _You must run your own consensus client_ and this projects sentry will connect to it via the consensus client's http server.
- [**Discovery**](./docs/discovery.md) - Client that uses the [Node Discovery Protocol v5](https://github.com/ethereum/devp2p/blob/master/discv5/discv5.md) and [Node Discovery Protocol v4](https://github.com/ethereum/devp2p/blob/master/discv4.md) to discovery nodes on the network. Also attempts to connect to execution layer nodes and collect meta data from them.
- [**Mimicry**](./docs/mimicry.md) - Client that collects data from the execution layer P2P network.
- [**Cannon**](./docs/cannon.md) - Client that runs along side a [Ethereum consensus client](https://ethereum.org/en/developers/docs/nodes-and-clients/#consensus-clients) and collects canonical finalized data via the consensus client's [Beacon API](https://ethereum.github.io/beacon-APIs/). _You must run your own consensus client_ and this projects cannon client will connect to it via the consensus client's http server.
- [**Sage**](./docs/sage.md) - Client that connects to an [Armiarma](https://github.com/migalabs/armiarma) instance (which itself connects to an Ethereum Beacon Chain P2P network) and creates events from the events Armiarma emits.

## Getting Started

### Download a release

Download the latest release from the [Releases page](https://github.com/ethpandaops/xatu/releases). Extract and run with:

```
./xatu <server|sentry|discovery|mimicry> --config your-config.yaml
```

### Docker

Available as a docker image at [ethpandaops/xatu](https://hub.docker.com/r/ethpandaops/xatu/tags)

#### Images

- `latest` - distroless, multiarch
- `latest-debian` - debian, multiarch
- `$version` - distroless, multiarch, pinned to a release (i.e. `0.4.0`)
- `$version-debian` - debian, multiarch, pinned to a release (i.e. `0.4.0-debian`)

### Kubernetes via Helm

[Read more](https://github.com/ethpandaops/ethereum-helm-charts/tree/master/charts/xatu)

```
helm repo add ethereum-helm-charts https://ethpandaops.github.io/ethereum-helm-charts

helm install xatu ethereum-helm-charts/xatu -f your_values.yaml
```

### Locally via docker compose

```bash
docker compose up --detach
```

This will setup a pipeline to import events from Xatu server into a clickhouse instance. This will **not** run `sentry`/`discovery`/`mimicry`/`cannon`/`sage` clients but allow you run them locally and connect to the server.

There is also a grafana instance running with dashboards that can be used to visualize the data.

Exposed ports:

- `8080` - Xatu server
- `9000` - Clickhouse native port
- `8123` - Clickhouse http port
- `3000` - Grafana

Links:

- [Clickhouse playground](http://localhost:8123/play)
- [Grafana](http://localhost:3000)

Example sentry config to connect to the server:

```yaml
logging: "info"
metricsAddr: ":9095"

name: example-instance

labels:
  ethpandaops: rocks

ntpServer: time.google.com

ethereum:
  # connect to your own consensus client
  beaconNodeAddress: http://localhost:5052

forkChoice:
  enabled: false

attestationData:
  enabled: false

beaconCommittees:
  enabled: false

outputs:
  - name: xatu
    type: xatu
    config:
      address: localhost:8080
      tls: false
      maxQueueSize: 51200
      batchTimeout: 5s
      exportTimeout: 30s
      maxExportBatchSize: 5
      connections: 3
```

### Local clickhouse

You can start up the clickhouse cluster only with migrations automatically applied. You might want to do this to play with out [Xatu data](https://github.com/ethpandaops/xatu-data) locally.

```bash
docker compose --profile clickhouse up --detach
```

## Contributing

Contributions are greatly appreciated! Pull requests will be reviewed and merged promptly if you're interested in improving Xatu!

1. Fork the project
2. Create your feature branch:
   - `git checkout -b feat/new-output`
3. Commit your changes:
   - `git commit -m 'feat(sentry): new output`
4. Push to the branch: -`git push origin feat/new-output`
5. Open a pull request

## Contact

Sam - [@samcmau](https://twitter.com/samcmau)

Andrew - [@savid](https://twitter.com/Savid)
