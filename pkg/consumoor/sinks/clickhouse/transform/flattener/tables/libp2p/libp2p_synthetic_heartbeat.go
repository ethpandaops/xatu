package libp2p

import (
	"time"

	chProto "github.com/ClickHouse/ch-go/proto"
	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/flattener"
	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/metadata"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

var libp2pSyntheticHeartbeatEventNames = []xatu.Event_Name{
	xatu.Event_LIBP2P_TRACE_SYNTHETIC_HEARTBEAT,
}

func init() {
	flattener.MustRegister(flattener.NewStaticRoute(
		libp2pSyntheticHeartbeatTableName,
		libp2pSyntheticHeartbeatEventNames,
		func() flattener.ColumnarBatch { return newlibp2pSyntheticHeartbeatBatch() },
	))
}

func (b *libp2pSyntheticHeartbeatBatch) FlattenTo(
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
	b.appendPayload(event, meta)
	b.appendServerAdditionalData(event)
	b.rows++

	return nil
}

func (b *libp2pSyntheticHeartbeatBatch) appendRuntime(event *xatu.DecoratedEvent) {
	b.UpdatedDateTime.Append(time.Now())

	if ts := event.GetEvent().GetDateTime(); ts != nil {
		b.EventDateTime.Append(ts.AsTime())
	} else {
		b.EventDateTime.Append(time.Time{})
	}
}

func (b *libp2pSyntheticHeartbeatBatch) appendPayload(
	event *xatu.DecoratedEvent,
	meta *metadata.CommonMetadata,
) {
	payload := event.GetLibp2PTraceSyntheticHeartbeat()
	if payload == nil {
		b.RemotePeerIDUniqueKey.Append(0)
		b.RemoteMaddrs.Append("")
		b.LatencyMs.Append(chProto.Nullable[int64]{})
		b.Direction.Append("")
		b.Protocols.Append(nil)
		b.ConnectionAgeMs.Append(chProto.Nullable[int64]{})
		b.RemoteAgentImplementation.Append("")
		b.RemoteAgentVersion.Append("")
		b.RemoteAgentVersionMajor.Append("")
		b.RemoteAgentVersionMinor.Append("")
		b.RemoteAgentVersionPatch.Append("")
		b.RemoteAgentPlatform.Append("")
		b.RemoteIP.Append(chProto.Nullable[chProto.IPv6]{})
		b.RemotePort.Append(chProto.Nullable[uint16]{})

		return
	}

	// Parse remote address fields from maddrs.
	if maddrs := wrappedStringValue(payload.GetRemoteMaddrs()); maddrs != "" {
		b.RemoteMaddrs.Append(maddrs)

		addr := parseMultiAddr(maddrs)
		b.RemoteIP.Append(chProto.NewNullable[chProto.IPv6](flattener.ParseIPv6(addr.IP)))

		if addr.Port > 0 {
			b.RemotePort.Append(chProto.NewNullable[uint16](uint16(addr.Port))) //nolint:gosec // port fits uint16
		} else {
			b.RemotePort.Append(chProto.Nullable[uint16]{})
		}
	} else {
		b.RemoteMaddrs.Append("")
		b.RemoteIP.Append(chProto.Nullable[chProto.IPv6]{})
		b.RemotePort.Append(chProto.Nullable[uint16]{})
	}

	if latency := payload.GetLatencyMs(); latency != nil {
		b.LatencyMs.Append(chProto.NewNullable[int64](latency.GetValue()))
	} else {
		b.LatencyMs.Append(chProto.Nullable[int64]{})
	}

	// Map direction enum to string.
	if direction := payload.GetDirection(); direction != nil {
		switch direction.GetValue() {
		case 1:
			b.Direction.Append("inbound")
		case 2:
			b.Direction.Append("outbound")
		default:
			b.Direction.Append("unknown")
		}
	} else {
		b.Direction.Append("")
	}

	if protocols := payload.GetProtocols(); len(protocols) > 0 {
		b.Protocols.Append(protocols)
	} else {
		b.Protocols.Append(nil)
	}

	if connectionAgeNs := payload.GetConnectionAgeNs(); connectionAgeNs != nil {
		b.ConnectionAgeMs.Append(chProto.NewNullable[int64](connectionAgeNs.GetValue() / 1_000_000))
	} else {
		b.ConnectionAgeMs.Append(chProto.Nullable[int64]{})
	}

	// Parse agent version fields.
	if agentVersionStr := wrappedStringValue(payload.GetAgentVersion()); agentVersionStr != "" {
		parsed := parseAgentVersion(agentVersionStr)
		b.RemoteAgentImplementation.Append(parsed.Implementation)
		b.RemoteAgentVersion.Append(parsed.Version)
		b.RemoteAgentVersionMajor.Append(parsed.Major)
		b.RemoteAgentVersionMinor.Append(parsed.Minor)
		b.RemoteAgentVersionPatch.Append(parsed.Patch)
		b.RemoteAgentPlatform.Append(parsed.Platform)
	} else {
		b.RemoteAgentImplementation.Append("")
		b.RemoteAgentVersion.Append("")
		b.RemoteAgentVersionMajor.Append("")
		b.RemoteAgentVersionMinor.Append("")
		b.RemoteAgentVersionPatch.Append("")
		b.RemoteAgentPlatform.Append("")
	}

	// Compute remote_peer_id_unique_key.
	remotePeer := wrappedStringValue(payload.GetRemotePeer())
	networkName := meta.MetaNetworkName

	if remotePeer != "" {
		b.RemotePeerIDUniqueKey.Append(computePeerIDUniqueKey(remotePeer, networkName))
	} else {
		b.RemotePeerIDUniqueKey.Append(0)
	}
}

func (b *libp2pSyntheticHeartbeatBatch) appendServerAdditionalData(event *xatu.DecoratedEvent) {
	if event == nil || event.GetMeta() == nil || event.GetMeta().GetServer() == nil {
		b.RemoteGeoCity.Append("")
		b.RemoteGeoCountry.Append("")
		b.RemoteGeoCountryCode.Append("")
		b.RemoteGeoContinentCode.Append("")
		b.RemoteGeoLongitude.Append(chProto.Nullable[float64]{})
		b.RemoteGeoLatitude.Append(chProto.Nullable[float64]{})
		b.RemoteGeoAutonomousSystemNumber.Append(chProto.Nullable[uint32]{})
		b.RemoteGeoAutonomousSystemOrganization.Append(chProto.Nullable[string]{})

		return
	}

	additional := event.GetMeta().GetServer().GetLIBP2P_TRACE_SYNTHETIC_HEARTBEAT()
	if additional == nil {
		b.RemoteGeoCity.Append("")
		b.RemoteGeoCountry.Append("")
		b.RemoteGeoCountryCode.Append("")
		b.RemoteGeoContinentCode.Append("")
		b.RemoteGeoLongitude.Append(chProto.Nullable[float64]{})
		b.RemoteGeoLatitude.Append(chProto.Nullable[float64]{})
		b.RemoteGeoAutonomousSystemNumber.Append(chProto.Nullable[uint32]{})
		b.RemoteGeoAutonomousSystemOrganization.Append(chProto.Nullable[string]{})

		return
	}

	if peer := additional.GetPeer(); peer != nil {
		if geo := peer.GetGeo(); geo != nil {
			b.RemoteGeoCity.Append(geo.GetCity())
			b.RemoteGeoCountry.Append(geo.GetCountry())
			b.RemoteGeoCountryCode.Append(geo.GetCountryCode())
			b.RemoteGeoContinentCode.Append(geo.GetContinentCode())
			b.RemoteGeoLongitude.Append(chProto.NewNullable[float64](float64(geo.GetLongitude())))
			b.RemoteGeoLatitude.Append(chProto.NewNullable[float64](float64(geo.GetLatitude())))
			b.RemoteGeoAutonomousSystemNumber.Append(chProto.NewNullable[uint32](geo.GetAutonomousSystemNumber()))
			b.RemoteGeoAutonomousSystemOrganization.Append(chProto.NewNullable[string](geo.GetAutonomousSystemOrganization()))

			return
		}
	}

	b.RemoteGeoCity.Append("")
	b.RemoteGeoCountry.Append("")
	b.RemoteGeoCountryCode.Append("")
	b.RemoteGeoContinentCode.Append("")
	b.RemoteGeoLongitude.Append(chProto.Nullable[float64]{})
	b.RemoteGeoLatitude.Append(chProto.Nullable[float64]{})
	b.RemoteGeoAutonomousSystemNumber.Append(chProto.Nullable[uint32]{})
	b.RemoteGeoAutonomousSystemOrganization.Append(chProto.Nullable[string]{})
}
