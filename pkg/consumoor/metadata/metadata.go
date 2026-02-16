package metadata

import (
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

// CommonMetadata holds shared metadata fields extracted from a
// DecoratedEvent. This replaces the VRL xatu_server_events_meta
// transform that was 200+ lines of string manipulation.
type CommonMetadata struct {
	// Client identification
	MetaClientName           string
	MetaClientID             string
	MetaClientVersion        string
	MetaClientImplementation string
	MetaClientOS             string
	MetaClientClockDrift     uint64
	MetaClientModuleName     string

	// Client IP (IPv6 normalized)
	MetaClientIP string

	// Client geo (from server-side lookup)
	MetaClientGeoCity                         string
	MetaClientGeoCountry                      string
	MetaClientGeoCountryCode                  string
	MetaClientGeoContinentCode                string
	MetaClientGeoLongitude                    float64
	MetaClientGeoLatitude                     float64
	MetaClientGeoAutonomousSystemNumber       uint32
	MetaClientGeoAutonomousSystemOrganization string

	// Network
	MetaNetworkID   uint64
	MetaNetworkName string

	// Consensus client
	MetaConsensusImplementation string
	MetaConsensusVersion        string
	MetaConsensusVersionMajor   string
	MetaConsensusVersionMinor   string
	MetaConsensusVersionPatch   string

	// Execution client
	MetaExecutionImplementation string
	MetaExecutionVersion        string
	MetaExecutionVersionMajor   string
	MetaExecutionVersionMinor   string
	MetaExecutionVersionPatch   string
	MetaExecutionForkIDHash     string
	MetaExecutionForkIDNext     string

	// Labels
	MetaLabels map[string]string
}

// Extract builds CommonMetadata from a DecoratedEvent using typed
// proto accessors. No JSON parsing or string field access needed.
func Extract(event *xatu.DecoratedEvent) *CommonMetadata {
	m := &CommonMetadata{}

	if event.GetMeta() == nil {
		return m
	}

	meta := event.GetMeta()

	// Client metadata
	if client := meta.GetClient(); client != nil {
		m.MetaClientName = client.GetName()
		m.MetaClientID = client.GetId()
		m.MetaClientVersion = client.GetVersion()
		m.MetaClientImplementation = client.GetImplementation()
		m.MetaClientOS = client.GetOs()
		m.MetaClientClockDrift = client.GetClockDrift()
		m.MetaClientModuleName = client.GetModuleName().String()
		m.MetaLabels = client.GetLabels()

		// Ethereum network metadata
		if eth := client.GetEthereum(); eth != nil {
			if network := eth.GetNetwork(); network != nil {
				m.MetaNetworkID = network.GetId()
				m.MetaNetworkName = network.GetName()
			}

			// Consensus client version parsing
			if consensus := eth.GetConsensus(); consensus != nil {
				m.MetaConsensusImplementation = consensus.GetImplementation()

				rawVersion := consensus.GetVersion()
				m.MetaConsensusVersion = rawVersion
				m.MetaConsensusVersionMajor,
					m.MetaConsensusVersionMinor,
					m.MetaConsensusVersionPatch = parseVersion(rawVersion)
			}

			// Execution client metadata
			if execution := eth.GetExecution(); execution != nil {
				m.MetaExecutionImplementation = execution.GetImplementation()
				m.MetaExecutionVersion = execution.GetVersion()
				m.MetaExecutionVersionMajor = execution.GetVersionMajor()
				m.MetaExecutionVersionMinor = execution.GetVersionMinor()
				m.MetaExecutionVersionPatch = execution.GetVersionPatch()

				if forkID := execution.GetForkId(); forkID != nil {
					m.MetaExecutionForkIDHash = forkID.GetHash()
					m.MetaExecutionForkIDNext = forkID.GetNext()
				}
			}
		}
	}

	// Server-side metadata (IP, geo)
	if server := meta.GetServer(); server != nil {
		if serverClient := server.GetClient(); serverClient != nil {
			m.MetaClientIP = normalizeIP(serverClient.GetIP())

			if geo := serverClient.GetGeo(); geo != nil {
				m.MetaClientGeoCity = geo.GetCity()
				m.MetaClientGeoCountry = geo.GetCountry()
				m.MetaClientGeoCountryCode = geo.GetCountryCode()
				m.MetaClientGeoContinentCode = geo.GetContinentCode()
				m.MetaClientGeoLongitude = geo.GetLongitude()
				m.MetaClientGeoLatitude = geo.GetLatitude()
				m.MetaClientGeoAutonomousSystemNumber = geo.GetAutonomousSystemNumber()
				m.MetaClientGeoAutonomousSystemOrganization = geo.GetAutonomousSystemOrganization()
			}
		}
	}

	return m
}

// ToMap returns the common metadata as a flat map suitable for merging
// into a ClickHouse row.
func (m *CommonMetadata) ToMap() map[string]any {
	row := map[string]any{
		"meta_client_name":           m.MetaClientName,
		"meta_client_id":             m.MetaClientID,
		"meta_client_version":        m.MetaClientVersion,
		"meta_client_implementation": m.MetaClientImplementation,
		"meta_client_os":             m.MetaClientOS,
		"meta_client_clock_drift":    m.MetaClientClockDrift,
		"meta_client_module_name":    m.MetaClientModuleName,
		"meta_client_ip":             m.MetaClientIP,

		"meta_client_geo_city":                           m.MetaClientGeoCity,
		"meta_client_geo_country":                        m.MetaClientGeoCountry,
		"meta_client_geo_country_code":                   m.MetaClientGeoCountryCode,
		"meta_client_geo_continent_code":                 m.MetaClientGeoContinentCode,
		"meta_client_geo_longitude":                      m.MetaClientGeoLongitude,
		"meta_client_geo_latitude":                       m.MetaClientGeoLatitude,
		"meta_client_geo_autonomous_system_number":       m.MetaClientGeoAutonomousSystemNumber,
		"meta_client_geo_autonomous_system_organization": m.MetaClientGeoAutonomousSystemOrganization,

		"meta_network_id":   m.MetaNetworkID,
		"meta_network_name": m.MetaNetworkName,

		"meta_consensus_implementation": m.MetaConsensusImplementation,
		"meta_consensus_version":        m.MetaConsensusVersion,
		"meta_consensus_version_major":  m.MetaConsensusVersionMajor,
		"meta_consensus_version_minor":  m.MetaConsensusVersionMinor,
		"meta_consensus_version_patch":  m.MetaConsensusVersionPatch,

		"meta_execution_implementation": m.MetaExecutionImplementation,
		"meta_execution_version":        m.MetaExecutionVersion,
		"meta_execution_version_major":  m.MetaExecutionVersionMajor,
		"meta_execution_version_minor":  m.MetaExecutionVersionMinor,
		"meta_execution_version_patch":  m.MetaExecutionVersionPatch,
		"meta_execution_fork_id_hash":   m.MetaExecutionForkIDHash,
		"meta_execution_fork_id_next":   m.MetaExecutionForkIDNext,
	}

	// Labels as map column
	if m.MetaLabels != nil {
		row["meta_labels"] = m.MetaLabels
	} else {
		row["meta_labels"] = map[string]string{}
	}

	return row
}

// normalizeIP converts IPv4 addresses to IPv6-mapped form for
// consistent ClickHouse storage. IPv6 addresses pass through unchanged.
func normalizeIP(ip string) string {
	if ip == "" {
		return ""
	}

	parsed := net.ParseIP(ip)
	if parsed == nil {
		return ip
	}

	// If it's IPv4, convert to IPv6-mapped form
	if parsed.To4() != nil {
		return fmt.Sprintf("::ffff:%s", parsed.To4().String())
	}

	return parsed.String()
}

// parseVersion extracts major, minor, patch from a version string.
// Handles formats like "v1.2.3", "Lighthouse/v4.5.6-abcdef/x86_64-linux",
// "teku/teku/v1.2.3".
func parseVersion(raw string) (major, minor, patch string) {
	if raw == "" {
		return "", "", ""
	}

	// Find the segment containing a version (starts with "v" or "V")
	// by splitting on "/" and finding the first segment with a "v" prefix.
	version := raw

	if strings.Contains(version, "/") {
		parts := strings.Split(version, "/")

		found := false

		for _, p := range parts {
			if strings.HasPrefix(p, "v") || strings.HasPrefix(p, "V") {
				version = p
				found = true

				break
			}
		}

		if !found {
			// Fall back to last segment
			version = parts[len(parts)-1]
		}
	}

	version = strings.TrimPrefix(version, "v")
	version = strings.TrimPrefix(version, "V")

	// Handle suffixes like "4.5.6-abcdef" by stripping after "-" or "+"
	if idx := strings.IndexAny(version, "-+"); idx != -1 {
		version = version[:idx]
	}

	parts := strings.SplitN(version, ".", 3)

	if len(parts) >= 1 {
		if _, err := strconv.Atoi(parts[0]); err == nil {
			major = parts[0]
		}
	}

	if len(parts) >= 2 {
		if _, err := strconv.Atoi(parts[1]); err == nil {
			minor = parts[1]
		}
	}

	if len(parts) >= 3 {
		if _, err := strconv.Atoi(parts[2]); err == nil {
			patch = parts[2]
		}
	}

	return major, minor, patch
}
