package mpesa

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

// Sentinel errors that callers can test with errors.Is.
var (
	// ErrMissingCredentials is returned when New is called without an app key
	// or app secret.
	ErrMissingCredentials = errors.New("mpesa: app key and app secret are required")

	// ErrInvalidEnvironment is returned by New when the Env is not one of the
	// supported values.
	ErrInvalidEnvironment = errors.New("mpesa: invalid environment")

	// ErrSandboxOnly is returned when an API that Safaricom only exposes in the
	// sandbox (C2B simulation) is called against production.
	ErrSandboxOnly = errors.New("mpesa: endpoint is only available in the sandbox environment")

	// ErrEmptyResponse is returned when the API replies with an empty body where
	// JSON was expected.
	ErrEmptyResponse = errors.New("mpesa: empty response body")
)

// ValidationError reports a request that failed client side validation, before
// any network call was made.
type ValidationError struct {
	// Field is the request field that is invalid.
	Field string
	// Reason explains what is wrong with the field.
	Reason string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("mpesa: invalid %s: %s", e.Field, e.Reason)
}

// APIError is returned when Daraja replies with a non 2xx HTTP status code.
type APIError struct {
	// StatusCode is the HTTP status code of the response.
	StatusCode int
	// RequestID is the Safaricom request id, when the response contains one.
	RequestID string
	// ErrorCode is the machine readable Daraja error code, when present.
	ErrorCode string
	// ErrorMessage is the human readable error message.
	ErrorMessage string
	// Body is the raw response body.
	Body []byte
}

func (e *APIError) Error() string {
	var b strings.Builder
	fmt.Fprintf(&b, "mpesa: api error (http %d", e.StatusCode)
	if e.ErrorCode != "" {
		fmt.Fprintf(&b, ", code %s", e.ErrorCode)
	}
	b.WriteString(")")
	if e.ErrorMessage != "" {
		fmt.Fprintf(&b, ": %s", e.ErrorMessage)
	} else if len(e.Body) > 0 && !looksLikeJSON(e.Body) {
		fmt.Fprintf(&b, ": %s", truncate(string(e.Body), 200))
	}
	if e.RequestID != "" {
		fmt.Fprintf(&b, " [request id: %s]", e.RequestID)
	}
	return b.String()
}

// Retryable reports whether the request may be retried safely. Only responses
// that indicate the request was not processed (rate limiting and gateway
// errors) are considered retryable; payment requests must never be replayed on
// ambiguous failures such as timeouts.
func (e *APIError) Retryable() bool {
	switch e.StatusCode {
	case http.StatusTooManyRequests,
		http.StatusBadGateway,
		http.StatusServiceUnavailable,
		http.StatusGatewayTimeout:
		return true
	}
	return false
}

// AsAPIError extracts an *APIError from err.
func AsAPIError(err error) (*APIError, bool) {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr, true
	}
	return nil, false
}

// IsNotFound reports whether err is an APIError with a 404 status code. Daraja
// answers 404 for expired or unknown resources (and for bad credentials).
func IsNotFound(err error) bool {
	apiErr, ok := AsAPIError(err)
	return ok && apiErr.StatusCode == http.StatusNotFound
}

// newAPIError builds an APIError from a response, using Daraja's error envelope
// when the body is JSON.
func newAPIError(status int, body []byte) *APIError {
	e := &APIError{StatusCode: status, Body: body}

	var payload struct {
		RequestID    string `json:"requestId"`
		ErrorCode    string `json:"errorCode"`
		ErrorMessage string `json:"errorMessage"`
		ResponseCode string `json:"ResponseCode"`
		ResponseDesc string `json:"ResponseDescription"`
		ResultDesc   string `json:"ResultDesc"`
	}
	if len(body) > 0 && json.Unmarshal(body, &payload) == nil {
		e.RequestID = payload.RequestID
		e.ErrorCode = payload.ErrorCode
		e.ErrorMessage = payload.ErrorMessage
		if e.ErrorMessage == "" {
			e.ErrorMessage = payload.ResponseDesc
		}
		if e.ErrorMessage == "" {
			e.ErrorMessage = payload.ResultDesc
		}
		if e.ErrorCode == "" {
			e.ErrorCode = payload.ResponseCode
		}
	}
	if e.ErrorMessage == "" {
		e.ErrorMessage = http.StatusText(status)
	}
	return e
}

func looksLikeJSON(b []byte) bool {
	trimmed := strings.TrimSpace(string(b))
	return strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "[")
}

func truncate(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
