package auth

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewUser(t *testing.T) {
	tests := []struct {
		name     string
		username string
		config   UserConfig
		wantErr  bool
	}{
		{
			name:     "valid user",
			username: "testuser",
			config:   UserConfig{Password: "testpass"},
			wantErr:  false,
		},
		{
			name:     "empty username",
			username: "",
			config:   UserConfig{Password: "testpass"},
			wantErr:  true,
		},
		{
			name:     "empty password",
			username: "testuser",
			config:   UserConfig{Password: ""},
			wantErr:  true,
		},
		{
			name:     "both empty",
			username: "",
			config:   UserConfig{Password: ""},
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user, err := NewUser(tt.username, tt.config)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, user)
				assert.Equal(t, tt.username, user.Username())
			}
		})
	}
}

func TestUser_ValidatePassword(t *testing.T) {
	user, err := NewUser("testuser", UserConfig{Password: "correctpass"})
	require.NoError(t, err)

	tests := []struct {
		name     string
		password string
		want     bool
	}{
		{
			name:     "correct password",
			password: "correctpass",
			want:     true,
		},
		{
			name:     "wrong password",
			password: "wrongpass",
			want:     false,
		},
		{
			name:     "empty password",
			password: "",
			want:     false,
		},
		{
			name:     "case sensitive",
			password: "CORRECTPASS",
			want:     false,
		},
		{
			name:     "extra spaces",
			password: " correctpass ",
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := user.ValidatePassword(tt.password)
			assert.Equal(t, tt.want, result)
		})
	}
}
