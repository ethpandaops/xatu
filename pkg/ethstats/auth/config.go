package auth

import "fmt"

type Config struct {
	Enabled bool                   `yaml:"enabled" default:"true"`
	Groups  map[string]GroupConfig `yaml:"groups"`
}

type GroupConfig struct {
	Users map[string]UserConfig `yaml:"users"`
}

type UserConfig struct {
	Password string `yaml:"password"`
}

func (c *Config) Validate() error {
	if !c.Enabled {
		return nil
	}

	if len(c.Groups) == 0 {
		return fmt.Errorf("at least one group must be configured when auth is enabled")
	}

	for groupName, group := range c.Groups {
		if err := group.Validate(); err != nil {
			return fmt.Errorf("group %s validation failed: %w", groupName, err)
		}
	}

	return nil
}

func (c *GroupConfig) Validate() error {
	if len(c.Users) == 0 {
		return fmt.Errorf("at least one user must be configured in the group")
	}

	for username, user := range c.Users {
		if err := user.Validate(); err != nil {
			return fmt.Errorf("user %s validation failed: %w", username, err)
		}
	}

	return nil
}

func (c *UserConfig) Validate() error {
	if c.Password == "" {
		return fmt.Errorf("password is required")
	}

	return nil
}
