package auth

import (
	"fmt"
)

type User struct {
	username string
	password string
}

func NewUser(username string, cfg UserConfig) (*User, error) {
	if username == "" {
		return nil, fmt.Errorf("username cannot be empty")
	}

	if cfg.Password == "" {
		return nil, fmt.Errorf("password cannot be empty")
	}

	return &User{
		username: username,
		password: cfg.Password,
	}, nil
}

func (u *User) Username() string {
	return u.username
}

func (u *User) ValidatePassword(password string) bool {
	return u.password == password
}
