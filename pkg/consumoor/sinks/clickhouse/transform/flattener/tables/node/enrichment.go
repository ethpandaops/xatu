package node

import (
	"net"
	"strings"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

// nodeGeo holds typed geo data extracted from server additional data.
type nodeGeo struct {
	City                         string
	Country                      string
	CountryCode                  string
	ContinentCode                string
	Longitude                    float64
	Latitude                     float64
	AutonomousSystemNumber       uint32
	AutonomousSystemOrganization string
}

func extractNodeGeo(event *xatu.DecoratedEvent, eventName xatu.Event_Name) nodeGeo {
	if event == nil || event.GetMeta() == nil || event.GetMeta().GetServer() == nil {
		return nodeGeo{}
	}

	server := event.GetMeta().GetServer()

	var geo *xatu.ServerMeta_Geo

	switch eventName {
	case xatu.Event_NODE_RECORD_CONSENSUS:
		if additional := server.GetNODE_RECORD_CONSENSUS(); additional != nil {
			geo = additional.GetGeo()
		}
	case xatu.Event_NODE_RECORD_EXECUTION:
		if additional := server.GetNODE_RECORD_EXECUTION(); additional != nil {
			geo = additional.GetGeo()
		}
	}

	if geo == nil {
		return nodeGeo{}
	}

	return nodeGeo{
		City:                         geo.GetCity(),
		Country:                      geo.GetCountry(),
		CountryCode:                  geo.GetCountryCode(),
		ContinentCode:                geo.GetContinentCode(),
		Longitude:                    geo.GetLongitude(),
		Latitude:                     geo.GetLatitude(),
		AutonomousSystemNumber:       geo.GetAutonomousSystemNumber(),
		AutonomousSystemOrganization: geo.GetAutonomousSystemOrganization(),
	}
}

func parseVersion(version string) (major, minor, patch string) {
	if version == "" {
		return "", "", ""
	}

	parts := strings.SplitN(version, ".", 3)
	if len(parts) >= 1 {
		major = strings.TrimPrefix(parts[0], "v")
	}

	if len(parts) >= 2 {
		minor = parts[1]
	}

	if len(parts) >= 3 {
		patch = parts[2]
		if idx := strings.IndexAny(patch, "-+ "); idx != -1 {
			patch = patch[:idx]
		}
	}

	return major, minor, patch
}

func normalizeIPv6Mapped(ip string) string {
	if ip == "" {
		return ""
	}

	parsed := net.ParseIP(ip)
	if parsed == nil {
		return ip
	}

	if ipv4 := parsed.To4(); ipv4 != nil {
		return "::ffff:" + ipv4.String()
	}

	return parsed.String()
}

func nodeStringValue(value interface{ GetValue() string }) string {
	if value == nil {
		return ""
	}

	return value.GetValue()
}
