package libp2p

import (
	"context"
	"errors"
	"net"
	"strings"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/ethpandaops/xatu/pkg/server/geoip"
	"github.com/sirupsen/logrus"
)

var TraceSyntheticHeartbeatType = xatu.Event_LIBP2P_TRACE_SYNTHETIC_HEARTBEAT.String()

type TraceSyntheticHeartbeat struct {
	log           logrus.FieldLogger
	event         *xatu.DecoratedEvent
	geoipProvider geoip.Provider
}

func NewTraceSyntheticHeartbeat(log logrus.FieldLogger, event *xatu.DecoratedEvent, geoipProvider geoip.Provider) *TraceSyntheticHeartbeat {
	return &TraceSyntheticHeartbeat{
		log:           log.WithField("event", TraceSyntheticHeartbeatType),
		event:         event,
		geoipProvider: geoipProvider,
	}
}

func (th *TraceSyntheticHeartbeat) Type() string {
	return TraceSyntheticHeartbeatType
}

func (th *TraceSyntheticHeartbeat) Validate(ctx context.Context) error {
	data, ok := th.event.Data.(*xatu.DecoratedEvent_Libp2PTraceSyntheticHeartbeat)
	if !ok {
		return errors.New("failed to cast event data to TraceSyntheticHeartbeat")
	}

	if data.Libp2PTraceSyntheticHeartbeat.GetRemotePeer() == nil {
		return errors.New("remote peer is nil")
	}

	return nil
}

func (th *TraceSyntheticHeartbeat) Filter(ctx context.Context) bool {
	return false
}

func (th *TraceSyntheticHeartbeat) AppendServerMeta(ctx context.Context, meta *xatu.ServerMeta) *xatu.ServerMeta {
	data, ok := th.event.Data.(*xatu.DecoratedEvent_Libp2PTraceSyntheticHeartbeat)
	if !ok {
		th.log.Error("failed to get remote maddrs")

		return meta
	}

	multiaddr := data.Libp2PTraceSyntheticHeartbeat.GetRemoteMaddrs().GetValue()
	if multiaddr == "" {
		return meta
	}

	addrParts := strings.Split(multiaddr, "/")
	if len(addrParts) < 3 {
		return meta
	}

	ipAddress := addrParts[2]

	if ipAddress != "" {
		ip := net.ParseIP(ipAddress)
		if ip != nil && th.geoipProvider != nil {
			geoipLookupResult, err := th.geoipProvider.LookupIP(ctx, ip)
			if err != nil {
				th.log.WithField("ip", ipAddress).WithError(err).Warn("failed to lookup geoip data")
			}

			if geoipLookupResult != nil {
				meta.AdditionalData = &xatu.ServerMeta_LIBP2P_TRACE_SYNTHETIC_HEARTBEAT{
					LIBP2P_TRACE_SYNTHETIC_HEARTBEAT: &xatu.ServerMeta_AdditionalLibP2PTraceSyntheticHeartbeatData{
						Peer: &xatu.ServerMeta_Peer{
							Geo: &xatu.ServerMeta_Geo{
								Country:                      geoipLookupResult.CountryName,
								CountryCode:                  geoipLookupResult.CountryCode,
								City:                         geoipLookupResult.CityName,
								Latitude:                     geoipLookupResult.Latitude,
								Longitude:                    geoipLookupResult.Longitude,
								ContinentCode:                geoipLookupResult.ContinentCode,
								AutonomousSystemNumber:       geoipLookupResult.AutonomousSystemNumber,
								AutonomousSystemOrganization: geoipLookupResult.AutonomousSystemOrganization,
							},
						},
					},
				}
			}
		}
	}

	return meta
}
