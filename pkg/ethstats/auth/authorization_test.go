package auth

import (
	"context"
	"encoding/base64"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAuthorization(t *testing.T) {
	log := logrus.New()

	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name: "valid config with groups",
			config: Config{
				Enabled: true,
				Groups: map[string]GroupConfig{
					"admin": {
						Users: map[string]UserConfig{
							"user1": {Password: "pass1"},
							"user2": {Password: "pass2"},
						},
					},
					"readonly": {
						Users: map[string]UserConfig{
							"viewer": {Password: "viewpass"},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "disabled auth",
			config: Config{
				Enabled: false,
			},
			wantErr: false,
		},
		{
			name: "invalid group config",
			config: Config{
				Enabled: true,
				Groups: map[string]GroupConfig{
					"": { // empty group name
						Users: map[string]UserConfig{
							"user1": {Password: "pass1"},
						},
					},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			auth, err := NewAuthorization(log, tt.config)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, auth)
				assert.Equal(t, tt.config.Enabled, auth.enabled)
			}
		})
	}
}

func TestAuthorization_Start(t *testing.T) {
	log := logrus.New()
	auth, err := NewAuthorization(log, Config{Enabled: true})
	require.NoError(t, err)

	ctx := context.Background()
	err = auth.Start(ctx)
	assert.NoError(t, err)
}

func TestAuthorization_AuthorizeSecret(t *testing.T) {
	log := logrus.New()

	config := Config{
		Enabled: true,
		Groups: map[string]GroupConfig{
			"admin": {
				Users: map[string]UserConfig{
					"admin": {Password: "adminpass"},
				},
			},
			"operators": {
				Users: map[string]UserConfig{
					"operator1": {Password: "operpass1"},
					"operator2": {Password: "operpass2"},
				},
			},
		},
	}

	auth, err := NewAuthorization(log, config)
	require.NoError(t, err)

	tests := []struct {
		name         string
		secret       string
		wantUsername string
		wantGroup    string
		wantErr      bool
	}{
		{
			name:         "valid admin plain",
			secret:       "admin:adminpass",
			wantUsername: "admin",
			wantGroup:    "admin",
		},
		{
			name:         "valid admin base64",
			secret:       base64.StdEncoding.EncodeToString([]byte("admin:adminpass")),
			wantUsername: "admin",
			wantGroup:    "admin",
		},
		{
			name:         "valid operator1",
			secret:       "operator1:operpass1",
			wantUsername: "operator1",
			wantGroup:    "operators",
		},
		{
			name:         "valid operator2",
			secret:       "operator2:operpass2",
			wantUsername: "operator2",
			wantGroup:    "operators",
		},
		{
			name:    "invalid password",
			secret:  "admin:wrongpass",
			wantErr: true,
		},
		{
			name:    "non-existent user",
			secret:  "nobody:pass",
			wantErr: true,
		},
		{
			name:    "malformed secret",
			secret:  "invalidformat",
			wantErr: true,
		},
		{
			name:    "empty username",
			secret:  ":password",
			wantErr: true,
		},
		{
			name:    "empty password",
			secret:  "username:",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			username, group, err := auth.AuthorizeSecret(tt.secret)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantUsername, username)
				assert.Equal(t, tt.wantGroup, group)
			}
		})
	}
}

func TestAuthorization_AuthorizeSecret_Disabled(t *testing.T) {
	log := logrus.New()
	auth, err := NewAuthorization(log, Config{Enabled: false})
	require.NoError(t, err)

	username, group, err := auth.AuthorizeSecret("anything")
	assert.NoError(t, err)
	assert.Empty(t, username)
	assert.Empty(t, group)
}

func TestAuthorization_parseSecret(t *testing.T) {
	auth := &Authorization{}

	tests := []struct {
		name         string
		secret       string
		wantUsername string
		wantPassword string
		wantErr      bool
	}{
		{
			name:         "plain format",
			secret:       "user:pass",
			wantUsername: "user",
			wantPassword: "pass",
		},
		{
			name:         "base64 encoded",
			secret:       base64.StdEncoding.EncodeToString([]byte("user:pass")),
			wantUsername: "user",
			wantPassword: "pass",
		},
		{
			name:         "with spaces",
			secret:       " user : pass ",
			wantUsername: "user",
			wantPassword: "pass",
		},
		{
			name:         "password with colon",
			secret:       "user:pass:with:colon",
			wantUsername: "user",
			wantPassword: "pass:with:colon",
		},
		{
			name:    "no colon",
			secret:  "invalid",
			wantErr: true,
		},
		{
			name:    "empty username",
			secret:  ":pass",
			wantErr: true,
		},
		{
			name:    "empty password",
			secret:  "user:",
			wantErr: true,
		},
		{
			name:    "empty secret",
			secret:  "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			username, password, err := auth.parseSecret(tt.secret)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantUsername, username)
				assert.Equal(t, tt.wantPassword, password)
			}
		})
	}
}
