package libp2p

import (
	"time"

	chProto "github.com/ClickHouse/ch-go/proto"
	"github.com/ethpandaops/xatu/pkg/consumoor/route"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

var libp2pIdentifyEventNames = []xatu.Event_Name{
	xatu.Event_LIBP2P_TRACE_IDENTIFY,
}

func init() {
	r, err := route.NewStaticRoute(
		libp2pIdentifyTableName,
		libp2pIdentifyEventNames,
		func() route.ColumnarBatch { return newlibp2pIdentifyBatch() },
	)
	if err != nil {
		route.RecordError(err)

		return
	}

	if err := route.Register(r); err != nil {
		route.RecordError(err)
	}
}

func (b *libp2pIdentifyBatch) FlattenTo(
	event *xatu.DecoratedEvent,
) error {
	if event == nil || event.GetEvent() == nil {
		return nil
	}

	b.appendRuntime(event)
	b.appendMetadata(event)
	b.appendPayload(event)
	b.appendServerAdditionalData(event)
	b.rows++

	return nil
}

func (b *libp2pIdentifyBatch) appendRuntime(event *xatu.DecoratedEvent) {
	b.UpdatedDateTime.Append(time.Now())

	if ts := event.GetEvent().GetDateTime(); ts != nil {
		b.EventDateTime.Append(ts.AsTime())
	} else {
		b.EventDateTime.Append(time.Time{})
	}
}

func (b *libp2pIdentifyBatch) appendPayload(event *xatu.DecoratedEvent) {
	payload := event.GetLibp2PTraceIdentify()
	if payload == nil {
		b.RemotePeerIDUniqueKey.Append(0)
		b.Success.Append(false)
		b.Error.Append(chProto.Nullable[string]{})
		b.RemoteProtocol.Append("")
		b.RemoteTransportProtocol.Append("")
		b.RemotePort.Append(0)
		b.RemoteIP.Append(chProto.Nullable[chProto.IPv6]{})
		b.RemoteAgentImplementation.Append("")
		b.RemoteAgentVersion.Append("")
		b.RemoteAgentVersionMajor.Append("")
		b.RemoteAgentVersionMinor.Append("")
		b.RemoteAgentVersionPatch.Append("")
		b.RemoteAgentPlatform.Append("")
		b.ProtocolVersion.Append("")
		b.Protocols.Append([]string{})
		b.ListenAddrs.Append([]string{})
		b.ObservedAddr.Append("")
		b.Transport.Append("")
		b.Security.Append("")
		b.Muxer.Append("")
		b.Direction.Append("")
		b.RemoteMultiaddr.Append("")

		return
	}

	b.Success.Append(payload.GetSuccess().GetValue())

	// Error (nullable string).
	if errVal := wrappedStringValue(payload.GetError()); errVal != "" {
		b.Error.Append(chProto.NewNullable[string](errVal))
	} else {
		b.Error.Append(chProto.Nullable[string]{})
	}

	// Parse remote address fields from remote_multiaddr.
	remoteMultiaddr := wrappedStringValue(payload.GetRemoteMultiaddr())
	b.RemoteMultiaddr.Append(remoteMultiaddr)

	if remoteMultiaddr != "" {
		addr := parseMultiAddr(remoteMultiaddr)
		b.RemoteProtocol.Append(addr.Protocol)
		b.RemoteIP.Append(chProto.NewNullable[chProto.IPv6](route.ParseIPv6(addr.IP)))
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

	b.ProtocolVersion.Append(wrappedStringValue(payload.GetProtocolVersion()))
	b.Protocols.Append(payload.GetProtocols())
	b.ListenAddrs.Append(payload.GetListenAddrs())
	b.ObservedAddr.Append(wrappedStringValue(payload.GetObservedAddr()))
	b.Transport.Append(wrappedStringValue(payload.GetTransport()))
	b.Security.Append(wrappedStringValue(payload.GetSecurity()))
	b.Muxer.Append(wrappedStringValue(payload.GetMuxer()))
	b.Direction.Append(wrappedStringValue(payload.GetDirection()))

	// Compute remote_peer_id_unique_key: prefer payload peer ID, fall back to metadata.
	remotePeer := wrappedStringValue(payload.GetRemotePeer())
	if remotePeer == "" {
		remotePeer = peerIDFromMetadata(event, func(c *xatu.ClientMeta) peerIDMetadataProvider {
			return c.GetLibp2PTraceIdentify()
		})
	}

	networkName := event.GetMeta().GetClient().GetEthereum().GetNetwork().GetName()
	b.RemotePeerIDUniqueKey.Append(computePeerIDUniqueKey(remotePeer, networkName))
}

func (b *libp2pIdentifyBatch) appendServerAdditionalData(event *xatu.DecoratedEvent) {
	if event == nil || event.GetMeta() == nil || event.GetMeta().GetServer() == nil {
		b.appendEmptyGeo()

		return
	}

	additional := event.GetMeta().GetServer().GetLIBP2P_TRACE_IDENTIFY()
	if additional == nil {
		b.appendEmptyGeo()

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

	b.appendEmptyGeo()
}

func (b *libp2pIdentifyBatch) appendEmptyGeo() {
	b.RemoteGeoCity.Append("")
	b.RemoteGeoCountry.Append("")
	b.RemoteGeoCountryCode.Append("")
	b.RemoteGeoContinentCode.Append("")
	b.RemoteGeoLongitude.Append(chProto.Nullable[float64]{})
	b.RemoteGeoLatitude.Append(chProto.Nullable[float64]{})
	b.RemoteGeoAutonomousSystemNumber.Append(chProto.Nullable[uint32]{})
	b.RemoteGeoAutonomousSystemOrganization.Append(chProto.Nullable[string]{})
}
