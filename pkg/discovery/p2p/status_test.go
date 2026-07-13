package p2p

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExecutionDialFailureReason(t *testing.T) {
	tests := []struct {
		name             string
		disconnectReason string
		response         error
		want             string
	}{
		{
			name:             "remote disconnect reason wins",
			disconnectReason: "too many peers",
			response:         errors.New("disconnected from peer (reason too many peers)"),
			want:             "too many peers",
		},
		{
			name:     "timeout",
			response: errDialTimeout,
			want:     dialFailureTimeout,
		},
		{
			name:     "context canceled",
			response: context.Canceled,
			want:     dialFailureCanceled,
		},
		{
			name:     "dial error",
			response: errors.New("connection refused"),
			want:     dialFailureError,
		},
		{
			name: "no response",
			want: dialFailureError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			peer := &ExecutionPeer{}

			if tt.disconnectReason != "" {
				peer.disconnectReason.Store(&tt.disconnectReason)
			}

			require.Equal(t, tt.want, executionDialFailureReason(peer, tt.response))
		})
	}
}
