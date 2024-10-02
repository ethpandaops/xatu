package coordinator

import (
	"context"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func (c *Client) validateAuth(ctx context.Context, md metadata.MD) error {
	if c.config.Auth.Enabled == nil || !*c.config.Auth.Enabled {
		return nil
	}

	authHeader := md.Get("Authorization")

	if len(authHeader) == 0 {
		return status.Errorf(codes.Unauthenticated, "missing authorization header")
	}

	token := authHeader[0]
	if token == "" {
		return status.Errorf(codes.Unauthenticated, "empty authorization header")
	}

	// Extract the secret from the token
	split := strings.Split(token, " ")
	if len(split) != 2 {
		return status.Errorf(codes.Unauthenticated, "invalid authorization header")
	}

	secret := split[1]

	if secret != c.config.Auth.Secret {
		return status.Errorf(codes.Unauthenticated, "invalid secret")
	}

	return nil
}
