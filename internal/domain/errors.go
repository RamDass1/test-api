package domain

import (
	"errors"
	"fmt"
)

type Code string

const (
	CodeValidation   Code = "validation_error"
	CodeUnauthorized Code = "unauthorized"
	CodeForbidden    Code = "forbidden"
	CodeNotFound     Code = "not_found"
	CodeConflict     Code = "conflict"
	CodeRateLimited  Code = "rate_limited"
	CodeInternal     Code = "internal_error"
)

var (
	ErrNotFound        = errors.New("not found")
	ErrAlreadyExists   = errors.New("already exists")
	ErrUnknownID       = errors.New("unknown id")
	ErrVersionConflict = errors.New("version conflict")
)

type Error struct {
	Code    Code
	Message string
	Err     error
}

func (e *Error) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *Error) Unwrap() error { return e.Err }

func Invalid(format string, args ...any) *Error {
	return &Error{Code: CodeValidation, Message: fmt.Sprintf(format, args...)}
}

func Unauthorized(format string, args ...any) *Error {
	return &Error{Code: CodeUnauthorized, Message: fmt.Sprintf(format, args...)}
}

func Forbidden(format string, args ...any) *Error {
	return &Error{Code: CodeForbidden, Message: fmt.Sprintf(format, args...)}
}

func NotFound(format string, args ...any) *Error {
	return &Error{Code: CodeNotFound, Message: fmt.Sprintf(format, args...)}
}

func Conflict(format string, args ...any) *Error {
	return &Error{Code: CodeConflict, Message: fmt.Sprintf(format, args...)}
}

func RateLimited(format string, args ...any) *Error {
	return &Error{Code: CodeRateLimited, Message: fmt.Sprintf(format, args...)}
}

func Internal(err error, format string, args ...any) *Error {
	return &Error{Code: CodeInternal, Message: fmt.Sprintf(format, args...), Err: err}
}
