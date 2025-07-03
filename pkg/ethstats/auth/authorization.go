package auth

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"
)

type Authorization struct {
	enabled bool
	groups  map[string]*Group
	log     logrus.FieldLogger
}

func NewAuthorization(log logrus.FieldLogger, cfg Config) (*Authorization, error) {
	auth := &Authorization{
		enabled: cfg.Enabled,
		groups:  make(map[string]*Group),
		log:     log,
	}

	if !cfg.Enabled {
		log.Info("Authorization is disabled")

		return auth, nil
	}

	for groupName, groupCfg := range cfg.Groups {
		group, err := NewGroup(groupName, groupCfg)
		if err != nil {
			return nil, fmt.Errorf("failed to create group %s: %w", groupName, err)
		}

		auth.groups[groupName] = group
	}

	log.WithField("groups", len(auth.groups)).Info("Authorization configured")

	return auth, nil
}

func (a *Authorization) Start(ctx context.Context) error {
	if !a.enabled {
		return nil
	}

	a.log.Info("Starting authorization service")

	return nil
}

func (a *Authorization) AuthorizeSecret(secret string) (username, group string, err error) {
	if !a.enabled {
		return "", "", nil
	}

	username, password, err := a.parseSecret(secret)
	if err != nil {
		return "", "", fmt.Errorf("failed to parse secret: %w", err)
	}

	for groupName, grp := range a.groups {
		if grp.ValidateUser(username, password) {
			return username, groupName, nil
		}
	}

	return "", "", fmt.Errorf("invalid credentials")
}

func (a *Authorization) parseSecret(secret string) (username, password string, err error){
	// The secret can be in format:
	// 1. base64(username:password)
	// 2. plain username:password (for compatibility)

	// Try base64 decode first
	decoded, err := base64.StdEncoding.DecodeString(secret)
	if err == nil {
		secret = string(decoded)
	}

	// Split username:password
	parts := strings.SplitN(secret, ":", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid secret format, expected username:password")
	}

	username = strings.TrimSpace(parts[0])
	password = strings.TrimSpace(parts[1])

	if username == "" || password == "" {
		return "", "", fmt.Errorf("username and password cannot be empty")
	}

	return username, password, nil
}
