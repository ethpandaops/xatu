package server

import (
	"fmt"
	"strings"
	"time"

	"google.golang.org/grpc/keepalive"
)

// KeepaliveParams represents the keepalive parameters for the gRPC server.
type KeepaliveParams struct {
	// Enabled is whether keepalive is enabled.
	Enabled *bool `yaml:"enabled" default:"true"`
	// MaxConnectionIdle is a duration for the amount of time after which an
	// idle connection would be closed by sending a GoAway. Idleness duration is
	// defined since the most recent time the number of outstanding RPCs became
	// zero or the connection establishment.
	MaxConnectionIdle *time.Duration `yaml:"maxConnectionIdle"`
	// MaxConnectionAge is a duration for the maximum amount of time a
	// connection may exist before it will be closed by sending a GoAway. A
	// random jitter of +/-10% will be added to MaxConnectionAge to spread out
	// connection storms.
	MaxConnectionAge *time.Duration `yaml:"maxConnectionAge"`
	// MaxConnectionAgeGrace is an additive period after MaxConnectionAge after
	// which the connection will be forcibly closed.
	MaxConnectionAgeGrace *time.Duration `yaml:"maxConnectionAgeGrace"`
	// After a duration of this time if the server doesn't see any activity it
	// pings the client to see if the transport is still alive.
	// If set below 1s, a minimum value of 1s will be used instead.
	Time *time.Duration `yaml:"time"`
	// After having pinged for keepalive check, the server waits for a duration
	// of Timeout and if no activity is seen even after that the connection is
	// closed.
	Timeout *time.Duration `yaml:"timeout"`
}

// Validate checks if the keepalive parameters are valid.
func (k *KeepaliveParams) Validate() error {
	if k == nil {
		return nil
	}

	if k.Time != nil && *k.Time < time.Second {
		return fmt.Errorf("time must be at least 1 second, got %v", *k.Time)
	}

	return nil
}

// ToGRPCKeepaliveParams converts the keepalive parameters to gRPC keepalive parameters.
func (k *KeepaliveParams) ToGRPCKeepaliveParams() (keepalive.ServerParameters, error) {
	params := keepalive.ServerParameters{}

	if k == nil {
		return params, nil
	}

	if k.MaxConnectionIdle != nil {
		params.MaxConnectionIdle = *k.MaxConnectionIdle
	}

	if k.MaxConnectionAge != nil {
		params.MaxConnectionAge = *k.MaxConnectionAge
	}

	if k.MaxConnectionAgeGrace != nil {
		params.MaxConnectionAgeGrace = *k.MaxConnectionAgeGrace
	}

	if k.Time != nil {
		params.Time = *k.Time
	}

	if k.Timeout != nil {
		params.Timeout = *k.Timeout
	}

	return params, nil
}

func (k *KeepaliveParams) String() string {
	params := []string{}

	if k.Time != nil {
		params = append(params, fmt.Sprintf("time: %s", *k.Time))
	}

	if k.Timeout != nil {
		params = append(params, fmt.Sprintf("timeout: %s", *k.Timeout))
	}

	if k.MaxConnectionIdle != nil {
		params = append(params, fmt.Sprintf("max_connection_idle: %s", *k.MaxConnectionIdle))
	}

	if k.MaxConnectionAge != nil {
		params = append(params, fmt.Sprintf("max_connection_age: %s", *k.MaxConnectionAge))
	}

	if k.MaxConnectionAgeGrace != nil {
		params = append(params, fmt.Sprintf("max_connection_age_grace: %s", *k.MaxConnectionAgeGrace))
	}

	return strings.Join(params, ", ")
}
