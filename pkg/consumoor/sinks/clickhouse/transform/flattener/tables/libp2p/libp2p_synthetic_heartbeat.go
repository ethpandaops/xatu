package libp2p

import (
	"fmt"
	"strconv"
	"time"

	"github.com/ClickHouse/ch-go/proto"
	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/flattener"
	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/metadata"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

func init() {
	flattener.MustRegister(flattener.NewStaticRoute(
		libp2pSyntheticHeartbeatTableName,
		[]xatu.Event_Name{xatu.Event_LIBP2P_TRACE_SYNTHETIC_HEARTBEAT},
		func() flattener.ColumnarBatch {
			return newlibp2pSyntheticHeartbeatBatch()
		},
	))
}

func (b *libp2pSyntheticHeartbeatBatch) FlattenTo(
	event *xatu.DecoratedEvent,
	meta *metadata.CommonMetadata,
) error {
	if event == nil || event.GetEvent() == nil {
		return nil
	}

	payload := event.GetLibp2PTraceSyntheticHeartbeat()
	if payload == nil {
		return fmt.Errorf("nil LibP2PTraceSyntheticHeartbeat payload: %w", flattener.ErrInvalidEvent)
	}

	if err := b.validate(event); err != nil {
		return err
	}

	remotePeerID := payload.GetRemotePeer().GetValue()
	_, ip, _, port := parseMaddrs(payload.GetRemoteMaddrs().GetValue())
	agent := parseAgentVersion(payload.GetAgentVersion().GetValue())

	b.UpdatedDateTime.Append(time.Now())
	b.EventDateTime.Append(event.GetEvent().GetDateTime().AsTime())
	b.RemotePeerIDUniqueKey.Append(flattener.SeaHashInt64(remotePeerID + meta.MetaNetworkName))
	b.RemoteMaddrs.Append(payload.GetRemoteMaddrs().GetValue())

	// LatencyMs is nullable.
	if latency := payload.GetLatencyMs(); latency != nil {
		b.LatencyMs.Append(proto.NewNullable(latency.GetValue()))
	} else {
		b.LatencyMs.Append(proto.Nullable[int64]{})
	}

	// Direction: UInt32 in proto, ColStr in ClickHouse.
	b.Direction.Append(strconv.FormatUint(uint64(payload.GetDirection().GetValue()), 10))

	b.Protocols.Append(payload.GetProtocols())

	// ConnectionAgeMs: proto has nanoseconds, ClickHouse has milliseconds.
	if ageNs := payload.GetConnectionAgeNs(); ageNs != nil {
		b.ConnectionAgeMs.Append(proto.NewNullable(ageNs.GetValue() / 1_000_000))
	} else {
		b.ConnectionAgeMs.Append(proto.Nullable[int64]{})
	}

	b.RemoteAgentImplementation.Append(agent.Implementation)
	b.RemoteAgentVersion.Append(agent.Version)
	b.RemoteAgentVersionMajor.Append(agent.Major)
	b.RemoteAgentVersionMinor.Append(agent.Minor)
	b.RemoteAgentVersionPatch.Append(agent.Patch)
	b.RemoteAgentPlatform.Append(agent.Platform)

	// RemoteIP is nullable.
	if ip != "" {
		b.RemoteIP.Append(proto.NewNullable(flattener.ParseIPv6(ip)))
	} else {
		b.RemoteIP.Append(proto.Nullable[proto.IPv6]{})
	}

	// RemotePort is nullable.
	if port > 0 {
		b.RemotePort.Append(proto.NewNullable(port))
	} else {
		b.RemotePort.Append(proto.Nullable[uint16]{})
	}

	// Remote geo from server-side enrichment.
	serverMeta := event.GetMeta().GetServer().GetLIBP2P_TRACE_SYNTHETIC_HEARTBEAT()
	b.appendRemoteGeo(serverMeta)

	b.appendMetadata(meta)
	b.rows++

	return nil
}

func (b *libp2pSyntheticHeartbeatBatch) validate(event *xatu.DecoratedEvent) error {
	addl := event.GetMeta().GetClient().GetLibp2PTraceSyntheticHeartbeat()
	if addl == nil {
		return fmt.Errorf("nil LibP2PTraceSyntheticHeartbeat additional data: %w", flattener.ErrInvalidEvent)
	}

	if addl.GetMetadata() == nil || addl.GetMetadata().GetPeerId() == nil {
		return fmt.Errorf("nil PeerId in metadata: %w", flattener.ErrInvalidEvent)
	}

	return nil
}

func (b *libp2pSyntheticHeartbeatBatch) appendRemoteGeo(
	serverMeta *xatu.ServerMeta_AdditionalLibP2PTraceSyntheticHeartbeatData,
) {
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
