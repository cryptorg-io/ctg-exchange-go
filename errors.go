package ctgexchange

import (
	"errors"
	"fmt"
)

// ErrMissingCredentials is returned when a private endpoint is called
// without an API key and secret configured.
var ErrMissingCredentials = errors.New(
	"ctgexchange: api key and secret are required for private endpoints")

// APIError is a non-2xx HTTP response from the API. Inspect StatusCode,
// or use the Is* helpers, to branch on the kind of failure:
//
//	var apiErr *ctgexchange.APIError
//	if errors.As(err, &apiErr) && apiErr.IsRateLimited() {
//	        time.Sleep(time.Duration(apiErr.RetryAfter) * time.Second)
//	}
type APIError struct {
	// StatusCode is the HTTP status of the response.
	StatusCode int
	// Code is the API's machine-readable "error" field.
	Code string
	// Message is the API's human-readable "message" field, if any.
	Message string
	// RequestID is the API's "request_id" — quote it when reporting.
	RequestID string
	// RetryAfter is the Retry-After header value in seconds on a 429,
	// or 0 when the server did not send one.
	RetryAfter int
}

func (e *APIError) Error() string {
	detail := e.Message
	if detail == "" {
		detail = e.Code
	}
	if detail == "" {
		detail = "request failed"
	}
	s := fmt.Sprintf("ctgexchange: [%d] %s", e.StatusCode, detail)
	if e.RequestID != "" {
		s += fmt.Sprintf(" (request_id=%s)", e.RequestID)
	}
	return s
}

// IsBadRequest reports whether the request was malformed (400).
func (e *APIError) IsBadRequest() bool { return e.StatusCode == 400 }

// IsUnauthorized reports a missing/expired/invalid API key (401).
func (e *APIError) IsUnauthorized() bool { return e.StatusCode == 401 }

// IsForbidden reports a wrong-scope key or off-allowlist IP (403).
func (e *APIError) IsForbidden() bool { return e.StatusCode == 403 }

// IsNotFound reports an unknown symbol or order (404).
func (e *APIError) IsNotFound() bool { return e.StatusCode == 404 }

// IsRateLimited reports a rate-limit response (429); see RetryAfter.
func (e *APIError) IsRateLimited() bool { return e.StatusCode == 429 }
