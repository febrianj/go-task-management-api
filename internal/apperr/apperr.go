package apperr

import (
	"fmt"
	"net/http"
)

const (
	CodeInvalidRequest     = "INVALID_REQUEST"
	CodeNotFound           = "NOT_FOUND"
	CodeInternal           = "INTERNAL_ERROR"
	CodeValidation         = "VALIDATION_ERROR"
	CodeUnauthorized       = "UNAUTHORIZED"
	CodeInvalidCredentials = "INVALID_CREDENTIALS"
	CodeEmailAlreadyExists = "EMAIL_ALREADY_EXISTS"
)

// FieldError describes a single field-level validation failure
type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// AppError is the error type every layer return upward.
// Err holds internal cause and MUST NOT reach client
type AppError struct {
	HTTPStatus int
	Code       string
	Message    string
	Details    []FieldError
	Err        error
}

// Error implements [error].
func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// exposes internal cause so errors.Is and errors.As can walk the full chain
func (e *AppError) Unwrap() error {
	return e.Err
}

// attach internal error cause
func (e *AppError) WithErr(err error) *AppError {
	e.Err = err
	return e
}

func New(status int, code, msg string) *AppError {
	return &AppError{HTTPStatus: status, Code: code, Message: msg}
}

func BadRequest(msg string) *AppError {
	return New(http.StatusBadRequest, CodeInvalidRequest, msg)
}

func NotFound(code, msg string) *AppError {
	return New(http.StatusNotFound, code, msg)
}

func Validation(details []FieldError) *AppError {
	return &AppError{
		HTTPStatus: http.StatusUnprocessableEntity,
		Code:       CodeValidation,
		Message:    "validation failed",
		Details:    details,
	}
}

func Unauthorized(code, msg string) *AppError {
	return New(http.StatusUnauthorized, code, msg)
}

func Conflict(code, msg string) *AppError {
	return New(http.StatusConflict, code, msg)
}

func Internal(err error) *AppError {
	return &AppError{
		HTTPStatus: http.StatusInternalServerError,
		Code:       CodeInternal,
		Message:    "internal server error",
		Err:        err,
	}
}
