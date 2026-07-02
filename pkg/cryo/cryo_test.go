package cryo

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
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
