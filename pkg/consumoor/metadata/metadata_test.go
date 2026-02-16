package metadata

import (
	"testing"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/stretchr/testify/assert"
)

func TestExtract(t *testing.T) {
	tests := []struct {
		name     string
		event    *xatu.DecoratedEvent
		validate func(t *testing.T, m *CommonMetadata)
	}{
		{
			name:  "nil event meta",
			event: &xatu.DecoratedEvent{},
			validate: func(t *testing.T, m *CommonMetadata) {
				t.Helper()

				assert.Empty(t, m.MetaClientName)
				assert.Empty(t, m.MetaClientIP)
			},
		},
		{
			name: "full client metadata",
			event: &xatu.DecoratedEvent{
				Meta: &xatu.Meta{
					Client: &xatu.ClientMeta{
						Name:           "test-sentry",
						Id:             "abc-123",
						Version:        "0.1.0",
						Implementation: "Xatu",
						Os:             "linux",
						ClockDrift:     42,
						Ethereum: &xatu.ClientMeta_Ethereum{
							Network: &xatu.ClientMeta_Ethereum_Network{
								Name: "mainnet",
								Id:   1,
							},
							Consensus: &xatu.ClientMeta_Ethereum_Consensus{
								Implementation: "lighthouse",
								Version:        "Lighthouse/v4.5.6-abcdef/x86_64-linux",
							},
							Execution: &xatu.ClientMeta_Ethereum_Execution{
								Implementation: "geth",
								Version:        "v1.13.4",
								VersionMajor:   "1",
								VersionMinor:   "13",
								VersionPatch:   "4",
								ForkId: &xatu.ForkID{
									Hash: "0xdce96c2d",
									Next: "0",
								},
							},
						},
						Labels: map[string]string{"env": "prod"},
					},
					Server: &xatu.ServerMeta{
						Client: &xatu.ServerMeta_Client{
							IP: "1.2.3.4",
							Geo: &xatu.ServerMeta_Geo{
								City:                         "Berlin",
								Country:                      "Germany",
								CountryCode:                  "DE",
								ContinentCode:                "EU",
								Latitude:                     52.52,
								Longitude:                    13.405,
								AutonomousSystemNumber:       12345,
								AutonomousSystemOrganization: "Test ISP",
							},
						},
					},
				},
			},
			validate: func(t *testing.T, m *CommonMetadata) {
				t.Helper()

				// Client fields
				assert.Equal(t, "test-sentry", m.MetaClientName)
				assert.Equal(t, "abc-123", m.MetaClientID)
				assert.Equal(t, "0.1.0", m.MetaClientVersion)
				assert.Equal(t, "Xatu", m.MetaClientImplementation)
				assert.Equal(t, "linux", m.MetaClientOS)
				assert.Equal(t, uint64(42), m.MetaClientClockDrift)

				// Network
				assert.Equal(t, "mainnet", m.MetaNetworkName)
				assert.Equal(t, uint64(1), m.MetaNetworkID)

				// Consensus version parsing
				assert.Equal(t, "lighthouse", m.MetaConsensusImplementation)
				assert.Equal(t, "4", m.MetaConsensusVersionMajor)
				assert.Equal(t, "5", m.MetaConsensusVersionMinor)
				assert.Equal(t, "6", m.MetaConsensusVersionPatch)

				// Execution
				assert.Equal(t, "geth", m.MetaExecutionImplementation)
				assert.Equal(t, "1", m.MetaExecutionVersionMajor)
				assert.Equal(t, "0xdce96c2d", m.MetaExecutionForkIDHash)

				// IP normalization (IPv4 → IPv6-mapped)
				assert.Equal(t, "::ffff:1.2.3.4", m.MetaClientIP)

				// Geo
				assert.Equal(t, "Berlin", m.MetaClientGeoCity)
				assert.Equal(t, "DE", m.MetaClientGeoCountryCode)
				assert.Equal(t, 52.52, m.MetaClientGeoLatitude)

				// Labels
				assert.Equal(t, "prod", m.MetaLabels["env"])
			},
		},
		{
			name: "ipv6 passthrough",
			event: &xatu.DecoratedEvent{
				Meta: &xatu.Meta{
					Server: &xatu.ServerMeta{
						Client: &xatu.ServerMeta_Client{
							IP: "2001:db8::1",
						},
					},
				},
			},
			validate: func(t *testing.T, m *CommonMetadata) {
				t.Helper()

				assert.Equal(t, "2001:db8::1", m.MetaClientIP)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := Extract(tt.event)
			tt.validate(t, m)
		})
	}
}

func TestParseVersion(t *testing.T) {
	tests := []struct {
		input               string
		major, minor, patch string
	}{
		{"v1.2.3", "1", "2", "3"},
		{"Lighthouse/v4.5.6-abcdef/x86_64-linux", "4", "5", "6"},
		{"teku/teku/v1.2.3", "1", "2", "3"},
		{"1.13.4", "1", "13", "4"},
		{"v0.1.0-rc1", "0", "1", "0"},
		{"", "", "", ""},
		{"not-a-version", "", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			major, minor, patch := parseVersion(tt.input)
			assert.Equal(t, tt.major, major, "major")
			assert.Equal(t, tt.minor, minor, "minor")
			assert.Equal(t, tt.patch, patch, "patch")
		})
	}
}

func TestNormalizeIP(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"1.2.3.4", "::ffff:1.2.3.4"},
		{"2001:db8::1", "2001:db8::1"},
		{"", ""},
		{"invalid", "invalid"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, normalizeIP(tt.input))
		})
	}
}

func TestCommonMetadataToMap(t *testing.T) {
	m := &CommonMetadata{
		MetaClientName:  "test",
		MetaNetworkName: "mainnet",
		MetaLabels:      map[string]string{"k": "v"},
	}

	row := m.ToMap()

	assert.Equal(t, "test", row["meta_client_name"])
	assert.Equal(t, "mainnet", row["meta_network_name"])
	assert.Equal(t, map[string]string{"k": "v"}, row["meta_labels"])
}
