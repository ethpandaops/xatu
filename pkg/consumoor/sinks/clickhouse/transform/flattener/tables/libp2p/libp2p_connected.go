package libp2p

import (
	"time"

	chProto "github.com/ClickHouse/ch-go/proto"
	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/flattener"
	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/metadata"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

var libp2pConnectedEventNames = []xatu.Event_Name{
	xatu.Event_LIBP2P_TRACE_CONNECTED,
}

func init() {
	flattener.MustRegister(flattener.NewStaticRoute(
		libp2pConnectedTableName,
		libp2pConnectedEventNames,
		func() flattener.ColumnarBatch { return newlibp2pConnectedBatch() },
	))
}

func (b *libp2pConnectedBatch) FlattenTo(
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

func (b *libp2pConnectedBatch) appendRuntime(event *xatu.DecoratedEvent) {
	b.UpdatedDateTime.Append(time.Now())

	if ts := event.GetEvent().GetDateTime(); ts != nil {
		b.EventDateTime.Append(ts.AsTime())
	} else {
		b.EventDateTime.Append(time.Time{})
	}
}

func (b *libp2pConnectedBatch) appendPayload(
	event *xatu.DecoratedEvent,
	meta *metadata.CommonMetadata,
) {
	payload := event.GetLibp2PTraceConnected()
	if payload == nil {
		b.Direction.Append("")
		b.Opened.Append(time.Time{})
		b.Transient.Append(false)
		b.RemoteProtocol.Append("")
		b.RemoteIP.Append(chProto.Nullable[chProto.IPv6]{})
		b.RemoteTransportProtocol.Append("")
		b.RemotePort.Append(0)
		b.RemoteAgentImplementation.Append("")
		b.RemoteAgentVersion.Append("")
		b.RemoteAgentVersionMajor.Append("")
		b.RemoteAgentVersionMinor.Append("")
		b.RemoteAgentVersionPatch.Append("")
		b.RemoteAgentPlatform.Append("")
		b.RemotePeerIDUniqueKey.Append(0)

		return
	}

	remotePeer := wrappedStringValue(payload.GetRemotePeer())
	b.Direction.Append(wrappedStringValue(payload.GetDirection()))

	if ts := payload.GetOpened(); ts != nil {
		b.Opened.Append(ts.AsTime())
	} else {
		b.Opened.Append(time.Time{})
	}

	if transient := payload.GetTransient(); transient != nil { //nolint:staticcheck // SA1019: field still needed for backward compat
		b.Transient.Append(transient.GetValue())
	} else {
		b.Transient.Append(false)
	}

	// Parse remote address fields from maddrs.
	if maddrs := wrappedStringValue(payload.GetRemoteMaddrs()); maddrs != "" {
		addr := parseMultiAddr(maddrs)
		b.RemoteProtocol.Append(addr.Protocol)
		b.RemoteIP.Append(chProto.NewNullable[chProto.IPv6](flattener.ParseIPv6(addr.IP)))
		b.RemoteTransportProtocol.Append(addr.Transport)

		if addr.Port > 0 {
			b.RemotePort.Append(uint16(addr.Port)) //nolint:gosec // port fits uint16
		} else {
			b.RemotePort.Append(0)
		}
	} else {
		b.RemoteProtocol.Append("")
		b.RemoteIP.Append(chProto.Nullable[chProto.IPv6]{})
		b.RemoteTransportProtocol.Append("")
		b.RemotePort.Append(0)
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
	networkName := meta.MetaNetworkName

	if remotePeer != "" {
		b.RemotePeerIDUniqueKey.Append(computePeerIDUniqueKey(remotePeer, networkName))
	} else {
		b.RemotePeerIDUniqueKey.Append(0)
	}
}

func (b *libp2pConnectedBatch) appendServerAdditionalData(event *xatu.DecoratedEvent) {
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

	additional := event.GetMeta().GetServer().GetLIBP2P_TRACE_CONNECTED()
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
