package libp2p

import (
	"strconv"
	"strings"

	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/flattener"
	libp2pProto "github.com/ethpandaops/xatu/pkg/proto/libp2p"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

// agentVersion holds parsed components of a peer agent version string.
type agentVersion struct {
	Implementation string
	Version        string
	Major          string
	Minor          string
	Patch          string
	Platform       string
}

// topicParsed holds parsed components of a libp2p gossipsub topic string.
type topicParsed struct {
	Layer           string
	ForkDigestValue string
	Name            string
	Encoding        string
}

// multiAddrParsed holds parsed components of a multiaddr string.
type multiAddrParsed struct {
	Protocol  string
	IP        string
	Transport string
	Port      uint64
}

// parseTopicFields parses a libp2p gossipsub topic string into its components.
// Expected format: /eth2/{fork_digest}/{topic_name}/{encoding}
func parseTopicFields(topic string) topicParsed {
	if topic == "" {
		return topicParsed{}
	}

	parts := strings.Split(topic, "/")
	if len(parts) != 5 {
		return topicParsed{}
	}

	return topicParsed{
		Layer:           parts[1],
		ForkDigestValue: parts[2],
		Name:            parts[3],
		Encoding:        parts[4],
	}
}

// parseMultiAddr parses a multiaddr string into protocol, IP, transport,
// and port.
func parseMultiAddr(addr string) multiAddrParsed {
	parts := strings.Split(addr, "/")
	if len(parts) < 5 {
		return multiAddrParsed{}
	}

	port, _ := strconv.ParseUint(parts[4], 10, 64)

	return multiAddrParsed{
		Protocol:  parts[1],
		IP:        parts[2],
		Transport: parts[3],
		Port:      port,
	}
}

// parseAgentVersion parses a user agent or agent version string into its
// components (implementation, version, platform, semver parts).
func parseAgentVersion(raw string) agentVersion {
	if raw == "" {
		return agentVersion{}
	}

	raw = strings.Replace(raw, "teku/teku", "teku", 1)

	parts := strings.Split(raw, "/")
	if len(parts) == 0 {
		return agentVersion{}
	}

	parsed := agentVersion{
		Implementation: strings.ToLower(parts[0]),
	}

	if len(parts) > 1 {
		parsed.Version = parts[1]
	}

	if len(parts) > 2 && parsed.Implementation != "prysm" {
		parsed.Platform = parts[2]
	}

	parsed.Major, parsed.Minor, parsed.Patch = splitSemver(parsed.Version)

	return parsed
}

// splitSemver splits a version string into major, minor, and patch components.
func splitSemver(version string) (major, minor, patch string) {
	if version == "" {
		return "", "", ""
	}

	version = strings.TrimPrefix(version, "v")
	if idx := strings.IndexAny(version, "-+ "); idx != -1 {
		version = version[:idx]
	}

	parts := strings.Split(version, ".")
	if len(parts) == 0 {
		return "", "", ""
	}

	major = parts[0]

	if len(parts) > 1 {
		minor = parts[1]
	}

	if len(parts) > 2 {
		patch = parts[2]
	}

	return major, minor, patch
}

// computePeerIDUniqueKey computes a SeaHash-based unique key from a peer ID
// and network name combination.
func computePeerIDUniqueKey(peerID, networkName string) int64 {
	return flattener.SeaHashInt64(peerID + networkName)
}

// computeRPCMetaUniqueKey computes a SeaHash-based unique key from a root
// event ID.
func computeRPCMetaUniqueKey(rootEventID string) int64 {
	return flattener.SeaHashInt64(rootEventID)
}

// computeRPCMetaChildUniqueKey builds a composite unique key matching
// Vector's VRL format for RPC meta child tables:
// seahash(to_string(seahash(rootEventID)) + "_" + suffix + "_" + idx1 + "_" + idx2)
func computeRPCMetaChildUniqueKey(rootEventID, tableSuffix, idx1, idx2 string) int64 {
	rootKey := strconv.FormatInt(computeRPCMetaUniqueKey(rootEventID), 10)

	return flattener.SeaHashInt64(rootKey + "_" + tableSuffix + "_" + idx1 + "_" + idx2)
}

// peerIDMetadataProvider is implemented by libp2p additional data types
// that contain trace metadata with a peer ID.
type peerIDMetadataProvider interface {
	GetMetadata() *libp2pProto.TraceEventMetadata
}

// peerIDFromMetadata extracts the peer ID from the metadata section of client
// additional data via the provided accessor function.
func peerIDFromMetadata(
	event *xatu.DecoratedEvent,
	accessor func(*xatu.ClientMeta) peerIDMetadataProvider,
) string {
	if event == nil || event.GetMeta() == nil || event.GetMeta().GetClient() == nil {
		return ""
	}

	provider := accessor(event.GetMeta().GetClient())
	if provider == nil {
		return ""
	}

	traceMeta := provider.GetMetadata()
	if traceMeta == nil || traceMeta.GetPeerId() == nil {
		return ""
	}

	return traceMeta.GetPeerId().GetValue()
}

// wrappedStringValue extracts the string value from a wrapped string proto.
func wrappedStringValue(value interface{ GetValue() string }) string {
	if value == nil {
		return ""
	}

	return value.GetValue()
}
