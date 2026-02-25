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

// flattenError wraps FlattenTo failures. Not matched by IsPermanentWriteError,
// so the source nack path retries via Kafka redelivery.
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
// retry: schema mismatches, type errors, conversion failures.
func IsPermanentWriteError(err error) bool {
	if err == nil {
		return false
	}

	var prepErr *inputPrepError
	if errors.As(err, &prepErr) {
		return true
	}

	if exc, ok := ch.AsException(err); ok {
		return exc.IsCode(
			proto.ErrUnknownTable,
			proto.ErrUnknownDatabase,
			proto.ErrNoSuchColumnInTable,
			proto.ErrThereIsNoColumn,
			proto.ErrUnknownIdentifier,
			proto.ErrTypeMismatch,
			proto.ErrCannotConvertType,
			proto.ErrCannotParseText,
			proto.ErrCannotParseNumber,
			proto.ErrCannotParseDate,
			proto.ErrCannotParseDatetime,
			proto.ErrCannotInsertNullInOrdinaryColumn,
			proto.ErrIncorrectData,
			proto.ErrValueIsOutOfRangeOfDataType,
			proto.ErrIncorrectNumberOfColumns,
			proto.ErrNumberOfColumnsDoesntMatch,
			proto.ErrIllegalColumn,
			proto.ErrIllegalTypeOfArgument,
			proto.ErrUnknownSetting,
			proto.ErrBadArguments,
			proto.ErrSyntaxError,
		)
	}

	return false
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
		"connection reset",
		"connection refused",
		"broken pipe",
		"eof",
		"timeout",
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
