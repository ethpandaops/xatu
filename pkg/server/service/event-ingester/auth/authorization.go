package auth

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

type AuthorizationConfig struct {
	Enabled bool

	Groups GroupsConfig
}

type Authorization struct {
	enabled bool
	groups  Groups
}

func NewAuthorization(config AuthorizationConfig) (*Authorization, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid authorization config: %w", err)
	}

	groups := make(Groups)

	for groupName, group := range config.Groups {
		g, err := NewGroup(groupName, group)
		if err != nil {
			return nil, fmt.Errorf("failed to create group %s: %w", groupName, err)
		}

		groups[groupName] = g
	}

	return &Authorization{
		enabled: config.Enabled,
		groups:  groups,
	}, nil
}

func (a *Authorization) Start(ctx context.Context) error {
	// Check if any of the groups have the same user
	userNames := make(map[string]bool)

	for _, group := range a.groups {
		for _, user := range group.Users().Usernames() {
			if userNames[user] {
				return fmt.Errorf("user %s already exists in multiple groups", user)
			}

			userNames[user] = true
		}
	}

	return nil
}

func (a *AuthorizationConfig) Validate() error {
	if !a.Enabled {
		return nil
	}

	if len(a.Groups) == 0 {
		return fmt.Errorf("no groups configured")
	}

	for groupName, group := range a.Groups {
		if err := group.Validate(); err != nil {
			return fmt.Errorf("group %s is invalid: %w", groupName, err)
		}
	}

	return nil
}

func (a *Authorization) IsAuthorized(token string) (bool, error) {
	parts := strings.SplitN(token, " ", 2)
	if len(parts) != 2 {
		return false, fmt.Errorf("invalid token format")
	}

	typ, value := parts[0], parts[1]

	switch typ {
	case "Basic":
		decodedBytes, err := base64.StdEncoding.DecodeString(value)
		if err != nil {
			return false, fmt.Errorf("failed to decode basic auth token: %w", err)
		}

		credentials := strings.SplitN(string(decodedBytes), ":", 2)
		if len(credentials) != 2 {
			return false, fmt.Errorf("invalid basic auth format")
		}

		username, password := credentials[0], credentials[1]

		return a.IsAuthorizedBasic(username, password)
	case "Bearer":
		return false, errors.New("bearer token not supported")
	default:
		return false, fmt.Errorf("unsupported token type: %s", typ)
	}
}

func (a *Authorization) IsAuthorizedBasic(username, password string) (bool, error) {
	if !a.enabled {
		return true, nil
	}

	for _, group := range a.groups {
		if group.Users().ValidUser(username, password) {
			return true, nil
		}
	}

	return false, nil
}
func (a *Authorization) GetUserAndGroup(username string) (*User, *Group, error) {
	for _, group := range a.groups {
		if user, ok := group.Users().GetUser(username); ok {
			return user, group, nil
		}
	}

	return nil, nil, fmt.Errorf("user %s not found", username)
}

func (a *Authorization) FilterEvents(user *User, group *Group, events []*xatu.DecoratedEvent) ([]*xatu.DecoratedEvent, error) {
	if !a.enabled {
		return events, nil
	}

	// Filter events for the user first since they're the most restrictive
	filteredUserEvents, err := user.ApplyFilter(events)
	if err != nil {
		return nil, fmt.Errorf("failed to filter events for user %s: %w", user.Username(), err)
	}

	// Then filter events for the group
	filteredGroupEvents, err := group.ApplyFilter(filteredUserEvents)
	if err != nil {
		return nil, fmt.Errorf("failed to filter events for group %s: %w", group.Name(), err)
	}

	return filteredGroupEvents, nil
}

func (a *Authorization) RedactEvents(group *Group, events []*xatu.DecoratedEvent) ([]*xatu.DecoratedEvent, error) {
	redactedEvents, err := group.ApplyRedacter(events)
	if err != nil {
		return nil, fmt.Errorf("failed to redact events for group %s: %w", group.Name(), err)
	}

	return redactedEvents, nil
}

func (a *Authorization) FilterAndRedactEvents(user *User, group *Group, events []*xatu.DecoratedEvent) ([]*xatu.DecoratedEvent, error) {
	// Filter first to save on processing
	filteredEvents, err := a.FilterEvents(user, group, events)
	if err != nil {
		return nil, fmt.Errorf("failed to filter events: %w", err)
	}

	redactedEvents, err := a.RedactEvents(group, filteredEvents)
	if err != nil {
		return nil, fmt.Errorf("failed to redact events: %w", err)
	}

	return redactedEvents, nil
}
