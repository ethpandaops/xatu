package libp2p

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/ethpandaops/xatu/pkg/server/geoip"
	"github.com/sirupsen/logrus"
)

var TraceHeartbeatType = xatu.Event_LIBP2P_TRACE_HEARTBEAT.String()

type TraceHeartbeat struct {
	log           logrus.FieldLogger
	event         *xatu.DecoratedEvent
	geoipProvider geoip.Provider
}

func NewTraceHeartbeat(log logrus.FieldLogger, event *xatu.DecoratedEvent, geoipProvider geoip.Provider) *TraceHeartbeat {
	return &TraceHeartbeat{
		log:           log.WithField("event", TraceHeartbeatType),
		event:         event,
		geoipProvider: geoipProvider,
	}
}

func (th *TraceHeartbeat) Type() string {
	return TraceHeartbeatType
}

func (th *TraceHeartbeat) Validate(ctx context.Context) error {
	_, ok := th.event.Data.(*xatu.DecoratedEvent_Libp2PTraceHeartbeat)
	if !ok {
		return errors.New("failed to cast event data to TraceHeartbeat")
	}

	return nil
}

func (th *TraceHeartbeat) Filter(ctx context.Context) bool {
	return false
}

func (th *TraceHeartbeat) AppendServerMeta(ctx context.Context, meta *xatu.ServerMeta) *xatu.ServerMeta {
	fmt.Printf("APPENDING META TO HEARTBEAT: %s - %v\n", meta, th.event.Data)

	data, ok := th.event.Data.(*xatu.DecoratedEvent_Libp2PTraceHeartbeat)
	if !ok {
		th.log.Error("failed to get remote maddrs")

		return meta
	}

	multiaddr := data.Libp2PTraceHeartbeat.GetRemoteMaddrs().GetValue()
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
				meta.AdditionalData = &xatu.ServerMeta_LIBP2P_TRACE_HEARTBEAT{
					LIBP2P_TRACE_HEARTBEAT: &xatu.ServerMeta_AdditionalLibP2PTraceHeartbeatData{
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
