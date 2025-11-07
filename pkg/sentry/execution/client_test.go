package execution

import (
	"testing"
)

func TestParseClientVersion(t *testing.T) {
	tests := []struct {
		name             string
		clientVersion    string
		wantImpl         string
		wantVersion      string
		wantVersionMajor string
		wantVersionMinor string
		wantVersionPatch string
	}{
		{
			name:             "Geth with stable suffix",
			clientVersion:    "Geth/v1.16.4-stable-41714b49/linux-amd64/go1.24.7",
			wantImpl:         "Geth",
			wantVersion:      "1.16.4-stable-41714b49",
			wantVersionMajor: "1",
			wantVersionMinor: "16",
			wantVersionPatch: "4",
		},
		{
			name:             "Erigon without v prefix",
			clientVersion:    "erigon/3.0.14/linux-amd64/go1.23.11",
			wantImpl:         "erigon",
			wantVersion:      "3.0.14",
			wantVersionMajor: "3",
			wantVersionMinor: "0",
			wantVersionPatch: "14",
		},
		{
			name:             "Nethermind with commit hash",
			clientVersion:    "Nethermind/v1.32.4+1c4c7c0a/linux-x64/dotnet9.0.7",
			wantImpl:         "Nethermind",
			wantVersion:      "1.32.4+1c4c7c0a",
			wantVersionMajor: "1",
			wantVersionMinor: "32",
			wantVersionPatch: "4",
		},
		{
			name:             "Besu standard format",
			clientVersion:    "besu/v25.7.0/linux-x86_64/openjdk-java-21",
			wantImpl:         "besu",
			wantVersion:      "25.7.0",
			wantVersionMajor: "25",
			wantVersionMinor: "7",
			wantVersionPatch: "0",
		},
		{
			name:             "Reth with commit hash",
			clientVersion:    "reth/v1.8.2-9c30bf7/x86_64-unknown-linux-gnu",
			wantImpl:         "reth",
			wantVersion:      "1.8.2-9c30bf7",
			wantVersionMajor: "1",
			wantVersionMinor: "8",
			wantVersionPatch: "2",
		},
		{
			name:             "Empty string",
			clientVersion:    "",
			wantImpl:         "",
			wantVersion:      "",
			wantVersionMajor: "",
			wantVersionMinor: "",
			wantVersionPatch: "",
		},
		{
			name:             "Only implementation name",
			clientVersion:    "CustomClient",
			wantImpl:         "CustomClient",
			wantVersion:      "",
			wantVersionMajor: "",
			wantVersionMinor: "",
			wantVersionPatch: "",
		},
		{
			name:             "Version without patch",
			clientVersion:    "TestClient/v1.2",
			wantImpl:         "TestClient",
			wantVersion:      "1.2",
			wantVersionMajor: "1",
			wantVersionMinor: "2",
			wantVersionPatch: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotImpl, gotVersion, gotMajor, gotMinor, gotPatch := parseClientVersion(tt.clientVersion)

			if gotImpl != tt.wantImpl {
				t.Errorf("implementation: got %q, want %q", gotImpl, tt.wantImpl)
			}

			if gotVersion != tt.wantVersion {
				t.Errorf("version: got %q, want %q", gotVersion, tt.wantVersion)
			}

			if gotMajor != tt.wantVersionMajor {
				t.Errorf("versionMajor: got %q, want %q", gotMajor, tt.wantVersionMajor)
			}

			if gotMinor != tt.wantVersionMinor {
				t.Errorf("versionMinor: got %q, want %q", gotMinor, tt.wantVersionMinor)
			}

			if gotPatch != tt.wantVersionPatch {
				t.Errorf("versionPatch: got %q, want %q", gotPatch, tt.wantVersionPatch)
			}
		})
	}
}

func TestSplitString(t *testing.T) {
	tests := []struct {
		name string
		s    string
		sep  string
		want []string
	}{
		{
			name: "Split by slash",
			s:    "Geth/v1.2.3/linux",
			sep:  "/",
			want: []string{"Geth", "v1.2.3", "linux"},
		},
		{
			name: "Split by dot",
			s:    "1.2.3",
			sep:  ".",
			want: []string{"1", "2", "3"},
		},
		{
			name: "Empty string",
			s:    "",
			sep:  "/",
			want: nil,
		},
		{
			name: "No separator found",
			s:    "noseparator",
			sep:  "/",
			want: []string{"noseparator"},
		},
		{
			name: "Multiple consecutive separators",
			s:    "a//b//c",
			sep:  "/",
			want: []string{"a", "", "b", "", "c"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := splitString(tt.s, tt.sep)

			if len(got) != len(tt.want) {
				t.Errorf("length mismatch: got %d, want %d", len(got), len(tt.want))

				return
			}

			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("element %d: got %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}
