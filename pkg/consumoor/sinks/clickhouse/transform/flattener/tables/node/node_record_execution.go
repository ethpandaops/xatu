package node

import (
	"fmt"
	"strings"
	"time"

	"github.com/ClickHouse/ch-go/proto"
	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/flattener"
	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/metadata"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

func init() {
	flattener.MustRegister(flattener.NewStaticRoute(
		nodeRecordExecutionTableName,
		[]xatu.Event_Name{xatu.Event_NODE_RECORD_EXECUTION},
		func() flattener.ColumnarBatch {
			return newnodeRecordExecutionBatch()
		},
	))
}

func (b *nodeRecordExecutionBatch) FlattenTo(
	event *xatu.DecoratedEvent,
	meta *metadata.CommonMetadata,
) error {
	if event == nil || event.GetEvent() == nil {
		return nil
	}

	payload := event.GetNodeRecordExecution()
	if payload == nil {
		return fmt.Errorf("nil NodeRecordExecution payload: %w", flattener.ErrInvalidEvent)
	}

	// Parse agent version from the Name field.
	name := payload.GetName().GetValue()
	impl, version, major, minor, patch := parseAgentVersion(name)

	// Parse capabilities string into array by splitting on comma.
	caps := parseCapabilities(payload.GetCapabilities().GetValue())

	b.UpdatedDateTime.Append(time.Now())
	b.EventDateTime.Append(event.GetEvent().GetDateTime().AsTime())
	b.Enr.Append(payload.GetEnr().GetValue())
	b.Name.Append(name)
	b.Version.Append(version)
	b.VersionMajor.Append(major)
	b.VersionMinor.Append(minor)
	b.VersionPatch.Append(patch)
	b.Implementation.Append(impl)
	b.Capabilities.Append(caps)
	b.ProtocolVersion.Append(payload.GetProtocolVersion().GetValue())
	b.TotalDifficulty.Append(payload.GetTotalDifficulty().GetValue())
	b.Head.Append(payload.GetHead().GetValue())
	b.Genesis.Append(payload.GetGenesis().GetValue())
	b.ForkIDHash.Append(payload.GetForkIdHash().GetValue())
	b.ForkIDNext.Append(payload.GetForkIdNext().GetValue())
	b.NodeID.Append(payload.GetNodeId().GetValue())

	// IP is nullable IPv6.
	ipStr := payload.GetIp().GetValue()
	if ipStr != "" {
		b.IP.Append(proto.NewNullable(flattener.ParseIPv6(ipStr)))
	} else {
		b.IP.Append(proto.Nullable[proto.IPv6]{})
	}

	// Tcp, Udp are nullable UInt16.
	if tcp := payload.GetTcp(); tcp != nil {
		b.Tcp.Append(proto.NewNullable(uint16(tcp.GetValue()))) //nolint:gosec // G115: tcp port fits uint16.
	} else {
		b.Tcp.Append(proto.Nullable[uint16]{})
	}

	if udp := payload.GetUdp(); udp != nil {
		b.Udp.Append(proto.NewNullable(uint16(udp.GetValue()))) //nolint:gosec // G115: udp port fits uint16.
	} else {
		b.Udp.Append(proto.Nullable[uint16]{})
	}

	b.HasIpv6.Append(payload.GetHasIpv6().GetValue())

	// Geo fields from server-side enrichment (stored in meta).
	b.GeoCity.Append(meta.MetaClientGeoCity)
	b.GeoCountry.Append(meta.MetaClientGeoCountry)
	b.GeoCountryCode.Append(meta.MetaClientGeoCountryCode)
	b.GeoContinentCode.Append(meta.MetaClientGeoContinentCode)
	b.GeoLongitude.Append(proto.NewNullable(meta.MetaClientGeoLongitude))
	b.GeoLatitude.Append(proto.NewNullable(meta.MetaClientGeoLatitude))
	b.GeoAutonomousSystemNumber.Append(proto.NewNullable(meta.MetaClientGeoAutonomousSystemNumber))
	b.GeoAutonomousSystemOrganization.Append(proto.NewNullable(meta.MetaClientGeoAutonomousSystemOrganization))

	b.appendMetadata(meta)
	b.rows++

	return nil
}

// parseCapabilities splits a comma-separated capabilities string into
// a string slice suitable for the Array(String) ClickHouse column.
func parseCapabilities(s string) []string {
	if s == "" {
		return []string{}
	}

	parts := strings.Split(s, ",")

	result := make([]string, 0, len(parts))
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}

	return result
}
