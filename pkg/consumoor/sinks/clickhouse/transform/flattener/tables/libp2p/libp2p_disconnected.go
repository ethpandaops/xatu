package libp2p

import (
	"fmt"
	"time"

	"github.com/ClickHouse/ch-go/proto"
	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/flattener"
	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/metadata"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

func init() {
	flattener.MustRegister(flattener.NewStaticRoute(
		libp2pDisconnectedTableName,
		[]xatu.Event_Name{xatu.Event_LIBP2P_TRACE_DISCONNECTED},
		func() flattener.ColumnarBatch {
			return newlibp2pDisconnectedBatch()
		},
	))
}

func (b *libp2pDisconnectedBatch) FlattenTo(
	event *xatu.DecoratedEvent,
	meta *metadata.CommonMetadata,
) error {
	if event == nil || event.GetEvent() == nil {
		return nil
	}

	payload := event.GetLibp2PTraceDisconnected()
	if payload == nil {
		return fmt.Errorf("nil LibP2PTraceDisconnected payload: %w", flattener.ErrInvalidEvent)
	}

	if err := b.validate(event); err != nil {
		return err
	}

	addl := event.GetMeta().GetClient().GetLibp2PTraceDisconnected()

	remotePeerID := payload.GetRemotePeer().GetValue()
	protocol, ip, transportProtocol, port := parseMaddrs(payload.GetRemoteMaddrs().GetValue())
	agent := parseAgentVersion(payload.GetAgentVersion().GetValue())

	b.UpdatedDateTime.Append(time.Now())
	b.EventDateTime.Append(event.GetEvent().GetDateTime().AsTime())
	b.RemotePeerIDUniqueKey.Append(flattener.SeaHashInt64(remotePeerID + meta.MetaNetworkName))
	b.RemoteProtocol.Append(protocol)
	b.RemoteTransportProtocol.Append(transportProtocol)
	b.RemotePort.Append(port)

	if ip != "" {
		b.RemoteIP.Append(proto.NewNullable(flattener.ParseIPv6(ip)))
	} else {
		b.RemoteIP.Append(proto.Nullable[proto.IPv6]{})
	}

	// Remote geo from server-side enrichment.
	serverMeta := event.GetMeta().GetServer().GetLIBP2P_TRACE_DISCONNECTED()
	b.appendRemoteGeo(serverMeta)

	b.RemoteAgentImplementation.Append(agent.Implementation)
	b.RemoteAgentVersion.Append(agent.Version)
	b.RemoteAgentVersionMajor.Append(agent.Major)
	b.RemoteAgentVersionMinor.Append(agent.Minor)
	b.RemoteAgentVersionPatch.Append(agent.Patch)
	b.RemoteAgentPlatform.Append(agent.Platform)
	b.Direction.Append(payload.GetDirection().GetValue())
	b.Opened.Append(payload.GetOpened().AsTime())
	b.Transient.Append(payload.GetTransient().GetValue()) //nolint:staticcheck // SA1019: field is deprecated but still required by the ClickHouse schema.

	_ = addl // addl.GetMetadata() used only for validation

	b.appendMetadata(meta)
	b.rows++

	return nil
}

func (b *libp2pDisconnectedBatch) validate(event *xatu.DecoratedEvent) error {
	addl := event.GetMeta().GetClient().GetLibp2PTraceDisconnected()
	if addl == nil {
		return fmt.Errorf("nil LibP2PTraceDisconnected additional data: %w", flattener.ErrInvalidEvent)
	}

	if addl.GetMetadata() == nil || addl.GetMetadata().GetPeerId() == nil {
		return fmt.Errorf("nil PeerId in metadata: %w", flattener.ErrInvalidEvent)
	}

	return nil
}

func (b *libp2pDisconnectedBatch) appendRemoteGeo(serverMeta *xatu.ServerMeta_AdditionalLibp2PTraceDisconnectedData) {
	if serverMeta == nil || serverMeta.GetPeer() == nil || serverMeta.GetPeer().GetGeo() == nil {
		b.RemoteGeoCity.Append("")
		b.RemoteGeoCountry.Append("")
		b.RemoteGeoCountryCode.Append("")
		b.RemoteGeoContinentCode.Append("")
		b.RemoteGeoLongitude.Append(proto.Nullable[float64]{})
		b.RemoteGeoLatitude.Append(proto.Nullable[float64]{})
		b.RemoteGeoAutonomousSystemNumber.Append(proto.Nullable[uint32]{})
		b.RemoteGeoAutonomousSystemOrganization.Append(proto.Nullable[string]{})

		return
	}

	geo := serverMeta.GetPeer().GetGeo()
	b.RemoteGeoCity.Append(geo.GetCity())
	b.RemoteGeoCountry.Append(geo.GetCountry())
	b.RemoteGeoCountryCode.Append(geo.GetCountryCode())
	b.RemoteGeoContinentCode.Append(geo.GetContinentCode())
	b.RemoteGeoLongitude.Append(proto.NewNullable(geo.GetLongitude()))
	b.RemoteGeoLatitude.Append(proto.NewNullable(geo.GetLatitude()))
	b.RemoteGeoAutonomousSystemNumber.Append(proto.NewNullable(geo.GetAutonomousSystemNumber()))
	b.RemoteGeoAutonomousSystemOrganization.Append(proto.NewNullable(geo.GetAutonomousSystemOrganization()))
}
