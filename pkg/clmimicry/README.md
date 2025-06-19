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
- **Single-level sharding**: No more two-level hierarchical complexity
- **Event categorization**: Events grouped by sharding capabilities (Groups A-D)
- **Fixed 512 shards**: Consistent distribution across all configurations
- **Topic-first design**: Prioritize topic-based sharding where available
- **Simplified metrics**: Only 5 core metrics for clear observability

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

## Metrics

The system provides 5 core metrics for clear observability:

### Core Metrics

```prometheus
# Total events received by type and network
xatu_cl_mimicry_events_total{event_type="GOSSIPSUB_BEACON_BLOCK", network="mainnet"}

# Events that passed sharding and were processed
xatu_cl_mimicry_events_processed_total{event_type="GOSSIPSUB_BEACON_ATTESTATION", network="mainnet"}

# Events filtered out by sharding
xatu_cl_mimicry_events_filtered_total{event_type="JOIN", network="mainnet"}

# Sharding decisions with reasons
xatu_cl_mimicry_sharding_decisions_total{event_type="PUBLISH_MESSAGE", reason="group_a_topic_match_true", network="mainnet"}

# Decorated events created (unchanged from V1)
xatu_cl_mimicry_libp2p_decorated_events_total{event_type="GOSSIPSUB_BEACON_BLOCK", network="mainnet"}
```

### Common Sharding Decision Reasons

- `sharding_disabled` - Sharding is turned off
- `group_a_topic_match_true/false` - Group A event matched/didn't match topic config
- `group_a_no_topic_config` - Group A event with no matching topic pattern
- `group_b_topic_match_true/false` - Group B event matched/didn't match topic config
- `group_b_no_config` - Group B event with no topic configuration
- `group_c_shard_X` - Group C event assigned to shard X
- `group_d_enabled/disabled` - Group D events enabled or disabled

## Examples

### Production Multi-Instance Setup

**Instance 1 - High Priority Events (Critical consensus data):**
```yaml
sharding:
  topics:
    # 100% of beacon blocks and finalized checkpoints
    ".*beacon_block.*|.*finalized_checkpoint.*":
      totalShards: 512
      activeShards: ["0-511"]  # 100% sampling
    
    # 10% of attestations (higher than instance 2)
    ".*beacon_attestation.*":
      totalShards: 512
      activeShards: ["0-50"]   # ~10% sampling
    
    # 50% of blob sidecars
    ".*blob_sidecar.*":
      totalShards: 512
      activeShards: ["0-255"]  # 50% sampling

  noShardingKeyEvents:
    enabled: true  # Capture all peer events

events:
  # Focus on consensus events
  gossipSubBeaconBlockEnabled: true
  gossipSubAttestationEnabled: true
  gossipSubBlobSidecarEnabled: true
  publishMessageEnabled: true
  deliverMessageEnabled: true
```

**Instance 2 - Network Analysis (Peer behavior & sampling):**
```yaml
sharding:
  topics:
    # Skip beacon blocks (handled by instance 1)
    ".*beacon_block.*":
      totalShards: 512
      activeShards: []  # 0% - skip entirely
    
    # Different attestation shards for redundancy
    ".*beacon_attestation.*":
      totalShards: 512
      activeShards: ["51-76"]  # ~5% sampling, different shards
    
    # Catch-all for other topics
    ".*":
      totalShards: 512
      activeShards: ["0-127"]  # 25% sampling

  noShardingKeyEvents:
    enabled: true

events:
  # Focus on network events
  joinEnabled: true
  leaveEnabled: true
  graftEnabled: true
  pruneEnabled: true
  addPeerEnabled: true
  removePeerEnabled: true
```

### Development Environment

**Local Testing (Minimal sampling):**
```yaml
sharding:
  topics:
    # Sample everything at 1%
    ".*":
      totalShards: 512
      activeShards: ["0-4"]  # 1% sampling
  
  noShardingKeyEvents:
    enabled: true

events:
  # Enable all events for testing
  all_events_enabled: true  # Simplified for example
```

### Specific Use Cases

**Attestation Subnet Analysis:**
```yaml
sharding:
  topics:
    # Different sampling per subnet
    ".*beacon_attestation_0.*":
      totalShards: 512
      activeShards: ["0-255"]  # 50% for subnet 0
    
    ".*beacon_attestation_1.*":
      totalShards: 512
      activeShards: ["0-127"]  # 25% for subnet 1
    
    ".*beacon_attestation.*":
      totalShards: 512
      activeShards: ["0-25"]   # 5% for other subnets
```

## Migration from V1

### Key Differences

1. **Configuration Structure**:
   - V1: Nested `traces.topics` with event patterns
   - Current: Flat `sharding.topics` with gossip topic patterns

2. **Event Selection**:
   - V1: Event type patterns in config
   - Current: Enable/disable events in `events` section

3. **Sharding Logic**:
   - V1: Two-level (event + topic) sharding
   - Current: Single-level with event categorization

### Migration Example

**V1 Config:**
```yaml
traces:
  enabled: true
  topics:
    "(?i).*gossipsub_beacon_attestation.*":
      topics:
        gossipTopics:
          ".*":
            totalShards: 512
            activeShards: ["0-25"]
```

**Current Config:**
```yaml
sharding:
  topics:
    ".*beacon_attestation.*":
      totalShards: 512
      activeShards: ["0-25"]

events:
  gossipSubAttestationEnabled: true
```

### Migration Steps

1. **Identify topic patterns**: Extract gossip topic patterns from V1 config
2. **Move to sharding section**: Place patterns under `sharding.topics`
3. **Enable events**: Set corresponding events to `true` in `events` section
4. **Test thoroughly**: Verify sharding distribution matches expectations
5. **Monitor metrics**: Use new metrics to validate behavior
