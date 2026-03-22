package errors

import "fmt"

// Code represents a typed error code.
type Code string

const (
	CodeTenantNotFound   Code = "TENANT_NOT_FOUND"
	CodeAgentNotFound    Code = "AGENT_NOT_FOUND"
	CodeAgentQuarantined Code = "AGENT_QUARANTINED"
	CodeTaskNotFound     Code = "TASK_NOT_FOUND"
	CodeUnauthorized     Code = "UNAUTHORIZED"
	CodeForbidden        Code = "FORBIDDEN"
	CodeValidation       Code = "VALIDATION_ERROR"
	CodeInternal         Code = "INTERNAL_ERROR"
	CodeCertExpired      Code = "CERT_EXPIRED"
	CodeCertRevoked      Code = "CERT_REVOKED"
	CodeRateLimited      Code = "RATE_LIMITED"
	CodeTierViolation    Code = "TIER_VIOLATION"
)

// Error is a typed error with a code and message.
type Error struct {
	Code    Code   `json:"code"`
	Message string `json:"message"`
	Err     error  `json:"-"`
}

func (e *Error) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *Error) Unwrap() error {
	return e.Err
}

// New creates a new typed error.
func New(code Code, message string) *Error {
	return &Error{Code: code, Message: message}
}

// Wrap wraps an existing error with a code.
func Wrap(code Code, message string, err error) *Error {
	return &Error{Code: code, Message: message, Err: err}
}

// Is checks if an error has a specific code.
func Is(err error, code Code) bool {
	e, ok := err.(*Error)
	return ok && e.Code == code
}
