// Package auth provides Basic HTTP authentication for the contributoor service.
// It supports simple username/password validation without complex group-based permissions.
package auth

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"
)

// AuthorizationConfig defines the authentication configuration for the contributoor service.
// When enabled, all requests must provide valid Basic Auth credentials.
type AuthorizationConfig struct {
	Enabled bool        `yaml:"enabled"`
	Users   UsersConfig `yaml:"users"`
}

// Authorization handles Basic HTTP authentication for the contributoor service.
// It validates username/password combinations against the configured user list.
type Authorization struct {
	enabled bool
	users   *Users
	log     logrus.FieldLogger
}

// NewAuthorization creates a new authorization handler with the provided configuration.
// It validates the config and initializes the user authentication system.
func NewAuthorization(log logrus.FieldLogger, config AuthorizationConfig) (*Authorization, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid authorization config: %w", err)
	}

	users := make(Users)

	for username, user := range config.Users {
		u, err := NewUser(username, user)
		if err != nil {
			return nil, fmt.Errorf("failed to create user %s: %w", username, err)
		}

		users[username] = u
	}

	return &Authorization{
		enabled: config.Enabled,
		users:   &users,
		log:     log.WithField("server/module", "contributoor/auth"),
	}, nil
}

// Start initializes the authorization system and logs the number of configured users.
func (a *Authorization) Start(ctx context.Context) error {
	a.log.WithField("users", len(a.users.Usernames())).Info("Loaded users for authorization")

	return nil
}

// Validate ensures the authorization configuration is valid.
// It checks that users are configured when authentication is enabled.
func (a *AuthorizationConfig) Validate() error {
	if !a.Enabled {
		return nil
	}

	if len(a.Users) == 0 {
		return fmt.Errorf("no users configured")
	}

	for username, user := range a.Users {
		if err := user.Validate(); err != nil {
			return fmt.Errorf("user %s is invalid: %w", username, err)
		}
	}

	return nil
}

// IsAuthorized validates an authorization token and returns the authenticated username.
// It supports Basic Auth tokens in the format "Basic <base64-encoded-credentials>".
func (a *Authorization) IsAuthorized(token string) (string, error) {
	parts := strings.SplitN(token, " ", 2)
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid token format")
	}

	typ, value := parts[0], parts[1]

	switch typ {
	case "Basic":
		decodedBytes, err := base64.StdEncoding.DecodeString(value)
		if err != nil {
			return "", fmt.Errorf("failed to decode basic auth token: %w", err)
		}

		credentials := strings.SplitN(string(decodedBytes), ":", 2)
		if len(credentials) != 2 {
			return "", fmt.Errorf("invalid basic auth format")
		}

		username, password := credentials[0], credentials[1]

		authorized, err := a.IsAuthorizedBasic(username, password)
		if err != nil {
			return "", fmt.Errorf("failed to authorize user %s: %w", username, err)
		}

		if authorized {
			return username, nil
		}

		return "", fmt.Errorf("user %s not authorized", username)
	case "Bearer":
		return "", errors.New("bearer token not supported")
	default:
		return "", fmt.Errorf("unsupported token type: %s", typ)
	}
}

// IsAuthorizedBasic validates a username and password combination.
// If authentication is disabled, it always returns true.
func (a *Authorization) IsAuthorizedBasic(username, password string) (bool, error) {
	if !a.enabled {
		return true, nil
	}

	return a.users.ValidUser(username, password), nil
}

// GetUser retrieves a user by username if they exist in the configured user list.
func (a *Authorization) GetUser(username string) (*User, error) {
	if user, ok := a.users.GetUser(username); ok {
		return user, nil
	}

	return nil, fmt.Errorf("user %s not found", username)
}
