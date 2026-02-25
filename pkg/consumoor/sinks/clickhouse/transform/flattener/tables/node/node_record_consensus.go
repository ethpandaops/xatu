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
		nodeRecordConsensusTableName,
		[]xatu.Event_Name{xatu.Event_NODE_RECORD_CONSENSUS},
		func() flattener.ColumnarBatch {
			return newnodeRecordConsensusBatch()
		},
	))
}

func (b *nodeRecordConsensusBatch) FlattenTo(
	event *xatu.DecoratedEvent,
	meta *metadata.CommonMetadata,
) error {
	if event == nil || event.GetEvent() == nil {
		return nil
	}

	payload := event.GetNodeRecordConsensus()
	if payload == nil {
		return fmt.Errorf("nil NodeRecordConsensus payload: %w", flattener.ErrInvalidEvent)
	}

	if err := b.validate(event); err != nil {
		return err
	}

	// Parse agent version from the Name field (e.g. "Lighthouse/v4.5.0/linux-x86_64").
	name := payload.GetName().GetValue()
	impl, version, major, minor, patch := parseAgentVersion(name)

	// NodeID is nullable.
	nodeID := payload.GetNodeId().GetValue()
	if nodeID != "" {
		b.NodeID.Append(proto.NewNullable(nodeID))
	} else {
		b.NodeID.Append(proto.Nullable[string]{})
	}

	// PeerIDUniqueKey: seahash of PeerId string, nullable.
	peerID := payload.GetPeerId().GetValue()
	if peerID != "" {
		b.PeerIDUniqueKey.Append(proto.NewNullable(flattener.SeaHashInt64(peerID)))
	} else {
		b.PeerIDUniqueKey.Append(proto.Nullable[int64]{})
	}

	b.UpdatedDateTime.Append(time.Now())
	b.EventDateTime.Append(event.GetEvent().GetDateTime().AsTime())
	b.Enr.Append(payload.GetEnr().GetValue())
	b.Timestamp.Append(payload.GetTimestamp().GetValue())
	b.Name.Append(name)
	b.Version.Append(version)
	b.VersionMajor.Append(major)
	b.VersionMinor.Append(minor)
	b.VersionPatch.Append(patch)
	b.Implementation.Append(impl)
	b.ForkDigest.Append(payload.GetForkDigest().GetValue())

	// NextForkDigest is nullable.
	nfd := payload.GetNextForkDigest().GetValue()
	if nfd != "" {
		b.NextForkDigest.Append(proto.NewNullable(nfd))
	} else {
		b.NextForkDigest.Append(proto.Nullable[string]{})
	}

	b.FinalizedRoot.Append(payload.GetFinalizedRoot().GetValue())
	b.FinalizedEpoch.Append(payload.GetFinalizedEpoch().GetValue())
	b.HeadRoot.Append(payload.GetHeadRoot().GetValue())
	b.HeadSlot.Append(payload.GetHeadSlot().GetValue())

	// Cgc is nullable.
	cgc := payload.GetCgc().GetValue()
	if cgc != "" {
		b.Cgc.Append(proto.NewNullable(cgc))
	} else {
		b.Cgc.Append(proto.Nullable[string]{})
	}

	// FinalizedEpochStartDateTime is nullable DateTime.
	if ts := payload.GetFinalizedEpochStartDateTime(); ts != nil && ts.IsValid() {
		b.FinalizedEpochStartDateTime.Append(proto.NewNullable(ts.AsTime()))
	} else {
		b.FinalizedEpochStartDateTime.Append(proto.Nullable[time.Time]{})
	}

	// HeadSlotStartDateTime is nullable DateTime.
	if ts := payload.GetHeadSlotStartDateTime(); ts != nil && ts.IsValid() {
		b.HeadSlotStartDateTime.Append(proto.NewNullable(ts.AsTime()))
	} else {
		b.HeadSlotStartDateTime.Append(proto.Nullable[time.Time]{})
	}

	// IP is nullable IPv6.
	ipStr := payload.GetIp().GetValue()
	if ipStr != "" {
		b.IP.Append(proto.NewNullable(flattener.ParseIPv6(ipStr)))
	} else {
		b.IP.Append(proto.Nullable[proto.IPv6]{})
	}

	// Tcp, Udp, Quic are nullable UInt16.
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

	if quic := payload.GetQuic(); quic != nil {
		b.Quic.Append(proto.NewNullable(uint16(quic.GetValue()))) //nolint:gosec // G115: quic port fits uint16.
	} else {
		b.Quic.Append(proto.Nullable[uint16]{})
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

func (b *nodeRecordConsensusBatch) validate(event *xatu.DecoratedEvent) error {
	payload := event.GetNodeRecordConsensus()

	if payload.GetTimestamp() == nil {
		return fmt.Errorf("nil Timestamp: %w", flattener.ErrInvalidEvent)
	}

	if payload.GetFinalizedEpoch() == nil {
		return fmt.Errorf("nil FinalizedEpoch: %w", flattener.ErrInvalidEvent)
	}

	if payload.GetHeadSlot() == nil {
		return fmt.Errorf("nil HeadSlot: %w", flattener.ErrInvalidEvent)
	}

	return nil
}

// parseAgentVersion extracts implementation, version, and semver parts from
// an agent version string like "Lighthouse/v4.5.0-abc/linux-x86_64".
func parseAgentVersion(name string) (impl, version, major, minor, patch string) {
	if name == "" {
		return "", "", "", "", ""
	}

	parts := strings.SplitN(name, "/", 3)
	if len(parts) >= 1 {
		impl = parts[0]
	}

	if len(parts) >= 2 {
		version = parts[1]
	}

	// Parse semver from version string.
	ver := version
	ver = strings.TrimPrefix(ver, "v")
	ver = strings.TrimPrefix(ver, "V")

	semParts := strings.SplitN(ver, ".", 3)
	if len(semParts) >= 1 {
		major = semParts[0]
	}

	if len(semParts) >= 2 {
		minor = semParts[1]
	}

	if len(semParts) >= 3 {
		patch = semParts[2]
		if idx := strings.IndexAny(patch, "-+ "); idx != -1 {
			patch = patch[:idx]
		}
	}

	return impl, version, major, minor, patch
}
