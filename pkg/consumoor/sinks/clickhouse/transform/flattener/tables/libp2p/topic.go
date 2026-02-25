package libp2p

import "strings"

// parseTopic splits a gossipsub topic string of the form
// "/eth2/<fork_digest>/<topic_name>/<encoding>" into its parts.
func parseTopic(topic string) (layer, forkDigest, name, encoding string) {
	parts := strings.Split(topic, "/")
	if len(parts) != 5 {
		return "", "", "", ""
	}

	return parts[1], parts[2], parts[3], parts[4]
}

// agentInfo holds the parsed components of an agent version string.
type agentInfo struct {
	Implementation string
	Version        string
	Major          string
	Minor          string
	Patch          string
	Platform       string
}

// parseAgentVersion extracts implementation, version, and semver parts from
// an agent version string like "Lighthouse/v4.5.0-abc/linux-x86_64".
func parseAgentVersion(name string) agentInfo {
	if name == "" {
		return agentInfo{}
	}

	// Normalize teku double-prefix.
	name = strings.Replace(name, "teku/teku", "teku", 1)

	var info agentInfo

	parts := strings.SplitN(name, "/", 4)
	if len(parts) >= 1 {
		info.Implementation = strings.ToLower(parts[0])
	}

	if len(parts) >= 2 {
		info.Version = parts[1]
	}

	if len(parts) >= 3 && info.Implementation != "prysm" {
		info.Platform = parts[2]
	}

	// Parse semver from version string.
	ver := strings.TrimPrefix(strings.TrimPrefix(info.Version, "v"), "V")

	semParts := strings.SplitN(ver, ".", 3)
	if len(semParts) >= 1 {
		info.Major = semParts[0]
	}

	if len(semParts) >= 2 {
		info.Minor = semParts[1]
	}

	if len(semParts) >= 3 {
		info.Patch = semParts[2]
		if idx := strings.IndexAny(info.Patch, "-+ "); idx != -1 {
			info.Patch = info.Patch[:idx]
		}
	}

	return info
}

// parseMaddrs splits a multiaddr string like "/ip4/1.2.3.4/tcp/9000"
// into protocol, IP, transport protocol, and port components.
func parseMaddrs(maddrs string) (protocol, ip, transportProtocol string, port uint16) {
	parts := strings.Split(maddrs, "/")
	if len(parts) < 5 {
		return "", "", "", 0
	}

	protocol = parts[1]
	ip = parts[2]
	transportProtocol = parts[3]

	// Parse port, ignoring errors (returns 0 on failure).
	var p uint64

	for _, c := range parts[4] {
		if c < '0' || c > '9' {
			break
		}

		p = p*10 + uint64(c-'0')
	}

	if p > 65535 {
		p = 0
	}

	return protocol, ip, transportProtocol, uint16(p) //nolint:gosec // G115: port fits uint16.
}
