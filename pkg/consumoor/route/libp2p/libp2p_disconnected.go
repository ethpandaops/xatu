package libp2p

import (
	"fmt"
	"time"

	chProto "github.com/ClickHouse/ch-go/proto"
	"github.com/ethpandaops/xatu/pkg/consumoor/route"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

var libp2pDisconnectedEventNames = []xatu.Event_Name{
	xatu.Event_LIBP2P_TRACE_DISCONNECTED,
}

func init() {
	r, err := route.NewStaticRoute(
		libp2pDisconnectedTableName,
		libp2pDisconnectedEventNames,
		func() route.ColumnarBatch { return newlibp2pDisconnectedBatch() },
	)
	if err != nil {
		route.RecordError(err)

		return
	}

	if err := route.Register(r); err != nil {
		route.RecordError(err)
	}
}

func (b *libp2pDisconnectedBatch) FlattenTo(
	event *xatu.DecoratedEvent,
) error {
	if event == nil || event.GetEvent() == nil {
		return nil
	}

	if event.GetLibp2PTraceDisconnected() == nil {
		return fmt.Errorf("nil libp2p_trace_disconnected payload: %w", route.ErrInvalidEvent)
	}

	if err := b.validate(event); err != nil {
		return err
	}

	b.appendRuntime(event)
	b.appendMetadata(event)
	b.appendPayload(event)
	b.appendServerAdditionalData(event)
	b.rows++

	return nil
}

func (b *libp2pDisconnectedBatch) validate(event *xatu.DecoratedEvent) error {
	payload := event.GetLibp2PTraceDisconnected()

	if payload.GetRemotePeer() == nil {
		return fmt.Errorf("nil RemotePeer: %w", route.ErrInvalidEvent)
	}

	return nil
}

func (b *libp2pDisconnectedBatch) appendRuntime(event *xatu.DecoratedEvent) {
	b.UpdatedDateTime.Append(time.Now())

	if ts := event.GetEvent().GetDateTime(); ts != nil {
		b.EventDateTime.Append(ts.AsTime())
	} else {
		b.EventDateTime.Append(time.Time{})
	}
}

func (b *libp2pDisconnectedBatch) appendPayload(
	event *xatu.DecoratedEvent,
) {
	payload := event.GetLibp2PTraceDisconnected()
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

	// Compute remote_peer_id_unique_key.
	networkName := event.GetMeta().GetClient().GetEthereum().GetNetwork().GetName()

	if remotePeer != "" {
		b.RemotePeerIDUniqueKey.Append(computePeerIDUniqueKey(remotePeer, networkName))
	} else {
		b.RemotePeerIDUniqueKey.Append(0)
	}
}

func (b *libp2pDisconnectedBatch) appendServerAdditionalData(event *xatu.DecoratedEvent) {
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

	additional := event.GetMeta().GetServer().GetLIBP2P_TRACE_DISCONNECTED()
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
