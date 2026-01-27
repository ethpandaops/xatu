# Ethstats

Server that receives data from Ethereum execution clients via the [ethstats protocol](https://github.com/ethereum/go-ethereum/wiki/Network-Status-Monitoring) and forwards events to configured output sinks.

This mode acts as an ethstats server, allowing execution clients (like Geth) to connect and report their status. Events are converted to Xatu's protobuf format and can be forwarded to various sinks.

## Table of contents

- [Usage](#usage)
- [Client Connection](#client-connection)
  - [Credential Format](#credential-format)
  - [Connection Examples](#connection-examples)
- [Configuration](#configuration)
  - [`xatu` output](#output-xatu-configuration)
  - [`http` output](#output-http-configuration)
  - [`kafka` output](#output-kafka-configuration)
  - [Simple example](#simple-example)
  - [With authorization example](#with-authorization-example)
  - [Complex example](#complex-example)
- [Events](#events)
- [Running locally](#running-locally)

## Usage

Ethstats requires a [config file](#configuration).

```bash
Usage:
  xatu ethstats [flags]

Flags:
      --config string   config file (default is ethstats.yaml) (default "ethstats.yaml")
  -h, --help            help for ethstats
```

## Client Connection

Execution clients connect to the ethstats server using a WebSocket connection with credentials.

### Credential Format

Clients connect using the format:

```
nodename:base64(username:password)@server:port
```

Where:
- **nodename**: A unique display name for this node (appears in logs and events)
- **base64(username:password)**: Base64-encoded credentials for authentication
- **server:port**: The ethstats server address

This format allows multiple nodes to share the same credentials while having unique display names. The server decodes the base64 secret to extract the actual username and password for authentication.

### Connection Examples

**Geth:**
```bash
# First, encode your credentials
CREDENTIALS=$(echo -n "myuser:mypassword" | base64)
# Result: bXl1c2VyOm15cGFzc3dvcmQ=

# Then connect with your node name and encoded credentials
geth --ethstats "my-geth-node:${CREDENTIALS}@localhost:3000"

# Or inline:
geth --ethstats "my-geth-node:$(echo -n 'myuser:mypassword' | base64)@localhost:3000"
```

**Multiple nodes with shared credentials:**
```bash
# Node 1
geth --ethstats "geth-node-us-east:bXl1c2VyOm15cGFzc3dvcmQ=@ethstats.example.com:3000"

# Node 2 (same credentials, different name)
geth --ethstats "geth-node-eu-west:bXl1c2VyOm15cGFzc3dvcmQ=@ethstats.example.com:3000"
```

## Configuration

Ethstats requires a single `yaml` config file. An example file can be found [here](../example_ethstats.yaml)

| Name | Type | Default | Description |
| --- | --- | --- | --- |
| logging | string | `info` | Log level (`panic`, `fatal`, `warn`, `info`, `debug`, `trace`) |
| metricsAddr | string | `:9090` | The address the metrics server will listen on |
| pprofAddr | string | | The address the [pprof](https://github.com/google/pprof) server will listen on. When omitted, the pprof server will not be started |
| name | string | | Unique name of this ethstats instance |
| addr | string | `:3000` | Address to listen for WebSocket connections |
| labels | object | | A key value map of labels to append to every event |
| ntpServer | string | `time.google.com` | NTP server to calculate clock drift for events |
| overrideNetworkName | string | | Override the network name for all events (useful for testnets) |
| clientNameSalt | string | | Salt for hashing client names (for privacy) |
| websocket.readBufferSize | int | `1024` | Buffer size for reading WebSocket messages |
| websocket.writeBufferSize | int | `1024` | Buffer size for writing WebSocket messages |
| websocket.pingInterval | string | `15s` | Interval between ping messages |
| websocket.readLimit | int | `15728640` | Maximum message size in bytes (15MB) |
| websocket.writeWait | string | `10s` | Write deadline duration |
| authorization | object | | Authorization configuration (see below) |
| geoip | object | | GeoIP configuration for geographic enrichment |
| outputs | array | | List of outputs for events |

### Authorization Configuration

| Name | Type | Default | Description |
| --- | --- | --- | --- |
| authorization.enabled | bool | `false` | Enable authorization |
| authorization.groups | object | | Map of group configurations |
| authorization.groups[name].users | object | | Map of username to user config |
| authorization.groups[name].users[username].password | string | | Password for the user |
| authorization.groups[name].users[username].eventFilter | object | | Optional event filtering |
| authorization.groups[name].obscureClientNames | bool | `false` | Hash client names for privacy |
| authorization.groups[name].precision | string | `full` | GeoIP precision (`full`, `city`, `country`, `continent`, `none`) |

### Output `xatu` configuration

Output configuration to send events to a [Xatu server](./server.md).

| Name | Type | Default | Description |
| --- | --- | --- | --- |
| outputs[].config.address | string | | The address of the server receiving events |
| outputs[].config.tls | bool | | Server requires TLS |
| outputs[].config.headers | object | | A key value map of headers to append to requests |
| outputs[].config.maxQueueSize | int | `51200` | The maximum queue size to buffer events |
| outputs[].config.batchTimeout | string | `5s` | The maximum duration for constructing a batch |
| outputs[].config.exportTimeout | string | `30s` | The maximum duration for exporting events |
| outputs[].config.maxExportBatchSize | int | `512` | Maximum number of events per batch |

### Output `http` configuration

Output configuration to send events to an HTTP server.

| Name | Type | Default | Description |
| --- | --- | --- | --- |
| outputs[].config.address | string | | The address of the server receiving events |
| outputs[].config.headers | object | | A key value map of headers to append to requests |
| outputs[].config.maxQueueSize | int | `51200` | The maximum queue size to buffer events |
| outputs[].config.batchTimeout | string | `5s` | The maximum duration for constructing a batch |
| outputs[].config.exportTimeout | string | `30s` | The maximum duration for exporting events |
| outputs[].config.maxExportBatchSize | int | `512` | Maximum number of events per batch |

### Output `kafka` configuration

Output configuration to send events to Kafka.

| Name | Type | Default | Allowed Values | Description |
| --- | --- | --- | --- | --- |
| outputs[].config.brokers | string | | | Comma delimited list of brokers |
| outputs[].config.topic | string | | | Name of the topic |
| outputs[].config.flushFrequency | string | `3s` | | Maximum time before flush |
| outputs[].config.flushMessages | int | `500` | | Maximum messages before flush |
| outputs[].config.flushBytes | int | `1000000` | | Maximum bytes before flush |
| outputs[].config.maxRetries | int | `3` | | Maximum retries for batch delivery |
| outputs[].config.compression | string | `none` | `none` `gzip` `snappy` `lz4` `zstd` | Compression to use |
| outputs[].config.requiredAcks | string | `leader` | `none` `leader` `all` | Required acks for delivery |
| outputs[].config.partitioning | string | `none` | `none` `random` | Partitioning strategy |

### Simple example

```yaml
name: my-ethstats-server

addr: ":3000"

outputs:
- name: standard-out
  type: stdout
```

### With authorization example

```yaml
name: my-ethstats-server

addr: ":3000"

authorization:
  enabled: true
  groups:
    production:
      users:
        myuser:
          password: "mypassword"

outputs:
- name: standard-out
  type: stdout
```

Clients connect with:
```bash
geth --ethstats "my-node:$(echo -n 'myuser:mypassword' | base64)@localhost:3000"
```

### Complex example

```yaml
logging: "debug"
metricsAddr: ":9090"
pprofAddr: ":6060"

name: ethstats-prod

addr: ":3000"

labels:
  environment: production
  region: us-east-1

ntpServer: time.google.com

websocket:
  readBufferSize: 2048
  writeBufferSize: 2048
  pingInterval: 15s
  readLimit: 15728640
  writeWait: 10s

authorization:
  enabled: true
  groups:
    team-alpha:
      obscureClientNames: true
      precision: "city"
      users:
        alpha-user:
          password: "alpha-secret"
          eventFilter:
            eventNames:
            - ETHSTATS_BLOCK
            - ETHSTATS_NODE_STATS
    team-beta:
      users:
        beta-user:
          password: "beta-secret"

geoip:
  enabled: true
  type: maxmind
  config:
    database:
      city: ./GeoLite2-City.mmdb
      asn: ./GeoLite2-ASN.mmdb

outputs:
- name: log
  type: stdout
- name: xatu-server
  type: xatu
  config:
    address: xatu.example.com:8080
    tls: true
    maxQueueSize: 51200
    batchTimeout: 5s
- name: kafka-sink
  type: kafka
  config:
    brokers: kafka.example.com:9092
    topic: ethstats-events
```

## Events

The ethstats server emits the following event types:

| Event | Description |
| --- | --- |
| `ETHSTATS_HELLO` | Emitted when a node connects and authenticates |
| `ETHSTATS_BLOCK` | Block information reported by the node |
| `ETHSTATS_NODE_STATS` | Node status (active, syncing, peers, etc.) |
| `ETHSTATS_PENDING` | Pending transaction count |
| `ETHSTATS_LATENCY` | Network latency measurement |
| `ETHSTATS_NEW_PAYLOAD` | Engine API new payload timing (if supported by client) |
| `ETHSTATS_HISTORY` | Historical block data |

## Running locally

```bash
# docker
docker run -d --name xatu-ethstats -v $HOST_DIR_CHANGE_ME/config.yaml:/opt/xatu/config.yaml -p 3000:3000 -p 9090:9090 -it ethpandaops/xatu:latest ethstats --config /opt/xatu/config.yaml

# build
go build -o dist/xatu main.go
./dist/xatu ethstats --config ethstats.yaml

# dev
go run main.go ethstats --config ethstats.yaml
```
