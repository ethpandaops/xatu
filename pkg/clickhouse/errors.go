package clickhouse

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"syscall"

	"github.com/ClickHouse/ch-go"
	"github.com/ClickHouse/ch-go/compress"
	"github.com/ClickHouse/ch-go/proto"
)

type tableWriteError struct {
	table string
	cause error
}

func (e *tableWriteError) Error() string {
	return fmt.Sprintf("table %s write failed: %v", e.table, e.cause)
}

func (e *tableWriteError) Unwrap() error {
	return e.cause
}

// inputPrepError wraps errors from batch factory initialization.
// These are always permanent — retrying won't resolve them.
type inputPrepError struct {
	cause error
}

func (e *inputPrepError) Error() string { return e.cause.Error() }
func (e *inputPrepError) Unwrap() error { return e.cause }

// flattenError wraps FlattenTo failures. Classified as permanent because
// corrupt data cannot fix itself on retry.
type flattenError struct {
	cause error
}

func (e *flattenError) Error() string { return fmt.Sprintf("flatten failed: %v", e.cause) }
func (e *flattenError) Unwrap() error { return e.cause }

// limiterRejectedError indicates the adaptive concurrency limiter rejected
// the request. This is not retryable (doWithRetry exits immediately) and not
// permanent — the table writer simply waits for the next flush cycle.
type limiterRejectedError struct {
	cause error
}

func (e *limiterRejectedError) Error() string {
	return fmt.Sprintf("adaptive limiter rejected: %v", e.cause)
}

func (e *limiterRejectedError) Unwrap() error { return e.cause }

// IsLimiterRejected reports whether err was caused by adaptive concurrency
// limiter rejection.
func IsLimiterRejected(err error) bool {
	var lre *limiterRejectedError

	return errors.As(err, &lre)
}

// IsPermanentWriteError returns true for errors that will never succeed on
// retry: data-quality problems (bad values, parse failures) and code bugs
// (syntax errors, bad arguments). Schema mismatches (unknown table, missing
// column, column count) are intentionally excluded because they can be
// transient during rolling deployments when migrations haven't been applied
// yet — NAK + Kafka redelivery lets them self-resolve.
//
// For joined errors (from multi-table concurrent flushes), all constituent
// errors must be permanent for the result to be permanent. A single transient
// failure in any table means the group should be NAK'd for retry.
func IsPermanentWriteError(err error) bool {
	if err == nil {
		return false
	}

	// Direct type checks — avoid errors.As which traverses into joined
	// error children and would incorrectly match a single permanent child
	// inside a mixed joined error.
	switch e := err.(type) {
	case *inputPrepError:
		return true
	case *flattenError:
		return true
	case *ch.Exception:
		return isPermanentCHException(e)
	}

	// Multi-unwrap (errors.Join, ch.Exception chains): only permanent if
	// ALL non-nil constituents are individually permanent. A single
	// transient failure in any table means the group should be NAK'd.
	if joined, ok := err.(interface{ Unwrap() []error }); ok {
		errs := joined.Unwrap()

		hasConcrete := false

		for _, e := range errs {
			if e == nil {
				continue
			}

			hasConcrete = true

			if !IsPermanentWriteError(e) {
				return false
			}
		}

		return hasConcrete
	}

	// Single-unwrap: walk the error chain (e.g. tableWriteError wrapping
	// an inputPrepError or ch.Exception).
	if unwrapper, ok := err.(interface{ Unwrap() error }); ok {
		return IsPermanentWriteError(unwrapper.Unwrap())
	}

	return false
}

func isPermanentCHException(exc *ch.Exception) bool {
	return exc.IsCode(
		// Data-quality errors: the message payload itself is invalid,
		// retrying the same data will never succeed.
		proto.ErrCannotConvertType,
		proto.ErrCannotParseText,
		proto.ErrCannotParseNumber,
		proto.ErrCannotParseDate,
		proto.ErrCannotParseDatetime,
		proto.ErrCannotInsertNullInOrdinaryColumn,
		proto.ErrIncorrectData,
		proto.ErrValueIsOutOfRangeOfDataType,
		proto.ErrIllegalTypeOfArgument,
		// Code/config bugs: no amount of retry will fix these.
		proto.ErrUnknownSetting,
		proto.ErrBadArguments,
		proto.ErrSyntaxError,
	)
}

func isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	// Context cancellation/deadlines are terminal for this call path.
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}

	// Closed pool/client is terminal.
	if errors.Is(err, ch.ErrClosed) {
		return false
	}

	// Classify server-side exceptions by ClickHouse code.
	if exc, ok := ch.AsException(err); ok {
		return exc.IsCode(
			proto.ErrTimeoutExceeded,
			proto.ErrNoFreeConnection,
			proto.ErrTooManySimultaneousQueries,
			proto.ErrSocketTimeout,
			proto.ErrNetworkError,
		)
	}

	// Corrupted compression payloads should not be retried.
	var corruptedErr *compress.CorruptedDataErr
	if errors.As(err, &corruptedErr) {
		return false
	}

	// Common transient transport errors.
	if errors.Is(err, syscall.ECONNRESET) ||
		errors.Is(err, syscall.ECONNREFUSED) ||
		errors.Is(err, syscall.EPIPE) ||
		errors.Is(err, io.EOF) ||
		errors.Is(err, io.ErrUnexpectedEOF) {
		return true
	}

	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}

	errStr := strings.ToLower(err.Error())
	transientPatterns := []string{
		"connection reset by peer",
		"connection refused",
		"broken pipe",
		"unexpected eof",
		"read: eof",
		"write: eof",
		"i/o timeout",
		"operation timed out",
		"temporary failure",
		"server is overloaded",
		"too many connections",
	}

	for _, pattern := range transientPatterns {
		if strings.Contains(errStr, pattern) {
			return true
		}
	}

	return false
}
