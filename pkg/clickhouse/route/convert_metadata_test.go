package route

import (
	"net/netip"
	"testing"

	"github.com/ClickHouse/ch-go/proto"
	"github.com/stretchr/testify/assert"
)

func TestNormalizeIPToIPv6(t *testing.T) {
	tests := []struct {
		input    string
		expected proto.IPv6
	}{
		{
			"1.2.3.4",
			proto.ToIPv6(netip.MustParseAddr("::ffff:1.2.3.4")),
		},
		{
			"2001:db8::1",
			proto.ToIPv6(netip.MustParseAddr("2001:db8::1")),
		},
		{"", proto.IPv6{}},
		{"invalid", proto.IPv6{}},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, NormalizeIPToIPv6(tt.input))
		})
	}
}

func TestNormalizeConsensusVersion(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Lighthouse/v4.5.6-abcdef/x86_64-linux", "v4.5.6-abcdef"},
		{"v1.2.3", "v1.2.3"},
		{"teku/teku/v1.2.3", "teku"},
		{"", ""},
		{"no-slash", "no-slash"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, NormalizeConsensusVersion(tt.input))
		})
	}
}

func TestConsensusVersionComponents(t *testing.T) {
	tests := []struct {
		input               string
		major, minor, patch string
	}{
		{"v1.2.3", "1", "2", "3"},
		{"Lighthouse/v4.5.6-abcdef/x86_64-linux", "4", "5", "6"},
		{"teku/teku/v1.2.3", "teku", "", ""},
		{"1.13.4", "1", "13", "4"},
		{"v0.1.0-rc1", "0", "1", "0"},
		{"", "", "", ""},
		{"not-a-version", "not-a-version", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.major, ConsensusVersionMajor(tt.input), "major")
			assert.Equal(t, tt.minor, ConsensusVersionMinor(tt.input), "minor")
			assert.Equal(t, tt.patch, ConsensusVersionPatch(tt.input), "patch")
		})
	}
}
