package node

import (
	"strings"
	"time"

	chProto "github.com/ClickHouse/ch-go/proto"
	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/flattener"
	catalog "github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/flattener/tables/catalog"
	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/metadata"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

var nodeRecordExecutionEventNames = []xatu.Event_Name{
	xatu.Event_NODE_RECORD_EXECUTION,
}

func init() {
	catalog.MustRegister(flattener.NewStaticRoute(
		nodeRecordExecutionTableName,
		nodeRecordExecutionEventNames,
		func() flattener.ColumnarBatch { return newnodeRecordExecutionBatch() },
	))
}

func (b *nodeRecordExecutionBatch) FlattenTo(
	event *xatu.DecoratedEvent,
	meta *metadata.CommonMetadata,
) error {
	if event == nil || event.GetEvent() == nil {
		return nil
	}

	if meta == nil {
		meta = metadata.Extract(event)
	}

	b.appendRuntime(event)
	b.appendMetadata(meta)
	b.appendPayload(event)
	b.appendServerGeo(event)
	b.rows++

	return nil
}

func (b *nodeRecordExecutionBatch) appendRuntime(event *xatu.DecoratedEvent) {
	b.UpdatedDateTime.Append(time.Now())

	if ts := event.GetEvent().GetDateTime(); ts != nil {
		b.EventDateTime.Append(ts.AsTime())
	} else {
		b.EventDateTime.Append(time.Time{})
	}
}

func (b *nodeRecordExecutionBatch) appendPayload(event *xatu.DecoratedEvent) {
	execution := event.GetNodeRecordExecution()
	if execution == nil {
		b.Enr.Append("")
		b.Name.Append("")
		b.Version.Append("")
		b.VersionMajor.Append("")
		b.VersionMinor.Append("")
		b.VersionPatch.Append("")
		b.Implementation.Append("")
		b.Capabilities.Append(nil)
		b.ProtocolVersion.Append("")
		b.TotalDifficulty.Append("")
		b.Head.Append("")
		b.Genesis.Append("")
		b.ForkIDHash.Append("")
		b.ForkIDNext.Append("")
		b.NodeID.Append("")
		b.IP.Append(chProto.Nullable[chProto.IPv6]{})
		b.Tcp.Append(chProto.Nullable[uint16]{})
		b.Udp.Append(chProto.Nullable[uint16]{})
		b.HasIpv6.Append(false)

		return
	}

	b.Enr.Append(nodeStringValue(execution.GetEnr()))

	// Name and parsed implementation/version.
	name := nodeStringValue(execution.GetName())
	b.Name.Append(name)

	implementation, version := parseExecutionName(name)
	major, minor, patch := parseVersion(version)

	b.Version.Append(version)
	b.VersionMajor.Append(major)
	b.VersionMinor.Append(minor)
	b.VersionPatch.Append(patch)
	b.Implementation.Append(implementation)

	// Capabilities is ColArr[string].
	capabilities := nodeStringValue(execution.GetCapabilities())
	if capabilities == "" {
		b.Capabilities.Append([]string{})
	} else {
		b.Capabilities.Append(strings.Split(capabilities, ","))
	}

	b.ProtocolVersion.Append(nodeStringValue(execution.GetProtocolVersion()))
	b.TotalDifficulty.Append(nodeStringValue(execution.GetTotalDifficulty()))
	b.Head.Append(nodeStringValue(execution.GetHead()))
	b.Genesis.Append(nodeStringValue(execution.GetGenesis()))
	b.ForkIDHash.Append(nodeStringValue(execution.GetForkIdHash()))
	b.ForkIDNext.Append(nodeStringValue(execution.GetForkIdNext()))
	b.NodeID.Append(nodeStringValue(execution.GetNodeId()))

	// IP is Nullable[IPv6].
	ip := nodeStringValue(execution.GetIp())
	normalizedIP := normalizeIPv6Mapped(ip)

	if normalizedIP != "" {
		b.IP.Append(chProto.NewNullable[chProto.IPv6](flattener.ParseIPv6(normalizedIP)))
	} else {
		b.IP.Append(chProto.Nullable[chProto.IPv6]{})
	}

	// Tcp, Udp are Nullable[uint16].
	if tcp := execution.GetTcp(); tcp != nil {
		b.Tcp.Append(chProto.NewNullable[uint16](uint16(tcp.GetValue()))) //nolint:gosec // proto uint32 narrowed to uint16 target field
	} else {
		b.Tcp.Append(chProto.Nullable[uint16]{})
	}

	if udp := execution.GetUdp(); udp != nil {
		b.Udp.Append(chProto.NewNullable[uint16](uint16(udp.GetValue()))) //nolint:gosec // proto uint32 narrowed to uint16 target field
	} else {
		b.Udp.Append(chProto.Nullable[uint16]{})
	}

	if hasIpv6 := execution.GetHasIpv6(); hasIpv6 != nil {
		b.HasIpv6.Append(hasIpv6.GetValue())
	} else {
		b.HasIpv6.Append(false)
	}
}

func (b *nodeRecordExecutionBatch) appendServerGeo(event *xatu.DecoratedEvent) {
	geo := extractNodeGeo(event, xatu.Event_NODE_RECORD_EXECUTION)

	b.GeoCity.Append(geo.City)
	b.GeoCountry.Append(geo.Country)
	b.GeoCountryCode.Append(geo.CountryCode)
	b.GeoContinentCode.Append(geo.ContinentCode)

	b.GeoLongitude.Append(chProto.NewNullable[float64](geo.Longitude))
	b.GeoLatitude.Append(chProto.NewNullable[float64](geo.Latitude))
	b.GeoAutonomousSystemNumber.Append(chProto.NewNullable[uint32](geo.AutonomousSystemNumber))
	b.GeoAutonomousSystemOrganization.Append(chProto.NewNullable[string](geo.AutonomousSystemOrganization))
}

func parseExecutionName(name string) (implementation, version string) {
	if name == "" {
		return "", ""
	}

	parts := strings.SplitN(name, "/", 4)
	if len(parts) <= 1 {
		return "", ""
	}

	implementation = strings.ToLower(parts[0])

	if implementation == "coregeth" && len(parts) > 2 {
		version = parts[2]
	} else {
		version = parts[1]
	}

	return implementation, version
}
