package libp2p

import (
	"fmt"
	"hash/fnv"
	"strconv"
	"strings"

	"github.com/ethpandaops/xatu/pkg/consumoor/metadata"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

const maxInt64AsUint = ^uint64(0) >> 1

type agentVersion struct {
	Implementation string
	Version        string
	Major          string
	Minor          string
	Patch          string
	Platform       string
}

// EnrichFields applies libp2p-specific derived columns.
func EnrichFields(event *xatu.DecoratedEvent, _ *metadata.CommonMetadata, row map[string]any) error {
	eventName := ""
	if event != nil && event.GetEvent() != nil {
		eventName = event.GetEvent().GetName().String()
	}

	networkName := asString(row["meta_network_name"])
	remotePeer := firstNonEmpty(row, "remote_peer", "peer_id")

	if remotePeer != "" {
		if _, ok := row["remote_peer_id_unique_key"]; !ok {
			row["remote_peer_id_unique_key"] = hashKey(remotePeer + networkName)
		}
	}

	if peerID := firstNonEmpty(row, "peer_id", "remote_peer"); peerID != "" {
		if _, ok := row["peer_id_unique_key"]; !ok {
			row["peer_id_unique_key"] = hashKey(peerID + networkName)
		}
	}

	if topic := firstNonEmpty(row, "topic", "topic_id", "metadata_topic"); topic != "" {
		forkDigest, topicName, topicEncoding := parseLibP2PTopic(topic)
		if _, ok := row["topic_fork_digest"]; !ok {
			row["topic_fork_digest"] = forkDigest
		}

		if _, ok := row["topic_name"]; !ok {
			row["topic_name"] = topicName
		}

		if _, ok := row["topic_encoding"]; !ok {
			row["topic_encoding"] = topicEncoding
		}
	}

	if maddrs := firstNonEmpty(row, "remote_maddrs", "meta_remote_maddrs"); maddrs != "" {
		protocol, ip, transport, port := parseMultiAddr(maddrs)
		if protocol != "" {
			setDefault(row, "remote_protocol", protocol)
		}

		if ip != "" {
			setDefault(row, "remote_ip", ip)
		}

		if transport != "" {
			setDefault(row, "remote_transport_protocol", transport)
		}

		if port > 0 {
			setDefault(row, "remote_port", port)
		}
	}

	if userAgent := firstNonEmpty(row, "peer_user_agent", "user_agent"); userAgent != "" {
		parsed := parseAgentVersion(userAgent)
		setDefault(row, "peer_implementation", parsed.Implementation)
		setDefault(row, "peer_version", parsed.Version)
		setDefault(row, "peer_version_major", parsed.Major)
		setDefault(row, "peer_version_minor", parsed.Minor)
		setDefault(row, "peer_version_patch", parsed.Patch)
		setDefault(row, "peer_platform", parsed.Platform)
	}

	if agentVersion := firstNonEmpty(row, "agent_version", "remote_agent_version"); agentVersion != "" {
		parsed := parseAgentVersion(agentVersion)
		setDefault(row, "remote_agent_implementation", parsed.Implementation)
		setDefault(row, "remote_agent_version", parsed.Version)
		setDefault(row, "remote_agent_version_major", parsed.Major)
		setDefault(row, "remote_agent_version_minor", parsed.Minor)
		setDefault(row, "remote_agent_version_patch", parsed.Patch)
		setDefault(row, "remote_agent_platform", parsed.Platform)
	}

	if eventName == xatu.Event_LIBP2P_TRACE_SYNTHETIC_HEARTBEAT.String() {
		if direction, ok := toInt(row["direction"]); ok {
			switch direction {
			case 1:
				row["direction"] = "inbound"
			case 2:
				row["direction"] = "outbound"
			default:
				row["direction"] = "unknown"
			}
		}

		if ns, ok := toInt(row["connection_age_ns"]); ok {
			row["connection_age_ms"] = ns / 1_000_000
		}
	}

	return nil
}

func parseLibP2PTopic(topic string) (forkDigest, topicName, topicEncoding string) {
	parts := strings.Split(topic, "/")
	if len(parts) < 5 {
		return "", "", ""
	}

	// /eth2/{fork_digest}/{topic_name}/{encoding}
	return parts[2], parts[3], parts[4]
}

func parseMultiAddr(addr string) (protocol, ip, transport string, port uint64) {
	parts := strings.Split(addr, "/")
	if len(parts) < 5 {
		return "", "", "", 0
	}

	port, _ = strconv.ParseUint(parts[4], 10, 64)

	return parts[1], parts[2], parts[3], port
}

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

func splitSemver(version string) (major, minor, patch string) {
	if version == "" {
		return "", "", ""
	}

	version = strings.TrimPrefix(version, "v")
	if idx := strings.IndexAny(version, "-+"); idx != -1 {
		version = version[:idx]
	}

	parts := strings.Split(version, ".")
	if len(parts) == 0 {
		return "", "", ""
	}

	major = parts[0]
	minor = ""
	patch = ""

	if len(parts) > 1 {
		minor = parts[1]
	}

	if len(parts) > 2 {
		patch = parts[2]
	}

	return major, minor, patch
}

func firstNonEmpty(row map[string]any, keys ...string) string {
	for _, key := range keys {
		if value := asString(row[key]); value != "" {
			return value
		}
	}

	return ""
}

func setDefault(row map[string]any, key string, value any) {
	if _, exists := row[key]; exists {
		return
	}

	if value == nil {
		return
	}

	v, ok := value.(string)
	if ok && v == "" {
		return
	}

	row[key] = value
}

func asString(value any) string {
	switch v := value.(type) {
	case string:
		return v
	case fmt.Stringer:
		return v.String()
	case []byte:
		return string(v)
	default:
		if v == nil {
			return ""
		}

		return fmt.Sprint(v)
	}
}

func toInt(value any) (int64, bool) {
	switch v := value.(type) {
	case int:
		return int64(v), true
	case int8:
		return int64(v), true
	case int16:
		return int64(v), true
	case int32:
		return int64(v), true
	case int64:
		return v, true
	case uint:
		if uint64(v) > maxInt64AsUint {
			return 0, false
		}

		//nolint:gosec // guarded by maxInt64AsUint check above.
		return int64(v), true
	case uint8:
		return int64(v), true
	case uint16:
		return int64(v), true
	case uint32:
		return int64(v), true
	case uint64:
		if v > maxInt64AsUint {
			return 0, false
		}

		return int64(v), true
	case float32:
		return int64(v), true
	case float64:
		return int64(v), true
	case string:
		i, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return 0, false
		}

		return i, true
	default:
		return 0, false
	}
}

func hashKey(s string) uint64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte(s))

	return h.Sum64()
}
