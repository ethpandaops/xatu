package libp2p

import "testing"

func TestParseAgentVersionKeepsCommit(t *testing.T) {
	tests := []struct {
		raw          string
		version      string
		platform     string
		majorVersion string
	}{
		{"Prysm/v7.3.1/96d418cd964b083e37519a9df274030a1beac7db", "v7.3.1", "96d418cd964b083e37519a9df274030a1beac7db", "7"},
		{"lighthouse/v8.2.3-6db6ae9/x86_64-linux", "v8.2.3-6db6ae9", "x86_64-linux", "8"},
		{"teku/teku/v26.10.3/linux-x86_64/-eclipseadoptium-openjdk64bitservervm-java-21", "v26.10.3", "linux-x86_64", "26"},
	}

	for _, tt := range tests {
		got := parseAgentVersion(tt.raw)
		if got.Version != tt.version || got.Platform != tt.platform || got.Major != tt.majorVersion {
			t.Errorf("parseAgentVersion(%q) = %+v, want version %q platform %q major %q", tt.raw, got, tt.version, tt.platform, tt.majorVersion)
		}
	}
}
