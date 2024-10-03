package eventingester

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/ethpandaops/xatu/pkg/server/geoip"
	"github.com/ethpandaops/xatu/pkg/server/service/event-ingester/auth"
	eventHandler "github.com/ethpandaops/xatu/pkg/server/service/event-ingester/event"
	"github.com/ethpandaops/xatu/pkg/server/store"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Handler struct {
	log            logrus.FieldLogger
	clockDrift     *time.Duration
	geoipProvider  geoip.Provider
	cache          store.Cache
	clientNameSalt string
	metrics        *Metrics

	eventRouter *eventHandler.EventRouter
}

func NewHandler(log logrus.FieldLogger, clockDrift *time.Duration, geoipProvider geoip.Provider, cache store.Cache, clientNameSalt string) *Handler {
	return &Handler{
		log:            log,
		clockDrift:     clockDrift,
		geoipProvider:  geoipProvider,
		cache:          cache,
		metrics:        NewMetrics("xatu_server_event_ingester"),
		eventRouter:    eventHandler.NewEventRouter(log, cache, geoipProvider),
		clientNameSalt: clientNameSalt,
	}
}

//nolint:gocyclo // Needs refactor
func (h *Handler) Events(ctx context.Context, events []*xatu.DecoratedEvent, user *auth.User, group *auth.Group) ([]*xatu.DecoratedEvent, error) {
	groupName := "unknown"
	if group != nil {
		groupName = group.Name()
	}

	username := "unknown"
	if user != nil {
		username = user.Username()
	}

	filteredEvents, err := h.filterEvents(ctx, events, user, group)
	if err != nil {
		return nil, fmt.Errorf("failed to filter events: %w", err)
	}

	events = filteredEvents

	// Redact the events. Redacting is done before and after the event is processed to ensure that the field is not leaked by processing such as geoip lookups.
	if group != nil {
		redactedEvents, err := group.ApplyRedacter(events)
		if err != nil {
			return nil, fmt.Errorf("failed to apply group redacter: %w", err)
		}

		events = redactedEvents
	}

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

	var clientGeo *xatu.ServerMeta_Geo

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
				clientGeo = &xatu.ServerMeta_Geo{
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

	for _, event := range events {
		if event == nil || event.Event == nil {
			continue
		}

		if event.Meta == nil {
			event.Meta = &xatu.Meta{}
		}

		eventName := event.Event.Name.String()

		e, err := h.eventRouter.Route(eventHandler.Type(eventName), event)
		if err != nil {
			h.log.WithError(err).WithField("event", eventName).Warn("failed to create event handler")

			return nil, fmt.Errorf("failed to create event for %s event handler: %w ", eventName, err)
		}

		if err := e.Validate(ctx); err != nil {
			h.log.WithError(err).WithField("event", eventName).Warn("failed to validate event")

			return nil, fmt.Errorf("%s event failed validation: %w", eventName, err)
		}

		if shouldFilter := e.Filter(ctx); shouldFilter {
			h.log.WithField("event", eventName).Debug("event filtered")

			continue
		}

		h.metrics.AddDecoratedEventReceived(1, eventName, groupName)
		h.metrics.AddDecoratedEventFromUserReceived(1, username, groupName)

		meta := xatu.ServerMeta{
			Event: &xatu.ServerMeta_Event{
				ReceivedDateTime: receivedAt,
			},
			Client: &xatu.ServerMeta_Client{
				IP:  ipAddress,
				Geo: clientGeo,
			},
		}

		if group != nil {
			meta.Client.Group = group.Name()

			if user != nil {
				event.Meta.Client.Name = group.ComputeClientName(username, h.clientNameSalt, event.GetMeta().GetClient().GetName())
			}
		}

		if user != nil {
			meta.Client.User = username
		}

		event.Meta.Server = e.AppendServerMeta(ctx, &meta)

		filteredEvents = append(filteredEvents, event)
	}

	// Redact the events again
	if group != nil {
		redactedEvents, err := group.ApplyRedacter(filteredEvents)
		if err != nil {
			return nil, fmt.Errorf("failed to apply group redacter: %w", err)
		}

		filteredEvents = redactedEvents
	}

	return filteredEvents, nil
}

func (h *Handler) filterEvents(_ context.Context, events []*xatu.DecoratedEvent, user *auth.User, group *auth.Group) ([]*xatu.DecoratedEvent, error) {
	filteredEvents := events

	// Apply the user filter
	if user != nil {
		ev, err := user.ApplyFilter(filteredEvents)
		if err != nil {
			return nil, fmt.Errorf("failed to apply user filter: %w", err)
		}

		filteredEvents = ev
	}

	// Apply the group filter
	if group != nil {
		ev, err := group.ApplyFilter(filteredEvents)
		if err != nil {
			return nil, fmt.Errorf("failed to apply group filter: %w", err)
		}

		filteredEvents = ev
	}

	return filteredEvents, nil
}
