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

	// Log configured groups and users
	for groupName, grp := range a.groups {
		userCount := 0
		usernames := []string{}

		for username := range grp.users {
			userCount++

			usernames = append(usernames, username)
		}

		a.log.WithFields(logrus.Fields{
			"group":      groupName,
			"user_count": userCount,
			"users":      usernames,
		}).Info("Configured group")
	}

	return nil
}

func (a *Authorization) AuthorizeSecret(secret string) (username, group string, err error) {
	if !a.enabled {
		a.log.Debug("Authorization disabled, allowing access")
		return "", "", nil
	}

	a.log.WithField("secret_length", len(secret)).Debug("Authorizing secret")

	username, password, err := a.parseSecret(secret)
	if err != nil {
		a.log.WithError(err).Warn("Failed to parse secret")
		return "", "", fmt.Errorf("failed to parse secret: %w", err)
	}

	a.log.WithFields(logrus.Fields{
		"username": username,
		"groups":   len(a.groups),
	}).Debug("Checking user against groups")

	for groupName, grp := range a.groups {
		a.log.WithFields(logrus.Fields{
			"username": username,
			"group":    groupName,
			"has_user": grp.HasUser(username),
		}).Debug("Checking user in group")

		if grp.ValidateUser(username, password) {
			a.log.WithFields(logrus.Fields{
				"username": username,
				"group":    groupName,
			}).Info("User authorized")
			return username, groupName, nil
		}
	}

	a.log.WithFields(logrus.Fields{
		"username":       username,
		"groups_checked": len(a.groups),
	}).Warn("Invalid credentials - user not found or password incorrect")
	return "", "", fmt.Errorf("invalid credentials")
}

func (a *Authorization) parseSecret(secret string) (username, password string, err error) {
	// The secret can be in format:
	// 1. base64(username:password)
	// 2. plain username:password (for compatibility)

	originalSecret := secret
	isBase64 := false

	// Try base64 decode first
	decoded, err := base64.StdEncoding.DecodeString(secret)
	if err == nil {
		secret = string(decoded)
		isBase64 = true
	}

	a.log.WithFields(logrus.Fields{
		"original_length": len(originalSecret),
		"decoded_length":  len(secret),
		"is_base64":       isBase64,
	}).Debug("Parsing secret")

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

	a.log.WithField("username", username).Debug("Parsed credentials")

	return username, password, nil
}
