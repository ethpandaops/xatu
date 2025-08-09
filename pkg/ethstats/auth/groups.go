package auth

import "fmt"

type Group struct {
	name  string
	users map[string]*User
}

func NewGroup(name string, cfg GroupConfig) (*Group, error) {
	if name == "" {
		return nil, fmt.Errorf("group name cannot be empty")
	}

	group := &Group{
		name:  name,
		users: make(map[string]*User),
	}

	for username, userCfg := range cfg.Users {
		user, err := NewUser(username, userCfg)
		if err != nil {
			return nil, fmt.Errorf("failed to create user %s: %w", username, err)
		}

		group.users[username] = user
	}

	return group, nil
}

func (g *Group) Name() string {
	return g.name
}

func (g *Group) ValidateUser(username, password string) bool {
	user, exists := g.users[username]
	if !exists {
		return false
	}

	return user.ValidatePassword(password)
}

func (g *Group) HasUser(username string) bool {
	_, exists := g.users[username]

	return exists
}
