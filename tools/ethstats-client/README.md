# Ethstats Test Client

A standalone test client for the ethstats server that simulates an Ethereum node reporting statistics.

**Note:** This tool requires the `tools` build tag to compile.

## Building

```bash
# Build with the tools tag
go build -tags tools -o ethstats-client .

# Or use the build script
../../tools/build.sh
```

## Usage

```bash
./ethstats-client --ethstats username:password@host:port
```

### Example

```bash
# Connect to local server
./ethstats-client --ethstats testuser:testpass@localhost:8081

# Connect to remote server
./ethstats-client --ethstats mynode:secretpass@ethstats.example.com:8081
```

## Features

- Performs initial authentication handshake
- Sends random events every 5 seconds:
  - Block reports (incrementing block numbers)
  - Node statistics (peers, syncing status)
  - Pending transaction counts
  - Latency reports
  - Historical block data (10% chance)
- Handles ping/pong messages every 30 seconds
- Graceful shutdown on Ctrl+C

## Event Types

The client simulates the following event types:

1. **Block Reports**: New blocks with incrementing numbers
2. **Stats Updates**: Node statistics including peer count and sync status
3. **Pending Transactions**: Random pending transaction counts
4. **Latency Reports**: Simulated network latency (10-110ms)
5. **History Reports**: Occasional historical block data

## Notes

- The client uses the node ID from the username part of the credentials
- Initial block number starts around 18,000,000 + random offset
- All random values use crypto/rand for secure randomness
- The client maintains the WebSocket connection and handles reconnection if needed