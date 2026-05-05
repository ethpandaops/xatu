package clickhouse

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAdaptiveConcurrencyLimiter_DisabledReturnsNil(t *testing.T) {
	limiter := newAdaptiveConcurrencyLimiter(AdaptiveLimiterConfig{
		Enabled: false,
	})

	assert.Nil(t, limiter)
}

func TestNewAdaptiveConcurrencyLimiter_EnabledReturnsLimiter(t *testing.T) {
	limiter := newAdaptiveConcurrencyLimiter(AdaptiveLimiterConfig{
		Enabled:                     true,
		MinLimit:                    1,
		MaxLimit:                    50,
		InitialLimit:                8,
		QueueInitialRejectionFactor: 2,
		QueueMaxRejectionFactor:     3,
	})

	require.NotNil(t, limiter)
	assert.Equal(t, 8, limiter.Limit())
	assert.Equal(t, 0, limiter.Inflight())
	assert.Equal(t, 0, limiter.Queued())
}

func TestAdaptiveConcurrencyLimiter_AcquireAndRecord(t *testing.T) {
	limiter := newAdaptiveConcurrencyLimiter(AdaptiveLimiterConfig{
		Enabled:      true,
		MinLimit:     1,
		MaxLimit:     50,
		InitialLimit: 8,
	})

	require.NotNil(t, limiter)

	err := limiter.doWithLimiter(context.Background(), func(_ context.Context) error {
		// While in-flight, inflight should be >= 1.
		assert.GreaterOrEqual(t, limiter.Inflight(), 1)

		return nil
	})

	require.NoError(t, err)
}

func TestAdaptiveConcurrencyLimiter_AcquireAndDropOnError(t *testing.T) {
	limiter := newAdaptiveConcurrencyLimiter(AdaptiveLimiterConfig{
		Enabled:      true,
		MinLimit:     1,
		MaxLimit:     50,
		InitialLimit: 8,
	})

	require.NotNil(t, limiter)

	sentinel := errors.New("simulated failure")

	err := limiter.doWithLimiter(context.Background(), func(_ context.Context) error {
		return sentinel
	})

	require.ErrorIs(t, err, sentinel)
	// After drop, inflight should return to 0.
	assert.Equal(t, 0, limiter.Inflight())
}

func TestAdaptiveConcurrencyLimiter_BlocksWhenFullAndCancels(t *testing.T) {
	// Create a limiter with limit=1 so we can saturate it.
	limiter := newAdaptiveConcurrencyLimiter(AdaptiveLimiterConfig{
		Enabled:      true,
		MinLimit:     1,
		MaxLimit:     1,
		InitialLimit: 1,
	})

	require.NotNil(t, limiter)

	// Acquire the single permit.
	permit, ok := limiter.limiter.TryAcquirePermit()
	require.True(t, ok, "should acquire the single permit")

	// Second attempt should block and then fail when context is cancelled.
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err := limiter.doWithLimiter(ctx, func(_ context.Context) error {
		return nil
	})

	require.Error(t, err)
	assert.True(t, IsLimiterRejected(err))

	// Release and verify we can acquire again.
	permit.Record()

	err = limiter.doWithLimiter(context.Background(), func(_ context.Context) error {
		return nil
	})

	require.NoError(t, err)
}
