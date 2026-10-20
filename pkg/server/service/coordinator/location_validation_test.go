package coordinator

import (
	"context"
	"errors"
	"testing"

	"github.com/creasty/defaults"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

// UpsertCannonLocation used to dereference req.Location unconditionally, so a
// request with the field omitted crashed the whole server. It must now
// reject a nil Location with InvalidArgument instead.
func TestUpsertCannonLocation_NilLocation_ReturnsInvalidArgument(t *testing.T) {
	cfg := &Config{}
	if err := defaults.Set(cfg); err != nil {
		t.Fatalf("defaults.Set: %v", err)
	}

	// persistence, geoipProvider, nodeRecord, metrics and healthServer are
	// intentionally left nil: the nil-Location check should reject the
	// request before any of them are touched.
	c := &Client{
		config: cfg,
	}

	req := &xatu.UpsertCannonLocationRequest{}

	resp, err := c.UpsertCannonLocation(context.Background(), req)

	if resp != nil {
		t.Fatalf("expected a nil response, got %v", resp)
	}

	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected codes.InvalidArgument, got %v (%v)", status.Code(err), err)
	}
}

// The nil-Location check must apply even when auth is enabled and the
// caller presents a valid secret. The auth gate runs first, so it should
// not end up bypassing the validation for authenticated callers.
func TestUpsertCannonLocation_NilLocation_ReturnsInvalidArgument_EvenWhenAuthenticated(t *testing.T) {
	enabled := true
	cfg := &Config{
		Auth: AuthConfig{
			Enabled: &enabled,
			Secret:  "correct-horse-battery-staple",
		},
	}

	c := &Client{config: cfg}

	md := metadata.New(map[string]string{"Authorization": "Bearer correct-horse-battery-staple"})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	resp, err := c.UpsertCannonLocation(ctx, &xatu.UpsertCannonLocationRequest{})

	if resp != nil {
		t.Fatalf("expected a nil response, got %v", resp)
	}

	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected codes.InvalidArgument, got %v (%v)", status.Code(err), err)
	}
}

// UpsertRelayMonitorLocation had the same bug and gets the same fix.
func TestUpsertRelayMonitorLocation_NilLocation_ReturnsInvalidArgument(t *testing.T) {
	cfg := &Config{}
	if err := defaults.Set(cfg); err != nil {
		t.Fatalf("defaults.Set: %v", err)
	}

	c := &Client{config: cfg}

	resp, err := c.UpsertRelayMonitorLocation(context.Background(), &xatu.UpsertRelayMonitorLocationRequest{})

	if resp != nil {
		t.Fatalf("expected a nil response, got %v", resp)
	}

	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected codes.InvalidArgument, got %v (%v)", status.Code(err), err)
	}
}

// Confirms the error comes from a genuine early return and not from a
// recovered panic further down the call chain.
func TestUpsertCannonLocation_NilLocation_IsNotARecoveredPanic(t *testing.T) {
	cfg := &Config{}
	_ = defaults.Set(cfg)

	c := &Client{config: cfg}

	_, err := c.UpsertCannonLocation(context.Background(), &xatu.UpsertCannonLocationRequest{})

	var runtimeErr interface{ RuntimeError() }
	if errors.As(err, &runtimeErr) {
		t.Fatalf("error chain contains a runtime.Error, expected a plain validation error: %v", err)
	}
}
