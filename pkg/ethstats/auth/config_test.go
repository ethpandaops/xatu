package auth

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: Config{
				Enabled: true,
				Groups: map[string]GroupConfig{
					"admin": {
						Users: map[string]UserConfig{
							"admin": {Password: "pass"},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "disabled config",
			config: Config{
				Enabled: false,
			},
			wantErr: false,
		},
		{
			name: "enabled without groups",
			config: Config{
				Enabled: true,
				Groups:  map[string]GroupConfig{},
			},
			wantErr: true,
		},
		{
			name: "group without users",
			config: Config{
				Enabled: true,
				Groups: map[string]GroupConfig{
					"empty": {
						Users: map[string]UserConfig{},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "user without password",
			config: Config{
				Enabled: true,
				Groups: map[string]GroupConfig{
					"test": {
						Users: map[string]UserConfig{
							"user": {Password: ""},
						},
					},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestGroupConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  GroupConfig
		wantErr bool
	}{
		{
			name: "valid group",
			config: GroupConfig{
				Users: map[string]UserConfig{
					"user1": {Password: "pass1"},
					"user2": {Password: "pass2"},
				},
			},
			wantErr: false,
		},
		{
			name: "empty users",
			config: GroupConfig{
				Users: map[string]UserConfig{},
			},
			wantErr: true,
		},
		{
			name: "nil users",
			config: GroupConfig{
				Users: nil,
			},
			wantErr: true,
		},
		{
			name: "user with empty password",
			config: GroupConfig{
				Users: map[string]UserConfig{
					"user": {Password: ""},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestUserConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  UserConfig
		wantErr bool
	}{
		{
			name:    "valid user",
			config:  UserConfig{Password: "validpass"},
			wantErr: false,
		},
		{
			name:    "empty password",
			config:  UserConfig{Password: ""},
			wantErr: true,
		},
		{
			name:    "whitespace password",
			config:  UserConfig{Password: "   "},
			wantErr: false, // whitespace is technically allowed
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
