package cryo

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCryoOutput(t *testing.T) {
	tests := []struct {
		name   string
		stdout string
		stderr string
		want   string
	}{
		{
			name:   "stdout only (cryo's normal failure path)",
			stdout: "Failed to get block: error sending request\n",
			stderr: "",
			want:   "Failed to get block: error sending request",
		},
		{
			name:   "stderr only",
			stdout: "",
			stderr: "some stderr message\n",
			want:   "some stderr message",
		},
		{
			name:   "both populated",
			stdout: "stdout cause",
			stderr: "stderr detail",
			want:   "stdout cause; stderr: stderr detail",
		},
		{
			name:   "both empty",
			stdout: "",
			stderr: "  \n",
			want:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cryoOutput(bytes.NewBufferString(tt.stdout), bytes.NewBufferString(tt.stderr))
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestRedactRPC(t *testing.T) {
	tests := []struct {
		name string
		in   string
		rpc  string
		want string
	}{
		{
			name: "full url replaced with redacted form",
			in:   "rpc url: https://user:s3cret@node.example:8545/path",
			rpc:  "https://user:s3cret@node.example:8545/path",
			want: "rpc url: https://user:xxxxx@node.example:8545/path",
		},
		{
			name: "reformatted url still has password masked",
			in:   "rpc url: https://user:s3cret@node.example:8545/path/",
			rpc:  "https://user:s3cret@node.example:8545/path",
			want: "rpc url: https://user:xxxxx@node.example:8545/path/",
		},
		{
			name: "url without credentials unchanged",
			in:   "rpc url: http://localhost:8545",
			rpc:  "http://localhost:8545",
			want: "rpc url: http://localhost:8545",
		},
		{
			name: "username only unchanged",
			in:   "rpc url: https://user@node.example",
			rpc:  "https://user@node.example",
			want: "rpc url: https://user@node.example",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, redactRPC(tt.in, tt.rpc))
		})
	}
}

func TestCollectFailureRedactsRPC(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fake cryo binary is a shell script")
	}

	dir := t.TempDir()
	bin := filepath.Join(dir, "cryo")

	// Fake cryo: echo the RPC URL it was given and its args like cryo's
	// parameter summary, then fail.
	script := "#!/bin/sh\necho \"rpc url: $ETH_RPC_URL\"\necho \"args: $*\"\nexit 1\n"
	require.NoError(t, os.WriteFile(bin, []byte(script), 0o700)) //nolint:gosec // test helper must be executable.

	const password = "s3cret-pass"

	rpc := "https://user:" + password + "@node.example:8545"
	r := New(logrus.New(), &Config{BinaryPath: bin, OutputDir: filepath.Join(dir, "out"), MaxRangeSize: 1}, rpc)

	_, err := r.Collect(context.Background(), "blocks", 1, 2, nil)
	require.Error(t, err)
	assert.NotContains(t, err.Error(), password)
	assert.Contains(t, err.Error(), "rpc url: https://user:xxxxx@node.example:8545")
	assert.NotContains(t, err.Error(), "--rpc")
}
