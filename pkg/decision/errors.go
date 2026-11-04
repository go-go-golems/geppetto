package decision

import (
	"errors"
	"fmt"
	"time"
)

// Sentinel error categories. Classify with errors.Is; wrap with safe context
// only (see the package safety contract).
var (
	// ErrInvalidRequest: the request is malformed, either caught locally or
	// rejected by the provider (400, 415).
	ErrInvalidRequest = errors.New("invalid decision request")
	// ErrInvalidResponse: the provider answered something unusable.
	ErrInvalidResponse = errors.New("invalid decision response")
	// ErrUnavailable: transport failure, timeout or 5xx.
	ErrUnavailable = errors.New("decision provider unavailable")
	// ErrRateLimited: the provider asked us to slow down (429).
	ErrRateLimited = errors.New("decision provider rate limited")
	// ErrNotGranted: missing or rejected credential, no credits, or decisions
	// disabled for the account (401, 402, 403).
	ErrNotGranted = errors.New("decision access not granted")
	// ErrModelNotFound: the provider does not know the model (404).
	ErrModelNotFound = errors.New("decision model not found")
	// ErrRequestTooLarge: the encoded request exceeds the configured limit,
	// or the provider rejected its size (413).
	ErrRequestTooLarge = errors.New("decision request too large")
	// ErrResponseTooLarge: the response body exceeds the configured limit.
	ErrResponseTooLarge = errors.New("decision response too large")
)

// MaxProviderMessage caps ProviderError.Message.
const MaxProviderMessage = 200

// ProviderError is a non-2xx provider response. It unwraps to one sentinel.
type ProviderError struct {
	Provider string
	Status   int
	// Code is the provider's error code, e.g. "systemone_invalid_request".
	Code string
	// Message is the provider's validation message, kept only for 400 and 404
	// responses and capped at MaxProviderMessage bytes; empty otherwise.
	Message   string
	RequestID string
	// RetryAfter is the provider's Retry-After hint (429, 503), or 0.
	RetryAfter time.Duration

	kind error
}

// NewProviderError classifies a non-2xx status. message is dropped unless
// the status is 400 or 404.
func NewProviderError(provider string, status int, code, message, requestID string, retryAfter time.Duration) *ProviderError {
	e := &ProviderError{Provider: provider, Status: status, Code: code, RequestID: requestID, RetryAfter: retryAfter, kind: KindForStatus(status)}
	if status == 400 || status == 404 {
		if len(message) > MaxProviderMessage {
			message = message[:MaxProviderMessage] + "…"
		}
		e.Message = message
	}
	return e
}

// KindForStatus maps an HTTP status to a sentinel.
func KindForStatus(status int) error {
	switch status {
	case 400, 415:
		return ErrInvalidRequest
	case 401, 402, 403:
		return ErrNotGranted
	case 404:
		return ErrModelNotFound
	case 413:
		return ErrRequestTooLarge
	case 429:
		return ErrRateLimited
	default:
		return ErrUnavailable
	}
}

func (e *ProviderError) Error() string {
	s := fmt.Sprintf("%s returned %d", e.Provider, e.Status)
	if e.Code != "" {
		s += " " + e.Code
	}
	if e.Message != "" {
		s += ": " + e.Message
	}
	if e.RetryAfter > 0 {
		s += fmt.Sprintf(" (retry after %s)", e.RetryAfter)
	}
	return s + ": " + e.kind.Error()
}

func (e *ProviderError) Unwrap() error { return e.kind }
