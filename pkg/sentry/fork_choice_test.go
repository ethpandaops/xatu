package sentry

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	eth2client "github.com/ethpandaops/go-eth2-client"
	"github.com/ethpandaops/go-eth2-client/api"
	eth2v1 "github.com/ethpandaops/go-eth2-client/api/v1"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsUnsupportedEndpoint(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "lighthouse rejects the version", err: &api.Error{StatusCode: http.StatusBadRequest}, want: true},
		{name: "nimbus has no route", err: &api.Error{StatusCode: http.StatusNotFound}, want: true},
		{name: "method not allowed", err: &api.Error{StatusCode: http.StatusMethodNotAllowed}, want: true},
		{name: "not implemented", err: &api.Error{StatusCode: http.StatusNotImplemented}, want: true},
		{name: "wrapped", err: fmt.Errorf("fetch: %w", &api.Error{StatusCode: http.StatusNotFound}), want: true},
		{name: "server error", err: &api.Error{StatusCode: http.StatusInternalServerError}, want: false},
		{name: "not an api error", err: errors.New("connection refused"), want: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.want, isUnsupportedEndpoint(test.err))
		})
	}
}

// stubForkChoiceV2 is a ForkChoiceV2Provider returning fixed results.
type stubForkChoiceV2 struct {
	calls int
	data  *eth2v1.ForkChoiceV2
	err   error
}

func (s *stubForkChoiceV2) ForkChoiceV2(context.Context, *api.ForkChoiceOpts) (*api.Response[*eth2v1.ForkChoiceV2], error) {
	s.calls++
	if s.err != nil {
		return nil, s.err
	}

	return &api.Response[*eth2v1.ForkChoiceV2]{Data: s.data}, nil
}

func TestFetchForkChoiceWithFallback(t *testing.T) {
	v1ForkChoice := &eth2v1.ForkChoice{}
	v2ForkChoice := &eth2v1.ForkChoiceV2{}
	invalid := fmt.Errorf("%w: fork choice data missing", eth2client.ErrInvalidResponse)

	tests := []struct {
		name      string
		v2Err     error
		wantV2    bool
		wantRetry bool
	}{
		{name: "v2 served", wantV2: true},
		{name: "v2 unsupported", v2Err: &api.Error{StatusCode: http.StatusNotFound}, wantRetry: true},
		{name: "v2 response not following the spec", v2Err: invalid, wantRetry: true},
		{name: "v2 transient failure", v2Err: &api.Error{StatusCode: http.StatusServiceUnavailable}},
		{name: "v2 connection failure", v2Err: errors.New("connection refused")},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var retryAt atomic.Int64

			provider := &stubForkChoiceV2{data: v2ForkChoice, err: test.v2Err}
			v1Calls := 0
			fetchV1 := func(context.Context) (*eth2v1.ForkChoice, error) {
				v1Calls++

				return v1ForkChoice, nil
			}

			forkChoice, forkChoiceV2, err := fetchForkChoiceWithFallback(context.Background(), logrus.New(), provider, fetchV1, &retryAt)
			require.NoError(t, err)

			if test.wantV2 {
				assert.Same(t, v2ForkChoice, forkChoiceV2)
				assert.Nil(t, forkChoice)
				assert.Zero(t, v1Calls)
			} else {
				assert.Same(t, v1ForkChoice, forkChoice)
				assert.Nil(t, forkChoiceV2)
				assert.Equal(t, 1, v1Calls)
			}

			// Only a node that cannot serve v2 backs off; otherwise the next fetch tries v2 again.
			_, _, err = fetchForkChoiceWithFallback(context.Background(), logrus.New(), provider, fetchV1, &retryAt)
			require.NoError(t, err)

			wantCalls := 2

			if test.wantRetry {
				wantCalls = 1

				assert.Greater(t, retryAt.Load(), time.Now().UnixNano())
			}

			assert.Equal(t, wantCalls, provider.calls)
		})
	}
}

func TestFetchForkChoiceWithFallbackNoProvider(t *testing.T) {
	var retryAt atomic.Int64

	v1ForkChoice := &eth2v1.ForkChoice{}
	forkChoice, forkChoiceV2, err := fetchForkChoiceWithFallback(context.Background(), logrus.New(), nil,
		func(context.Context) (*eth2v1.ForkChoice, error) { return v1ForkChoice, nil }, &retryAt)
	require.NoError(t, err)
	assert.Same(t, v1ForkChoice, forkChoice)
	assert.Nil(t, forkChoiceV2)
}

// A v1 failure is returned, not masked by the fallback.
func TestFetchForkChoiceWithFallbackV1Error(t *testing.T) {
	var retryAt atomic.Int64

	v1Err := errors.New("v1 failed")
	_, _, err := fetchForkChoiceWithFallback(context.Background(), logrus.New(),
		&stubForkChoiceV2{err: &api.Error{StatusCode: http.StatusNotFound}},
		func(context.Context) (*eth2v1.ForkChoice, error) { return nil, v1Err }, &retryAt)
	require.ErrorIs(t, err, v1Err)
}

// A cancelled fetch neither backs off nor falls back to v1.
func TestFetchForkChoiceWithFallbackCancelled(t *testing.T) {
	var retryAt atomic.Int64

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	v1Calls := 0
	_, _, err := fetchForkChoiceWithFallback(ctx, logrus.New(), &stubForkChoiceV2{err: context.Canceled},
		func(context.Context) (*eth2v1.ForkChoice, error) {
			v1Calls++

			return &eth2v1.ForkChoice{}, nil
		}, &retryAt)
	require.ErrorIs(t, err, context.Canceled)
	assert.Zero(t, v1Calls)
	assert.Zero(t, retryAt.Load())
}
