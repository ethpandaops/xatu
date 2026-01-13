package auth

import (
	"context"
	"fmt"
	"strings"

	"github.com/ethpandaops/xatu/pkg/observability"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type UserConfig struct {
	Password    string                  `yaml:"password"`
	EventFilter *xatu.EventFilterConfig `yaml:"eventFilter"`
}

type UsersConfig map[string]UserConfig

func (u *UserConfig) Validate() error {
	if u.Password == "" {
		return fmt.Errorf("password is required")
	}

	return nil
}

type Users map[string]*User

type User struct {
	username    string
	password    string
	eventFilter xatu.EventFilter
}

func (u *User) ValidClientName(clientName string) bool {
	return strings.HasPrefix(clientName, u.username)
}

func NewUser(username string, u UserConfig) (*User, error) {
	user := &User{
		username: username,
		password: u.Password,
	}

	if u.EventFilter != nil {
		filter, err := xatu.NewEventFilter(u.EventFilter)
		if err != nil {
			return nil, fmt.Errorf("failed to create event filter: %w", err)
		}

		user.eventFilter = filter
	}

	return user, nil
}

func (u *User) Username() string {
	return u.username
}

func (u *User) Password() string {
	return u.password
}

func (u *Users) Usernames() []string {
	usernames := make([]string, 0, len(*u))
	for username := range *u {
		usernames = append(usernames, username)
	}

	return usernames
}

func (u *Users) ValidUser(username, password string) bool {
	user, ok := (*u)[username]
	if !ok {
		return false
	}

	return user.password == password
}

func (u *Users) GetUser(username string) (*User, bool) {
	user, ok := (*u)[username]

	return user, ok
}

func (u *User) ApplyFilter(ctx context.Context, events []*xatu.DecoratedEvent) ([]*xatu.DecoratedEvent, error) {
	if u.eventFilter == nil {
		return events, nil
	}

	_, span := observability.Tracer().Start(ctx,
		"User.ApplyFilter",
		trace.WithAttributes(attribute.Int64("events", int64(len(events)))),
	)
	defer span.End()

	filteredEvents := make([]*xatu.DecoratedEvent, 0)

	for _, event := range events {
		shouldBeDropped, err := u.eventFilter.ShouldBeDropped(event)
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
