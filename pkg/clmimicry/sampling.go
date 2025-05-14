package clmimicry

import (
	"crypto/sha256"
	"encoding/binary"
	"math/rand"
	"regexp"
	"strings"

	"github.com/OffchainLabs/prysm/v6/beacon-chain/p2p"
	pubsubpb "github.com/libp2p/go-libp2p-pubsub/pb"
)

// ShouldSampleMessage determines whether a message with the given MsgID should be included
// in the sample based on the configured sampling settings.
func (m *Mimicry) ShouldSampleMessage(msgID string, eventType string) bool {
	// If no msgID, we can't sample.
	if msgID == "" {
		return true
	}

	// Get sampling config for this event type.
	config, enabled := m.isSamplingEnabled(eventType)

	// If sampling is not enabled for this event type, process all messages.
	if !enabled {
		return true
	}

	// Check if the message matches the pattern based on the mode.
	// If the message doesn't match the pattern, don't process it.
	if !matchesPattern(msgID, config.Pattern, m.Config.Events.SamplingMode) {
		return false
	}

	// If pattern matches, apply sampling rate.
	// This ensures we only process a percentage of matching messages.
	if config.SampleRate >= 1.0 {
		// Process all matching messages.
		return true
	} else if config.SampleRate <= 0.0 {
		// Process no matching messages.
		return false
	} else {
		// Sample based on sample rate.
		if m.Config.Events.SamplingMode == "modulo" {
			// For modulo mode, use deterministic sampling.
			return deterministicSample(msgID, config.SampleRate)
		} else {
			// For other modes, use random sampling.
			return rand.Float64() < config.SampleRate
		}
	}
}

// isSamplingEnabled checks if sampling is enabled for the given event type
// and returns the sampling config if it is.
func (m *Mimicry) isSamplingEnabled(eventType string) (*SamplingConfig, bool) {
	switch eventType {
	case p2p.GossipBlockMessage:
		if m.Config.Events.GossipSubBeaconBlock.Sampling == nil || !m.Config.Events.GossipSubBeaconBlock.Sampling.Enabled {
			return nil, false
		}

		return m.Config.Events.GossipSubBeaconBlock.Sampling, true
	case p2p.GossipAttestationMessage:
		if m.Config.Events.GossipSubAttestation.Sampling == nil || !m.Config.Events.GossipSubAttestation.Sampling.Enabled {
			return nil, false
		}

		return m.Config.Events.GossipSubAttestation.Sampling, true
	case p2p.GossipBlobSidecarMessage:
		if m.Config.Events.GossipSubBlobSidecar.Sampling == nil || !m.Config.Events.GossipSubBlobSidecar.Sampling.Enabled {
			return nil, false
		}

		return m.Config.Events.GossipSubBlobSidecar.Sampling, true
	case pubsubpb.TraceEvent_ADD_PEER.String():
		if m.Config.Events.AddPeer.Sampling == nil || !m.Config.Events.AddPeer.Sampling.Enabled {
			return nil, false
		}

		return m.Config.Events.AddPeer.Sampling, true
	case pubsubpb.TraceEvent_REMOVE_PEER.String():
		if m.Config.Events.RemovePeer.Sampling == nil || !m.Config.Events.RemovePeer.Sampling.Enabled {
			return nil, false
		}

		return m.Config.Events.RemovePeer.Sampling, true
	case pubsubpb.TraceEvent_JOIN.String():
		if m.Config.Events.Join.Sampling == nil || !m.Config.Events.Join.Sampling.Enabled {
			return nil, false
		}

		return m.Config.Events.Join.Sampling, true
	case pubsubpb.TraceEvent_RECV_RPC.String():
		if m.Config.Events.RecvRPC.Sampling == nil || !m.Config.Events.RecvRPC.Sampling.Enabled {
			return nil, false
		}

		return m.Config.Events.RecvRPC.Sampling, true
	case pubsubpb.TraceEvent_SEND_RPC.String():
		if m.Config.Events.SendRPC.Sampling == nil || !m.Config.Events.SendRPC.Sampling.Enabled {
			return nil, false
		}

		return m.Config.Events.SendRPC.Sampling, true
	case "CONNECTED":
		if m.Config.Events.Connected.Sampling == nil || !m.Config.Events.Connected.Sampling.Enabled {
			return nil, false
		}

		return m.Config.Events.Connected.Sampling, true
	case "DISCONNECTED":
		if m.Config.Events.Disconnected.Sampling == nil || !m.Config.Events.Disconnected.Sampling.Enabled {
			return nil, false
		}

		return m.Config.Events.Disconnected.Sampling, true
	case "HANDLE_METADATA":
		if m.Config.Events.HandleMetadata.Sampling == nil || !m.Config.Events.HandleMetadata.Sampling.Enabled {
			return nil, false
		}

		return m.Config.Events.HandleMetadata.Sampling, true
	case "HANDLE_STATUS":
		if m.Config.Events.HandleStatus.Sampling == nil || !m.Config.Events.HandleStatus.Sampling.Enabled {
			return nil, false
		}

		return m.Config.Events.HandleStatus.Sampling, true
	default:
		return nil, false
	}
}

// matchesPattern checks if the MsgID matches the pattern according to the specified mode.
func matchesPattern(msgID, pattern, mode string) bool {
	switch mode {
	case "prefix":
		return strings.HasPrefix(msgID, pattern)
	case "regex":
		// Compile the regex pattern.
		// Note: In a real implementation, you would want to cache the compiled regex.
		re, err := regexp.Compile(pattern)
		if err != nil {
			// If the pattern is invalid, don't match anything.
			return false
		}
		return re.MatchString(msgID)
	case "modulo":
		// For modulo, the pattern is a divisor (as a string).
		// If MsgID hash % divisor == 0, then it matches.
		divisor := 100 // Default.
		hash := hashMsgID(msgID)
		return hash%divisor == 0
	default:
		// Unknown mode, don't match anything.
		return false
	}
}

// deterministicSample provides deterministic sampling based on a hash of the MsgID.
// This ensures the same MsgID will always get the same sampling decision.
func deterministicSample(msgID string, sampleRate float64) bool {
	// Hash the MsgID to get a deterministic value.
	hash := hashMsgID(msgID)

	// Convert sample rate to an integer threshold (out of 1000).
	threshold := int(sampleRate * 1000)

	// If hash % 1000 is less than threshold, process the message.
	return hash%1000 < threshold
}

// hashMsgID creates a deterministic integer hash from a MsgID.
func hashMsgID(msgID string) int {
	// Create a SHA-256 hash of the MsgID.
	hasher := sha256.New()
	hasher.Write([]byte(msgID))
	hashBytes := hasher.Sum(nil)

	// Convert the first 4 bytes to an integer.
	return int(binary.BigEndian.Uint32(hashBytes[:4]))
}
