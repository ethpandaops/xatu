package relay

import (
	"errors"
	"fmt"
	"net/http"
)

// Sentinel errors for specific conditions.
var (
	ErrValidatorNotRegistered = errors.New("validator not registered")
	ErrRateLimited            = errors.New("rate limited")
)

// RelayError represents an error from the relay API with status code information.
type RelayError struct {
	StatusCode int
	Message    string
	Endpoint   string
}

func (e *RelayError) Error() string {
	return fmt.Sprintf("relay API error on %s: %s (status: %d)", e.Endpoint, e.Message, e.StatusCode)
}

// IsRetryable returns true if the error is likely transient and the request should be retried.
func (e *RelayError) IsRetryable() bool {
	// 5xx errors are server-side and typically transient
	if e.StatusCode >= 500 && e.StatusCode < 600 {
		return true
	}

	// 429 is rate limiting - should retry with backoff
	if e.StatusCode == http.StatusTooManyRequests {
		return true
	}

	// 408 Request Timeout is retryable
	if e.StatusCode == http.StatusRequestTimeout {
		return true
	}

	return false
}

// IsRateLimited returns true if the error is due to rate limiting.
func (e *RelayError) IsRateLimited() bool {
	return e.StatusCode == http.StatusTooManyRequests
}

// IsClientError returns true if the error is a 4xx client error (excluding rate limiting).
func (e *RelayError) IsClientError() bool {
	return e.StatusCode >= 400 && e.StatusCode < 500 && e.StatusCode != http.StatusTooManyRequests
}

// IsServerError returns true if the error is a 5xx server error.
func (e *RelayError) IsServerError() bool {
	return e.StatusCode >= 500 && e.StatusCode < 600
}

// NewRelayError creates a new RelayError.
func NewRelayError(statusCode int, message, endpoint string) *RelayError {
	return &RelayError{
		StatusCode: statusCode,
		Message:    message,
		Endpoint:   endpoint,
	}
}

// IsRelayError checks if an error is a RelayError and returns it.
func IsRelayError(err error) (*RelayError, bool) {
	var relayErr *RelayError
	if errors.As(err, &relayErr) {
		return relayErr, true
	}

	return nil, false
}

// IsRetryableError checks if an error is retryable (either a retryable RelayError or a network error).
func IsRetryableError(err error) bool {
	if err == nil {
		return false
	}

	// Check if it's a RelayError
	if relayErr, ok := IsRelayError(err); ok {
		return relayErr.IsRetryable()
	}

	// Network errors are generally retryable
	// This catches things like connection refused, timeout, etc.
	return true
}
