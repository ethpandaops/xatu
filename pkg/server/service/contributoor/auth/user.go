package auth

import (
	"fmt"
)

// UserConfig defines the configuration for a single user account.
// Currently only supports password-based authentication.
type UserConfig struct {
	Password string `yaml:"password"`
}

// UsersConfig is a map of usernames to their respective user configurations.
type UsersConfig map[string]UserConfig

// Validate ensures the user configuration is valid by checking required fields.
func (u *UserConfig) Validate() error {
	if u.Password == "" {
		return fmt.Errorf("password is required")
	}

	return nil
}

// Users is a map of usernames to their respective User instances for runtime use.
type Users map[string]*User

// User represents an authenticated user with their credentials.
type User struct {
	username string
	password string
}

// NewUser creates a new User instance from the provided username and configuration.
func NewUser(username string, u UserConfig) (*User, error) {
	return &User{
		username: username,
		password: u.Password,
	}, nil
}

// Username returns the username for this user.
func (u *User) Username() string {
	return u.username
}

// Password returns the password for this user.
func (u *User) Password() string {
	return u.password
}

// Usernames returns a slice of all configured usernames.
func (u *Users) Usernames() []string {
	usernames := make([]string, 0, len(*u))
	for username := range *u {
		usernames = append(usernames, username)
	}

	return usernames
}

// ValidUser checks if the provided username and password combination is valid.
func (u *Users) ValidUser(username, password string) bool {
	user, ok := (*u)[username]
	if !ok {
		return false
	}

	return user.password == password
}

// GetUser retrieves a user by username, returning the user and whether it was found.
func (u *Users) GetUser(username string) (*User, bool) {
	user, ok := (*u)[username]

	return user, ok
}
