package eventingester

import (
	"context"
	"errors"
	"net"
	"strings"
	"time"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/ethpandaops/xatu/pkg/server/geoip"
	eventHandler "github.com/ethpandaops/xatu/pkg/server/service/event-ingester/event"
	"github.com/ethpandaops/xatu/pkg/server/store"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Handler struct {
	log           logrus.FieldLogger
	clockDrift    *time.Duration
	geoipProvider geoip.Provider
	cache         store.Cache

	metrics *Metrics
}

func NewHandler(log logrus.FieldLogger, clockDrift *time.Duration, geoipProvider geoip.Provider, cache store.Cache) *Handler {
	return &Handler{
		log:           log,
		clockDrift:    clockDrift,
		geoipProvider: geoipProvider,
		cache:         cache,
		metrics:       NewMetrics("xatu_server_event_ingester"),
	}
}

func (h *Handler) Events(ctx context.Context, clientID string, events []*xatu.DecoratedEvent) ([]*xatu.DecoratedEvent, error) {
	now := time.Now()
	if h.clockDrift != nil {
		now = now.Add(*h.clockDrift)
	}

	receivedAt := timestamppb.New(now)

	p, ok := peer.FromContext(ctx)
	if !ok {
		return nil, errors.New("failed to get grpc peer")
	}

	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, errors.New("failed to get metadata from context")
	}

	var ipAddress string

	realIP := md.Get("x-real-ip")
	if len(realIP) > 0 {
		ipAddress = realIP[0]
	}

	forwardedFor := md.Get("x-forwarded-for")
	if len(forwardedFor) > 0 {
		ipAddress = forwardedFor[0]
	}

	if ipAddress == "" {
		switch addr := p.Addr.(type) {
		case *net.UDPAddr:
			ipAddress = addr.IP.String()
		case *net.TCPAddr:
			ipAddress = addr.IP.String()
		}
	}

	var sGeo *xatu.ServerMeta_Client_Geo

	if ipAddress != "" {
		// grab the first ip if there are multiple
		ips := strings.Split(ipAddress, ",")
		ipAddress = strings.TrimSpace(ips[0])

		ip := net.ParseIP(ipAddress)
		// validate
		if ip == nil {
			ipAddress = ""
		} else if h.geoipProvider != nil {
			// get geoip locational data
			geoipLookupResult, err := h.geoipProvider.LookupIP(ctx, ip)
			if err != nil {
				h.log.WithField("ip", ipAddress).WithError(err).Warn("failed to lookup geoip data")
			}

			if geoipLookupResult != nil {
				sGeo = &xatu.ServerMeta_Client_Geo{
					Country:                      geoipLookupResult.CountryName,
					CountryCode:                  geoipLookupResult.CountryCode,
					City:                         geoipLookupResult.CityName,
					Latitude:                     geoipLookupResult.Latitude,
					Longitude:                    geoipLookupResult.Longitude,
					ContinentCode:                geoipLookupResult.ContinentCode,
					AutonomousSystemNumber:       geoipLookupResult.AutonomousSystemNumber,
					AutonomousSystemOrganization: geoipLookupResult.AutonomousSystemOrganization,
				}
			}
		}
	}

	meta := xatu.ServerMeta{
		Event: &xatu.ServerMeta_Event{
			ReceivedDateTime: receivedAt,
		},
		Client: &xatu.ServerMeta_Client{
			IP:  ipAddress,
			Geo: sGeo,
		},
	}

	filteredEvents := make([]*xatu.DecoratedEvent, 0, len(events))

	for _, event := range events {
		if event == nil || event.Event == nil {
			continue
		}

		if event.Meta == nil {
			event.Meta = &xatu.Meta{}
		}

		eventName := event.Event.Name.String()

		e, err := eventHandler.New(eventHandler.Type(eventName), h.log, event, h.cache)
		if err != nil {
			h.log.WithError(err).WithField("event", eventName).Warn("failed to create event handler")

			continue
		}

		if err := e.Validate(ctx); err != nil {
			h.log.WithError(err).WithField("event", eventName).Warn("failed to validate event")

			continue
		}

		if shouldFilter := e.Filter(ctx); shouldFilter {
			h.log.WithField("event", eventName).Debug("event filtered")

			continue
		}

		h.metrics.AddDecoratedEventReceived(1, eventName, clientID)

		event.Meta.Server = &meta

		filteredEvents = append(filteredEvents, event)
	}

	return filteredEvents, nil
}
