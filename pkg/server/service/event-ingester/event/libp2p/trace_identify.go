package libp2p

import (
	"context"
	"errors"
	"net"
	"strings"

	"github.com/ethpandaops/xatu/pkg/observability"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/ethpandaops/xatu/pkg/server/geoip"
	"github.com/ethpandaops/xatu/pkg/server/geoip/lookup"
)

var TraceIdentifyType = xatu.Event_LIBP2P_TRACE_IDENTIFY.String()

type TraceIdentify struct {
	log           observability.ContextualLogger
	event         *xatu.DecoratedEvent
	geoipProvider geoip.Provider
}

func NewTraceIdentify(log observability.ContextualLogger, event *xatu.DecoratedEvent, geoipProvider geoip.Provider) *TraceIdentify {
	return &TraceIdentify{
		log:           log.WithField("event", TraceIdentifyType),
		event:         event,
		geoipProvider: geoipProvider,
	}
}

func (ti *TraceIdentify) Type() string {
	return TraceIdentifyType
}

func (ti *TraceIdentify) Validate(ctx context.Context) error {
	data, ok := ti.event.GetData().(*xatu.DecoratedEvent_Libp2PTraceIdentify)
	if !ok {
		return errors.New("failed to cast event data to TraceIdentify")
	}

	if data.Libp2PTraceIdentify.GetRemotePeer() == nil {
		return errors.New("remote peer is nil")
	}

	return nil
}

func (ti *TraceIdentify) Filter(ctx context.Context) bool {
	return false
}

func (ti *TraceIdentify) AppendServerMeta(ctx context.Context, meta *xatu.ServerMeta) *xatu.ServerMeta {
	data, ok := ti.event.GetData().(*xatu.DecoratedEvent_Libp2PTraceIdentify)
	if !ok {
		ti.log.WithContext(ctx).Error("failed to cast event data")

		return meta
	}

	multiaddr := data.Libp2PTraceIdentify.GetRemoteMultiaddr().GetValue()
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
		if ip != nil && ti.geoipProvider != nil {
			geoipLookupResult, err := ti.geoipProvider.LookupIP(ctx, ip, lookup.PrecisionFull)
			if err != nil {
				ti.log.WithField("ip", ipAddress).WithError(err).WithContext(ctx).Warn("failed to lookup geoip data")
			}

			if geoipLookupResult != nil {
				meta.AdditionalData = &xatu.ServerMeta_LIBP2P_TRACE_IDENTIFY{
					LIBP2P_TRACE_IDENTIFY: &xatu.ServerMeta_AdditionalLibp2PTraceIdentifyData{
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
