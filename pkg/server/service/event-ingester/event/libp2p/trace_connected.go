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
	TraceConnectedType = xatu.Event_LIBP2P_TRACE_CONNECTED.String()
)

type TraceConnected struct {
	log           logrus.FieldLogger
	event         *xatu.DecoratedEvent
	geoipProvider geoip.Provider
}

func NewTraceConnected(log logrus.FieldLogger, event *xatu.DecoratedEvent, geoipProvider geoip.Provider) *TraceConnected {
	return &TraceConnected{
		log:           log.WithField("event", TraceConnectedType),
		event:         event,
		geoipProvider: geoipProvider,
	}
}

func (tc *TraceConnected) Type() string {
	return TraceConnectedType
}

func (tc *TraceConnected) Validate(ctx context.Context) error {
	_, ok := tc.event.Data.(*xatu.DecoratedEvent_Libp2PTraceConnected)
	if !ok {
		return errors.New("failed to cast event data to TraceConnected")
	}

	return nil
}

func (tc *TraceConnected) Filter(ctx context.Context) bool {
	return false
}

func (tc *TraceConnected) AppendServerMeta(ctx context.Context, meta *xatu.ServerMeta) *xatu.ServerMeta {
	data, ok := tc.event.Data.(*xatu.DecoratedEvent_Libp2PTraceConnected)
	if !ok {
		tc.log.Error("failed to get remote maddrs")

		return meta
	}

	multiaddr := data.Libp2PTraceConnected.GetRemoteMaddrs().GetValue()
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
		if ip != nil && tc.geoipProvider != nil {
			geoipLookupResult, err := tc.geoipProvider.LookupIP(ctx, ip)
			if err != nil {
				tc.log.WithField("ip", ipAddress).WithError(err).Warn("failed to lookup geoip data")
			}

			if geoipLookupResult != nil {
				meta.AdditionalData = &xatu.ServerMeta_LIBP2P_TRACE_CONNECTED{
					LIBP2P_TRACE_CONNECTED: &xatu.ServerMeta_AdditionalLibp2PTraceConnectedData{
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
