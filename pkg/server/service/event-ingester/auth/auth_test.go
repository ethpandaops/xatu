package auth_test

import (
	"context"
	"reflect"
	"testing"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/ethpandaops/xatu/pkg/server/service/event-ingester/auth"
	"google.golang.org/protobuf/proto"
)

func TestNewAuthorization(t *testing.T) {
	tests := []struct {
		name    string
		config  auth.AuthorizationConfig
		wantErr bool
	}{
		{
			name: "Authorization disabled",
			config: auth.AuthorizationConfig{
				Enabled: false,
			},
			wantErr: false,
		},
		{
			name: "Missing groups when enabled",
			config: auth.AuthorizationConfig{
				Enabled: true,
				Groups:  nil,
			},
			wantErr: true,
		},
		{
			name: "Valid configuration with multiple groups and users",
			config: auth.AuthorizationConfig{
				Enabled: true,
				Groups: auth.GroupsConfig{
					"group1": {
						Users: auth.UsersConfig{
							"user1": {Password: "password1"},
							"user2": {Password: "password2"},
						},
					},
					"group2": {
						Users: auth.UsersConfig{
							"user3": {Password: "password3"},
						},
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := auth.NewAuthorization(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewAuthorization() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAuthorization_IsAuthorized(t *testing.T) {
	authConfig := auth.AuthorizationConfig{
		Enabled: true,
		Groups: auth.GroupsConfig{
			"group1": {
				Users: auth.UsersConfig{
					"user1": {Password: "password1"},
					"user2": {Password: "password2"},
				},
			},
		},
	}

	authorization, err := auth.NewAuthorization(authConfig)
	if err != nil {
		t.Fatalf("Failed to create Authorization: %v", err)
	}

	tests := []struct {
		name     string
		username string
		password string
		want     bool
	}{
		{"Valid user1 credentials", "user1", "password1", true},
		{"Valid user2 credentials", "user2", "password2", true},
		{"Invalid password for user1", "user1", "wrongpassword", false},
		{"Non-existent user", "user3", "password3", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, errr := authorization.IsAuthorizedBasic(tt.username, tt.password)
			if errr != nil {
				t.Errorf("IsAuthorizedBasic(%s, %s) unexpected error: %v", tt.username, tt.password, errr)
			}

			if got != tt.want {
				t.Errorf("IsAuthorized(%s, %s) = %v, want %v", tt.username, tt.password, got, tt.want)
			}
		})
	}

	// Test with authorization disabled
	disabledAuthConfig := auth.AuthorizationConfig{
		Enabled: false,
	}

	disabledAuth, err := auth.NewAuthorization(disabledAuthConfig)
	if err != nil {
		t.Fatalf("Failed to create disabled Authorization: %v", err)
	}

	if err := disabledAuth.Start(context.Background()); err != nil {
		t.Fatalf("Failed to start disabled Authorization: %v", err)
	}

	t.Run("Authorization disabled - any credentials authorized", func(t *testing.T) {
		got, err := disabledAuth.IsAuthorizedBasic("anyuser", "anypassword")
		if err != nil {
			t.Errorf("IsAuthorizedBasic with disabled auth unexpected error: %v", err)
		}

		if !got {
			t.Errorf("IsAuthorized with disabled auth = %v, want true", got)
		}
	})
}

func TestAuthorization_GetUserAndGroup(t *testing.T) {
	authConfig := auth.AuthorizationConfig{
		Enabled: true,
		Groups: auth.GroupsConfig{
			"group1": {
				Users: auth.UsersConfig{
					"user1": {Password: "password1"},
					"user2": {Password: "password2"},
				},
			},
			"group2": {
				Users: auth.UsersConfig{
					"user3": {Password: "password3"},
				},
			},
		},
	}

	authorization, err := auth.NewAuthorization(authConfig)
	if err != nil {
		t.Fatalf("Failed to create Authorization: %v", err)
	}

	tests := []struct {
		name          string
		username      string
		wantUserFound bool
		wantGroupName string
	}{
		{"Retrieve existing user1", "user1", true, "group1"},
		{"Retrieve existing user2", "user2", true, "group1"},
		{"Retrieve existing user3", "user3", true, "group2"},
		{"Retrieve non-existent user4", "user4", false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user, group, err := authorization.GetUserAndGroup(tt.username)
			if tt.wantUserFound {
				if err != nil {
					t.Errorf("GetUserAndGroup(%s) unexpected error: %v", tt.username, err)
				}

				if user == nil {
					t.Errorf("GetUserAndGroup(%s) user is nil, expected non-nil", tt.username)
				}

				if group == nil {
					t.Errorf("GetUserAndGroup(%s) group is nil, expected non-nil", tt.username)
				}

				if group != nil && group.Name() != tt.wantGroupName {
					t.Errorf("GetUserAndGroup(%s) group name = %s, want %s", tt.username, group.Name(), tt.wantGroupName)
				}
			} else {
				if err == nil {
					t.Errorf("GetUserAndGroup(%s) expected error, got nil", tt.username)
				}

				if user != nil || group != nil {
					t.Errorf("GetUserAndGroup(%s) expected nil user and group for non-existent user", tt.username)
				}
			}
		})
	}
}

func TestUserConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  auth.UserConfig
		wantErr bool
	}{
		{
			name: "Valid user config",
			config: auth.UserConfig{
				Password: "securepassword",
			},
			wantErr: false,
		},
		{
			name: "Missing password",
			config: auth.UserConfig{
				Password: "",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("UserConfig.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestUser_ValidClientName(t *testing.T) {
	user, err := auth.NewUser("clientUser", auth.UserConfig{
		Password: "password",
	})
	if err != nil {
		t.Fatalf("Failed to create User: %v", err)
	}

	tests := []struct {
		name       string
		clientName string
		want       bool
	}{
		{"Valid prefix", "clientUser123", true},
		{"Exact match", "clientUser", true},
		{"Invalid prefix", "otherUser", false},
		{"Empty client name", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := user.ValidClientName(tt.clientName)
			if got != tt.want {
				t.Errorf("ValidClientName(%s) = %v, want %v", tt.clientName, got, tt.want)
			}
		})
	}
}

func TestGroup_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  auth.GroupConfig
		wantErr bool
	}{
		{
			name: "Valid group config",
			config: auth.GroupConfig{
				Users: auth.UsersConfig{
					"user1": {Password: "password1"},
					"user2": {Password: "password2"},
				},
				EventFilter: &xatu.EventFilterConfig{},
			},
			wantErr: false,
		},
		{
			name: "Empty users",
			config: auth.GroupConfig{
				Users:       auth.UsersConfig{},
				EventFilter: &xatu.EventFilterConfig{},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("GroupConfig.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAuthorization_Start(t *testing.T) {
	tests := []struct {
		name    string
		config  auth.AuthorizationConfig
		wantErr bool
	}{
		{
			name: "Valid configuration",
			config: auth.AuthorizationConfig{
				Enabled: true,
				Groups: auth.GroupsConfig{
					"group1": {
						Users: auth.UsersConfig{
							"user1": {Password: "password1"},
							"user2": {Password: "password2"},
						},
					},
					"group2": {
						Users: auth.UsersConfig{
							"user3": {Password: "password3"},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Duplicate users across groups",
			config: auth.AuthorizationConfig{
				Enabled: true,
				Groups: auth.GroupsConfig{
					"group1": {
						Users: auth.UsersConfig{
							"user1": {Password: "password1"},
							"user2": {Password: "password2"},
						},
					},
					"group2": {
						Users: auth.UsersConfig{
							"user1": {Password: "password3"},
						},
					},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a, errr := auth.NewAuthorization(tt.config)
			if errr != nil {
				t.Fatalf("Failed to create Authorization: %v", errr)
			}

			errr = a.Start(context.Background())
			if (errr != nil) != tt.wantErr {
				t.Errorf("Authorization.Start() error = %v, wantErr %v", errr, tt.wantErr)
			}
		})
	}
}

func TestAuthorization_MultipleFilters(t *testing.T) {
	username := "user1"
	password := "password1"

	authConfig := auth.AuthorizationConfig{
		Enabled: true,
		Groups: auth.GroupsConfig{
			"group1": {
				Users: auth.UsersConfig{
					username: {
						Password: password,
						EventFilter: &xatu.EventFilterConfig{
							EventNames: []string{
								"BEACON_API_ETH_V2_BEACON_BLOCK_VOLUNTARY_EXIT",
								"BEACON_API_ETH_V2_BEACON_BLOCK_ATTESTER_SLASHING",
							},
						},
					},
				},
				EventFilter: &xatu.EventFilterConfig{
					EventNames: []string{"BEACON_API_ETH_V2_BEACON_BLOCK_ATTESTER_SLASHING"},
				},
			},
		},
	}

	tests := []struct {
		name   string
		input  []*xatu.DecoratedEvent
		output []*xatu.DecoratedEvent
	}{
		{
			"Event in both filters",
			[]*xatu.DecoratedEvent{
				{
					Event: &xatu.Event{
						Name: xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_ATTESTER_SLASHING,
					},
				},
				{
					Event: &xatu.Event{
						Name: xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_VOLUNTARY_EXIT,
					},
				},
			},
			[]*xatu.DecoratedEvent{
				{
					Event: &xatu.Event{
						Name: xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_ATTESTER_SLASHING,
					},
				},
			},
		},
	}

	authorization, err := auth.NewAuthorization(authConfig)
	if err != nil {
		t.Fatalf("Failed to create Authorization: %v", err)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user, group, err := authorization.GetUserAndGroup(username)
			if err != nil {
				t.Fatalf("Failed to get user and group: %v", err)
			}

			got, err := authorization.FilterEvents(user, group, tt.input)
			if err != nil {
				t.Fatalf("Failed to filter events: %v", err)
			}

			if !reflect.DeepEqual(got, tt.output) {
				t.Errorf("FilterEvents(%v) = %v, want %v", tt.input, got, tt.output)
			}
		})
	}
}

func TestAuthorization_NoFilter(t *testing.T) {
	username := "user1"
	password := "password1"

	authConfig := auth.AuthorizationConfig{
		Enabled: true,
		Groups: auth.GroupsConfig{
			"group1": {
				Users: auth.UsersConfig{
					username: {
						Password: password,
						// No EventFilter specified for the user
					},
				},
				// No EventFilter specified for the group
			},
		},
	}

	tests := []struct {
		name   string
		input  []*xatu.DecoratedEvent
		output []*xatu.DecoratedEvent
	}{
		{
			"All events allowed",
			[]*xatu.DecoratedEvent{
				{
					Event: &xatu.Event{
						Name: xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_ATTESTER_SLASHING,
					},
				},
				{
					Event: &xatu.Event{
						Name: xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_VOLUNTARY_EXIT,
					},
				},
				{
					Event: &xatu.Event{
						Name: xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK,
					},
				},
			},
			[]*xatu.DecoratedEvent{
				{
					Event: &xatu.Event{
						Name: xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_ATTESTER_SLASHING,
					},
				},
				{
					Event: &xatu.Event{
						Name: xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_VOLUNTARY_EXIT,
					},
				},
				{
					Event: &xatu.Event{
						Name: xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK,
					},
				},
			},
		},
	}

	authorization, err := auth.NewAuthorization(authConfig)
	if err != nil {
		t.Fatalf("Failed to create Authorization: %v", err)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user, group, err := authorization.GetUserAndGroup(username)
			if err != nil {
				t.Fatalf("Failed to get user and group: %v", err)
			}

			got, err := authorization.FilterEvents(user, group, tt.input)
			if err != nil {
				t.Fatalf("Failed to filter events: %v", err)
			}

			if !reflect.DeepEqual(got, tt.output) {
				t.Errorf("FilterEvents(%v) = %v, want %v", tt.input, got, tt.output)
			}
		})
	}
}

func TestAuthorization_FilterAndRedactEvents(t *testing.T) {
	username := "user1"
	password := "password1"

	authConfig := auth.AuthorizationConfig{
		Enabled: true,
		Groups: auth.GroupsConfig{
			"group1": {
				Users: auth.UsersConfig{
					username: {
						Password: password,
						EventFilter: &xatu.EventFilterConfig{
							EventNames: []string{
								"BEACON_API_ETH_V2_BEACON_BLOCK_VOLUNTARY_EXIT",
								"BEACON_API_ETH_V2_BEACON_BLOCK_ATTESTER_SLASHING",
							},
						},
					},
				},
				EventFilter: &xatu.EventFilterConfig{
					EventNames: []string{"BEACON_API_ETH_V2_BEACON_BLOCK_ATTESTER_SLASHING"},
				},
				Redacter: &xatu.RedacterConfig{
					FieldPaths: []string{"meta.server.client.IP"},
				},
			},
		},
	}

	tests := []struct {
		name   string
		input  []*xatu.DecoratedEvent
		output []*xatu.DecoratedEvent
	}{
		{
			"Event in both filters",
			[]*xatu.DecoratedEvent{
				{
					Event: &xatu.Event{
						Name: xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_ATTESTER_SLASHING,
					},
					Meta: &xatu.Meta{
						Server: &xatu.ServerMeta{
							Client: &xatu.ServerMeta_Client{
								IP: "192.168.1.1",
								Geo: &xatu.ServerMeta_Geo{
									Country: "US",
								},
							},
						},
					},
				},
				{
					Event: &xatu.Event{
						Name: xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_VOLUNTARY_EXIT,
					},
					Meta: &xatu.Meta{
						Server: &xatu.ServerMeta{
							Client: &xatu.ServerMeta_Client{
								Geo: &xatu.ServerMeta_Geo{
									Country: "US",
								},
								IP: "192.168.1.1",
							},
						},
					},
				},
			},
			[]*xatu.DecoratedEvent{
				{
					Event: &xatu.Event{
						Name: xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_ATTESTER_SLASHING,
					},
					Meta: &xatu.Meta{
						Server: &xatu.ServerMeta{
							Client: &xatu.ServerMeta_Client{
								Geo: &xatu.ServerMeta_Geo{
									Country: "US",
								},
							},
						},
					},
				},
			},
		},
	}

	authorization, err := auth.NewAuthorization(authConfig)
	if err != nil {
		t.Fatalf("Failed to create Authorization: %v", err)
	}

	if err := authorization.Start(context.Background()); err != nil {
		t.Fatalf("Failed to start Authorization: %v", err)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			authorized, err := authorization.IsAuthorizedBasic(username, password)
			if err != nil {
				t.Fatalf("Failed to check authorization: %v", err)
			}

			if !authorized {
				t.Fatalf("User %s is not authorized", username)
			}

			user, group, err := authorization.GetUserAndGroup(username)
			if err != nil {
				t.Fatalf("Failed to get user and group: %v", err)
			}

			got, err := authorization.FilterAndRedactEvents(user, group, tt.input)
			if err != nil {
				t.Fatalf("Failed to filter events: %v", err)
			}

			for i, event := range got {
				if !proto.Equal(event, tt.output[i]) {
					t.Errorf("FilterEvents(%v) = %v, want %v", tt.name, got, tt.output)
				}
			}
		})
	}
}
