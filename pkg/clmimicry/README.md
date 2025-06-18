# CL-Mimicry: Ethereum Consensus Layer P2P Network Monitoring

CL-Mimicry is a sophisticated consensus layer P2P network monitoring client that mimics validator behavior to collect libp2p and gossipsub events from Ethereum consensus networks. It provides advanced trace-based sampling and sharding capabilities for scalable, distributed monitoring of Ethereum network activity.

## Table of Contents

- [Overview](#overview)
- [Event Processing](#event-processing)
- [Sharding System](#sharding-system)
- [Configuration](#configuration)
- [Supported Events](#supported-events)
- [Examples](#examples)
- [Performance](#performance)
- [Best Practices](#best-practices)

## Overview

CL-Mimicry connects to Ethereum consensus network nodes and captures libp2p trace events, providing insights into:

- **Gossipsub Message Flow**: Beacon blocks, attestations, and blob sidecars
- **Peer Behavior**: Connection patterns, message propagation, and network topology
- **Protocol Performance**: Message timing, duplicate detection, and peer interactions
- **Network Health**: RPC communication, consensus participation, and validator activity

The system uses **consistent hashing** with **SipHash-2-4** algorithm to enable distributed processing across multiple instances while maintaining deterministic message routing.

## Event Processing

CL-Mimicry processes events through a hierarchical handler system that categorizes events into four main types:

### 1. GossipSub Protocol Events
Events that carry Ethereum consensus data with full **hierarchical sharding support**:

- **Beacon Blocks**: Critical consensus events (typically 100% sampled)
- **Attestations**: Validator votes and consensus participation
- **Blob Sidecars**: EIP-4844 data availability sampling
- **Sync Committee Messages**: Light client support data

### 2. LibP2P Pubsub Protocol Level Events
Message flow events with **hierarchical and simple sharding support**:

- **Message Flow (Hierarchical)**: `PUBLISH_MESSAGE`, `DELIVER_MESSAGE`, `DUPLICATE_MESSAGE`, `REJECT_MESSAGE` 
- **Topic Management (Hierarchical)**: `JOIN`, `GRAFT`, `PRUNE`
- **Peer Management (Simple)**: `ADD_PEER`, `REMOVE_PEER`
- **RPC Communication (Simple)**: `RECV_RPC`, `SEND_RPC`, `DROP_RPC`

### 3. LibP2P Core Networking Events
Low-level connection events with **simple sharding support**:

- **Connection Lifecycle**: `CONNECTED`, `DISCONNECTED`
- **Protocol Negotiation**: Transport and stream management

### 4. Request/Response (RPC) Protocol Events
Ethereum-specific RPC events with **simple sharding support**:

- **Metadata Exchange**: `HANDLE_METADATA`
- **Status Synchronization**: `HANDLE_STATUS`

## Sharding System

### How Sharding Works

CL-Mimicry uses **SipHash-2-4** consistent hashing to distribute events across shards:

```
Event → Extract Sharding Key → SipHash → Shard Number → Active Check → Process/Skip
```

**Process Flow:**
1. **Extract Sharding Key**: Get MsgID or PeerID from event
2. **Apply SipHash**: `hash = SipHash(key, shardingKey)`
3. **Calculate Shard**: `shard = hash % totalShards`
4. **Check Active**: `if shard in activeShards then process`

### Sharding Key Types

- **MsgID** (default): Uses message ID for consistent message routing
- **PeerID**: Uses peer ID for peer-based event clustering

### Benefits

- **Horizontal Scaling**: Distribute load across multiple instances
- **Consistent Routing**: Same message always goes to same shard
- **Load Balancing**: Even distribution across all shards
- **Fault Tolerance**: Overlap shards for redundancy

## Configuration

### Simple Configuration

Uniform sampling across all messages of an event type:

```yaml
traces:
  enabled: true
  topics:
    "(?i).*duplicate_message.*":
      shardingKey: "MsgID"
      totalShards: 512
      activeShards: ["0-255"]  # 50% sampling
```

### Hierarchical Configuration

Different sampling rates based on gossip topic patterns:

```yaml
traces:
  enabled: true
  topics:
    "(?i).*duplicate_message.*":
      topics:
        gossipTopics:
          # Critical: 100% sampling for beacon blocks
          ".*beacon_block.*":
            shardingKey: "MsgID"
            totalShards: 512
            activeShards: ["0-511"]

          # Important: 50% sampling for attestations
          ".*beacon_attestation.*":
            shardingKey: "MsgID"
            totalShards: 512
            activeShards: ["0-255"]

          # Medium: 25% sampling for blob sidecars
          ".*blob_sidecar.*":
            shardingKey: "MsgID"
            totalShards: 512
            activeShards: ["0-127"]

        # Low priority: 1% sampling for unmatched topics
        fallback:
          shardingKey: "MsgID"
          totalShards: 512
          activeShards: [0]
```

### Active Shards Syntax

Flexible syntax for specifying which shards to process:

```yaml
# Individual shards
activeShards: [0, 1, 5, 10]

# Range syntax (inclusive)
activeShards: ["0-255"]

# Mixed syntax
activeShards: [0, "10-20", 50, "100-150"]

# Single shard (0.2% of 512 total)
activeShards: [0]

# All shards (100% sampling)
activeShards: ["0-511"]
```

## Supported Events

### Hierarchical Sharding Supported

Events that include gossip topic information and support topic-based sampling:

> **Note**: Message flow events (`DUPLICATE_MESSAGE`, `DELIVER_MESSAGE`, `PUBLISH_MESSAGE`, `REJECT_MESSAGE`) support **both** hierarchical and simple sharding. Use hierarchical when you need different sampling rates per gossip topic, or simple for uniform sampling across all messages.

| Event Type | Topic Information | Primary Use Case |
|------------|------------------|------------------|
| `LIBP2P_TRACE_DUPLICATE_MESSAGE` | ✅ Gossip topics | Duplicate detection with topic-aware sampling |
| `LIBP2P_TRACE_DELIVER_MESSAGE` | ✅ Gossip topics | Message delivery with topic-aware sampling |
| `LIBP2P_TRACE_PUBLISH_MESSAGE` | ✅ Gossip topics | Message publishing with topic-aware sampling |
| `LIBP2P_TRACE_REJECT_MESSAGE` | ✅ Gossip topics | Message rejection with topic-aware sampling |
| `LIBP2P_TRACE_GOSSIPSUB_BEACON_ATTESTATION` | ✅ Subnet patterns | Validator participation monitoring |
| `LIBP2P_TRACE_GOSSIPSUB_BEACON_BLOCK` | ✅ Network variants | Critical consensus tracking |
| `LIBP2P_TRACE_GOSSIPSUB_BLOB_SIDECAR` | ✅ Sidecar indices | EIP-4844 data availability |
| `LIBP2P_TRACE_JOIN` | ✅ Topic patterns | Topic subscription tracking |
| `LIBP2P_TRACE_LEAVE` | ✅ Topic patterns | Topic unsubscription tracking |
| `LIBP2P_TRACE_GRAFT` | ✅ Topic patterns | Mesh grafting with topic-aware sampling |
| `LIBP2P_TRACE_PRUNE` | ✅ Topic patterns | Mesh pruning with topic-aware sampling |
| `LIBP2P_TRACE_RPC_META_CONTROL_IHAVE` | ✅ Topic patterns | Message availability tracking |
| `LIBP2P_TRACE_RPC_META_CONTROL_GRAFT` | ✅ Topic patterns | RPC graft control messages |
| `LIBP2P_TRACE_RPC_META_CONTROL_PRUNE` | ✅ Topic patterns | Peer mesh optimization |
| `LIBP2P_TRACE_RPC_META_SUBSCRIPTION` | ✅ Topic patterns | Subscription behavior |
| `LIBP2P_TRACE_RPC_META_MESSAGE` | ✅ Topic patterns | Direct message flow |

### Simple Sharding Supported

Events with uniform sampling across the entire event type:

| Event Type | Available Sharding Keys | Primary Use Case |
|------------|------------------------|------------------|
| `LIBP2P_TRACE_ADD_PEER` | PeerID | Peer discovery monitoring |
| `LIBP2P_TRACE_REMOVE_PEER` | PeerID | Peer churn analysis |
| `LIBP2P_TRACE_RECV_RPC` | PeerID + Meta Messages | Inbound RPC monitoring |
| `LIBP2P_TRACE_SEND_RPC` | PeerID + Meta Messages | Outbound RPC monitoring |
| `LIBP2P_TRACE_DROP_RPC` | PeerID + Meta Messages | RPC failure analysis |
| `LIBP2P_TRACE_CONNECTED` | PeerID (Remote) | Connection establishment |
| `LIBP2P_TRACE_DISCONNECTED` | PeerID (Remote) | Connection termination |
| `LIBP2P_TRACE_HANDLE_METADATA` | PeerID | Metadata exchange |
| `LIBP2P_TRACE_HANDLE_STATUS` | PeerID | Status synchronization |
| `LIBP2P_TRACE_RPC_META_CONTROL_IWANT` | MsgID | Message request tracking |
| `LIBP2P_TRACE_RPC_META_CONTROL_IDONTWANT` | MsgID | Message rejection tracking |

> **Note**: All events now support some form of sharding. Events with topic information support hierarchical sharding, while all others support simple sharding via MsgID or PeerID keys.

## Examples

### Production Multi-Instance Setup

**Instance 1 (Primary Consensus Monitoring):**
```yaml
traces:
  enabled: true
  topics:
    # Full consensus event coverage
    "(?i).*(beacon_block|finalized_checkpoint).*":
      shardingKey: "MsgID"
      totalShards: 512
      activeShards: ["0-511"]  # 100% sampling

    # High attestation coverage
    "(?i).*attestation.*":
      shardingKey: "MsgID"
      totalShards: 512
      activeShards: ["0-255"]  # 50% sampling
```

**Instance 2 (Network Analysis):**
```yaml
traces:
  enabled: true
  topics:
    # Peer behavior analysis
    "(?i).*(add_peer|remove_peer|connected|disconnected).*":
      shardingKey: "PeerID"
      totalShards: 512
      activeShards: ["256-511"]  # Second half, 50% sampling

    # RPC communication analysis
    "(?i).*(recv_rpc|send_rpc).*":
      shardingKey: "PeerID"
      totalShards: 512
      activeShards: ["128-383"]  # Middle section, 50% sampling
```

### Development and Testing

**Local Development (Full Capture):**
```yaml
traces:
  enabled: true
  topics:
    # Capture everything for development
    "(?i).*":
      shardingKey: "MsgID"
      totalShards: 1
      activeShards: [0]  # 100% sampling, single shard
```

**Hierarchical Testing:**
```yaml
traces:
  enabled: true
  topics:
    "(?i).*duplicate_message.*":
      topics:
        gossipTopics:
          # Test beacon block processing
          ".*beacon_block.*":
            totalShards: 4
            activeShards: ["0-3"]  # 100% of 4 shards
            shardingKey: "MsgID"

          # Test attestation sampling
          ".*beacon_attestation.*":
            totalShards: 4
            activeShards: [0, 1]   # 50% of 4 shards
            shardingKey: "MsgID"

        fallback:
          totalShards: 4
          activeShards: [0]        # 25% of 4 shards
          shardingKey: "MsgID"
```
