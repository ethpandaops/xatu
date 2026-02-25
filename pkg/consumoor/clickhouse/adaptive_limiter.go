package clickhouse

import (
	"context"

	"github.com/failsafe-go/failsafe-go/adaptivelimiter"
)

// adaptiveConcurrencyLimiter wraps failsafe-go's AdaptiveLimiter to provide
// per-table AIMD-based concurrency control around ClickHouse INSERTs.
type adaptiveConcurrencyLimiter struct {
	limiter adaptivelimiter.AdaptiveLimiter[any]
}

// newAdaptiveConcurrencyLimiter creates a limiter from config. Returns nil
// when disabled, which callers treat as a no-op passthrough.
func newAdaptiveConcurrencyLimiter(
	cfg AdaptiveLimiterConfig,
) *adaptiveConcurrencyLimiter {
	if !cfg.Enabled {
		return nil
	}

	builder := adaptivelimiter.NewBuilder[any]().
		WithLimits(cfg.MinLimit, cfg.MaxLimit, cfg.InitialLimit)

	if cfg.QueueInitialRejectionFactor > 0 && cfg.QueueMaxRejectionFactor > 0 {
		builder = builder.WithQueueing(
			cfg.QueueInitialRejectionFactor,
			cfg.QueueMaxRejectionFactor,
		)
	}

	return &adaptiveConcurrencyLimiter{
		limiter: builder.Build(),
	}
}

// Limit returns the current concurrency limit.
func (l *adaptiveConcurrencyLimiter) Limit() int {
	return l.limiter.Limit()
}

// Inflight returns the number of in-flight permits.
func (l *adaptiveConcurrencyLimiter) Inflight() int {
	return l.limiter.Inflight()
}

// Queued returns the number of queued permit requests.
func (l *adaptiveConcurrencyLimiter) Queued() int {
	return l.limiter.Queued()
}

// doWithLimiter acquires a permit, executes fn, and records the outcome.
// On success the permit is recorded (RTT measured); on failure the permit
// is dropped (signals the limiter to decrease concurrency).
// Returns limiterRejectedError if the limiter rejects the request.
func (l *adaptiveConcurrencyLimiter) doWithLimiter(
	ctx context.Context,
	fn func(context.Context) error,
) error {
	permit, ok := l.limiter.TryAcquirePermit()
	if !ok {
		return &limiterRejectedError{
			cause: adaptivelimiter.ErrExceeded,
		}
	}

	err := fn(ctx)
	if err != nil {
		permit.Drop()

		return err
	}

	permit.Record()

	return nil
}
