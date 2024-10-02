package auth

import (
	"context"
	"fmt"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

type GroupsConfig map[string]GroupConfig

type GroupConfig struct {
	Users              UsersConfig
	EventFilter        *xatu.EventFilterConfig `yaml:"eventFilter"`
	Redacter           *xatu.RedacterConfig    `yaml:"redacter"`
	ObscureClientNames bool                    `yaml:"obscureClientNames"`
}

type Groups map[string]*Group

type Group struct {
	users              *Users
	name               string
	eventFilter        xatu.EventFilter
	redacter           xatu.Redacter
	metrics            *GroupMetrics
	obscureClientNames bool
	clientNames        *clientNameCache
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

func NewGroup(name string, c GroupConfig) (*Group, error) {
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
		users:              &users,
		name:               name,
		metrics:            DefaultGroupMetrics,
		obscureClientNames: c.ObscureClientNames,
		clientNames:        newClientNameCache(3000),
	}

	if c.EventFilter != nil {
		filter, err := xatu.NewEventFilter(c.EventFilter)
		if err != nil {
			return nil, fmt.Errorf("failed to create event filter: %w", err)
		}

		g.eventFilter = filter
	}

	if c.Redacter != nil {
		redacter, err := xatu.NewRedacter(c.Redacter)
		if err != nil {
			return nil, fmt.Errorf("failed to create redacter: %w", err)
		}

		g.redacter = redacter
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

func (g *Group) ApplyFilter(events []*xatu.DecoratedEvent) ([]*xatu.DecoratedEvent, error) {
	if g.eventFilter == nil {
		return events, nil
	}

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

func (g *Group) ApplyRedacter(events []*xatu.DecoratedEvent) ([]*xatu.DecoratedEvent, error) {
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
