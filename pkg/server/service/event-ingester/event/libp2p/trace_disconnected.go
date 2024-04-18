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

var (
	TraceDisconnectedType = xatu.Event_LIBP2P_TRACE_DISCONNECTED.String()
)

type TraceDisconnected struct {
	log           logrus.FieldLogger
	event         *xatu.DecoratedEvent
	geoipProvider geoip.Provider
}

func NewTraceDisconnected(log logrus.FieldLogger, event *xatu.DecoratedEvent, geoipProvider geoip.Provider) *TraceDisconnected {
	return &TraceDisconnected{
		log:           log.WithField("event", TraceDisconnectedType),
		event:         event,
		geoipProvider: geoipProvider,
	}
}

func (td *TraceDisconnected) Type() string {
	return TraceDisconnectedType
}

func (td *TraceDisconnected) Validate(ctx context.Context) error {
	_, ok := td.event.Data.(*xatu.DecoratedEvent_Libp2PTraceDisconnected)
	if !ok {
		return errors.New("failed to cast event data to TraceDisconnected")
	}

	return nil
}

func (td *TraceDisconnected) Filter(ctx context.Context) bool {
	return false
}

func (td *TraceDisconnected) AppendServerMeta(ctx context.Context, meta *xatu.ServerMeta) *xatu.ServerMeta {
	data, ok := td.event.Data.(*xatu.DecoratedEvent_Libp2PTraceDisconnected)
	if !ok {
		td.log.Error("failed to get remote maddrs")

		return meta
	}

	multiaddr := data.Libp2PTraceDisconnected.GetRemoteMaddrs().GetValue()
	if multiaddr != "" {
		return meta
	}

	addrParts := strings.Split(multiaddr, "/")
	if len(addrParts) < 3 {
		return meta
	}

	ipAddress := addrParts[2]

	if ipAddress != "" {
		ip := net.ParseIP(ipAddress)
		if ip != nil && td.geoipProvider != nil {
			geoipLookupResult, err := td.geoipProvider.LookupIP(ctx, ip)
			if err != nil {
				td.log.WithField("ip", ipAddress).WithError(err).Warn("failed to lookup geoip data")
			}

			if geoipLookupResult != nil {
				meta.AdditionalData = &xatu.ServerMeta_LIBP2P_TRACE_DISCONNECTED{
					LIBP2P_TRACE_DISCONNECTED: &xatu.ServerMeta_AdditionalLibp2PTraceDisconnectedData{
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
