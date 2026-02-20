package auth

import (
	"context"
	"fmt"

	"github.com/ethpandaops/xatu/pkg/observability"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/ethpandaops/xatu/pkg/server/geoip/lookup"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type GroupsConfig map[string]GroupConfig

type GroupConfig struct {
	Users              UsersConfig             `yaml:"users"`
	EventFilter        *xatu.EventFilterConfig `yaml:"eventFilter"`
	Redacter           *xatu.RedacterConfig    `yaml:"redacter"`
	ObscureClientNames bool                    `yaml:"obscureClientNames"`
	Precision          *lookup.Precision       `yaml:"precision"` // Optional: defaults to full if not specified
	ASN                *bool                   `yaml:"asn"`       // Optional: defaults to true if not specified
}

type Groups map[string]*Group

type Group struct {
	log                logrus.FieldLogger
	users              *Users
	name               string
	eventFilter        xatu.EventFilter
	redacter           xatu.Redacter
	metrics            *GroupMetrics
	obscureClientNames bool
	clientNames        *clientNameCache
	geoPrecision       lookup.Precision
	includeASN         bool
}

func (g *Group) ComputeClientName(user, salt, clientName string) string {
	if !g.obscureClientNames {
		return fmt.Sprintf("%s/%s/%s", g.name, user, clientName)
	}

	if computedName, ok := g.clientNames.Get(clientName); ok {
		return computedName
	}

	computedClientName := ComputeClientName(user, g.name, salt, clientName)
	g.clientNames.Set(fmt.Sprintf("%s/%s/%s", g.name, user, clientName), computedClientName)

	return computedClientName
}

func NewGroup(log logrus.FieldLogger, name string, c GroupConfig) (*Group, error) {
	if err := c.Validate(); err != nil {
		return nil, fmt.Errorf("group config is invalid: %w", err)
	}

	users := make(Users)

	for username, user := range c.Users {
		u, err := NewUser(username, user)
		if err != nil {
			return nil, fmt.Errorf("failed to create user %s: %w", username, err)
		}

		users[username] = u
	}

	g := &Group{
		log: log.WithFields(logrus.Fields{
			"group":         name,
			"server/module": "event-ingester/auth",
		}),
		users:              &users,
		name:               name,
		metrics:            DefaultGroupMetrics,
		obscureClientNames: c.ObscureClientNames,
		clientNames:        newClientNameCache(3000),
		geoPrecision:       lookup.PrecisionFull, // Default to full precision
	}

	if c.EventFilter != nil {
		filter, err := xatu.NewEventFilter(c.EventFilter)
		if err != nil {
			return nil, fmt.Errorf("failed to create event filter: %w", err)
		}

		g.log.WithFields(logrus.Fields{
			"event_names":         c.EventFilter.EventNames,
			"exclude_event_names": c.EventFilter.ExcludeEventNames,
			"modules":             c.EventFilter.Modules,
			"exclude_modules":     c.EventFilter.ExcludeModules,
		}).Info("Created a new event filter")

		g.eventFilter = filter
	}

	if c.Redacter != nil {
		redacter, err := xatu.NewRedacter(c.Redacter)
		if err != nil {
			return nil, fmt.Errorf("failed to create redacter: %w", err)
		}

		g.log.WithFields(logrus.Fields{
			"field_paths": c.Redacter.FieldPaths,
		}).Info("Created a new redacter")

		g.redacter = redacter
	}

	// Set geo precision from explicit config or default to full
	if c.Precision != nil {
		g.geoPrecision = *c.Precision
		g.log.WithField("geo_precision", g.geoPrecision.String()).Info("Geo precision set from config")
	} else {
		g.geoPrecision = lookup.PrecisionFull
		g.log.WithField("geo_precision", "full").Info("Geo precision defaulting to full")
	}

	// Set ASN inclusion from config or default to true
	if c.ASN != nil {
		g.includeASN = *c.ASN
		g.log.WithField("include_asn", g.includeASN).Info("ASN inclusion set from config")
	} else {
		g.includeASN = true
		g.log.WithField("include_asn", true).Info("ASN inclusion defaulting to true")
	}

	return g, nil
}

func (g *Group) Start(ctx context.Context) {
	g.clientNames.Start(ctx)
}

func (g *GroupConfig) Validate() error {
	if len(g.Users) == 0 {
		return fmt.Errorf("no users configured")
	}

	if g.EventFilter != nil {
		if err := g.EventFilter.Validate(); err != nil {
			return fmt.Errorf("event filter is invalid: %w", err)
		}
	}

	if g.Redacter != nil {
		if err := g.Redacter.Validate(); err != nil {
			return fmt.Errorf("redacter is invalid: %w", err)
		}
	}

	for username, user := range g.Users {
		if err := user.Validate(); err != nil {
			return fmt.Errorf("user %s is invalid: %w", username, err)
		}
	}

	// Check for duplicate users
	userNames := make(map[string]bool)
	for username := range g.Users {
		if userNames[username] {
			return fmt.Errorf("user %s already exists", username)
		}

		userNames[username] = true
	}

	return nil
}

func (g *Group) ShouldObscureClientName() bool {
	return g.obscureClientNames
}

func (g *Group) Name() string {
	return g.name
}

func (g *Group) Users() *Users {
	return g.users
}

func (g *Group) ValidUser(username, password string) bool {
	return g.users.ValidUser(username, password)
}

func (g *Group) EventFilter() xatu.EventFilter {
	return g.eventFilter
}

func (g *Group) ApplyFilter(ctx context.Context, events []*xatu.DecoratedEvent) ([]*xatu.DecoratedEvent, error) {
	if g.eventFilter == nil {
		return events, nil
	}

	_, span := observability.Tracer().Start(ctx,
		"Auth/Group.ApplyFilter",
		trace.WithAttributes(attribute.Int64("events", int64(len(events)))),
	)
	defer span.End()

	filteredEvents := make([]*xatu.DecoratedEvent, 0)

	for _, event := range events {
		shouldBeDropped, err := g.eventFilter.ShouldBeDropped(event)
		if err != nil {
			return nil, fmt.Errorf("failed to apply event filter: %w", err)
		}

		if shouldBeDropped {
			continue
		}

		filteredEvents = append(filteredEvents, event)
	}

	return filteredEvents, nil
}

func (g *Group) ApplyRedacter(ctx context.Context, events []*xatu.DecoratedEvent) ([]*xatu.DecoratedEvent, error) {
	_, span := observability.Tracer().Start(ctx,
		"Auth/Group.ApplyRedacter",
		trace.WithAttributes(attribute.Int64("events", int64(len(events)))),
	)
	defer span.End()

	if g.redacter == nil {
		return events, nil
	}

	for _, event := range events {
		fields := g.redacter.Apply(event)

		for _, field := range fields {
			g.metrics.IncFieldsRedacted(g.name, field)
		}
	}

	return events, nil
}

// GetGeoPrecision returns the geo precision level for this group
func (g *Group) GetGeoPrecision() lookup.Precision {
	return g.geoPrecision
}

func (g *Group) IncludeASN() bool {
	return g.includeASN
}
