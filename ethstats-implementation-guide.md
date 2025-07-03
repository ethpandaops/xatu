# Ethstats Implementation Guide for Go Server

## Overview

Based on analysis of 5 major Ethereum execution layer clients (go-ethereum, Nethermind, Erigon, Besu, and reth), this guide provides comprehensive requirements for building a Go ethstats server that clients can connect to.

**Note**: Reth does not currently implement ethstats support.

## Core Protocol Requirements

### 1. WebSocket Connection

- **Protocol**: WebSocket (must support both `ws://` and `wss://`)
- **Endpoint**: Most clients expect `/api` endpoint (some will try both root and `/api`)
- **Message Size Limit**: 15 MB (go-ethereum requirement)
- **Timeout**: 5-second handshake timeout is standard
- **Ping/Pong**: Must handle WebSocket-level ping/pong for connection health

### 2. Authentication

**URL Format**: `nodename:secret@host:port`

**Authentication Flow**:
1. Client connects via WebSocket
2. Client sends "hello" message with node info and secret
3. Server validates secret and responds with "ready" message
4. If authentication fails, close connection

### 3. Message Protocol

All messages use JSON format with an "emit" wrapper:

```json
{
  "emit": ["message_type", data_object]
}
```

## Message Types

### Incoming Messages (Client → Server)

#### 1. **hello** - Initial Authentication
```json
{
  "emit": ["hello", {
    "id": "nodename",
    "info": {
      "name": "nodename",
      "node": "Geth/v1.10.0",
      "port": 30303,
      "net": "1",
      "protocol": "eth/66,eth/67",
      "api": "no",
      "os": "linux",
      "os_v": "x64",
      "client": "0.1.1",
      "canUpdateHistory": true,
      "contact": "admin@example.com"  // Optional, Besu/Nethermind specific
    },
    "secret": "your-secret"
  }]
}
```

#### 2. **block** - New Block Report
```json
{
  "emit": ["block", {
    "id": "nodename",
    "block": {
      "number": 12345678,
      "hash": "0x...",
      "parentHash": "0x...",
      "timestamp": 1234567890,
      "miner": "0x...",
      "gasUsed": 15000000,
      "gasLimit": 30000000,
      "difficulty": "1234567890",
      "totalDifficulty": "12345678901234567890",
      "transactions": [
        {"hash": "0x..."},
        {"hash": "0x..."}
      ],
      "transactionsRoot": "0x...",
      "stateRoot": "0x...",
      "uncles": []
    }
  }]
}
```

#### 3. **pending** - Pending Transactions Count
```json
{
  "emit": ["pending", {
    "id": "nodename",
    "stats": {
      "pending": 150
    }
  }]
}
```

#### 4. **stats** - Node Statistics
```json
{
  "emit": ["stats", {
    "id": "nodename",
    "stats": {
      "active": true,
      "syncing": false,
      "mining": false,
      "hashrate": 0,
      "peers": 25,
      "gasPrice": 30000000000,
      "uptime": 100
    }
  }]
}
```

#### 5. **history** - Historical Blocks
```json
{
  "emit": ["history", {
    "id": "nodename",
    "history": [
      // Array of block objects (same structure as block report)
    ]
  }]
}
```

#### 6. **node-ping** - Latency Measurement Request
```json
{
  "emit": ["node-ping", {
    "id": "nodename",
    "clientTime": "timestamp_string"
  }]
}
```

#### 7. **latency** - Latency Report
```json
{
  "emit": ["latency", {
    "id": "nodename",
    "latency": 25
  }]
}
```

#### 8. **primus::pong::timestamp** - Heartbeat Response
```json
"primus::pong::1234567890123"
```

### Outgoing Messages (Server → Client)

#### 1. **ready** - Authentication Success
```json
{
  "emit": ["ready"]
}
```

#### 2. **node-pong** - Latency Measurement Response
```json
{
  "emit": ["node-pong", {
    "id": "nodename",
    "clientTime": "original_timestamp",
    "serverTime": "server_timestamp"
  }]
}
```

#### 3. **history** - Request Historical Data
```json
{
  "emit": ["history", {
    "max": 50,
    "min": 1
  }]
}
```

#### 4. **primus::ping::timestamp** - Heartbeat Request
```json
"primus::ping::1234567890123"
```

## Implementation Requirements

### 1. Connection Management

- Support multiple concurrent client connections
- Implement connection pooling with unique node identification
- Handle reconnections gracefully (clients will automatically reconnect)
- Track connection state per client

### 2. Data Handling

- **Block Numbers**: Can be hex strings or numbers (handle both)
- **Timestamps**: Unix timestamps (seconds since epoch)
- **Difficulty**: String representation of big integers
- **Gas Price**: Integer (wei), may need to handle very large values
- **Hashes**: Always 0x-prefixed hex strings

### 3. Update Frequencies

- **Full Stats Report**: Every 15 seconds (go-ethereum, Erigon, Nethermind)
- **Block Updates**: Real-time on new blocks
- **Pending Updates**: Throttled to max once per second (go-ethereum)
- **Heartbeat**: Primus ping/pong every few seconds

### 4. Special Considerations

#### Primus Protocol Support
Some clients expect Primus WebSocket protocol compatibility:
- Messages prefixed with `primus::ping::` are heartbeats
- Must respond with `primus::pong::` + same timestamp
- These are raw strings, not JSON

#### Client Variations
- **Nethermind**: May send "contact" field in node info
- **Besu**: Sends "contact" field, reports Clique signer status in mining field
- **Go-ethereum**: Throttles transaction updates, reports light client status
- **Erigon**: Always reports mining=false, gasPrice=0

#### Error Handling
- Invalid authentication should close connection immediately
- Malformed messages should be logged but not crash server
- Support graceful degradation for missing optional fields

### 5. Security Considerations

- Validate all input data to prevent injection attacks
- Implement rate limiting per client
- Use TLS (wss://) in production
- Store secrets securely (hashed, not plaintext)
- Implement connection limits to prevent DoS

### 6. Monitoring and Logging

- Log all client connections/disconnections
- Track message rates per client
- Monitor WebSocket connection health
- Implement metrics for server performance

## Sample Go Server Structure

```go
type EthstatsServer struct {
    clients map[string]*Client
    mu      sync.RWMutex
}

type Client struct {
    ID       string
    NodeInfo NodeInfo
    Conn     *websocket.Conn
    LastSeen time.Time
}

type NodeInfo struct {
    Name     string `json:"name"`
    Node     string `json:"node"`
    Port     int    `json:"port"`
    Network  string `json:"net"`
    Protocol string `json:"protocol"`
    // ... other fields
}

type Message struct {
    Emit []interface{} `json:"emit"`
}
```

## Testing Recommendations

1. Test with multiple client implementations
2. Verify handling of large block numbers (post-merge)
3. Test reconnection scenarios
4. Validate Primus protocol compatibility
5. Stress test with many concurrent connections
6. Test malformed message handling

## Common Gotchas

1. **Message Format**: The "emit" field is always an array with exactly 2 elements
2. **Empty Arrays**: Uncle arrays should be `[]` not `null`
3. **Network ID**: Sometimes sent as string, sometimes as number
4. **Gas Price**: May overflow int64, use big.Int or uint64
5. **History Requests**: Clients may expect server to request history on login
6. **ID Field**: Some clients generate complex IDs (e.g., "name-keccakhash")
7. **Protocol Versions**: Format varies (e.g., "eth/66,eth/67" vs "eth/66, eth/67")

## Additional Features to Consider

1. **Web Dashboard**: Display connected nodes and statistics
2. **Persistence**: Store historical data for analytics
3. **Alerting**: Notify on node disconnections or issues
4. **API**: RESTful API for querying node status
5. **Clustering**: Support multiple server instances
6. **Export**: Prometheus metrics or other monitoring integrations