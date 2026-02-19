# Sentry Logs

Vector-based log collection for Ethereum client structured logs. This component uses [Vector](https://vector.dev) to collect structured logs from Ethereum execution clients and send them to the Xatu server via HTTP.

Unlike other Xatu components, Sentry Logs runs as a standalone Docker container (not part of the main Xatu binary) and is designed to collect execution client metrics from structured log output.

## Table of contents

- [Usage](#usage)
- [Requirements](#requirements)
- [Configuration](#configuration)
  - [Environment Variables](#environment-variables)
  - [Log Sources](#log-sources)
- [Supported Log Formats](#supported-log-formats)
- [Execution Client Configuration](#execution-client-configuration)
  - [Geth](#geth)
- [Development](#development)
- [Building](#building)
- [Supported Events](#supported-events)

## Usage

Sentry Logs is distributed as a Docker image.

### Docker

```bash
docker run -d \
  -e XATU_CLIENT_NAME=my-sentry-logs \
  -e XATU_NETWORK_NAME=mainnet \
  -e XATU_SERVER_URL=http://your-xatu-server:8087/v1/events \
  -e XATU_AUTH=$(echo -n "user:password" | base64) \
  -v /path/to/geth/logs:/var/log/geth:ro \
  -v /path/to/sources.d:/etc/xatu/sources.d:ro \
  ethpandaops/xatu:sentry-logs-latest
```

### Docker Compose

```yaml
services:
  sentry-logs:
    image: ethpandaops/xatu:sentry-logs-latest
    environment:
      XATU_CLIENT_NAME: my-sentry-logs
      XATU_NETWORK_NAME: mainnet
      XATU_SERVER_URL: http://xatu-server:8087/v1/events
      XATU_AUTH: dXNlcjpwYXNzd29yZA==  # base64(user:password)
    volumes:
      - /var/log/geth:/var/log/geth:ro
      - ./sources.d:/etc/xatu/sources.d:ro
```

## Requirements

- Ethereum execution client with block execution metrics logging enabled
- [Xatu server](./server.md) with HTTP ingester enabled
- Docker runtime

## Configuration

### Environment Variables

| Name | Required | Default | Description |
| --- | --- | --- | --- |
| XATU_CLIENT_NAME | **Yes** | | Unique name of the sentry-logs instance |
| XATU_NETWORK_NAME | **Yes** | | Ethereum network name e.g. `mainnet`, `holesky`, `sepolia` |
| XATU_SERVER_URL | **Yes** | | Xatu server HTTP endpoint (e.g., `http://xatu-server:8087/v1/events`) |
| XATU_AUTH | **Yes** | | Base64-encoded `username:password` for Basic authentication |

The process will crash on startup if required environment variables are not set.
| XATU_VERSION | No | Build version | Version string reported in event metadata |
| XATU_COMPRESSION | No | `gzip` | Compression algorithm: `gzip`, `none` |
| XATU_BATCH_MAX_EVENTS | No | `5000` | Maximum events per batch |
| XATU_BATCH_TIMEOUT_SECS | No | `5` | Batch timeout in seconds |

**Note:** Requests are gzip-compressed by default for bandwidth efficiency. Set `XATU_COMPRESSION=none` to disable.

### Log Sources

Log sources are configured via YAML files in the `/etc/xatu/sources.d/` directory. Create a `file.yaml` with your source configuration:

#### File Source

```yaml
sources:
  ethereum_geth:
    type: file
    include:
      - /var/log/geth/*.log
```

#### Docker Logs Source

```yaml
sources:
  ethereum_geth_docker:
    type: docker_logs
    include_containers:
      - geth
```

#### Journald Source

```yaml
sources:
  ethereum_geth_journald:
    type: journald
    include_units:
      - geth.service
```

**Important:** Source names must start with `ethereum_` to be processed by the Vector pipeline.

## Supported Log Formats

Sentry Logs automatically detects and parses the following log formats:

| Format | Example |
| --- | --- |
| Raw JSON | `{"level":"warn","msg":"Slow block",...}` |
| slog JSON (`--log.format json`) | `{"t":"...","lvl":"warn","msg":"{\"level\":\"warn\",...}"}` |
| Terminal (`--log.format terminal`, default) | `WARN [01-28\|12:58:41.123] {"level":"warn","msg":"Slow block",...}` |
| Logfmt (`--log.format logfmt`) | `t=2026-01-28T... lvl=warn msg="{\"level\":\"warn\",...}"` |

No specific `--log.format` flag is required. All geth log formats are supported.

## Execution Client Configuration

### Geth

To enable block execution metrics logging in geth:

```bash
geth --debug.logslowblock 0
```

| Flag | Description |
| --- | --- |
| `--debug.logslowblock 0` | Logs metrics for all blocks (threshold of 0ms means every block is logged) |

To enable state size delta and MPT depth metrics, add the `--vmtrace` flag:

```bash
geth --debug.logslowblock 0 --vmtrace statesize
```

| Flag | Description |
| --- | --- |
| `--vmtrace statesize` | Enables the statesize tracer which logs state size deltas and trie depth stats per block |

Any `--log.format` value is supported (json, terminal, logfmt). If omitted, geth defaults to terminal format.

## Development

```bash
# Start the dev stack (xatu-server, clickhouse, kafka, vector, sentry-logs)
make sentry-logs-dev

# Log file is at: deploy/local/docker-compose/sentry-logs/logs/geth.log
# Append JSON logs to test the pipeline

# View sentry-logs output
docker logs -f xatu-sentry-logs

# Query ClickHouse for results
docker exec xatu-clickhouse-01 clickhouse-client --query \
  'SELECT block_number, total_ms, mgas_per_sec FROM default.execution_block_metrics'
```

## Building

```bash
# Build Docker image
make sentry-logs-build
```

## Supported Events

| Event | ID | Description |
| --- | --- | --- |
| `EXECUTION_BLOCK_METRICS` | 87 | Block execution performance metrics including timing, state reads/writes, and cache statistics |
| `EXECUTION_STATE_SIZE_DELTA` | 88 | State size delta per block: account, storage, trienode, and contract code count/byte changes |
| `EXECUTION_MPT_DEPTH` | 89 | Merkle Patricia Trie depth metrics: per-depth node counts and byte sizes for written/deleted trie nodes |
