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

// DefaultErrorClassifier classifies ClickHouse write errors for source retry
// and reject handling.
type DefaultErrorClassifier struct{}

// IsPermanent returns true if the error is a permanent write error.
func (DefaultErrorClassifier) IsPermanent(err error) bool {
	return IsPermanentWriteError(err)
}

// Table extracts the table name from a write error.
func (DefaultErrorClassifier) Table(err error) string {
	return WriteErrorTable(err)
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

// WriteErrorTable extracts the table name from a write error.
func WriteErrorTable(err error) string {
	if err == nil {
		return ""
	}

	var tableErr *tableWriteError
	if errors.As(err, &tableErr) {
		return tableErr.table
	}

	return ""
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
