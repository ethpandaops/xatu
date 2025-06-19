# CL-Mimicry: Ethereum Consensus Layer P2P Network Monitoring

CL-Mimicry is a sophisticated consensus layer P2P network monitoring client that mimics validator behavior to collect libp2p and gossipsub events from Ethereum consensus networks. It provides advanced trace-based sampling and sharding capabilities for scalable, distributed monitoring of Ethereum network activity.

## Table of Contents

- [Overview](#overview)
- [Sharding System](#sharding-system)
- [Event Categorization](#event-categorization)
- [Configuration](#configuration)
- [Metrics](#metrics)
- [Examples](#examples)
- [Migration from V1](#migration-from-v1)

## Overview

CL-Mimicry connects to Ethereum consensus network nodes and captures libp2p trace events, providing insights into:

- **Gossipsub Message Flow**: Beacon blocks, attestations, and blob sidecars
- **Peer Behavior**: Connection patterns, message propagation, and network topology
- **Protocol Performance**: Message timing, duplicate detection, and peer interactions
- **Network Health**: RPC communication, consensus participation, and validator activity

The system uses **consistent hashing** with **SipHash-2-4** algorithm to enable distributed processing across multiple instances while maintaining deterministic message routing.

## Sharding System

CL-Mimicry uses a simplified, unified sharding system based on a streamlined event categorization model.
- **Event categorization**: Events grouped by sharding capabilities (Groups A-D)
- **Configurable shards**: Consistent distribution across all configurations (defaulting to 512)
- **Topic-first design**: Prioritize topic-based sharding where available

### How It Works

```
┌─────────────┐
│Event Arrives│
└──────┬──────┘
       │
       ▼
┌──────────────┐     ┌─────────────────┐
│Get Event Info├────►│ Event Category? │
└──────────────┘     └────────┬────────┘
                              │
        ┌─────────────────────┼─────────────────────┬─────────────────────┐
        ▼                     ▼                     ▼                     ▼
   ┌─────────┐         ┌─────────┐           ┌─────────┐           ┌─────────┐
   │ Group A │         │ Group B │           │ Group C │           │ Group D │
   │Topic+Msg│         │Topic Only│          │Msg Only │           │ No Keys │
   └────┬────┘         └────┬────┘           └────┬────┘           └────┬────┘
        │                   │                     │                     │
        ▼                   ▼                     ▼                     ▼
   Topic Config?       Topic Config?         Default Shard         Enabled?
        │                   │                     │                     │
     Yes/No              Yes/No                   │                  Yes/No
        │                   │                     │                     │
        ▼                   ▼                     ▼                     ▼
   Shard by Msg       Shard by Topic        Shard by Msg         Process/Drop
```

## Event Categorization

Events are categorized into four groups based on their available sharding keys:

### Group A: Topic + MsgID Events
Events with both topic and message ID, enabling full sharding flexibility:
- `PUBLISH_MESSAGE`, `DELIVER_MESSAGE`, `DUPLICATE_MESSAGE`, `REJECT_MESSAGE`
- `GOSSIPSUB_BEACON_BLOCK`, `GOSSIPSUB_BEACON_ATTESTATION`, `GOSSIPSUB_BLOB_SIDECAR`
- `RPC_META_MESSAGE`, `RPC_META_CONTROL_IHAVE`

**Sharding**: Uses message ID for sharding, with topic-based configuration

### Group B: Topic-Only Events
Events with only topic information:
- `JOIN`, `LEAVE`, `GRAFT`, `PRUNE`
- `RPC_META_CONTROL_GRAFT`, `RPC_META_CONTROL_PRUNE`, `RPC_META_SUBSCRIPTION`

**Sharding**: Uses topic hash for sharding decisions

### Group C: MsgID-Only Events
Events with only message ID:
- `RPC_META_CONTROL_IWANT`, `RPC_META_CONTROL_IDONTWANT`

**Sharding**: Uses message ID with default configuration

### Group D: No Sharding Key Events
Events without sharding keys:
- `ADD_PEER`, `REMOVE_PEER`, `CONNECTED`, `DISCONNECTED`
- `RECV_RPC`, `SEND_RPC`, `DROP_RPC` (parent events only)
- `HANDLE_METADATA`, `HANDLE_STATUS`

**Sharding**: All-or-nothing based on configuration

## Configuration

The configuration focuses on topic-based patterns with simplified sharding:

### Basic Structure

```yaml
sharding:
  # Topic-based sharding configuration
  topics:
    ".*beacon_block.*":
      totalShards: 512          # Always 512 for consistency
      activeShards: ["0-511"]   # 100% sampling

    ".*beacon_attestation.*":
      totalShards: 512
      activeShards: ["0-25"]    # 26/512 = ~5% sampling

    ".*":                       # Catch-all pattern
      totalShards: 512
      activeShards: ["0-127"]   # 25% sampling

  # Events without sharding keys (Group D)
  noShardingKeyEvents:
    enabled: true               # Process all Group D events

events:
  # Enable/disable specific event types
  recvRpcEnabled: true
  gossipSubBeaconBlockEnabled: true
  gossipSubAttestationEnabled: true
  # ... etc
```

### Active Shards Syntax

Flexible syntax for specifying which shards to process:

```yaml
# Range syntax (recommended)
activeShards: ["0-255"]         # 256 shards = 50% sampling

# Individual shards
activeShards: [0, 1, 5, 10]     # Specific shards only

# Mixed syntax
activeShards: ["0-10", 50, "100-150"]  # Ranges and individuals

# Common sampling rates with 512 total shards:
activeShards: ["0-511"]         # 100% (all shards)
activeShards: ["0-255"]         # 50%  (256 shards)
activeShards: ["0-127"]         # 25%  (128 shards)
activeShards: ["0-25"]          # 5%   (26 shards)
activeShards: ["0-4"]           # 1%   (5 shards)
activeShards: [0]               # 0.2% (1 shard)
```

### Pattern Matching

Topics use regex patterns with highest-match-wins:

```yaml
topics:
  # Most specific patterns first
  ".*beacon_attestation_[0-9]+.*":  # Subnet-specific
    activeShards: ["0-12"]          # 2.5% sampling

  ".*beacon_attestation.*":         # General attestations
    activeShards: ["0-25"]          # 5% sampling

  ".*":                            # Everything else
    activeShards: ["0-127"]        # 25% sampling
```
