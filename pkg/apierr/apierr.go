// Package apierr is the error contract: one response shape for every error,
// and the table that maps domain errors onto it.
package apierr

import (
	"errors"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
)

// Code is the machine-readable error code a client switches on.
type Code string

// Error codes. A client never parses Message; it reads Code.
const (
	CodeValidation   Code = "VALIDATION"
	CodeUnauthorized Code = "UNAUTHORIZED"
	CodeForbidden    Code = "FORBIDDEN"
	CodeNotFound     Code = "NOT_FOUND"
	CodeConflict     Code = "CONFLICT"
	CodeInternal     Code = "INTERNAL"
)

// HumaError is the one error body the API ever returns.
type HumaError struct {
	Code    Code     `json:"code" doc:"Machine-readable error code"`
	Message string   `json:"message" doc:"Human-readable error message"`
	Errors  []string `json:"errors,omitempty" doc:"Individual problems, when more than one applies"`
	status  int
}

func (e *HumaError) Error() string  { return e.Message }
func (e *HumaError) GetStatus() int { return e.status }

// New builds a HumaError.
func New(status int, code Code, message string) *HumaError {
	return &HumaError{Code: code, Message: message, status: status}
}

// Huma builds its own errors (bad JSON, a failed schema tag) through
// huma.NewError. Replacing it makes those carry the same shape as ours, so a
// client parses one body whatever rejected the request.
func init() {
	huma.NewError = func(status int, msg string, errs ...error) huma.StatusError {
		out := &HumaError{Code: codeFor(status), Message: msg, status: status}
		for _, e := range errs {
			if e != nil {
				out.Errors = append(out.Errors, e.Error())
			}
		}
		return out
	}
}

func codeFor(status int) Code {
	switch status {
	case http.StatusBadRequest, http.StatusUnprocessableEntity:
		return CodeValidation
	case http.StatusUnauthorized:
		return CodeUnauthorized
	case http.StatusForbidden:
		return CodeForbidden
	case http.StatusNotFound:
		return CodeNotFound
	case http.StatusConflict:
		return CodeConflict
	default:
		return CodeInternal
	}
}

// Mapper turns domain errors into HumaErrors. Each domain registers the errors
// it can return; anything unregistered becomes a 500.
type Mapper struct {
	mappings map[error]*HumaError
}

// NewMapper returns an empty Mapper.
func NewMapper() *Mapper { return &Mapper{mappings: map[error]*HumaError{}} }

// Register maps a domain error to a response.
func (m *Mapper) Register(domainErr error, status int, code Code) {
	m.mappings[domainErr] = New(status, code, domainErr.Error())
}

// Map returns the response for err, matching wrapped errors too, or a 500 when
// nothing was registered for it.
func (m *Mapper) Map(err error) *HumaError {
	var he *HumaError
	if errors.As(err, &he) {
		return he
	}
	for domainErr, resp := range m.mappings {
		if errors.Is(err, domainErr) {
			out := *resp
			return &out
		}
	}
	return New(http.StatusInternalServerError, CodeInternal, "internal error")
}
