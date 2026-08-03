// Package errors provides a framework-agnostic application error type
// that carries an HTTP-independent code, letting the presentation layer
// decide how to translate it into a transport-specific response.
package errors

import "fmt"

type Code string

const (
	CodeNotFound     Code = "NOT_FOUND"
	CodeInvalidInput Code = "INVALID_INPUT"
	CodeConflict     Code = "CONFLICT"
	CodeUnauthorized Code = "UNAUTHORIZED"
	CodeForbidden    Code = "FORBIDDEN"
	CodeInternal     Code = "INTERNAL"
)

// AppError is the standard error type returned by domain and application layers.
type AppError struct {
	Code    Code
	Message string
	Err     error
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *AppError) Unwrap() error {
	return e.Err
}

func New(code Code, message string) *AppError {
	return &AppError{Code: code, Message: message}
}

func Wrap(code Code, message string, err error) *AppError {
	return &AppError{Code: code, Message: message, Err: err}
}

func NotFound(message string) *AppError     { return New(CodeNotFound, message) }
func InvalidInput(message string) *AppError { return New(CodeInvalidInput, message) }
func Conflict(message string) *AppError     { return New(CodeConflict, message) }
func Internal(err error) *AppError          { return Wrap(CodeInternal, "internal error", err) }

// Unavailable signals a dependency (e.g. database) is not configured/reachable.
// Presentation layers without a dedicated HTTP status for it should map
// CodeUnavailable to 503, falling back to 500 otherwise.
const CodeUnavailable Code = "UNAVAILABLE"

func Unavailable(message string) *AppError { return New(CodeUnavailable, message) }
