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

var nodeRecordConsensusEventNames = []xatu.Event_Name{
	xatu.Event_NODE_RECORD_CONSENSUS,
}

func init() {
	catalog.MustRegister(flattener.NewStaticRoute(
		nodeRecordConsensusTableName,
		nodeRecordConsensusEventNames,
		func() flattener.ColumnarBatch { return newnodeRecordConsensusBatch() },
	))
}

func (b *nodeRecordConsensusBatch) FlattenTo(
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
	b.appendServerGeo(event)
	b.rows++

	return nil
}

func (b *nodeRecordConsensusBatch) appendRuntime(event *xatu.DecoratedEvent) {
	b.UpdatedDateTime.Append(time.Now())

	if ts := event.GetEvent().GetDateTime(); ts != nil {
		b.EventDateTime.Append(ts.AsTime())
	} else {
		b.EventDateTime.Append(time.Time{})
	}
}

func (b *nodeRecordConsensusBatch) appendPayload(
	event *xatu.DecoratedEvent,
	meta *metadata.CommonMetadata,
) {
	consensus := event.GetNodeRecordConsensus()
	if consensus == nil {
		b.Enr.Append("")
		b.NodeID.Append(chProto.Nullable[string]{})
		b.PeerIDUniqueKey.Append(chProto.Nullable[int64]{})
		b.Timestamp.Append(0)
		b.Name.Append("")
		b.Version.Append("")
		b.VersionMajor.Append("")
		b.VersionMinor.Append("")
		b.VersionPatch.Append("")
		b.Implementation.Append("")
		b.ForkDigest.Append("")
		b.NextForkDigest.Append(chProto.Nullable[string]{})
		b.FinalizedRoot.Append("")
		b.FinalizedEpoch.Append(0)
		b.HeadRoot.Append("")
		b.HeadSlot.Append(0)
		b.Cgc.Append(chProto.Nullable[string]{})
		b.FinalizedEpochStartDateTime.Append(chProto.Nullable[time.Time]{})
		b.HeadSlotStartDateTime.Append(chProto.Nullable[time.Time]{})
		b.IP.Append(chProto.Nullable[chProto.IPv6]{})
		b.Tcp.Append(chProto.Nullable[uint16]{})
		b.Udp.Append(chProto.Nullable[uint16]{})
		b.Quic.Append(chProto.Nullable[uint16]{})
		b.HasIpv6.Append(false)

		return
	}

	b.Enr.Append(nodeStringValue(consensus.GetEnr()))

	// NodeID is Nullable[string].
	nodeID := nodeStringValue(consensus.GetNodeId())
	if nodeID != "" {
		b.NodeID.Append(chProto.NewNullable[string](nodeID))
	} else {
		b.NodeID.Append(chProto.Nullable[string]{})
	}

	// PeerIDUniqueKey: hash of peerID + network.
	peerID := nodeStringValue(consensus.GetPeerId())
	network := ""

	if meta != nil && meta.MetaNetworkName != "" {
		network = meta.MetaNetworkName
	}

	if peerID != "" {
		b.PeerIDUniqueKey.Append(chProto.NewNullable[int64](flattener.SeaHashInt64(peerID + network)))
	} else {
		b.PeerIDUniqueKey.Append(chProto.Nullable[int64]{})
	}

	// Timestamp.
	if ts := consensus.GetTimestamp(); ts != nil {
		b.Timestamp.Append(ts.GetValue())
	} else {
		b.Timestamp.Append(0)
	}

	// Name and parsed implementation/version.
	name := nodeStringValue(consensus.GetName())
	b.Name.Append(name)

	implementation, version := parseConsensusName(name)
	major, minor, patch := parseVersion(version)

	b.Version.Append(version)
	b.VersionMajor.Append(major)
	b.VersionMinor.Append(minor)
	b.VersionPatch.Append(patch)
	b.Implementation.Append(implementation)

	b.ForkDigest.Append(nodeStringValue(consensus.GetForkDigest()))

	// NextForkDigest is Nullable[string].
	nextForkDigest := nodeStringValue(consensus.GetNextForkDigest())
	if nextForkDigest != "" {
		b.NextForkDigest.Append(chProto.NewNullable[string](nextForkDigest))
	} else {
		b.NextForkDigest.Append(chProto.Nullable[string]{})
	}

	b.FinalizedRoot.Append(nodeStringValue(consensus.GetFinalizedRoot()))

	if finalizedEpoch := consensus.GetFinalizedEpoch(); finalizedEpoch != nil {
		b.FinalizedEpoch.Append(finalizedEpoch.GetValue())
	} else {
		b.FinalizedEpoch.Append(0)
	}

	b.HeadRoot.Append(nodeStringValue(consensus.GetHeadRoot()))

	if headSlot := consensus.GetHeadSlot(); headSlot != nil {
		b.HeadSlot.Append(headSlot.GetValue())
	} else {
		b.HeadSlot.Append(0)
	}

	// Cgc is Nullable[string].
	cgc := nodeStringValue(consensus.GetCgc())
	if cgc != "" {
		b.Cgc.Append(chProto.NewNullable[string](cgc))
	} else {
		b.Cgc.Append(chProto.Nullable[string]{})
	}

	// FinalizedEpochStartDateTime and HeadSlotStartDateTime: prefer additional
	// data (meta.client) over payload values. The old code set from payload
	// then overwrote from additional data; in batch mode we resolve once.
	finalizedEpochStartDT := chProto.Nullable[time.Time]{}
	headSlotStartDT := chProto.Nullable[time.Time]{}

	// Start with payload values.
	if fes := consensus.GetFinalizedEpochStartDateTime(); fes != nil {
		finalizedEpochStartDT = chProto.NewNullable[time.Time](fes.AsTime())
	}

	if hss := consensus.GetHeadSlotStartDateTime(); hss != nil {
		headSlotStartDT = chProto.NewNullable[time.Time](hss.AsTime())
	}

	// Override from additional data if present.
	if additional := event.GetMeta().GetClient().GetNodeRecordConsensus(); additional != nil {
		if fe := additional.GetFinalizedEpoch(); fe != nil {
			if startDT := fe.GetStartDateTime(); startDT != nil {
				finalizedEpochStartDT = chProto.NewNullable[time.Time](startDT.AsTime())
			}
		}

		if hs := additional.GetHeadSlot(); hs != nil {
			if startDT := hs.GetStartDateTime(); startDT != nil {
				headSlotStartDT = chProto.NewNullable[time.Time](startDT.AsTime())
			}
		}
	}

	b.FinalizedEpochStartDateTime.Append(finalizedEpochStartDT)
	b.HeadSlotStartDateTime.Append(headSlotStartDT)

	// IP is Nullable[IPv6].
	ip := nodeStringValue(consensus.GetIp())
	normalizedIP := normalizeIPv6Mapped(ip)

	if normalizedIP != "" {
		b.IP.Append(chProto.NewNullable[chProto.IPv6](flattener.ParseIPv6(normalizedIP)))
	} else {
		b.IP.Append(chProto.Nullable[chProto.IPv6]{})
	}

	// Tcp, Udp, Quic are Nullable[uint16].
	if tcp := consensus.GetTcp(); tcp != nil {
		b.Tcp.Append(chProto.NewNullable[uint16](uint16(tcp.GetValue()))) //nolint:gosec // proto uint32 narrowed to uint16 target field
	} else {
		b.Tcp.Append(chProto.Nullable[uint16]{})
	}

	if udp := consensus.GetUdp(); udp != nil {
		b.Udp.Append(chProto.NewNullable[uint16](uint16(udp.GetValue()))) //nolint:gosec // proto uint32 narrowed to uint16 target field
	} else {
		b.Udp.Append(chProto.Nullable[uint16]{})
	}

	if quic := consensus.GetQuic(); quic != nil {
		b.Quic.Append(chProto.NewNullable[uint16](uint16(quic.GetValue()))) //nolint:gosec // proto uint32 narrowed to uint16 target field
	} else {
		b.Quic.Append(chProto.Nullable[uint16]{})
	}

	if hasIpv6 := consensus.GetHasIpv6(); hasIpv6 != nil {
		b.HasIpv6.Append(hasIpv6.GetValue())
	} else {
		b.HasIpv6.Append(false)
	}
}

func (b *nodeRecordConsensusBatch) appendServerGeo(event *xatu.DecoratedEvent) {
	geo := extractNodeGeo(event, xatu.Event_NODE_RECORD_CONSENSUS)

	b.GeoCity.Append(geo.City)
	b.GeoCountry.Append(geo.Country)
	b.GeoCountryCode.Append(geo.CountryCode)
	b.GeoContinentCode.Append(geo.ContinentCode)

	b.GeoLongitude.Append(chProto.NewNullable[float64](geo.Longitude))
	b.GeoLatitude.Append(chProto.NewNullable[float64](geo.Latitude))
	b.GeoAutonomousSystemNumber.Append(chProto.NewNullable[uint32](geo.AutonomousSystemNumber))
	b.GeoAutonomousSystemOrganization.Append(chProto.NewNullable[string](geo.AutonomousSystemOrganization))
}

func parseConsensusName(name string) (implementation, version string) {
	if name == "" {
		return "", ""
	}

	parts := strings.SplitN(name, "/", 4)
	if len(parts) <= 1 {
		return "", ""
	}

	implementation = strings.ToLower(parts[0])

	if len(parts) > 2 && parts[0] == parts[1] {
		version = parts[2]
	} else {
		version = parts[1]
	}

	return implementation, version
}
