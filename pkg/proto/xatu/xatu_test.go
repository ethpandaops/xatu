package xatu

import "testing"

func TestWithModule(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		module ModuleName
		want   string
	}{
		{
			name:   "server",
			module: ModuleName_SERVER,
			want:   "xatu-server",
		},
		{
			name:   "el mimicry",
			module: ModuleName_EL_MIMICRY,
			want:   "xatu-el-mimicry",
		},
		{
			name:   "relay monitor",
			module: ModuleName_RELAY_MONITOR,
			want:   "xatu-relay-monitor",
		},
		{
			name:   "rpc snooper",
			module: ModuleName_RPC_SNOOPER,
			want:   "xatu-rpc-snooper",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := WithModule(tt.module); got != tt.want {
				t.Fatalf("WithModule(%s) = %q, want %q", tt.module, got, tt.want)
			}
		})
	}
}

func TestWithComponent(t *testing.T) {
	t.Parallel()

	if got := WithComponent("consumoor"); got != "xatu-consumoor" {
		t.Fatalf("WithComponent(%q) = %q, want %q", "consumoor", got, "xatu-consumoor")
	}
}

func TestFullWithModule(t *testing.T) {
	originalRelease := Release
	originalGitCommit := GitCommit

	t.Cleanup(func() {
		Release = originalRelease
		GitCommit = originalGitCommit
	})

	Release = "v1.2.3"
	GitCommit = "abcdef"

	if got := FullWithModule(ModuleName_SERVER); got != "xatu-server/v1.2.3-abcdef" {
		t.Fatalf("FullWithModule(%s) = %q, want %q", ModuleName_SERVER, got, "xatu-server/v1.2.3-abcdef")
	}
}
