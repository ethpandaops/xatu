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
| ethereum.beacon.blockCacheSize | int | `1000` | The maximum number of blocks to cache                                                                                                      |
| ethereum.beacon.blockCacheTtl | string | `1h` | The maximum duration to cache blocks                                                                                                       |
| ethereum.beacon.blockPreloadWorkers | int | `5` | The number of workers to use for preloading blocks                                                                                         |
| ethereum.beacon.blockPreloadQueueSize | int | `5000` | The maximum number of blocks to queue for preloading                                                                                       |
| ethereum.blockArchive.enabled | bool | `false` | Serve finalized beacon blocks from standard CL era files and Gloas execution payload envelopes from per-era `<era basename>.envelopes.e2s` sidecars. Each fetched window must link block to block by parent root and match the beacon node's block root and signature at one slot (one header request per window, held off for 10s after a failed request). The beacon node still answers slots no listed era covers (including the current era), missed slots at an era's edges that no block in the file proves, envelopes of eras without a verified sidecar, withheld payloads no later block in the era proves, and anything that fails |
| ethereum.blockArchive.url | string |  | The directory serving the era files (fetched as `<url>/<file>`), e.g. `https://mainnet.era.nimbus.team` or block-archiver's `https://<archive>/era/<network>` |
| ethereum.blockArchive.listingUrl | string |  | The listing of era files and sidecars. Any HTML directory index or JSON naming the files works, such as block-archiver's `https://<archive>/api/v1/era?network=<network>`, and a paged JSON listing (`eras`, `limit`, `offset`) is read page by page; other paged indexes, such as S3 XML listings, are read as their first page only. Defaults to `url`; set it for block-archiver, whose `/era/<network>/` page lists only its newest eras |
| ethereum.blockArchive.network | string |  | The network name era files are named with; defaults to the beacon node's network |
| ethereum.blockArchive.headers | object |  | A key value map of headers to send to the archive |
| ethereum.blockArchive.timeout | string | `5m` | Timeout for one fetch from the archive, including a whole era's block section |
| ethereum.blockArchive.listingTtl | string | `5m` | How long the listing, archive failures, and files or windows that failed to verify are trusted |
| ethereum.blockArchive.maxConcurrentFetches | int | `2` | The maximum number of parallel archive fetches |
| ethereum.blockArchive.rangeSlots | int | `256` | Fetch aligned windows of this many slots (at most 8192) with HTTP Range requests; `0` reads each era's whole block section in one request. On mainnet a 256-slot window is about 25MB to download and 70MB decoded, while `0` downloads about 600MB per era, peaks near 3GiB of memory per in-flight fetch (up to `maxConcurrentFetches` at once) and keeps about 1.4GiB of decoded blocks, since the block cache grows to hold one fetch |
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
| derivers.consensus.phase0Reward.enabled | bool | `false` | Enable the phase0 reward deriver: per-validator attestation rewards (head, target, source, inclusion delay, inactivity) and per-block proposer rewards for the epochs before Altair, computed from the beacon state at the last slot of each epoch with the consensus spec's phase0 rules. Downloads one full state per epoch; the head stays at the epoch before Altair, so enable backfill to cover the rest |
| derivers.consensus.phase0Reward.workers | int | `1` | How many beacon states, and how many epoch writes, a phase0 reward backfill batch has in flight at once (at most 16). Each holds a full state while it is decoded. Only helps with `iterator.backfill.epochsPerBatch` above 1 |
| derivers.consensus.beaconBlockGraffiti.enabled | bool | `true` | Enable the beacon block graffiti deriver (proposer index and raw graffiti, one row per block, from phase0) |
| derivers.consensus.beaconStatePtcMember.enabled | bool | `true` | Enable the Gloas payload timeliness committee deriver (ordered committee, one row per slot and position). Activates at Gloas and shares one beacon state download per epoch with the other Gloas state derivers |
| derivers.consensus.beaconStateBuilder.enabled | bool | `true` | Enable the Gloas builder registry snapshot deriver (one row per builder per epoch). Activates at Gloas and shares one beacon state download per epoch with the other Gloas state derivers |
| derivers.consensus.beaconStateBuilderPendingPayment.enabled | bool | `true` | Enable the Gloas builder pending payments snapshot deriver (non-empty entries only). Activates at Gloas and shares one beacon state download per epoch with the other Gloas state derivers |
| derivers.consensus.beaconStateBuilderPendingWithdrawal.enabled | bool | `true` | Enable the Gloas builder pending withdrawals queue snapshot deriver. Activates at Gloas and shares one beacon state download per epoch with the other Gloas state derivers |
| derivers.consensus.beaconStateExecutionPayloadAvailability.enabled | bool | `true` | Enable the Gloas execution payload availability deriver (one row per slot, two epochs behind). Activates at Gloas and shares one beacon state download per epoch with the other Gloas state derivers |
| derivers.proposerSlashing.enabled | bool | `true` | Enable the proposer slashing deriver                                                                                                       |
| derivers.proposerSlashing.headSlotLag | int | `5` | The number of slots to lag behind the head                                                                                                 |
| derivers.voluntaryExit.enabled | bool | `true` | Enable the voluntary exit deriver                                                                                                          |
| derivers.voluntaryExit.headSlotLag | int | `5` | The number of slots to lag behind the head                                                                                                 |
| derivers.consensus.executionRequestBuilderDeposit.enabled | bool | `true` | Enable the EIP-8282 builder deposit request deriver (activates at Gloas)                                                                   |
| derivers.consensus.executionRequestBuilderExit.enabled | bool | `true` | Enable the EIP-8282 builder exit request deriver (activates at Gloas)                                                                      |
| derivers.consensus.blockAccessListSummary.enabled | bool | `true` | Enable the per-block EIP-7928 block access list summary deriver (activates at Gloas)                                                       |
| derivers.consensus.&lt;deriver&gt;.iterator.backfill.enabled | bool | `false` | Also walk the deriver's history downward from where it started, once the head is caught up |
| derivers.consensus.&lt;deriver&gt;.iterator.backfill.toEpoch | int | `0` | The lowest epoch to backfill; never earlier than the deriver's activation fork |
| derivers.consensus.&lt;deriver&gt;.iterator.backfill.epochsPerBatch | int | `1` | Epochs derived and written as one unit per backfill step (at most 64), so one ClickHouse insert per table carries the whole batch. The backfill marker moves past a batch only after all of it is written; a batch whose marker update fails is recorded on the next step without being written again, and a batch that fails to derive or write is retried one epoch at a time (counted in `xatu_cannon_epoch_iterator_backfill_batch_fallbacks_total`). Head steps stay one epoch. A batch's events stay in memory until written. Each batching deriver preloads the blocks of its current and next batch into the block cache shared by every deriver, so the sum of `2 × epochsPerBatch × 32` over enabled, backfilling derivers must fit `ethereum.beacon.blockCacheSize` (validated at startup); block archive windows also land in that cache whole, rounded up to `ethereum.blockArchive.rangeSlots`. Inserts larger than the server's `async_insert_max_data_size` (10 MiB by default) skip the async insert queue and write synchronously; `executionTransaction` crosses that at single-digit batch sizes on mainnet. Not supported by `beaconSyncCommittee`, `beaconValidators`, `blockAccessList` and the Gloas beacon state derivers, which share one state download per epoch |
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
