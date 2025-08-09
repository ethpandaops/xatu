package auth

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewGroup(t *testing.T) {
	tests := []struct {
		name    string
		group   string
		config  GroupConfig
		wantErr bool
	}{
		{
			name:  "valid group",
			group: "admins",
			config: GroupConfig{
				Users: map[string]UserConfig{
					"admin1": {Password: "pass1"},
					"admin2": {Password: "pass2"},
				},
			},
			wantErr: false,
		},
		{
			name:    "empty group name",
			group:   "",
			config:  GroupConfig{},
			wantErr: true,
		},
		{
			name:  "invalid user in group",
			group: "test",
			config: GroupConfig{
				Users: map[string]UserConfig{
					"": {Password: "pass"}, // empty username
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			group, err := NewGroup(tt.group, tt.config)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, group)
				assert.Equal(t, tt.group, group.Name())
			}
		})
	}
}

func TestGroup_ValidateUser(t *testing.T) {
	config := GroupConfig{
		Users: map[string]UserConfig{
			"user1": {Password: "correctpass"},
			"user2": {Password: "anotherpass"},
		},
	}

	group, err := NewGroup("testgroup", config)
	require.NoError(t, err)

	tests := []struct {
		name     string
		username string
		password string
		want     bool
	}{
		{
			name:     "valid user1",
			username: "user1",
			password: "correctpass",
			want:     true,
		},
		{
			name:     "valid user2",
			username: "user2",
			password: "anotherpass",
			want:     true,
		},
		{
			name:     "wrong password",
			username: "user1",
			password: "wrongpass",
			want:     false,
		},
		{
			name:     "non-existent user",
			username: "user3",
			password: "anypass",
			want:     false,
		},
		{
			name:     "empty username",
			username: "",
			password: "correctpass",
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := group.ValidateUser(tt.username, tt.password)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestGroup_HasUser(t *testing.T) {
	config := GroupConfig{
		Users: map[string]UserConfig{
			"user1": {Password: "pass1"},
			"user2": {Password: "pass2"},
		},
	}

	group, err := NewGroup("testgroup", config)
	require.NoError(t, err)

	assert.True(t, group.HasUser("user1"))
	assert.True(t, group.HasUser("user2"))
	assert.False(t, group.HasUser("user3"))
	assert.False(t, group.HasUser(""))
}
